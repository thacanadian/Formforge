package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu      sync.RWMutex
	path    string
	dataDir string
	DB      Database
}

type backupEnvelope struct {
	Format   string            `json:"format"`
	Database json.RawMessage   `json:"database"`
	Files    map[string]string `json:"files,omitempty"`
}

func OpenStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dataDir, "formforge.db.json")
	s := &Store{path: path, dataDir: dataDir, DB: NewDatabase()}
	if err := s.load(); err != nil {
		return nil, err
	}
	if len(s.DB.Workouts) == 0 {
		s.DB.Workouts = BuiltInWorkouts()
		if err := s.persistLocked(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) DataDir() string { return s.dataDir }

func (s *Store) load() error {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.persistLocked()
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &s.DB); err == nil {
		before := s.DB.SchemaVersion
		if before < SchemaVersion {
			_ = s.writeMigrationSafetyCopy(b, before)
		}
		if _, err := MigrateDatabase(&s.DB); err != nil {
			return err
		}
		s.normalize()
		if before < SchemaVersion {
			return s.persistLocked()
		}
		return nil
	}
	prev := s.path + ".prev"
	if pb, perr := os.ReadFile(prev); perr == nil {
		var recovered Database
		if json.Unmarshal(pb, &recovered) == nil {
			if _, err := MigrateDatabase(&recovered); err != nil {
				return err
			}
			s.DB = recovered
			s.normalize()
			_ = os.Rename(s.path, s.path+".corrupt-"+time.Now().UTC().Format("20060102-150405"))
			return s.persistLocked()
		}
	}
	_ = os.Rename(s.path, s.path+".corrupt-"+time.Now().UTC().Format("20060102-150405"))
	s.DB = NewDatabase()
	return s.persistLocked()
}

func (s *Store) writeMigrationSafetyCopy(raw []byte, version int) error {
	dir := filepath.Join(s.dataDir, "migration-backups")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	name := fmt.Sprintf("FormForge-pre-migration-v%d-%s.json", version, time.Now().UTC().Format("20060102-150405"))
	return os.WriteFile(filepath.Join(dir, name), raw, 0600)
}

func (s *Store) normalize() {
	if s.DB.Users == nil {
		s.DB.Users = map[string]User{}
	}
	if s.DB.Profiles == nil {
		s.DB.Profiles = map[string]Profile{}
	}
	if s.DB.Workouts == nil {
		s.DB.Workouts = map[string]Workout{}
	}
	if s.DB.WeeklyPlans == nil {
		s.DB.WeeklyPlans = map[string]map[string]string{}
	}
	if s.DB.Sessions == nil {
		s.DB.Sessions = map[string]Session{}
	}
	if s.DB.LoginAttempts == nil {
		s.DB.LoginAttempts = map[string]LoginAttempt{}
	}
	if s.DB.ChatMessages == nil {
		s.DB.ChatMessages = []ChatMessage{}
	}
	if s.DB.CoachPreferences == nil {
		s.DB.CoachPreferences = map[string]CoachPreferences{}
	}
	if s.DB.CoachSources == nil {
		s.DB.CoachSources = []CoachSource{}
	}
	if s.DB.CustomCoachProfiles == nil {
		s.DB.CustomCoachProfiles = map[string]CustomCoachProfile{}
	}
	if s.DB.TakedownRequests == nil {
		s.DB.TakedownRequests = []TakedownRequest{}
	}
	if s.DB.AIUsage == nil {
		s.DB.AIUsage = []AIUsage{}
	}
	if s.DB.Migrations == nil {
		s.DB.Migrations = []MigrationRecord{}
	}
	if s.DB.PainFlags == nil {
		s.DB.PainFlags = []PainFlag{}
	}
	if s.DB.ProgressPhotos == nil {
		s.DB.ProgressPhotos = []ProgressPhoto{}
	}
	if s.DB.MealPlans == nil {
		s.DB.MealPlans = []MealPlan{}
	}
	if s.DB.WearableConnections == nil {
		s.DB.WearableConnections = []WearableConnection{}
	}
	if s.DB.HealthMetrics == nil {
		s.DB.HealthMetrics = []HealthMetric{}
	}
	if s.DB.Nudges == nil {
		s.DB.Nudges = []Nudge{}
	}
	if s.DB.SharedWorkouts == nil {
		s.DB.SharedWorkouts = []SharedWorkout{}
	}
	if s.DB.UserBlocks == nil {
		s.DB.UserBlocks = []UserBlock{}
	}
	if s.DB.ContentReports == nil {
		s.DB.ContentReports = []ContentReport{}
	}
	if s.DB.ModerationActions == nil {
		s.DB.ModerationActions = []ModerationAction{}
	}
	if s.DB.Jobs == nil {
		s.DB.Jobs = []BackgroundJob{}
	}
	if s.DB.ThemePreferences == nil {
		s.DB.ThemePreferences = map[string]ThemePreferences{}
	}
	if s.DB.AgentTasks == nil {
		s.DB.AgentTasks = []AgentTask{}
	}
	if s.DB.AgentMemories == nil {
		s.DB.AgentMemories = []AgentMemory{}
	}
	if s.DB.MarketplaceItems == nil {
		s.DB.MarketplaceItems = []MarketplaceItem{}
	}
	if s.DB.VisionAnalyses == nil {
		s.DB.VisionAnalyses = []VisionAnalysis{}
	}
	if s.DB.GroceryLists == nil {
		s.DB.GroceryLists = []GroceryList{}
	}
	if s.DB.Settings.Port == 0 {
		s.DB.Settings.Port = 8443
	}
	if strings.TrimSpace(s.DB.Settings.AIMode) == "" {
		s.DB.Settings.AIMode = "auto"
	}
	if strings.TrimSpace(s.DB.Settings.AIBaseURL) == "" {
		s.DB.Settings.AIBaseURL = "https://api.openai.com/v1"
	}
	if strings.TrimSpace(s.DB.Settings.AIModel) == "" {
		s.DB.Settings.AIModel = "gpt-4o-mini"
	}
	if s.DB.Settings.AIDailyTokenCap <= 0 {
		s.DB.Settings.AIDailyTokenCap = 50000
	}
	if s.DB.Settings.AIDailyCostCapMicros <= 0 {
		s.DB.Settings.AIDailyCostCapMicros = 2500000
	}
	if s.DB.Settings.BackupIntervalHours <= 0 {
		s.DB.Settings.BackupIntervalHours = 24
	}
	if s.DB.Settings.AgentModel == "" {
		s.DB.Settings.AgentModel = "llama3.1:8b"
	}
	if s.DB.Settings.AgentBaseURL == "" {
		s.DB.Settings.AgentBaseURL = "http://127.0.0.1:11434/v1"
	}
	if s.DB.Settings.AgentMaxSteps <= 0 {
		s.DB.Settings.AgentMaxSteps = 8
	}
	if s.DB.Settings.TermsVersion == "" {
		s.DB.Settings.TermsVersion = "1.4"
	}
	for id, u := range s.DB.Users {
		if u.Role == "admin" {
			u.PlanTier = "pro"
		} else if u.PlanTier == "" {
			u.PlanTier = "free"
		}
		s.DB.Users[id] = u
	}
	s.DB.SchemaVersion = SchemaVersion
}

func (s *Store) Read(fn func(Database) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fn(s.DB)
}

func (s *Store) Update(fn func(*Database) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.DB); err != nil {
		return err
	}
	s.DB.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return s.persistLocked()
}

func (s *Store) persistLocked() error {
	b, err := json.MarshalIndent(s.DB, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	cerr := f.Close()
	if err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	prev := s.path + ".prev"
	_ = os.Remove(prev)
	if _, err := os.Stat(s.path); err == nil {
		_ = os.Rename(s.path, prev)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Rename(prev, s.path)
		return err
	}
	return nil
}

func (s *Store) Snapshot(includeSessions bool) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyDB := s.DB
	if !includeSessions {
		copyDB.Sessions = map[string]Session{}
		copyDB.LoginAttempts = map[string]LoginAttempt{}
	}
	return json.Marshal(copyDB)
}

func (s *Store) ReplaceFromBytes(b []byte) error {
	var env backupEnvelope
	dbBytes := b
	files := map[string]string{}
	if json.Unmarshal(b, &env) == nil && env.Format == "FORMFORGE-BACKUP-V2" && len(env.Database) > 0 {
		dbBytes = env.Database
		files = env.Files
	}
	var db Database
	if err := json.Unmarshal(dbBytes, &db); err != nil {
		return fmt.Errorf("invalid database backup: %w", err)
	}
	if db.SchemaVersion < 1 || db.Users == nil {
		return errors.New("backup failed integrity validation")
	}
	if _, err := MigrateDatabase(&db); err != nil {
		return fmt.Errorf("backup migration failed: %w", err)
	}
	photoRoot := filepath.Join(s.dataDir, "photos")
	if len(files) > 0 {
		tmpRoot := photoRoot + ".restore-tmp"
		_ = os.RemoveAll(tmpRoot)
		if err := os.MkdirAll(tmpRoot, 0700); err != nil {
			return err
		}
		for rel, encoded := range files {
			clean := filepath.Clean(rel)
			if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
				_ = os.RemoveAll(tmpRoot)
				return errors.New("backup contains an unsafe file path")
			}
			data, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				_ = os.RemoveAll(tmpRoot)
				return errors.New("backup contains an invalid file")
			}
			dst := filepath.Join(tmpRoot, clean)
			if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
				_ = os.RemoveAll(tmpRoot)
				return err
			}
			if err := os.WriteFile(dst, data, 0600); err != nil {
				_ = os.RemoveAll(tmpRoot)
				return err
			}
		}
		_ = os.RemoveAll(photoRoot + ".restore-prev")
		_ = os.Rename(photoRoot, photoRoot+".restore-prev")
		if err := os.Rename(tmpRoot, photoRoot); err != nil {
			_ = os.Rename(photoRoot+".restore-prev", photoRoot)
			return err
		}
		_ = os.RemoveAll(photoRoot + ".restore-prev")
	}
	for i := range db.ProgressPhotos {
		db.ProgressPhotos[i].EncryptedPath = filepath.Join(photoRoot, db.ProgressPhotos[i].UserID, db.ProgressPhotos[i].ID+".ffphoto")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	db.Sessions = map[string]Session{}
	db.LoginAttempts = map[string]LoginAttempt{}
	s.DB = db
	s.normalize()
	return s.persistLocked()
}

func RandomID(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func RandomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func EnsureMasterKey(dataDir string) ([]byte, bool, error) {
	path := filepath.Join(dataDir, "master.key")
	b, err := os.ReadFile(path)
	if err == nil {
		decoded, derr := base64.RawURLEncoding.DecodeString(strings.TrimSpace(string(b)))
		if derr != nil || len(decoded) != 32 {
			return nil, false, errors.New("master.key is invalid")
		}
		return decoded, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(path, []byte(base64.RawURLEncoding.EncodeToString(key)), 0600); err != nil {
		return nil, false, err
	}
	return key, true, nil
}

func EncryptBackup(key, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	hash := sha256.Sum256(plain)
	ciphertext := gcm.Seal(nil, nonce, plain, []byte("FORMFORGE-BACKUP-V1"))
	out := append([]byte("FFBK1\n"), nonce...)
	out = append(out, hash[:]...)
	out = append(out, ciphertext...)
	return out, nil
}

func DecryptBackup(key, blob []byte) ([]byte, error) {
	if len(blob) < 6+12+32 || string(blob[:6]) != "FFBK1\n" {
		return nil, errors.New("not a FormForge backup")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < 6+ns+32 {
		return nil, errors.New("backup truncated")
	}
	nonce := blob[6 : 6+ns]
	expected := blob[6+ns : 6+ns+32]
	plain, err := gcm.Open(nil, nonce, blob[6+ns+32:], []byte("FORMFORGE-BACKUP-V1"))
	if err != nil {
		return nil, errors.New("backup decryption failed; recovery key may be wrong")
	}
	h := sha256.Sum256(plain)
	if !equalBytes(h[:], expected) {
		return nil, errors.New("backup integrity check failed")
	}
	return plain, nil
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var x byte
	for i := range a {
		x |= a[i] ^ b[i]
	}
	return x == 0
}

func EncryptSecret(key []byte, plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), []byte("FORMFORGE-SECRET-V1"))
	return "FFSEC1." + base64.RawURLEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func DecryptSecret(key []byte, encoded string) (string, error) {
	if strings.TrimSpace(encoded) == "" {
		return "", nil
	}
	if !strings.HasPrefix(encoded, "FFSEC1.") {
		return "", errors.New("unsupported encrypted secret format")
	}
	blob, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encoded, "FFSEC1."))
	if err != nil {
		return "", errors.New("encrypted secret is invalid")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(blob) < gcm.NonceSize() {
		return "", errors.New("encrypted secret is truncated")
	}
	plain, err := gcm.Open(nil, blob[:gcm.NonceSize()], blob[gcm.NonceSize():], []byte("FORMFORGE-SECRET-V1"))
	if err != nil {
		return "", errors.New("encrypted secret could not be decrypted")
	}
	return string(plain), nil
}

func (s *Store) CreateBackup(masterKey []byte, copyPath string) (string, error) {
	dbBytes, err := s.Snapshot(false)
	if err != nil {
		return "", err
	}
	env := backupEnvelope{Format: "FORMFORGE-BACKUP-V2", Database: dbBytes, Files: map[string]string{}}
	photoRoot := filepath.Join(s.dataDir, "photos")
	_ = filepath.Walk(photoRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(photoRoot, path)
		if err != nil {
			return nil
		}
		b, err := os.ReadFile(path)
		if err == nil {
			env.Files[filepath.ToSlash(rel)] = base64.StdEncoding.EncodeToString(b)
		}
		return nil
	})
	plain, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	blob, err := EncryptBackup(masterKey, plain)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	name := "FormForge-" + time.Now().UTC().Format("20060102-150405") + ".ffbackup"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, blob, 0600); err != nil {
		return "", err
	}
	if _, err := DecryptBackup(masterKey, blob); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if strings.TrimSpace(copyPath) != "" {
		if err := os.MkdirAll(copyPath, 0700); err == nil {
			_ = copyFile(path, filepath.Join(copyPath, name))
		}
	}
	_ = rotateBackups(dir)
	return path, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	cerr := out.Close()
	if err == nil {
		err = cerr
	}
	return err
}

func rotateBackups(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type item struct {
		name string
		mod  time.Time
	}
	var items []item
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ffbackup") {
			continue
		}
		info, err := e.Info()
		if err == nil {
			items = append(items, item{e.Name(), info.ModTime()})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.After(items[j].mod) })
	keep := map[string]bool{}
	for i, it := range items {
		if i < 14 {
			keep[it.name] = true
		}
	}
	weeks := map[string]bool{}
	for _, it := range items {
		y, w := it.mod.ISOWeek()
		key := fmt.Sprintf("%d-%02d", y, w)
		if !weeks[key] && len(weeks) < 8 {
			keep[it.name] = true
			weeks[key] = true
		}
	}
	for _, it := range items {
		if !keep[it.name] {
			_ = os.Remove(filepath.Join(dir, it.name))
		}
	}
	return nil
}

func (s *Store) ListBackups() ([]map[string]any, error) {
	dir := filepath.Join(s.dataDir, "backups")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ffbackup") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, map[string]any{"name": e.Name(), "size": info.Size(), "modifiedAt": info.ModTime().UTC().Format(time.RFC3339)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["modifiedAt"].(string) > out[j]["modifiedAt"].(string) })
	return out, nil
}
