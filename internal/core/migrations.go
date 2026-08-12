package core

import (
	"errors"
	"fmt"
	"time"
)

type Migration struct {
	Version int
	Name    string
	Apply   func(*Database) error
}

var migrations = []Migration{
	{2, "shared-user-and-audit-foundation", func(*Database) error { return nil }},
	{3, "coaching-team-and-ai-history", func(*Database) error { return nil }},
	{4, "custom-influencer-import", func(db *Database) error {
		if db.CustomCoachProfiles == nil {
			db.CustomCoachProfiles = map[string]CustomCoachProfile{}
		}
		return nil
	}},
	{5, "security-health-social-commercial-controls", func(db *Database) error {
		for id, u := range db.Users {
			if u.PlanTier == "" {
				u.PlanTier = "free"
			}
			db.Users[id] = u
		}
		return nil
	}},
	{6, "plans-terms-and-ai-usage-controls", func(db *Database) error {
		for id, u := range db.Users {
			if u.Role == "admin" {
				u.PlanTier = "pro"
			} else if u.PlanTier == "" {
				u.PlanTier = "free"
			}
			db.Users[id] = u
		}
		if db.Settings.TermsVersion == "" {
			db.Settings.TermsVersion = "1.4"
		}
		return nil
	}},
	{7, "local-agent-theme-and-marketplace", func(db *Database) error {
		if db.ThemePreferences == nil {
			db.ThemePreferences = map[string]ThemePreferences{}
		}
		return nil
	}},
	{8, "measurement-system-and-focused-member-interface", func(db *Database) error {
		if db.ThemePreferences == nil {
			db.ThemePreferences = map[string]ThemePreferences{}
		}
		for id, pref := range db.ThemePreferences {
			if pref.MeasurementSystem == "" {
				pref.MeasurementSystem = "imperial"
			}
			if pref.NavigationMode == "" {
				pref.NavigationMode = "focused"
			}
			db.ThemePreferences[id] = pref
		}
		return nil
	}},
	{9, "legal-consent-account-deletion-and-community-moderation", func(db *Database) error {
		if db.UserBlocks == nil {
			db.UserBlocks = []UserBlock{}
		}
		if db.ContentReports == nil {
			db.ContentReports = []ContentReport{}
		}
		if db.ModerationActions == nil {
			db.ModerationActions = []ModerationAction{}
		}
		if db.Settings.TermsVersion == "" || db.Settings.TermsVersion == "1.4" || db.Settings.TermsVersion == "1.0" {
			db.Settings.TermsVersion = "2.0"
		}
		if db.Settings.PrivacyVersion == "" {
			db.Settings.PrivacyVersion = "1.0"
		}
		if db.Settings.CommunityVersion == "" {
			db.Settings.CommunityVersion = "1.0"
		}
		if db.Settings.SubscriptionVersion == "" {
			db.Settings.SubscriptionVersion = "1.0"
		}
		for i := range db.Nudges {
			if db.Nudges[i].ModerationStatus == "" {
				db.Nudges[i].ModerationStatus = "visible"
			}
		}
		for i := range db.SharedWorkouts {
			if db.SharedWorkouts[i].ModerationStatus == "" {
				db.SharedWorkouts[i].ModerationStatus = "visible"
			}
		}
		return nil
	}},
}

func MigrateDatabase(db *Database) ([]MigrationRecord, error) {
	if db.SchemaVersion <= 0 {
		db.SchemaVersion = 1
	}
	if db.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("database schema %d is newer than supported schema %d", db.SchemaVersion, SchemaVersion)
	}
	var out []MigrationRecord
	for _, m := range migrations {
		if m.Version <= db.SchemaVersion {
			continue
		}
		if m.Version != db.SchemaVersion+1 {
			return out, errors.New("database migration chain is incomplete")
		}
		if err := m.Apply(db); err != nil {
			return out, fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
		}
		r := MigrationRecord{Version: m.Version, Name: m.Name, AppliedAt: time.Now().UTC().Format(time.RFC3339)}
		db.Migrations = append(db.Migrations, r)
		out = append(out, r)
		db.SchemaVersion = m.Version
	}
	return out, nil
}
