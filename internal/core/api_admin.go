package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (s *Server) exportJSON(w http.ResponseWriter, cu *contextUser) {
	var payload map[string]any
	_ = s.Store.Read(func(db Database) error {
		payload = map[string]any{"version": s.Version, "exportedAt": nowISO(), "profile": db.Profiles[cu.User.ID], "plan": db.WeeklyPlans[cu.User.ID]}
		workouts := []Workout{}
		for _, x := range db.Workouts {
			if x.OwnerID == cu.User.ID {
				workouts = append(workouts, x)
			}
		}
		wl := []WorkoutLog{}
		for _, x := range db.WorkoutLogs {
			if x.UserID == cu.User.ID {
				wl = append(wl, x)
			}
		}
		n := []NutritionEntry{}
		for _, x := range db.Nutrition {
			if x.UserID == cu.User.ID {
				n = append(n, x)
			}
		}
		h := []Habit{}
		for _, x := range db.Habits {
			if x.UserID == cu.User.ID {
				h = append(h, x)
			}
		}
		hl := []HabitLog{}
		for _, x := range db.HabitLogs {
			if x.UserID == cu.User.ID {
				hl = append(hl, x)
			}
		}
		p := []ProgressEntry{}
		for _, x := range db.Progress {
			if x.UserID == cu.User.ID {
				p = append(p, x)
			}
		}
		c := []CheckIn{}
		for _, x := range db.CheckIns {
			if x.UserID == cu.User.ID {
				c = append(c, x)
			}
		}
		payload["customWorkouts"] = workouts
		payload["workoutLogs"] = wl
		payload["nutrition"] = n
		payload["habits"] = h
		payload["habitLogs"] = hl
		payload["progress"] = p
		payload["checkIns"] = c
		chats := []ChatMessage{}
		for _, x := range db.ChatMessages {
			if x.UserID == cu.User.ID {
				chats = append(chats, x)
			}
		}
		payload["chatMessages"] = chats
		payload["coachingPreferences"] = preferencesFor(db, cu.User.ID)
		payload["coachSources"] = approvedCoachSources(db, cu.User.ID)
		return nil
	})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="FormForge-user-export.json"`)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	kind := r.URL.Query().Get("type")
	if kind == "" {
		kind = "nutrition"
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="FormForge-%s.csv"`, kind))
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = s.Store.Read(func(db Database) error {
		switch kind {
		case "nutrition":
			_ = cw.Write([]string{"date", "name", "serving", "calories", "protein", "carbs", "fat"})
			for _, x := range db.Nutrition {
				if x.UserID == cu.User.ID {
					_ = cw.Write([]string{x.Date, x.Name, x.Serving, fmt.Sprint(x.Calories), fmt.Sprint(x.Protein), fmt.Sprint(x.Carbs), fmt.Sprint(x.Fat)})
				}
			}
		case "progress":
			_ = cw.Write([]string{"date", "weightKg", "bodyFat", "notes"})
			for _, x := range db.Progress {
				if x.UserID == cu.User.ID {
					_ = cw.Write([]string{x.Date, fmt.Sprint(x.WeightKG), fmt.Sprint(x.BodyFat), x.Notes})
				}
			}
		case "workouts":
			_ = cw.Write([]string{"date", "workout", "duration", "notes"})
			for _, x := range db.WorkoutLogs {
				if x.UserID == cu.User.ID {
					_ = cw.Write([]string{x.Date, x.WorkoutName, strconv.Itoa(x.Duration), x.Notes})
				}
			}
		default:
			_ = cw.Write([]string{"error"})
			_ = cw.Write([]string{"unsupported export type"})
		}
		return nil
	})
}

func (s *Server) importJSON(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct {
		Data json.RawMessage `json:"data"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var payload struct {
		Profile             Profile           `json:"profile"`
		Plan                map[string]string `json:"plan"`
		CustomWorkouts      []Workout         `json:"customWorkouts"`
		WorkoutLogs         []WorkoutLog      `json:"workoutLogs"`
		Nutrition           []NutritionEntry  `json:"nutrition"`
		Habits              []Habit           `json:"habits"`
		HabitLogs           []HabitLog        `json:"habitLogs"`
		Progress            []ProgressEntry   `json:"progress"`
		CheckIns            []CheckIn         `json:"checkIns"`
		ChatMessages        []ChatMessage     `json:"chatMessages"`
		CoachingPreferences CoachPreferences  `json:"coachingPreferences"`
	}
	if err := json.Unmarshal(in.Data, &payload); err != nil {
		jsonError(w, 400, "invalid_import", err.Error())
		return
	}
	counts := map[string]int{}
	err := s.Store.Update(func(db *Database) error {
		if payload.Profile.Name != "" {
			db.Profiles[cu.User.ID] = normalizeProfile(payload.Profile, cu.User)
		}
		if payload.Plan != nil {
			db.WeeklyPlans[cu.User.ID] = payload.Plan
		}
		for _, x := range payload.CustomWorkouts {
			x.ID = RandomID("workout_")
			x.OwnerID = cu.User.ID
			x.BuiltIn = false
			db.Workouts[x.ID] = x
			counts["workouts"]++
		}
		for _, x := range payload.WorkoutLogs {
			x.ID = RandomID("wlog_")
			x.UserID = cu.User.ID
			db.WorkoutLogs = append(db.WorkoutLogs, x)
			counts["workoutLogs"]++
		}
		for _, x := range payload.Nutrition {
			x.ID = RandomID("food_")
			x.UserID = cu.User.ID
			db.Nutrition = append(db.Nutrition, x)
			counts["nutrition"]++
		}
		habitMap := map[string]string{}
		for _, x := range payload.Habits {
			old := x.ID
			x.ID = RandomID("habit_")
			x.UserID = cu.User.ID
			habitMap[old] = x.ID
			db.Habits = append(db.Habits, x)
			counts["habits"]++
		}
		for _, x := range payload.HabitLogs {
			x.ID = RandomID("hlog_")
			x.UserID = cu.User.ID
			if n := habitMap[x.HabitID]; n != "" {
				x.HabitID = n
			}
			db.HabitLogs = append(db.HabitLogs, x)
		}
		for _, x := range payload.Progress {
			x.ID = RandomID("progress_")
			x.UserID = cu.User.ID
			db.Progress = append(db.Progress, x)
			counts["progress"]++
		}
		for _, x := range payload.CheckIns {
			x.ID = RandomID("checkin_")
			x.UserID = cu.User.ID
			db.CheckIns = append(db.CheckIns, x)
			counts["checkIns"]++
		}
		if len(payload.CoachingPreferences.Influences) > 0 {
			cp, _ := normalizeCoachPreferences(payload.CoachingPreferences, cu.User.ID)
			db.CoachPreferences[cu.User.ID] = cp
			counts["coachingPreferences"] = 1
		}
		for _, x := range payload.ChatMessages {
			if x.Role != "user" && x.Role != "assistant" {
				continue
			}
			x.ID = RandomID("chat_")
			x.UserID = cu.User.ID
			if x.At == "" {
				x.At = nowISO()
			}
			db.ChatMessages = append(db.ChatMessages, x)
			counts["chatMessages"]++
		}
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "data.import", u.ID, clientIP(r), map[string]any{"counts": counts})
		return nil
	})
	if err != nil {
		jsonError(w, 500, "import_failed", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true, "imported": counts})
}

func (s *Server) adminUsers(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	out := []map[string]any{}
	_ = s.Store.Read(func(db Database) error {
		for _, u := range db.Users {
			out = append(out, publicUser(u))
		}
		sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) < out[j]["name"].(string) })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) adminCreateUser(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct {
		Name, Email, Password, Role, PlanTier string
		Profile                               Profile `json:"profile"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Role != "admin" {
		in.Role = "user"
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		jsonError(w, 400, "weak_password", err.Error())
		return
	}
	in.Email = normalizeEmail(in.Email)
	var u User
	err = s.Store.Update(func(db *Database) error {
		for _, x := range db.Users {
			if x.Email == in.Email {
				return errors.New("email is already in use")
			}
		}
		now := nowISO()
		tier := strings.ToLower(strings.TrimSpace(in.PlanTier))
		if tier != "pro" {
			tier = "free"
		}
		if in.Role == "admin" {
			tier = "pro"
		}
		u = User{ID: RandomID("user_"), Name: strings.TrimSpace(in.Name), Email: in.Email, PasswordHash: hash, Role: in.Role, Active: true, PlanTier: tier, CreatedAt: now, UpdatedAt: now}
		db.Users[u.ID] = u
		db.Profiles[u.ID] = normalizeProfile(in.Profile, u)
		db.CoachPreferences[u.ID] = defaultCoachPreferences(u.ID)
		for _, h := range DefaultHabits(u.ID) {
			h.CreatedAt = now
			h.UpdatedAt = now
			db.Habits = append(db.Habits, h)
		}
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "admin.user_create", u.ID, clientIP(r), map[string]any{"role": u.Role})
		return nil
	})
	if err != nil {
		jsonError(w, 409, "create_failed", err.Error())
		return
	}
	jsonOut(w, 201, publicUser(u))
}
func (s *Server) adminUpdateUser(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct {
		Name, Email, Password, Role, PlanTier string
		Active                                *bool `json:"active"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var out User
	err := s.Store.Update(func(db *Database) error {
		u, ok := db.Users[id]
		if !ok {
			return errors.New("user not found")
		}
		if strings.TrimSpace(in.Name) != "" {
			u.Name = strings.TrimSpace(in.Name)
		}
		if strings.TrimSpace(in.Email) != "" {
			u.Email = normalizeEmail(in.Email)
		}
		if in.Role == "admin" || in.Role == "user" {
			u.Role = in.Role
		}
		if in.PlanTier == "free" || in.PlanTier == "pro" {
			u.PlanTier = in.PlanTier
		}
		if u.Role == "admin" {
			u.PlanTier = "pro"
		}
		if in.Active != nil {
			u.Active = *in.Active
		}
		if in.Password != "" {
			h, e := HashPassword(in.Password)
			if e != nil {
				return e
			}
			u.PasswordHash = h
			for k, sess := range db.Sessions {
				if sess.UserID == id {
					delete(db.Sessions, k)
				}
			}
		}
		u.UpdatedAt = nowISO()
		db.Users[id] = u
		out = u
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "admin.user_update", id, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 400, "update_failed", err.Error())
		return
	}
	jsonOut(w, 200, publicUser(out))
}
func (s *Server) adminDeleteUser(w http.ResponseWriter, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	if id == cu.User.ID {
		jsonError(w, 400, "cannot_delete_self", "Use the Security page for self-deletion. The sole administrator cannot be deleted while the service remains active.")
		return
	}
	if err := s.deleteAccount(id, cu.User, ""); err != nil {
		jsonError(w, 400, "deletion_failed", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func filterWorkoutLogs(xs []WorkoutLog, id string) []WorkoutLog {
	n := xs[:0]
	for _, x := range xs {
		if x.UserID != id {
			n = append(n, x)
		}
	}
	return n
}
func filterNutrition(xs []NutritionEntry, id string) []NutritionEntry {
	n := xs[:0]
	for _, x := range xs {
		if x.UserID != id {
			n = append(n, x)
		}
	}
	return n
}
func filterHabits(xs []Habit, id string) []Habit {
	n := xs[:0]
	for _, x := range xs {
		if x.UserID != id {
			n = append(n, x)
		}
	}
	return n
}
func filterHabitLogs(xs []HabitLog, id string) []HabitLog {
	n := xs[:0]
	for _, x := range xs {
		if x.UserID != id {
			n = append(n, x)
		}
	}
	return n
}
func filterProgress(xs []ProgressEntry, id string) []ProgressEntry {
	n := xs[:0]
	for _, x := range xs {
		if x.UserID != id {
			n = append(n, x)
		}
	}
	return n
}
func filterCheckins(xs []CheckIn, id string) []CheckIn {
	n := xs[:0]
	for _, x := range xs {
		if x.UserID != id {
			n = append(n, x)
		}
	}
	return n
}
func filterChatMessages(xs []ChatMessage, id string) []ChatMessage {
	n := xs[:0]
	for _, x := range xs {
		if x.UserID != id {
			n = append(n, x)
		}
	}
	return n
}
func (s *Server) adminAudit(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	limit := 200
	if n, _ := strconv.Atoi(r.URL.Query().Get("limit")); n > 0 && n <= 1000 {
		limit = n
	}
	out := []AuditEvent{}
	_ = s.Store.Read(func(db Database) error {
		out = append(out, db.Audit...)
		sort.Slice(out, func(i, j int) bool { return out[i].At > out[j].At })
		if len(out) > limit {
			out = out[:limit]
		}
		return nil
	})
	jsonOut(w, 200, out)
}

func (s *Server) getSettings(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	_ = s.Store.Read(func(db Database) error { jsonOut(w, 200, db.Settings); return nil })
}
func (s *Server) putSettings(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in Settings
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Port < 1024 || in.Port > 65535 {
		jsonError(w, 400, "invalid_port", "Port must be between 1024 and 65535.")
		return
	}
	var out Settings
	_ = s.Store.Update(func(db *Database) error {
		db.Settings.LANEnabled = in.LANEnabled
		db.Settings.Port = in.Port
		db.Settings.BackupCopyPath = strings.TrimSpace(in.BackupCopyPath)
		if in.AIDailyTokenCap >= 0 {
			db.Settings.AIDailyTokenCap = in.AIDailyTokenCap
		}
		if in.AIDailyCostCapMicros >= 0 {
			db.Settings.AIDailyCostCapMicros = in.AIDailyCostCapMicros
		}
		if in.FreeDailyTokenCap >= 0 {
			db.Settings.FreeDailyTokenCap = in.FreeDailyTokenCap
		}
		if in.FreeDailyCostCapMicros >= 0 {
			db.Settings.FreeDailyCostCapMicros = in.FreeDailyCostCapMicros
		}
		db.Settings.UpdateManifestURL = strings.TrimSpace(in.UpdateManifestURL)
		db.Settings.UpdatePublicKey = strings.TrimSpace(in.UpdatePublicKey)
		db.Settings.CrashReportingEnabled = in.CrashReportingEnabled
		db.Settings.CrashEndpoint = strings.TrimSpace(in.CrashEndpoint)
		if in.BackupIntervalHours > 0 && in.BackupIntervalHours <= 720 {
			db.Settings.BackupIntervalHours = in.BackupIntervalHours
		}
		if strings.TrimSpace(in.TakedownContact) != "" {
			db.Settings.TakedownContact = strings.TrimSpace(in.TakedownContact)
		}
		if strings.TrimSpace(in.TermsVersion) != "" {
			db.Settings.TermsVersion = strings.TrimSpace(in.TermsVersion)
		}
		out = db.Settings
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "settings.update", "application", clientIP(r), map[string]any{"restartRequired": true})
		return nil
	})
	jsonOut(w, 200, map[string]any{"settings": out, "restartRequired": true})
}
func (s *Server) listBackups(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	x, err := s.Store.ListBackups()
	if err != nil {
		jsonError(w, 500, "backup_list_failed", err.Error())
		return
	}
	jsonOut(w, 200, x)
}
func (s *Server) createBackup(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var copyPath string
	_ = s.Store.Read(func(db Database) error { copyPath = db.Settings.BackupCopyPath; return nil })
	path, err := s.Store.CreateBackup(s.MasterKey, copyPath)
	if err != nil {
		jsonError(w, 500, "backup_failed", err.Error())
		return
	}
	_ = s.Store.Update(func(db *Database) error {
		db.Settings.LastAutoBackupAt = nowISO()
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "backup.create", filepath.Base(path), clientIP(r), nil)
		return nil
	})
	jsonOut(w, 201, map[string]any{"ok": true, "name": filepath.Base(path)})
}
func (s *Server) downloadBackup(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	name := filepath.Base(r.URL.Query().Get("name"))
	if !strings.HasSuffix(name, ".ffbackup") {
		jsonError(w, 400, "invalid_name", "Invalid backup name.")
		return
	}
	b, err := os.ReadFile(filepath.Join(s.Store.DataDir(), "backups", name))
	if err != nil {
		jsonError(w, 404, "not_found", "Backup not found.")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	_, _ = w.Write(b)
}
func (s *Server) restoreBackup(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct{ Data, RecoveryKey string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	blob, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		jsonError(w, 400, "invalid_backup", "Backup file could not be decoded.")
		return
	}
	key := s.MasterKey
	if strings.TrimSpace(in.RecoveryKey) != "" {
		key, err = base64.RawURLEncoding.DecodeString(strings.TrimSpace(in.RecoveryKey))
		if err != nil || len(key) != 32 {
			jsonError(w, 400, "invalid_recovery_key", "Recovery key format is invalid.")
			return
		}
	}
	plain, err := DecryptBackup(key, blob)
	if err != nil {
		jsonError(w, 400, "restore_failed", err.Error())
		return
	}
	_, _ = s.Store.CreateBackup(s.MasterKey, "")
	if err := s.Store.ReplaceFromBytes(plain); err != nil {
		jsonError(w, 400, "restore_failed", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true, "message": "Backup restored. All sessions were signed out; restart FormForge."})
}
func (s *Server) exportRecoveryKey(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	key := base64.RawURLEncoding.EncodeToString(s.MasterKey)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="FormForge-Recovery-Key.txt"`)
	_, _ = fmt.Fprintf(w, "FORMFORGE RECOVERY KEY\n\n%s\n\nKeep this offline. Anyone with this key and a backup can decrypt the backup and reset an administrator password.\n", key)
}
func (s *Server) rotateRecoveryKey(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	newKey := make([]byte, 32)
	_, _ = io.ReadFull(rand.Reader, newKey)
	path := filepath.Join(s.Store.DataDir(), "master.key")
	if err := os.WriteFile(path, []byte(base64.RawURLEncoding.EncodeToString(newKey)), 0600); err != nil {
		jsonError(w, 500, "rotate_failed", err.Error())
		return
	}
	s.MasterKey = newKey
	h := sha256.Sum256(newKey)
	_ = s.Store.Update(func(db *Database) error {
		db.Settings.RecoveryHash = hex.EncodeToString(h[:])
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "recovery_key.rotate", "application", "", nil)
		return nil
	})
	jsonOut(w, 200, map[string]any{"recoveryKey": base64.RawURLEncoding.EncodeToString(newKey), "warning": "Existing backups still require the old key. Create a new backup now."})
}
