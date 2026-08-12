package core

import (
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

var communityCategories = map[string]bool{"harassment": true, "hate": true, "sexual_content": true, "threat": true, "self_harm": true, "spam": true, "impersonation": true, "privacy": true, "copyright": true, "other": true}
var blockedTextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(kill yourself|kys|go die)\b`),
	regexp.MustCompile(`(?i)\b(rape|child porn|cp link|nudes? of a minor)\b`),
	regexp.MustCompile(`(?i)\b(nigger|faggot|retard(?:ed)?)\b`),
	regexp.MustCompile(`(?i)\b(i will kill you|i'm going to kill you|shoot up)\b`),
}
var spamPattern = regexp.MustCompile(`(?i)(https?://|www\.)\S+.*(https?://|www\.)\S+`)

func moderateCommunityText(text string) error {
	x := strings.TrimSpace(text)
	if x == "" {
		return errors.New("message is required")
	}
	for _, p := range blockedTextPatterns {
		if p.MatchString(x) {
			return errors.New("this message contains prohibited abusive, threatening, hateful, or sexual content")
		}
	}
	if spamPattern.MatchString(x) {
		return errors.New("multiple promotional links are not allowed")
	}
	return nil
}

func communityRestricted(u User) bool {
	if u.CommunityBanned {
		return true
	}
	if u.CommunitySuspendedUntil != "" {
		if t, err := time.Parse(time.RFC3339, u.CommunitySuspendedUntil); err == nil && time.Now().Before(t) {
			return true
		}
	}
	return false
}

func legalCommunityEligible(db Database, u User, minimumAge int) bool {
	p := db.Profiles[u.ID]
	return u.TermsAcceptedVersion == db.Settings.TermsVersion && u.PrivacyAcceptedVersion == db.Settings.PrivacyVersion && u.CommunityAcceptedVersion == db.Settings.CommunityVersion && u.AgeConfirmedAt != "" && p.Age >= minimumAge
}

func isBlocked(db Database, a, b string) bool {
	for _, x := range db.UserBlocks {
		if (x.BlockerID == a && x.BlockedID == b) || (x.BlockerID == b && x.BlockedID == a) {
			return true
		}
	}
	return false
}

func hiddenForReporter(db Database, viewer, kind, id string) bool {
	for _, r := range db.ContentReports {
		if r.ReporterID == viewer && r.TargetType == kind && r.TargetID == id && r.Status == "open" {
			return true
		}
	}
	return false
}

func (s *Server) communityGate(w http.ResponseWriter, cu *contextUser) bool {
	var u User
	var eligible bool
	_ = s.Store.Read(func(db Database) error {
		u = db.Users[cu.User.ID]
		eligible = legalCommunityEligible(db, u, s.legalConfig().MinimumAge)
		return nil
	})
	if !eligible {
		jsonError(w, 403, "community_consent_required", "Accept the current Terms, Privacy Notice, Community Standards, and age confirmation before using community features.")
		return false
	}
	if communityRestricted(u) {
		jsonError(w, 403, "community_restricted", "Community access is suspended or banned. Private fitness features remain available.")
		return false
	}
	return true
}

func (s *Server) listBlocks(w http.ResponseWriter, cu *contextUser) {
	out := []map[string]any{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.UserBlocks {
			if x.BlockerID == cu.User.ID {
				u := db.Users[x.BlockedID]
				out = append(out, map[string]any{"id": x.ID, "userId": x.BlockedID, "name": u.Name, "createdAt": x.CreatedAt})
			}
		}
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) blockUser(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !s.communityGate(w, cu) {
		return
	}
	var in struct {
		UserID string `json:"userId"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.UserID == "" || in.UserID == cu.User.ID {
		jsonError(w, 400, "invalid_input", "Choose another user.")
		return
	}
	err := s.Store.Update(func(db *Database) error {
		if u, ok := db.Users[in.UserID]; !ok || !u.Active {
			return errors.New("user not found")
		}
		for _, x := range db.UserBlocks {
			if x.BlockerID == cu.User.ID && x.BlockedID == in.UserID {
				return nil
			}
		}
		x := UserBlock{ID: RandomID("block_"), BlockerID: cu.User.ID, BlockedID: in.UserID, CreatedAt: nowISO()}
		db.UserBlocks = append(db.UserBlocks, x)
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "social.block", in.UserID, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 201, map[string]any{"ok": true})
}
func (s *Server) unblockUser(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		before := len(db.UserBlocks)
		db.UserBlocks = filterBy(db.UserBlocks, func(x UserBlock) bool { return !(x.BlockerID == cu.User.ID && (x.ID == id || x.BlockedID == id)) })
		if len(db.UserBlocks) == before {
			return errors.New("block not found")
		}
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "social.unblock", id, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func reportTarget(db Database, kind, id string) (string, bool) {
	switch kind {
	case "nudge":
		for _, x := range db.Nudges {
			if x.ID == id && x.ModerationStatus != "removed" {
				return x.FromUserID, true
			}
		}
	case "shared_workout":
		for _, x := range db.SharedWorkouts {
			if x.ID == id && x.ModerationStatus != "removed" {
				return x.CreatedBy, true
			}
		}
	case "user":
		if u, ok := db.Users[id]; ok && u.Active {
			return u.ID, true
		}
	}
	return "", false
}

func moderationTargetSummary(db Database, kind, id string) string {
	switch kind {
	case "nudge":
		for _, x := range db.Nudges {
			if x.ID == id {
				return x.Message
			}
		}
	case "shared_workout":
		for _, x := range db.SharedWorkouts {
			if x.ID == id {
				return x.WorkoutName + " · " + x.Date + " " + x.StartTime
			}
		}
	case "user":
		if u, ok := db.Users[id]; ok {
			return "User account: " + u.Name
		}
	}
	return ""
}
func (s *Server) createReport(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !s.communityGate(w, cu) {
		return
	}
	var in struct{ TargetType, TargetID, Category, Details string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	in.TargetType = strings.TrimSpace(in.TargetType)
	in.TargetID = strings.TrimSpace(in.TargetID)
	in.Category = strings.TrimSpace(in.Category)
	in.Details = strings.TrimSpace(in.Details)
	if !communityCategories[in.Category] || len(in.Details) > 1000 {
		jsonError(w, 400, "invalid_input", "Choose a valid report category and keep details under 1,000 characters.")
		return
	}
	var out ContentReport
	err := s.Store.Update(func(db *Database) error {
		targetUser, ok := reportTarget(*db, in.TargetType, in.TargetID)
		if !ok {
			return errors.New("content not found")
		}
		if targetUser == cu.User.ID {
			return errors.New("you cannot report your own content")
		}
		for _, x := range db.ContentReports {
			if x.ReporterID == cu.User.ID && x.TargetType == in.TargetType && x.TargetID == in.TargetID && x.Status == "open" {
				return errors.New("you already reported this content")
			}
		}
		now := nowISO()
		out = ContentReport{ID: RandomID("report_"), ReporterID: cu.User.ID, TargetType: in.TargetType, TargetID: in.TargetID, TargetUserID: targetUser, Category: in.Category, Details: in.Details, Status: "open", CreatedAt: now, UpdatedAt: now}
		db.ContentReports = append(db.ContentReports, out)
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "social.report", out.ID, clientIP(r), map[string]any{"targetType": in.TargetType, "category": in.Category})
		return nil
	})
	if err != nil {
		jsonError(w, 400, "report_failed", err.Error())
		return
	}
	jsonOut(w, 201, out)
}
func (s *Server) myReports(w http.ResponseWriter, cu *contextUser) {
	out := []ContentReport{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.ContentReports {
			if x.ReporterID == cu.User.ID {
				out = append(out, x)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) adminReports(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	out := []map[string]any{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.ContentReports {
			reporter := db.Users[x.ReporterID].Name
			target := db.Users[x.TargetUserID].Name
			out = append(out, map[string]any{"id": x.ID, "reporterId": x.ReporterID, "reporterName": reporter, "targetUserId": x.TargetUserID, "targetUserName": target, "targetType": x.TargetType, "targetId": x.TargetID, "targetContent": moderationTargetSummary(db, x.TargetType, x.TargetID), "category": x.Category, "details": x.Details, "status": x.Status, "resolution": x.Resolution, "createdAt": x.CreatedAt, "reviewedAt": x.ReviewedAt})
		}
		sort.Slice(out, func(i, j int) bool { return out[i]["createdAt"].(string) > out[j]["createdAt"].(string) })
		return nil
	})
	jsonOut(w, 200, out)
}

func removeModeratedContent(db *Database, typ, id, moderator, reason string) error {
	now := nowISO()
	switch typ {
	case "nudge":
		for i := range db.Nudges {
			if db.Nudges[i].ID == id {
				db.Nudges[i].ModerationStatus = "removed"
				db.Nudges[i].RemovedAt = now
				db.Nudges[i].RemovedBy = moderator
				db.Nudges[i].RemovalReason = reason
				return nil
			}
		}
	case "shared_workout":
		for i := range db.SharedWorkouts {
			if db.SharedWorkouts[i].ID == id {
				db.SharedWorkouts[i].ModerationStatus = "removed"
				db.SharedWorkouts[i].RemovedAt = now
				db.SharedWorkouts[i].RemovedBy = moderator
				db.SharedWorkouts[i].RemovalReason = reason
				return nil
			}
		}
	}
	return errors.New("content not found")
}
func (s *Server) resolveReport(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct{ Action, Reason string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	allowed := map[string]bool{"dismiss": true, "remove_content": true, "warn": true, "suspend_7d": true, "suspend_30d": true, "ban": true}
	if !allowed[in.Action] {
		jsonError(w, 400, "invalid_action", "Choose a supported moderation action.")
		return
	}
	err := s.Store.Update(func(db *Database) error {
		idx := -1
		for i := range db.ContentReports {
			if db.ContentReports[i].ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errors.New("report not found")
		}
		x := db.ContentReports[idx]
		if x.Status != "open" {
			return errors.New("report is already resolved")
		}
		removeRequested := in.Action == "remove_content" || in.Action == "suspend_7d" || in.Action == "suspend_30d" || in.Action == "ban"
		if removeRequested && x.TargetType != "user" {
			if err := removeModeratedContent(db, x.TargetType, x.TargetID, cu.User.ID, in.Reason); err != nil {
				return err
			}
		}
		if in.Action == "remove_content" && x.TargetType == "user" {
			return errors.New("a user report must be dismissed, warned, suspended, or banned")
		}
		u := db.Users[x.TargetUserID]
		switch in.Action {
		case "suspend_7d":
			u.CommunitySuspendedUntil = time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)
			u.CommunityBanReason = in.Reason
		case "suspend_30d":
			u.CommunitySuspendedUntil = time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
			u.CommunityBanReason = in.Reason
		case "ban":
			u.CommunityBanned = true
			u.CommunityBanReason = in.Reason
		}
		db.Users[u.ID] = u
		now := nowISO()
		x.Status = "resolved"
		x.Resolution = in.Action
		x.ReviewedBy = cu.User.ID
		x.ReviewedAt = now
		x.UpdatedAt = now
		db.ContentReports[idx] = x
		db.ModerationActions = append(db.ModerationActions, ModerationAction{ID: RandomID("mod_"), ModeratorID: cu.User.ID, Action: in.Action, TargetType: x.TargetType, TargetID: x.TargetID, TargetUserID: x.TargetUserID, Reason: in.Reason, CreatedAt: now})
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "moderation."+in.Action, id, clientIP(r), map[string]any{"targetUserId": x.TargetUserID})
		return nil
	})
	if err != nil {
		jsonError(w, 400, "moderation_failed", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
