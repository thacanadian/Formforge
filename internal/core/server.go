package core

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Server struct {
	Store      *Store
	MasterKey  []byte
	Version    string
	Web        fs.FS
	Certs      CertPaths
	Cloud      bool
	TrustProxy bool
	SetupToken string
	Legal      LegalConfig
}

type contextUser struct {
	User        User
	Session     Session
	CookieToken string
}

func NewServer(store *Store, masterKey []byte, version string, web fs.FS, certs CertPaths) *Server {
	return &Server{Store: store, MasterKey: masterKey, Version: version, Web: web, Certs: certs, Legal: LegalConfigFromEnv()}
}

func (s *Server) Handler() http.Handler { return http.HandlerFunc(s.serveHTTP) }

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if s.TrustProxy {
		applyTrustedProxyClientIP(r)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; manifest-src 'self'; worker-src 'self'")
	w.Header().Set("Cache-Control", "no-store")
	if s.Cloud {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}

	defer func() {
		if v := recover(); v != nil {
			s.reportPanic(r, v)
			jsonError(w, 500, "internal_error", "An internal error was logged. Your data was not deleted.")
		}
	}()
	if strings.HasPrefix(r.URL.Path, "/legal") || r.URL.Path == "/account-deletion" {
		s.serveLegalPage(w, r)
		return
	}
	if r.URL.Path == "/health" {
		s.systemHealth(w)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		s.handleAPI(w, r)
		return
	}
	s.serveStatic(w, r)
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(filepath.ToSlash(r.URL.Path), "/")
	if path == "" {
		path = "index.html"
	}
	if strings.Contains(path, "..") {
		http.NotFound(w, r)
		return
	}
	b, err := fs.ReadFile(s.Web, path)
	if err != nil {
		b, err = fs.ReadFile(s.Web, "index.html")
		if err != nil {
			http.Error(w, "UI unavailable", 500)
			return
		}
		path = "index.html"
	}
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if strings.HasSuffix(path, ".js") {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}
	if strings.HasSuffix(path, ".webmanifest") {
		w.Header().Set("Content-Type", "application/manifest+json")
	}
	// Local applications update in place. Avoid stale service-worker/API mismatches
	// after a repair or upgrade by requiring revalidation for every UI asset.
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	_, _ = w.Write(b)
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = "/api"
	}
	public := map[string]bool{"/api/system/status": true, "/api/auth/setup": true, "/api/auth/login": true, "/api/auth/recover": true, "/api/system/ca": true, "/api/coaching/takedown": true, "/api/legal/terms": true, "/api/account-deletion/request": true}
	var cu *contextUser
	if !public[path] {
		var err error
		cu, err = s.authenticate(r)
		if err != nil {
			jsonError(w, 401, "authentication_required", err.Error())
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if r.Header.Get("X-CSRF-Token") == "" || r.Header.Get("X-CSRF-Token") != cu.Session.CSRF {
				jsonError(w, 403, "csrf_failed", "The request could not be verified. Refresh and try again.")
				return
			}
		}
	}

	switch {
	case path == "/api/system/status" && r.Method == "GET":
		s.systemStatus(w)
	case path == "/api/system/health" && r.Method == "GET":
		s.systemHealth(w)
	case path == "/api/system/client-log" && r.Method == "POST":
		s.clientLog(w, r, cu)
	case path == "/api/update/status" && r.Method == "GET":
		s.updateStatus(w, cu)
	case path == "/api/update/stage" && r.Method == "POST":
		s.stageUpdate(w, r, cu)
	case path == "/api/system/ca" && r.Method == "GET":
		s.downloadCA(w)
	case path == "/api/system/mobile" && r.Method == "GET":
		s.mobileInfo(w, r, cu)
	case path == "/api/legal/terms" && r.Method == "GET":
		s.getTerms(w)
	case path == "/api/account-deletion/request" && r.Method == "POST":
		s.publicDeleteAccount(w, r)
	case path == "/api/auth/setup" && r.Method == "POST":
		s.setup(w, r)
	case path == "/api/auth/login" && r.Method == "POST":
		s.login(w, r)
	case path == "/api/auth/logout" && r.Method == "POST":
		s.logout(w, r, cu)
	case path == "/api/auth/session" && r.Method == "GET":
		s.sessionInfo(w, cu)
	case path == "/api/auth/recover" && r.Method == "POST":
		s.recover(w, r)
	case path == "/api/security/2fa" && r.Method == "GET":
		s.twoFactorStatus(w, cu)
	case path == "/api/security/2fa/begin" && r.Method == "POST":
		s.beginTwoFactor(w, r, cu)
	case path == "/api/security/2fa/confirm" && r.Method == "POST":
		s.confirmTwoFactor(w, r, cu)
	case path == "/api/security/2fa" && r.Method == "DELETE":
		s.disableTwoFactor(w, r, cu)
	case path == "/api/security/sessions" && r.Method == "GET":
		s.listSessions(w, cu)
	case strings.HasPrefix(path, "/api/security/sessions/") && r.Method == "DELETE":
		s.revokeSession(w, r, cu, strings.TrimPrefix(path, "/api/security/sessions/"))
	case path == "/api/dashboard" && r.Method == "GET":
		s.dashboard(w, cu)
	case path == "/api/profile" && r.Method == "GET":
		s.getProfile(w, cu)
	case path == "/api/profile" && r.Method == "PUT":
		s.putProfile(w, r, cu)
	case path == "/api/workouts" && r.Method == "GET":
		s.listWorkouts(w, cu)
	case path == "/api/workouts" && r.Method == "POST":
		s.createWorkout(w, r, cu)
	case strings.HasPrefix(path, "/api/workouts/") && r.Method == "PUT":
		s.updateWorkout(w, r, cu, strings.TrimPrefix(path, "/api/workouts/"))
	case strings.HasPrefix(path, "/api/workouts/") && r.Method == "DELETE":
		s.deleteWorkout(w, r, cu, strings.TrimPrefix(path, "/api/workouts/"))
	case path == "/api/plan" && r.Method == "GET":
		s.getPlan(w, cu)
	case path == "/api/plan" && r.Method == "PUT":
		s.putPlan(w, r, cu)
	case path == "/api/workout-logs" && r.Method == "GET":
		s.listWorkoutLogs(w, r, cu)
	case path == "/api/workout-logs" && r.Method == "POST":
		s.createWorkoutLog(w, r, cu)
	case strings.HasPrefix(path, "/api/workout-logs/") && r.Method == "DELETE":
		s.deleteWorkoutLog(w, cu, strings.TrimPrefix(path, "/api/workout-logs/"))
	case path == "/api/nutrition" && r.Method == "GET":
		s.listNutrition(w, r, cu)
	case path == "/api/nutrition" && r.Method == "POST":
		s.createNutrition(w, r, cu)
	case strings.HasPrefix(path, "/api/nutrition/") && r.Method == "PUT":
		s.updateNutrition(w, r, cu, strings.TrimPrefix(path, "/api/nutrition/"))
	case strings.HasPrefix(path, "/api/nutrition/") && r.Method == "DELETE":
		s.deleteNutrition(w, cu, strings.TrimPrefix(path, "/api/nutrition/"))
	case path == "/api/food-search" && r.Method == "GET":
		s.foodSearch(w, r)
	case path == "/api/food-lookup" && r.Method == "GET":
		s.foodLookup(w, r)
	case path == "/api/habits" && r.Method == "GET":
		s.listHabits(w, r, cu)
	case path == "/api/habits" && r.Method == "POST":
		s.createHabit(w, r, cu)
	case strings.HasSuffix(path, "/toggle") && strings.HasPrefix(path, "/api/habits/") && r.Method == "POST":
		s.toggleHabit(w, r, cu, strings.TrimSuffix(strings.TrimPrefix(path, "/api/habits/"), "/toggle"))
	case strings.HasPrefix(path, "/api/habits/") && r.Method == "PUT":
		s.updateHabit(w, r, cu, strings.TrimPrefix(path, "/api/habits/"))
	case strings.HasPrefix(path, "/api/habits/") && r.Method == "DELETE":
		s.deleteHabit(w, cu, strings.TrimPrefix(path, "/api/habits/"))
	case path == "/api/progress" && r.Method == "GET":
		s.listProgress(w, cu)
	case path == "/api/progress" && r.Method == "POST":
		s.createProgress(w, r, cu)
	case strings.HasPrefix(path, "/api/progress/") && r.Method == "PUT":
		s.updateProgress(w, r, cu, strings.TrimPrefix(path, "/api/progress/"))
	case strings.HasPrefix(path, "/api/progress/") && r.Method == "DELETE":
		s.deleteProgress(w, cu, strings.TrimPrefix(path, "/api/progress/"))
	case path == "/api/checkins" && r.Method == "GET":
		s.listCheckins(w, cu)
	case path == "/api/checkins" && r.Method == "POST":
		s.createCheckin(w, r, cu)
	case path == "/api/health/providers" && r.Method == "GET":
		s.healthProviders(w)
	case path == "/api/health" && r.Method == "GET":
		s.listHealth(w, r, cu)
	case path == "/api/health/import" && r.Method == "POST":
		s.importHealth(w, r, cu)
	case path == "/api/pain-flags" && r.Method == "GET":
		s.listPainFlags(w, cu)
	case path == "/api/pain-flags" && r.Method == "POST":
		s.savePainFlag(w, r, cu)
	case strings.HasPrefix(path, "/api/pain-flags/") && r.Method == "DELETE":
		s.deletePainFlag(w, r, cu, strings.TrimPrefix(path, "/api/pain-flags/"))
	case path == "/api/training/progression" && r.Method == "GET":
		s.progressionSuggestions(w, cu)
	case path == "/api/meal-plans" && r.Method == "GET":
		s.mealPlans(w, cu)
	case path == "/api/meal-plans" && r.Method == "POST":
		s.createMealPlan(w, r, cu)
	case strings.HasPrefix(path, "/api/meal-plans/") && r.Method == "DELETE":
		s.deleteMealPlan(w, cu, strings.TrimPrefix(path, "/api/meal-plans/"))
	case path == "/api/progress-photos" && r.Method == "GET":
		s.listPhotos(w, cu)
	case path == "/api/progress-photos" && r.Method == "POST":
		s.uploadPhoto(w, r, cu)
	case strings.HasPrefix(path, "/api/progress-photos/") && strings.HasSuffix(path, "/data") && r.Method == "GET":
		s.downloadPhoto(w, cu, strings.TrimSuffix(strings.TrimPrefix(path, "/api/progress-photos/"), "/data"))
	case strings.HasPrefix(path, "/api/progress-photos/") && r.Method == "DELETE":
		s.deletePhoto(w, cu, strings.TrimPrefix(path, "/api/progress-photos/"))
	case path == "/api/social/users" && r.Method == "GET":
		s.socialUsers(w, cu)
	case path == "/api/social/leaderboard" && r.Method == "GET":
		s.leaderboard(w, cu)
	case path == "/api/social/nudges" && r.Method == "GET":
		s.listNudges(w, cu)
	case path == "/api/social/nudges" && r.Method == "POST":
		s.createNudge(w, r, cu)
	case strings.HasPrefix(path, "/api/social/nudges/") && r.Method == "PUT":
		s.markNudge(w, cu, strings.TrimPrefix(path, "/api/social/nudges/"))
	case path == "/api/social/workouts" && r.Method == "GET":
		s.listSharedWorkouts(w, cu)
	case path == "/api/social/workouts" && r.Method == "POST":
		s.createSharedWorkout(w, r, cu)
	case strings.HasPrefix(path, "/api/social/workouts/") && r.Method == "DELETE":
		s.deleteSharedWorkout(w, cu, strings.TrimPrefix(path, "/api/social/workouts/"))
	case path == "/api/social/blocks" && r.Method == "GET":
		s.listBlocks(w, cu)
	case path == "/api/social/blocks" && r.Method == "POST":
		s.blockUser(w, r, cu)
	case strings.HasPrefix(path, "/api/social/blocks/") && r.Method == "DELETE":
		s.unblockUser(w, r, cu, strings.TrimPrefix(path, "/api/social/blocks/"))
	case path == "/api/social/reports" && r.Method == "GET":
		s.myReports(w, cu)
	case path == "/api/social/reports" && r.Method == "POST":
		s.createReport(w, r, cu)
	case path == "/api/coaching/team" && r.Method == "GET":
		s.coachingTeam(w, cu)
	case path == "/api/coaching/preferences" && r.Method == "PUT":
		s.putCoachingPreferences(w, r, cu)
	case path == "/api/coaching/pack" && r.Method == "GET":
		s.coachingPack(w, cu)
	case path == "/api/coaching/link-preview" && r.Method == "POST":
		s.previewCoachLink(w, r, cu)
	case path == "/api/coaching/profiles" && r.Method == "GET":
		s.listCustomCoachProfiles(w, cu)
	case path == "/api/coaching/profiles" && r.Method == "POST":
		s.createCustomCoachProfile(w, r, cu)
	case strings.HasPrefix(path, "/api/coaching/profiles/") && r.Method == "PUT":
		s.updateCustomCoachProfile(w, r, cu, strings.TrimPrefix(path, "/api/coaching/profiles/"))
	case strings.HasPrefix(path, "/api/coaching/profiles/") && r.Method == "DELETE":
		s.deleteCustomCoachProfile(w, r, cu, strings.TrimPrefix(path, "/api/coaching/profiles/"))
	case path == "/api/coaching/takedown" && r.Method == "POST":
		s.submitTakedown(w, r)
	case path == "/api/coaching/takedowns" && r.Method == "GET":
		s.listTakedowns(w, cu)
	case strings.HasPrefix(path, "/api/coaching/takedowns/") && r.Method == "PUT":
		s.updateTakedown(w, r, cu, strings.TrimPrefix(path, "/api/coaching/takedowns/"))
	case path == "/api/coaching/sources" && r.Method == "POST":
		s.createCoachSource(w, r, cu)
	case strings.HasPrefix(path, "/api/coaching/sources/") && r.Method == "PUT":
		s.updateCoachSource(w, r, cu, strings.TrimPrefix(path, "/api/coaching/sources/"))
	case strings.HasPrefix(path, "/api/coaching/sources/") && r.Method == "DELETE":
		s.deleteCoachSource(w, r, cu, strings.TrimPrefix(path, "/api/coaching/sources/"))
	case path == "/api/legal/accept" && r.Method == "POST":
		s.acceptTerms(w, r, cu)
	case path == "/api/legal/status" && r.Method == "GET":
		s.legalStatus(w, cu)
	case path == "/api/account" && r.Method == "DELETE":
		s.deleteOwnAccount(w, r, cu)
	case path == "/api/ai/status" && r.Method == "GET":
		s.aiStatus(w, cu)
	case path == "/api/ai/settings" && r.Method == "GET":
		s.getAISettings(w, cu)
	case path == "/api/ai/settings" && r.Method == "PUT":
		s.putAISettings(w, r, cu)
	case path == "/api/ai/test" && r.Method == "POST":
		s.testAI(w, r, cu)
	case path == "/api/ai/chat" && r.Method == "POST":
		s.chatAI(w, r, cu)
	case path == "/api/ai/history" && r.Method == "GET":
		s.listAIHistory(w, r, cu)
	case path == "/api/ai/history" && r.Method == "DELETE":
		s.clearAIHistory(w, r, cu)
	case strings.HasPrefix(path, "/api/ai/history/") && r.Method == "DELETE":
		s.deleteAIHistoryItem(w, r, cu, strings.TrimPrefix(path, "/api/ai/history/"))
	case path == "/api/ai/usage" && r.Method == "GET":
		s.aiUsageStatus(w, cu)
	case path == "/api/export/json" && r.Method == "GET":
		s.exportJSON(w, cu)
	case path == "/api/export/protected" && r.Method == "POST":
		s.protectedExport(w, r, cu)
	case path == "/api/import/protected" && r.Method == "POST":
		s.protectedImport(w, r, cu)
	case path == "/api/export/csv" && r.Method == "GET":
		s.exportCSV(w, r, cu)
	case path == "/api/import/json" && r.Method == "POST":
		s.importJSON(w, r, cu)
	case path == "/api/admin/users" && r.Method == "GET":
		s.adminUsers(w, cu)
	case path == "/api/admin/users" && r.Method == "POST":
		s.adminCreateUser(w, r, cu)
	case strings.HasPrefix(path, "/api/admin/users/") && r.Method == "PUT":
		s.adminUpdateUser(w, r, cu, strings.TrimPrefix(path, "/api/admin/users/"))
	case strings.HasPrefix(path, "/api/admin/users/") && r.Method == "DELETE":
		s.adminDeleteUser(w, cu, strings.TrimPrefix(path, "/api/admin/users/"))
	case path == "/api/admin/audit" && r.Method == "GET":
		s.adminAudit(w, r, cu)
	case path == "/api/admin/moderation/reports" && r.Method == "GET":
		s.adminReports(w, cu)
	case strings.HasPrefix(path, "/api/admin/moderation/reports/") && r.Method == "PUT":
		s.resolveReport(w, r, cu, strings.TrimPrefix(path, "/api/admin/moderation/reports/"))
	case path == "/api/theme" && r.Method == "GET":
		s.getTheme(w, cu)
	case path == "/api/theme" && r.Method == "PUT":
		s.putTheme(w, r, cu)
	case path == "/api/recovery-score" && r.Method == "GET":
		s.getRecovery(w, cu)
	case path == "/api/knowledge/status" && r.Method == "GET":
		s.knowledgeStatus(w, cu)
	case path == "/api/agent/memories" && r.Method == "GET":
		s.listMemories(w, cu)
	case path == "/api/agent/memories" && r.Method == "POST":
		s.saveMemory(w, r, cu)
	case strings.HasPrefix(path, "/api/agent/memories/") && r.Method == "DELETE":
		s.deleteMemory(w, cu, strings.TrimPrefix(path, "/api/agent/memories/"))
	case path == "/api/grocery-lists" && r.Method == "POST":
		s.grocery(w, r, cu)
	case path == "/api/marketplace" && (r.Method == "GET" || r.Method == "POST"):
		s.marketplace(w, r, cu)
	case path == "/api/agent/settings" && (r.Method == "GET" || r.Method == "PUT"):
		s.agentSettings(w, r, cu)
	case path == "/api/agent/tasks" && (r.Method == "GET" || r.Method == "POST"):
		s.agentTasks(w, r, cu)
	case path == "/api/settings" && r.Method == "GET":
		s.getSettings(w, cu)
	case path == "/api/settings" && r.Method == "PUT":
		s.putSettings(w, r, cu)
	case path == "/api/backups" && r.Method == "GET":
		s.listBackups(w, cu)
	case path == "/api/backups" && r.Method == "POST":
		s.createBackup(w, r, cu)
	case path == "/api/backups/download" && r.Method == "GET":
		s.downloadBackup(w, r, cu)
	case path == "/api/backups/restore" && r.Method == "POST":
		s.restoreBackup(w, r, cu)
	case path == "/api/recovery-key" && r.Method == "GET":
		s.exportRecoveryKey(w, cu)
	case path == "/api/recovery-key/rotate" && r.Method == "POST":
		s.rotateRecoveryKey(w, cu)
	default:
		jsonError(w, 404, "not_found", "API endpoint not found")
	}
}

func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func jsonError(w http.ResponseWriter, status int, code, msg string) {
	jsonOut(w, status, map[string]any{"error": code, "message": msg})
}
func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
func nowISO() string                 { return time.Now().UTC().Format(time.RFC3339) }
func todayISO() string               { return time.Now().Format("2006-01-02") }
func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
func applyTrustedProxyClientIP(r *http.Request) {
	raw := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	if ip := net.ParseIP(raw); ip != nil {
		r.RemoteAddr = net.JoinHostPort(ip.String(), "0")
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

func (s *Server) audit(db *Database, user *User, action, target, ip string, details map[string]any) {
	e := AuditEvent{ID: RandomID("audit_"), At: nowISO(), Action: action, Target: target, IP: ip, Details: details}
	if user != nil {
		e.ActorID = user.ID
		e.ActorName = user.Name
	}
	db.Audit = append(db.Audit, e)
	if len(db.Audit) > 10000 {
		db.Audit = db.Audit[len(db.Audit)-10000:]
	}
}

func (s *Server) authenticate(r *http.Request) (*contextUser, error) {
	c, err := r.Cookie("ff_session")
	if err != nil || c.Value == "" {
		return nil, errors.New("Please sign in.")
	}
	h := HashToken(c.Value)
	var out *contextUser
	err = s.Store.Read(func(db Database) error {
		sess, ok := db.Sessions[h]
		if !ok {
			return errors.New("Your session has expired.")
		}
		exp, err := time.Parse(time.RFC3339, sess.ExpiresAt)
		if err != nil || time.Now().After(exp) {
			return errors.New("Your session has expired.")
		}
		u, ok := db.Users[sess.UserID]
		if !ok || !u.Active {
			return errors.New("This account is disabled.")
		}
		out = &contextUser{User: u, Session: sess, CookieToken: c.Value}
		return nil
	})
	return out, err
}

func requireAdmin(w http.ResponseWriter, cu *contextUser) bool {
	if cu == nil || cu.User.Role != "admin" {
		jsonError(w, 403, "admin_required", "Administrator permission is required.")
		return false
	}
	return true
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: "ff_session", Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: 12 * 60 * 60})
}
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "ff_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}
