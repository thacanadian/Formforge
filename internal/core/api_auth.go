package core

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
)

func (s *Server) systemStatus(w http.ResponseWriter) {
	var users int
	var settings Settings
	_ = s.Store.Read(func(db Database) error { users = len(db.Users); settings = db.Settings; return nil })
	jsonOut(w, 200, map[string]any{"setupRequired": users == 0, "setupTokenRequired": users == 0 && s.Cloud && strings.TrimSpace(s.SetupToken) != "", "version": s.Version, "lanEnabled": settings.LANEnabled, "port": settings.Port, "cloud": s.Cloud})
}

func (s *Server) downloadCA(w http.ResponseWriter) {
	if s.Cloud {
		jsonError(w, 404, "not_available", "A local CA certificate is not used for the hosted version.")
		return
	}
	b, err := os.ReadFile(s.Certs.CADER)
	if err != nil {
		jsonError(w, 500, "ca_unavailable", "Local CA certificate is unavailable.")
		return
	}
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="FormForge-Local-CA.cer"`)
	_, _ = w.Write(b)
}

func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name, Email, Password, SetupToken string
		Profile                           Profile          `json:"profile"`
		CoachingPreferences               CoachPreferences `json:"coachingPreferences"`
		AppearancePreferences             ThemePreferences `json:"appearancePreferences"`
		AcceptTerms                       bool             `json:"acceptTerms"`
		AcceptPrivacy                     bool             `json:"acceptPrivacy"`
		AcceptCommunity                   bool             `json:"acceptCommunity"`
		AgeConfirmed                      bool             `json:"ageConfirmed"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if s.Cloud && strings.TrimSpace(s.SetupToken) != "" {
		want, got := []byte(strings.TrimSpace(s.SetupToken)), []byte(strings.TrimSpace(in.SetupToken))
		if len(want) != len(got) || subtle.ConstantTimeCompare(want, got) != 1 {
			jsonError(w, 403, "invalid_setup_token", "The hosted setup token is incorrect.")
			return
		}
	}
	in.Email = normalizeEmail(in.Email)
	in.Name = strings.TrimSpace(in.Name)
	if in.Profile.Age < s.legalConfig().MinimumAge {
		jsonError(w, 403, "age_restricted", "FormForge public accounts require users to meet the minimum age shown in the legal notice.")
		return
	}
	if s.Cloud && (!in.AcceptTerms || !in.AcceptPrivacy || !in.AgeConfirmed) {
		jsonError(w, 400, "consent_required", "Hosted setup requires acceptance of the Terms, Privacy Notice, and age confirmation.")
		return
	}
	if in.Name == "" || !strings.Contains(in.Email, "@") {
		jsonError(w, 400, "invalid_input", "Name and a valid email are required.")
		return
	}
	hash, err := HashPassword(in.Password)
	if err != nil {
		jsonError(w, 400, "weak_password", err.Error())
		return
	}
	var user User
	var recovery string
	err = s.Store.Update(func(db *Database) error {
		if len(db.Users) > 0 {
			return errors.New("setup is already complete")
		}
		now := nowISO()
		id := RandomID("user_")
		user = User{ID: id, Name: in.Name, Email: in.Email, PasswordHash: hash, Role: "admin", Active: true, PlanTier: "pro", CreatedAt: now, UpdatedAt: now}
		// Local installs are grandfathered for backward compatibility. Hosted setup requires explicit checkboxes above.
		if (in.AcceptTerms && in.AcceptPrivacy && in.AgeConfirmed) || (!s.Cloud && in.Profile.Age >= s.legalConfig().MinimumAge) {
			user.TermsAcceptedVersion, user.TermsAcceptedAt = db.Settings.TermsVersion, now
			user.PrivacyAcceptedVersion, user.PrivacyAcceptedAt = db.Settings.PrivacyVersion, now
			user.AgeConfirmedAt = now
			if !s.Cloud || in.AcceptCommunity {
				user.CommunityAcceptedVersion, user.CommunityAcceptedAt = db.Settings.CommunityVersion, now
			}
		}
		db.Users[id] = user
		p := normalizeProfile(in.Profile, user)
		db.Profiles[id] = p
		prefs := in.AppearancePreferences
		if prefs.Preset == "" {
			prefs = defaultThemeForUser(id)
		}
		prefs, prefErr := normalizeThemePreferences(prefs, id)
		if prefErr != nil {
			return prefErr
		}
		db.ThemePreferences[id] = prefs
		cp, _ := normalizeCoachPreferences(in.CoachingPreferences, id)
		db.CoachPreferences[id] = cp
		for _, h := range DefaultHabits(id) {
			h.CreatedAt = now
			h.UpdatedAt = now
			db.Habits = append(db.Habits, h)
		}
		recovery = base64.RawURLEncoding.EncodeToString(s.MasterKey)
		rh := sha256.Sum256(s.MasterKey)
		db.Settings.RecoveryHash = hex.EncodeToString(rh[:])
		s.audit(db, &user, "system.setup", "application", clientIP(r), map[string]any{"role": "admin"})
		return nil
	})
	if err != nil {
		jsonError(w, 409, "setup_failed", err.Error())
		return
	}
	token, csrf := s.newSession(user, r)
	setSessionCookie(w, token)
	jsonOut(w, 201, map[string]any{"user": publicUser(user), "csrf": csrf, "recoveryKey": recovery, "message": "Save the recovery key now. It unlocks encrypted backups and administrator recovery."})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password, TOTP string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	email := normalizeEmail(in.Email)
	ip := clientIP(r)
	key := ip + "|" + email
	var found User
	var ok bool
	var lockedMsg string
	_ = s.Store.Read(func(db Database) error {
		if a, exists := db.LoginAttempts[key]; exists && a.LockedUntil != "" {
			if t, e := time.Parse(time.RFC3339, a.LockedUntil); e == nil && time.Now().Before(t) {
				lockedMsg = "Too many failed attempts. Try again later."
			}
		}
		for _, u := range db.Users {
			if u.Email == email {
				found = u
				ok = true
				break
			}
		}
		if ok && found.LockedUntil != "" {
			if t, e := time.Parse(time.RFC3339, found.LockedUntil); e == nil && time.Now().Before(t) {
				lockedMsg = "This account is temporarily locked."
			}
		}
		return nil
	})
	if lockedMsg != "" {
		jsonError(w, 429, "login_locked", lockedMsg)
		return
	}
	if !ok || !found.Active || !VerifyPassword(found.PasswordHash, in.Password) {
		_ = s.recordFailedLogin(key, email, ip, ok, found)
		jsonError(w, 401, "invalid_credentials", "Email or password is incorrect.")
		return
	}
	if found.Role == "admin" && found.TOTPEnabled {
		secret, _ := DecryptSecret(s.MasterKey, found.TOTPSecretEncrypted)
		valid := validateTOTP(secret, in.TOTP, time.Now())
		usedRecovery := -1
		if !valid && strings.TrimSpace(in.TOTP) != "" {
			h := recoveryHash(in.TOTP)
			for i, x := range found.TOTPRecoveryHashes {
				if x == h {
					valid = true
					usedRecovery = i
					break
				}
			}
		}
		if !valid {
			if strings.TrimSpace(in.TOTP) == "" {
				jsonError(w, 401, "totp_required", "Enter the six-digit authenticator code or a recovery code.")
			} else {
				_ = s.recordFailedLogin(key, email, ip, true, found)
				jsonError(w, 401, "invalid_totp", "The authenticator or recovery code is incorrect.")
			}
			return
		}
		if usedRecovery >= 0 {
			_ = s.Store.Update(func(db *Database) error {
				u := db.Users[found.ID]
				u.TOTPRecoveryHashes = append(u.TOTPRecoveryHashes[:usedRecovery], u.TOTPRecoveryHashes[usedRecovery+1:]...)
				db.Users[u.ID] = u
				found = u
				return nil
			})
		}
	}
	_ = s.Store.Update(func(db *Database) error {
		u := db.Users[found.ID]
		u.FailedAttempts = 0
		u.LockedUntil = ""
		u.UpdatedAt = nowISO()
		db.Users[u.ID] = u
		delete(db.LoginAttempts, key)
		s.audit(db, &u, "auth.login", "account", ip, nil)
		found = u
		return nil
	})
	token, csrf := s.newSession(found, r)
	setSessionCookie(w, token)
	jsonOut(w, 200, map[string]any{"user": publicUser(found), "csrf": csrf})
}

func (s *Server) recordFailedLogin(key, email, ip string, userExists bool, found User) error {
	return s.Store.Update(func(db *Database) error {
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
		if userExists {
			u := db.Users[found.ID]
			u.FailedAttempts++
			if u.FailedAttempts >= 5 {
				u.LockedUntil = now.Add(15 * time.Minute).UTC().Format(time.RFC3339)
			}
			u.UpdatedAt = nowISO()
			db.Users[u.ID] = u
		}
		s.audit(db, nil, "auth.login_failed", email, ip, map[string]any{"userExists": userExists, "attempt": a.Count})
		return nil
	})
}

func (s *Server) newSession(user User, r *http.Request) (string, string) {
	token := RandomToken(32)
	csrf := RandomToken(24)
	h := HashToken(token)
	_ = s.Store.Update(func(db *Database) error {
		for k, v := range db.Sessions {
			if exp, e := time.Parse(time.RFC3339, v.ExpiresAt); e != nil || time.Now().After(exp) {
				delete(db.Sessions, k)
			}
		}
		db.Sessions[h] = Session{TokenHash: h, UserID: user.ID, CSRF: csrf, ExpiresAt: time.Now().Add(12 * time.Hour).UTC().Format(time.RFC3339), CreatedAt: nowISO(), LastSeenAt: nowISO(), IP: clientIP(r), UserAgent: r.UserAgent(), DeviceName: deviceLabel(r.UserAgent())}
		return nil
	})
	return token, csrf
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	_ = s.Store.Update(func(db *Database) error {
		delete(db.Sessions, HashToken(cu.CookieToken))
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "auth.logout", "account", clientIP(r), nil)
		return nil
	})
	clearSessionCookie(w)
	jsonOut(w, 200, map[string]any{"ok": true})
}

func (s *Server) sessionInfo(w http.ResponseWriter, cu *contextUser) {
	jsonOut(w, 200, map[string]any{"user": publicUser(cu.User), "csrf": cu.Session.CSRF})
}

func (s *Server) recover(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, RecoveryKey, NewPassword string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(in.RecoveryKey))
	if err != nil || len(key) != 32 {
		jsonError(w, 400, "invalid_recovery_key", "Recovery key format is invalid.")
		return
	}
	h := sha256.Sum256(key)
	hash, err := HashPassword(in.NewPassword)
	if err != nil {
		jsonError(w, 400, "weak_password", err.Error())
		return
	}
	var changed User
	err = s.Store.Update(func(db *Database) error {
		if db.Settings.RecoveryHash != "" && db.Settings.RecoveryHash != hex.EncodeToString(h[:]) {
			return errors.New("recovery key is incorrect")
		}
		for id, u := range db.Users {
			if u.Email == normalizeEmail(in.Email) && u.Role == "admin" {
				u.PasswordHash = hash
				u.FailedAttempts = 0
				u.LockedUntil = ""
				u.UpdatedAt = nowISO()
				db.Users[id] = u
				changed = u
				s.audit(db, &u, "auth.recovery", "account", clientIP(r), nil)
				return nil
			}
		}
		return errors.New("administrator account not found")
	})
	if err != nil {
		jsonError(w, 403, "recovery_failed", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true, "user": publicUser(changed)})
}

func deviceLabel(ua string) string {
	x := strings.ToLower(ua)
	switch {
	case strings.Contains(x, "iphone"):
		return "iPhone"
	case strings.Contains(x, "ipad"):
		return "iPad"
	case strings.Contains(x, "android"):
		return "Android device"
	case strings.Contains(x, "windows"):
		return "Windows PC"
	case strings.Contains(x, "macintosh"):
		return "Mac"
	default:
		return "Browser session"
	}
}

func publicUser(u User) map[string]any {
	return map[string]any{"id": u.ID, "name": u.Name, "email": u.Email, "role": u.Role, "active": u.Active, "createdAt": u.CreatedAt, "totpEnabled": u.TOTPEnabled, "planTier": u.PlanTier, "termsAcceptedVersion": u.TermsAcceptedVersion, "privacyAcceptedVersion": u.PrivacyAcceptedVersion, "communityAcceptedVersion": u.CommunityAcceptedVersion, "communityBanned": u.CommunityBanned, "communitySuspendedUntil": u.CommunitySuspendedUntil}
}

func normalizeProfile(p Profile, u User) Profile {
	p.UserID = u.ID
	if p.Name == "" {
		p.Name = u.Name
	}
	if p.DaysPerWeek < 2 || p.DaysPerWeek > 6 {
		p.DaysPerWeek = 3
	}
	if p.Experience == "" {
		p.Experience = "beginner"
	}
	if p.Goal == "" {
		p.Goal = "General fitness"
	}
	if p.Equipment == "" {
		p.Equipment = "Full gym"
	}
	if p.CalorieGoal <= 0 {
		p.CalorieGoal = calculateCalories(p)
	}
	if p.ProteinGoal <= 0 && p.WeightKG > 0 {
		p.ProteinGoal = int(p.WeightKG*1.8 + .5)
	}
	if p.CarbGoal <= 0 {
		p.CarbGoal = int(float64(p.CalorieGoal) * .4 / 4)
	}
	if p.FatGoal <= 0 {
		p.FatGoal = int(float64(p.CalorieGoal) * .28 / 9)
	}
	p.UpdatedAt = nowISO()
	return p
}

func calculateCalories(p Profile) int {
	if p.WeightKG <= 0 || p.HeightCM <= 0 || p.Age <= 0 {
		return 2200
	}
	bmr := 10*p.WeightKG + 6.25*p.HeightCM - 5*float64(p.Age) - 161
	if strings.EqualFold(p.Gender, "male") {
		bmr += 166
	}
	tdee := int(bmr*1.55 + .5)
	switch strings.ToLower(p.Goal) {
	case "lose fat":
		tdee -= 400
	case "build muscle":
		tdee += 300
	}
	if tdee < 1200 {
		tdee = 1200
	}
	return tdee
}
