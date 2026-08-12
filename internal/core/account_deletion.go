package core

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func filterBy[T any](items []T, keep func(T) bool) []T {
	out := make([]T, 0, len(items))
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

func (s *Server) deleteOwnAccount(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct{ Password, Confirmation string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Confirmation != "DELETE" {
		jsonError(w, 400, "confirmation_required", "Type DELETE to confirm permanent deletion.")
		return
	}
	if !VerifyPassword(cu.User.PasswordHash, in.Password) {
		jsonError(w, 401, "invalid_credentials", "Password is incorrect.")
		return
	}
	if err := s.deleteAccount(cu.User.ID, cu.User, clientIP(r)); err != nil {
		jsonError(w, 400, "deletion_failed", err.Error())
		return
	}
	clearSessionCookie(w)
	jsonOut(w, 200, map[string]any{"ok": true, "message": "Account and active data deleted. Cancel any Apple or Google subscription separately."})
}

func (s *Server) deletionAttemptLocked(key string) bool {
	locked := false
	_ = s.Store.Read(func(db Database) error {
		if a, ok := db.LoginAttempts[key]; ok && a.LockedUntil != "" {
			if until, err := time.Parse(time.RFC3339, a.LockedUntil); err == nil && time.Now().Before(until) {
				locked = true
			}
		}
		return nil
	})
	return locked
}

func (s *Server) recordFailedDeletion(key, email, ip string) {
	_ = s.Store.Update(func(db *Database) error {
		now := time.Now()
		a := db.LoginAttempts[key]
		start, _ := time.Parse(time.RFC3339, a.WindowStart)
		if a.WindowStart == "" || now.Sub(start) > 15*time.Minute {
			a = LoginAttempt{Key: key, WindowStart: now.UTC().Format(time.RFC3339)}
		}
		a.Count++
		if a.Count >= 5 {
			a.LockedUntil = now.Add(15 * time.Minute).UTC().Format(time.RFC3339)
		}
		db.LoginAttempts[key] = a
		s.audit(db, nil, "account.delete_failed", email, ip, map[string]any{"attempt": a.Count})
		return nil
	})
}

func (s *Server) publicDeleteAccount(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password, Confirmation string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Confirmation != "DELETE" {
		jsonError(w, 400, "confirmation_required", "Type DELETE to confirm permanent deletion.")
		return
	}
	email := normalizeEmail(in.Email)
	ip := clientIP(r)
	attemptKey := "delete|" + ip + "|" + email
	if s.deletionAttemptLocked(attemptKey) {
		jsonError(w, 429, "deletion_locked", "Too many failed deletion attempts. Try again later or contact support.")
		return
	}
	var u User
	found := false
	_ = s.Store.Read(func(db Database) error {
		for _, candidate := range db.Users {
			if candidate.Email == email {
				u = candidate
				found = true
				break
			}
		}
		return nil
	})
	if !found || !u.Active || !VerifyPassword(u.PasswordHash, in.Password) {
		s.recordFailedDeletion(attemptKey, email, ip)
		jsonError(w, 401, "invalid_credentials", "Email or password is incorrect.")
		return
	}
	if err := s.deleteAccount(u.ID, u, ip); err != nil {
		jsonError(w, 400, "deletion_failed", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func (s *Server) deleteAccount(userID string, actor User, ip string) error {
	var paths []string
	err := s.Store.Update(func(db *Database) error {
		u, ok := db.Users[userID]
		if !ok {
			return errors.New("user not found")
		}
		if u.Role == "admin" {
			admins := 0
			for _, x := range db.Users {
				if x.Active && x.Role == "admin" {
					admins++
				}
			}
			if admins <= 1 {
				return errors.New("the sole administrator cannot self-delete; transfer administration or close the service with support")
			}
		}
		for _, p := range db.ProgressPhotos {
			if p.UserID == userID {
				paths = append(paths, p.EncryptedPath)
			}
		}
		now := nowISO()
		// Keep a minimal, non-profiled deletion audit record. Remove names and direct identifiers from all records tied to the deleted account.
		s.audit(db, &actor, "account.delete", userID, ip, map[string]any{"at": now})
		for i := range db.Audit {
			if db.Audit[i].ActorID == userID {
				db.Audit[i].ActorID = ""
				db.Audit[i].ActorName = "Deleted user"
				db.Audit[i].IP = ""
			}
			if db.Audit[i].Target == userID {
				db.Audit[i].Target = "deleted-user"
			}
		}
		delete(db.Users, userID)
		delete(db.Profiles, userID)
		delete(db.CoachPreferences, userID)
		delete(db.WeeklyPlans, userID)
		delete(db.ThemePreferences, userID)
		for token, session := range db.Sessions {
			if session.UserID == userID {
				delete(db.Sessions, token)
			}
		}
		for key := range db.LoginAttempts {
			if strings.Contains(key, u.Email) {
				delete(db.LoginAttempts, key)
			}
		}
		for id, x := range db.Workouts {
			if x.OwnerID == userID {
				delete(db.Workouts, id)
			}
		}
		removedProfiles := map[string]bool{}
		for id, x := range db.CustomCoachProfiles {
			if x.AddedBy == userID {
				removedProfiles[id] = true
				delete(db.CustomCoachProfiles, id)
			}
		}
		db.CoachSources = filterBy(db.CoachSources, func(x CoachSource) bool { return !removedProfiles[x.ProfileID] })
		db.WorkoutLogs = filterBy(db.WorkoutLogs, func(x WorkoutLog) bool { return x.UserID != userID })
		db.Nutrition = filterBy(db.Nutrition, func(x NutritionEntry) bool { return x.UserID != userID })
		db.Habits = filterBy(db.Habits, func(x Habit) bool { return x.UserID != userID })
		db.HabitLogs = filterBy(db.HabitLogs, func(x HabitLog) bool { return x.UserID != userID })
		db.Progress = filterBy(db.Progress, func(x ProgressEntry) bool { return x.UserID != userID })
		db.CheckIns = filterBy(db.CheckIns, func(x CheckIn) bool { return x.UserID != userID })
		db.ChatMessages = filterBy(db.ChatMessages, func(x ChatMessage) bool { return x.UserID != userID })
		db.AIUsage = filterBy(db.AIUsage, func(x AIUsage) bool { return x.UserID != userID })
		db.PainFlags = filterBy(db.PainFlags, func(x PainFlag) bool { return x.UserID != userID })
		db.ProgressPhotos = filterBy(db.ProgressPhotos, func(x ProgressPhoto) bool { return x.UserID != userID })
		db.MealPlans = filterBy(db.MealPlans, func(x MealPlan) bool { return x.UserID != userID })
		db.WearableConnections = filterBy(db.WearableConnections, func(x WearableConnection) bool { return x.UserID != userID })
		db.HealthMetrics = filterBy(db.HealthMetrics, func(x HealthMetric) bool { return x.UserID != userID })
		db.Nudges = filterBy(db.Nudges, func(x Nudge) bool { return x.FromUserID != userID && x.ToUserID != userID })
		shared := make([]SharedWorkout, 0, len(db.SharedWorkouts))
		for _, x := range db.SharedWorkouts {
			if x.CreatedBy == userID {
				continue
			}
			ids := make([]string, 0, len(x.ParticipantIDs))
			for _, id := range x.ParticipantIDs {
				if id != userID {
					ids = append(ids, id)
				}
			}
			x.ParticipantIDs = ids
			if len(ids) > 0 {
				shared = append(shared, x)
			}
		}
		db.SharedWorkouts = shared
		db.UserBlocks = filterBy(db.UserBlocks, func(x UserBlock) bool { return x.BlockerID != userID && x.BlockedID != userID })
		db.ContentReports = filterBy(db.ContentReports, func(x ContentReport) bool { return x.ReporterID != userID && x.TargetUserID != userID })
		for i := range db.ModerationActions {
			if db.ModerationActions[i].ModeratorID == userID {
				db.ModerationActions[i].ModeratorID = "deleted-user"
			}
			if db.ModerationActions[i].TargetUserID == userID {
				db.ModerationActions[i].TargetUserID = "deleted-user"
			}
		}
		db.AgentTasks = filterBy(db.AgentTasks, func(x AgentTask) bool { return x.UserID != userID })
		db.AgentMemories = filterBy(db.AgentMemories, func(x AgentMemory) bool { return x.UserID != userID })
		db.MarketplaceItems = filterBy(db.MarketplaceItems, func(x MarketplaceItem) bool { return x.OwnerID != userID })
		db.VisionAnalyses = filterBy(db.VisionAnalyses, func(x VisionAnalysis) bool { return x.UserID != userID })
		db.GroceryLists = filterBy(db.GroceryLists, func(x GroceryList) bool { return x.UserID != userID })
		return nil
	})
	if err != nil {
		return err
	}
	for _, path := range paths {
		_ = os.Remove(path)
	}
	_ = os.RemoveAll(filepath.Join(s.Store.DataDir(), "photos", userID))
	return nil
}
