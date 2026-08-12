package core

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (s *Server) twoFactorStatus(w http.ResponseWriter, cu *contextUser) {
	jsonOut(w, 200, map[string]any{"enabled": cu.User.TOTPEnabled, "adminOnly": true})
}
func (s *Server) beginTwoFactor(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	secret, err := newTOTPSecret()
	if err != nil {
		jsonError(w, 500, "totp_failed", "Could not create a two-factor secret.")
		return
	}
	enc, err := EncryptSecret(s.MasterKey, secret)
	if err != nil {
		jsonError(w, 500, "totp_failed", "Could not protect the two-factor secret.")
		return
	}
	_ = s.Store.Update(func(db *Database) error {
		u := db.Users[cu.User.ID]
		u.TOTPEnabled = false
		u.TOTPSecretEncrypted = enc
		u.TOTPRecoveryHashes = nil
		u.UpdatedAt = nowISO()
		db.Users[u.ID] = u
		s.audit(db, &u, "security.2fa.begin", u.ID, clientIP(r), nil)
		return nil
	})
	jsonOut(w, 200, map[string]any{"secret": secret, "otpauthUri": totpURI(secret, cu.User.Email)})
}
func (s *Server) confirmTwoFactor(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var secret string
	_ = s.Store.Read(func(db Database) error {
		secret, _ = DecryptSecret(s.MasterKey, db.Users[cu.User.ID].TOTPSecretEncrypted)
		return nil
	})
	if secret == "" || !validateTOTP(secret, in.Code, time.Now()) {
		jsonError(w, 400, "invalid_totp", "The authenticator code is incorrect or expired.")
		return
	}
	codes := newRecoveryCodes(8)
	hashes := make([]string, len(codes))
	for i, c := range codes {
		hashes[i] = recoveryHash(c)
	}
	_ = s.Store.Update(func(db *Database) error {
		u := db.Users[cu.User.ID]
		u.TOTPEnabled = true
		u.TOTPRecoveryHashes = hashes
		u.UpdatedAt = nowISO()
		db.Users[u.ID] = u
		s.audit(db, &u, "security.2fa.enable", u.ID, clientIP(r), nil)
		return nil
	})
	jsonOut(w, 200, map[string]any{"enabled": true, "recoveryCodes": codes})
}
func (s *Server) disableTwoFactor(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if !VerifyPassword(cu.User.PasswordHash, in.Password) {
		jsonError(w, 403, "invalid_password", "Password is incorrect.")
		return
	}
	secret, _ := DecryptSecret(s.MasterKey, cu.User.TOTPSecretEncrypted)
	if cu.User.TOTPEnabled && !validateTOTP(secret, in.Code, time.Now()) {
		jsonError(w, 403, "invalid_totp", "Authenticator code is incorrect.")
		return
	}
	_ = s.Store.Update(func(db *Database) error {
		u := db.Users[cu.User.ID]
		u.TOTPEnabled = false
		u.TOTPSecretEncrypted = ""
		u.TOTPRecoveryHashes = nil
		u.UpdatedAt = nowISO()
		db.Users[u.ID] = u
		s.audit(db, &u, "security.2fa.disable", u.ID, clientIP(r), nil)
		return nil
	})
	jsonOut(w, 200, map[string]any{"enabled": false})
}
func (s *Server) listSessions(w http.ResponseWriter, cu *contextUser) {
	out := []map[string]any{}
	cur := HashToken(cu.CookieToken)
	_ = s.Store.Read(func(db Database) error {
		for h, x := range db.Sessions {
			if x.UserID != cu.User.ID {
				continue
			}
			id := h
			if len(id) > 16 {
				id = id[:16]
			}
			out = append(out, map[string]any{"id": id, "current": h == cur, "ip": x.IP, "userAgent": x.UserAgent, "deviceName": x.DeviceName, "createdAt": x.CreatedAt, "lastSeenAt": x.LastSeenAt, "expiresAt": x.ExpiresAt})
		}
		sort.Slice(out, func(i, j int) bool { return out[i]["createdAt"].(string) > out[j]["createdAt"].(string) })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	id = strings.TrimSpace(id)
	if len(id) < 8 {
		jsonError(w, 400, "invalid_session", "Invalid session identifier.")
		return
	}
	cur := HashToken(cu.CookieToken)
	revokedCurrent := false
	err := s.Store.Update(func(db *Database) error {
		found := ""
		for h, x := range db.Sessions {
			if x.UserID == cu.User.ID && strings.HasPrefix(h, id) {
				if found != "" {
					return errors.New("session identifier is ambiguous")
				}
				found = h
			}
		}
		if found == "" {
			return errors.New("session not found")
		}
		delete(db.Sessions, found)
		revokedCurrent = found == cur
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "security.session.revoke", id, clientIP(r), map[string]any{"current": revokedCurrent})
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	if revokedCurrent {
		clearSessionCookie(w)
	}
	jsonOut(w, 200, map[string]any{"ok": true, "currentRevoked": revokedCurrent})
}
