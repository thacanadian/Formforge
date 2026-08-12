package core

import (
	"errors"
	"net/http"
	"sort"
	"strings"
)

type coachingTeamResponse struct {
	Profiles    []CoachProfile   `json:"profiles"`
	Preferences CoachPreferences `json:"preferences"`
	Sources     []CoachSource    `json:"sources"`
	Blend       string           `json:"blend"`
	Disclosure  string           `json:"disclosure"`
}

func (s *Server) coachingTeam(w http.ResponseWriter, cu *contextUser) {
	var out coachingTeamResponse
	_ = s.Store.Read(func(db Database) error {
		prefs := preferencesFor(db, cu.User.ID)
		counts := map[string]int{}
		for _, src := range db.CoachSources {
			counts[src.ProfileID]++
		}
		profiles := ExpandedCoachCatalog()
		for _, cp := range db.CustomCoachProfiles {
			if cp.Status != "removed" {
				profiles = append(profiles, coachProfileFromCustom(cp, counts[cp.ID]))
			}
		}
		for i := range profiles {
			profiles[i].SourceCount = counts[profiles[i].ID]
		}
		sources := append([]CoachSource(nil), db.CoachSources...)
		sort.Slice(sources, func(i, j int) bool { return sources[i].UpdatedAt > sources[j].UpdatedAt })
		out = coachingTeamResponse{
			Profiles:    profiles,
			Preferences: prefs,
			Sources:     sources,
			Blend:       coachBlendSummary(db, cu.User.ID),
			Disclosure:  "Editorial profiles summarize broad training themes and do not imply sponsorship, endorsement, or identity imitation. Exact quotes are used only when an administrator marks a source as verified.",
		}
		return nil
	})
	jsonOut(w, 200, out)
}

func (s *Server) putCoachingPreferences(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in CoachPreferences
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var clean CoachPreferences
	var err error
	_ = s.Store.Read(func(db Database) error { clean, err = normalizeCoachPreferencesForDB(in, cu.User.ID, db); return nil })
	if err != nil {
		jsonError(w, 400, "invalid_input", err.Error())
		return
	}
	if err := s.Store.Update(func(db *Database) error {
		db.CoachPreferences[cu.User.ID] = clean
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "coaching.preferences_update", cu.User.ID, clientIP(r), map[string]any{"style": clean.ResponseStyle, "influences": len(clean.Influences)})
		return nil
	}); err != nil {
		jsonError(w, 500, "save_failed", err.Error())
		return
	}
	jsonOut(w, 200, clean)
}

func (s *Server) coachingPack(w http.ResponseWriter, cu *contextUser) {
	var out map[string]any
	_ = s.Store.Read(func(db Database) error {
		out = map[string]any{
			"preferences": preferencesFor(db, cu.User.ID),
			"profiles":    selectedCoachProfiles(db, cu.User.ID),
			"sources":     approvedCoachSources(db, cu.User.ID),
			"blend":       coachBlendSummary(db, cu.User.ID),
			"version":     s.Version,
			"offline":     true,
			"disclosure":  "This coach pack uses editorial summaries and approved sources. It does not reproduce a creator's exact identity or imply endorsement.",
		}
		return nil
	})
	jsonOut(w, 200, out)
}

func (s *Server) createCoachSource(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in CoachSource
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	clean, err := validateCoachSource(in)
	if err != nil {
		jsonError(w, 400, "invalid_input", err.Error())
		return
	}
	clean.ID = RandomID("coachsrc_")
	clean.AddedBy = cu.User.ID
	clean.CreatedAt = nowISO()
	clean.UpdatedAt = clean.CreatedAt
	if err := s.Store.Update(func(db *Database) error {
		if _, ok := coachCatalogMapFor(*db)[clean.ProfileID]; !ok {
			return errors.New("choose a valid coaching profile")
		}
		db.CoachSources = append(db.CoachSources, clean)
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "coaching.source_create", clean.ID, clientIP(r), map[string]any{"profileId": clean.ProfileID, "kind": clean.Kind, "licensed": clean.Licensed, "quoteVerified": clean.QuoteVerified})
		return nil
	}); err != nil {
		jsonError(w, 500, "save_failed", err.Error())
		return
	}
	jsonOut(w, 201, clean)
}

func (s *Server) updateCoachSource(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	var in CoachSource
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	clean, err := validateCoachSource(in)
	if err != nil {
		jsonError(w, 400, "invalid_input", err.Error())
		return
	}
	var out CoachSource
	err = s.Store.Update(func(db *Database) error {
		if _, ok := coachCatalogMapFor(*db)[clean.ProfileID]; !ok {
			return errors.New("choose a valid coaching profile")
		}
		for i, src := range db.CoachSources {
			if src.ID != id {
				continue
			}
			clean.ID = src.ID
			clean.AddedBy = src.AddedBy
			clean.CreatedAt = src.CreatedAt
			clean.UpdatedAt = nowISO()
			db.CoachSources[i] = clean
			out = clean
			u := db.Users[cu.User.ID]
			s.audit(db, &u, "coaching.source_update", id, clientIP(r), map[string]any{"profileId": clean.ProfileID, "licensed": clean.Licensed, "quoteVerified": clean.QuoteVerified})
			return nil
		}
		return errors.New("source not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}

func (s *Server) deleteCoachSource(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	id = strings.TrimSpace(id)
	err := s.Store.Update(func(db *Database) error {
		for i, src := range db.CoachSources {
			if src.ID != id {
				continue
			}
			db.CoachSources = append(db.CoachSources[:i], db.CoachSources[i+1:]...)
			u := db.Users[cu.User.ID]
			s.audit(db, &u, "coaching.source_delete", id, clientIP(r), map[string]any{"profileId": src.ProfileID})
			return nil
		}
		return errors.New("source not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
