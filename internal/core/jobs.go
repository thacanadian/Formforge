package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func startJob(store *Store, kind string) string {
	id := RandomID("job_")
	_ = store.Update(func(db *Database) error {
		db.Jobs = append(db.Jobs, BackgroundJob{ID: id, Type: kind, Status: "running", RunAt: nowISO(), Attempts: 1, CreatedAt: nowISO(), UpdatedAt: nowISO()})
		if len(db.Jobs) > 200 {
			db.Jobs = db.Jobs[len(db.Jobs)-200:]
		}
		return nil
	})
	return id
}
func finishJob(store *Store, id string, err error) {
	_ = store.Update(func(db *Database) error {
		for i, j := range db.Jobs {
			if j.ID == id {
				if err != nil {
					j.Status = "failed"
					j.LastError = err.Error()
				} else {
					j.Status = "completed"
					j.LastError = ""
				}
				j.UpdatedAt = nowISO()
				db.Jobs[i] = j
				break
			}
		}
		return nil
	})
}
func VerifyBackups(store *Store, key []byte) error {
	dir := filepath.Join(store.DataDir(), "backups")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ffbackup") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if _, err := DecryptBackup(key, b); err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}
	}
	return nil
}
func RunMaintenanceCycle(store *Store, key []byte) {
	var due bool
	var copyPath, last string
	hours := 24
	_ = store.Read(func(db Database) error {
		copyPath = db.Settings.BackupCopyPath
		last = db.Settings.LastAutoBackupAt
		if db.Settings.BackupIntervalHours > 0 {
			hours = db.Settings.BackupIntervalHours
		}
		if len(db.Users) == 0 {
			return nil
		}
		if last == "" {
			due = true
		} else if t, err := time.Parse(time.RFC3339, last); err != nil || time.Since(t) >= time.Duration(hours)*time.Hour {
			due = true
		}
		return nil
	})
	if due {
		id := startJob(store, "encrypted_backup")
		_, err := store.CreateBackup(key, copyPath)
		if err == nil {
			_ = store.Update(func(db *Database) error { db.Settings.LastAutoBackupAt = nowISO(); return nil })
		}
		finishJob(store, id, err)
	}
	id := startJob(store, "backup_integrity")
	finishJob(store, id, VerifyBackups(store, key))
}
