package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type HealthReport struct {
	Status    string         `json:"status"`
	OK        bool           `json:"ok"`
	Product   string         `json:"product"`
	Version   string         `json:"version"`
	CheckedAt string         `json:"checkedAt"`
	Checks    map[string]any `json:"checks"`
	Issues    []string       `json:"issues"`
}

func (s *Server) healthReport() HealthReport {
	h := HealthReport{Status: "up", OK: true, Product: "formforge", Version: s.Version, CheckedAt: nowISO(), Checks: map[string]any{}, Issues: []string{}}
	err := s.Store.Read(func(db Database) error {
		h.Checks["schemaVersion"] = db.SchemaVersion
		h.Checks["users"] = len(db.Users)
		h.Checks["lastBackupAt"] = db.Settings.LastAutoBackupAt
		if db.SchemaVersion != SchemaVersion {
			h.Issues = append(h.Issues, "database schema does not match application")
		}
		failed := 0
		for _, j := range db.Jobs {
			if j.Status == "failed" {
				failed++
			}
		}
		h.Checks["failedJobs"] = failed
		if failed > 0 {
			h.Issues = append(h.Issues, "one or more background jobs failed")
		}
		if len(db.Users) > 0 {
			hours := db.Settings.BackupIntervalHours
			if hours <= 0 {
				hours = 24
			}
			if db.Settings.LastAutoBackupAt == "" {
				h.Issues = append(h.Issues, "automatic backup has not completed yet")
			} else if t, e := time.Parse(time.RFC3339, db.Settings.LastAutoBackupAt); e != nil || time.Since(t) > time.Duration(hours*2)*time.Hour {
				h.Issues = append(h.Issues, "automatic backup is overdue")
			}
		}
		return nil
	})
	if err != nil {
		h.Status = "down"
		h.OK = false
		h.Issues = append(h.Issues, "database could not be read")
	}
	if _, err := os.Stat(s.Store.DataDir()); err != nil {
		h.Status = "down"
		h.OK = false
		h.Issues = append(h.Issues, "data directory is unavailable")
	}
	if h.Status != "down" && len(h.Issues) > 0 {
		h.Status = "degraded"
	}
	return h
}
func (s *Server) systemHealth(w http.ResponseWriter) {
	h := s.healthReport()
	code := 200
	if h.Status == "down" {
		code = 503
	}
	jsonOut(w, code, h)
}

func appendJSONLog(dataDir, kind string, payload map[string]any) error {
	dir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	payload["at"] = nowISO()
	payload["kind"] = kind
	b, _ := json.Marshal(payload)
	f, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}
func (s *Server) clientLog(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct {
		Level, Message, Page, Stack string
		Details                     map[string]any
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if len(in.Message) > 2000 || len(in.Stack) > 8000 {
		jsonError(w, 400, "too_large", "Log event is too large.")
		return
	}
	_ = appendJSONLog(s.Store.DataDir(), "client", map[string]any{"level": in.Level, "message": in.Message, "page": in.Page, "stack": in.Stack, "details": in.Details, "userId": cu.User.ID, "ip": clientIP(r)})
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (s *Server) reportPanic(r *http.Request, v any) {
	payload := map[string]any{"message": fmt.Sprint(v), "path": r.URL.Path, "method": r.Method, "ip": clientIP(r)}
	_ = appendJSONLog(s.Store.DataDir(), "server_panic", payload)
	var settings Settings
	_ = s.Store.Read(func(db Database) error { settings = db.Settings; return nil })
	if !settings.CrashReportingEnabled || strings.TrimSpace(settings.CrashEndpoint) == "" {
		return
	}
	u, err := url.Parse(settings.CrashEndpoint)
	if err != nil || u.Scheme != "https" {
		return
	}
	go func() {
		b, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", u.String(), strings.NewReader(string(b)))
		req.Header.Set("Content-Type", "application/json")
		c := &http.Client{Timeout: 5 * time.Second}
		if resp, e := c.Do(req); e == nil {
			resp.Body.Close()
		}
	}()
}

type UpdateManifest struct {
	Version   string `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"`
	Notes     string `json:"notes,omitempty"`
}

func compareVersion(a, b string) int {
	pa := strings.Split(strings.TrimPrefix(a, "v"), ".")
	pb := strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		ai, bi := 0, 0
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d", &ai)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d", &bi)
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}
func verifyManifest(m UpdateManifest, publicKey string) error {
	pk, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil || len(pk) != ed25519.PublicKeySize {
		return errors.New("update public key is invalid")
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(m.Signature))
	if err != nil {
		return errors.New("update signature is invalid")
	}
	msg := []byte(m.Version + "\n" + m.URL + "\n" + strings.ToLower(m.SHA256))
	if !ed25519.Verify(ed25519.PublicKey(pk), msg, sig) {
		return errors.New("update manifest signature verification failed")
	}
	return nil
}
func (s *Server) fetchUpdateManifest() (UpdateManifest, error) {
	var settings Settings
	_ = s.Store.Read(func(db Database) error { settings = db.Settings; return nil })
	if settings.UpdateManifestURL == "" || settings.UpdatePublicKey == "" {
		return UpdateManifest{}, errors.New("signed update channel is not configured")
	}
	u, err := url.Parse(settings.UpdateManifestURL)
	if err != nil || u.Scheme != "https" {
		return UpdateManifest{}, errors.New("update manifest URL must use HTTPS")
	}
	c := &http.Client{Timeout: 10 * time.Second}
	resp, err := c.Get(u.String())
	if err != nil {
		return UpdateManifest{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return UpdateManifest{}, fmt.Errorf("update server returned %s", resp.Status)
	}
	var m UpdateManifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&m); err != nil {
		return m, err
	}
	if err := verifyManifest(m, settings.UpdatePublicKey); err != nil {
		return m, err
	}
	if _, err := validatePublicURL(m.URL); err != nil {
		return m, fmt.Errorf("update download URL: %w", err)
	}
	if len(m.SHA256) != 64 {
		return m, errors.New("update SHA-256 is invalid")
	}
	return m, nil
}
func (s *Server) updateStatus(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	m, err := s.fetchUpdateManifest()
	if err != nil {
		jsonOut(w, 200, map[string]any{"configured": false, "currentVersion": s.Version, "message": err.Error()})
		return
	}
	jsonOut(w, 200, map[string]any{"configured": true, "currentVersion": s.Version, "manifest": m, "updateAvailable": compareVersion(s.Version, m.Version) < 0})
}
func (s *Server) stageUpdate(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	m, err := s.fetchUpdateManifest()
	if err != nil {
		jsonError(w, 400, "update_unavailable", err.Error())
		return
	}
	if compareVersion(s.Version, m.Version) >= 0 {
		jsonError(w, 409, "already_current", "FormForge is already current.")
		return
	}
	c := &http.Client{Timeout: 2 * time.Minute}
	resp, err := c.Get(m.URL)
	if err != nil {
		jsonError(w, 502, "download_failed", err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		jsonError(w, 502, "download_failed", resp.Status)
		return
	}
	blob, err := io.ReadAll(io.LimitReader(resp.Body, 200<<20))
	if err != nil {
		jsonError(w, 502, "download_failed", err.Error())
		return
	}
	sum := sha256.Sum256(blob)
	if hex.EncodeToString(sum[:]) != strings.ToLower(m.SHA256) {
		jsonError(w, 400, "hash_mismatch", "Downloaded installer failed SHA-256 verification.")
		return
	}
	dir := filepath.Join(s.Store.DataDir(), "updates")
	_ = os.MkdirAll(dir, 0700)
	path := filepath.Join(dir, "FormForge-Setup-"+m.Version+".exe")
	if err := os.WriteFile(path, blob, 0700); err != nil {
		jsonError(w, 500, "save_failed", err.Error())
		return
	}
	u := cu.User
	_ = s.Store.Update(func(db *Database) error {
		s.audit(db, &u, "update.stage", m.Version, clientIP(r), map[string]any{"path": path})
		return nil
	})
	launched := false
	if runtime.GOOS == "windows" {
		if err := exec.Command(path, "/SILENT", "/CLOSEAPPLICATIONS", "/RESTARTAPPLICATIONS").Start(); err == nil {
			launched = true
		}
	}
	jsonOut(w, 200, map[string]any{"ok": true, "version": m.Version, "path": path, "installerLaunched": launched, "message": "The signed installer was verified and staged. Application data remains outside the installation folder."})
}
