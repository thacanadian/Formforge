package core

import (
	"errors"
	"net/http"
	"strings"
)

func (s *Server) previewCoachLink(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct {
		URL string `json:"url"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	x, err := previewPublicCreatorURL(in.URL)
	if err != nil {
		jsonError(w, 400, "invalid_link", err.Error())
		return
	}
	jsonOut(w, 200, x)
}

func (s *Server) listCustomCoachProfiles(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	out := []CustomCoachProfile{}
	_ = s.Store.Read(func(db Database) error {
		for _, p := range db.CustomCoachProfiles {
			if p.Status != "removed" {
				out = append(out, p)
			}
		}
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createCustomCoachProfile(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	var in CustomCoachProfile
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	clean, err := validateCustomCoach(in)
	if err != nil {
		jsonError(w, 400, "invalid_input", err.Error())
		return
	}
	clean.ID = "custom-" + strings.TrimPrefix(RandomID(""), "")
	clean.AddedBy = cu.User.ID
	clean.CreatedAt = nowISO()
	clean.UpdatedAt = clean.CreatedAt
	err = s.Store.Update(func(db *Database) error {
		for _, p := range ExpandedCoachCatalog() {
			if strings.EqualFold(p.Name, clean.Name) {
				return errors.New("a built-in profile already uses this name")
			}
		}
		for _, p := range db.CustomCoachProfiles {
			if strings.EqualFold(p.Name, clean.Name) {
				return errors.New("a custom profile already uses this name")
			}
		}
		db.CustomCoachProfiles[clean.ID] = clean
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "coaching.profile_create", clean.ID, clientIP(r), map[string]any{"links": len(clean.Links), "licensed": clean.Licensed})
		return nil
	})
	if err != nil {
		jsonError(w, 409, "duplicate", err.Error())
		return
	}
	jsonOut(w, 201, clean)
}
func (s *Server) updateCustomCoachProfile(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	var in CustomCoachProfile
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	clean, err := validateCustomCoach(in)
	if err != nil {
		jsonError(w, 400, "invalid_input", err.Error())
		return
	}
	var out CustomCoachProfile
	err = s.Store.Update(func(db *Database) error {
		old, ok := db.CustomCoachProfiles[id]
		if !ok {
			return errors.New("custom profile not found")
		}
		clean.ID = id
		clean.AddedBy = old.AddedBy
		clean.CreatedAt = old.CreatedAt
		clean.UpdatedAt = nowISO()
		db.CustomCoachProfiles[id] = clean
		out = clean
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "coaching.profile_update", id, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}
func (s *Server) deleteCustomCoachProfile(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	err := s.Store.Update(func(db *Database) error {
		p, ok := db.CustomCoachProfiles[id]
		if !ok {
			return errors.New("custom profile not found")
		}
		p.Status = "removed"
		p.UpdatedAt = nowISO()
		db.CustomCoachProfiles[id] = p
		for uid, prefs := range db.CoachPreferences {
			n := prefs.Influences[:0]
			for _, x := range prefs.Influences {
				if x.ProfileID != id {
					n = append(n, x)
				}
			}
			prefs.Influences = n
			if prefs.PreferredCoachID == id {
				prefs.PreferredCoachID = "formforge-balanced"
			}
			db.CoachPreferences[uid] = prefs
		}
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "coaching.profile_remove", id, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func (s *Server) submitTakedown(w http.ResponseWriter, r *http.Request) {
	var in TakedownRequest
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	in.ProfileID = strings.TrimSpace(in.ProfileID)
	in.RequesterName = strings.TrimSpace(in.RequesterName)
	in.Contact = strings.TrimSpace(in.Contact)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.ProfileID == "" || in.RequesterName == "" || in.Contact == "" || len(in.Reason) < 20 {
		jsonError(w, 400, "invalid_input", "Profile, requester name, contact information, and a detailed reason are required.")
		return
	}
	in.ID = RandomID("takedown_")
	in.Status = "open"
	in.CreatedAt = nowISO()
	_ = s.Store.Update(func(db *Database) error {
		db.TakedownRequests = append(db.TakedownRequests, in)
		s.audit(db, nil, "coaching.takedown_submit", in.ProfileID, clientIP(r), map[string]any{"contact": in.Contact})
		return nil
	})
	jsonOut(w, 201, map[string]any{"id": in.ID, "status": in.Status, "message": "The request was recorded for administrator review."})
}
func (s *Server) listTakedowns(w http.ResponseWriter, cu *contextUser) {
	if !requireAdmin(w, cu) {
		return
	}
	out := []TakedownRequest{}
	_ = s.Store.Read(func(db Database) error { out = append(out, db.TakedownRequests...); return nil })
	jsonOut(w, 200, out)
}
func (s *Server) updateTakedown(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	if !requireAdmin(w, cu) {
		return
	}
	var in struct{ Status, Notes string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Status != "open" && in.Status != "reviewing" && in.Status != "resolved" && in.Status != "rejected" {
		jsonError(w, 400, "invalid_status", "Invalid status.")
		return
	}
	var out TakedownRequest
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.TakedownRequests {
			if x.ID == id {
				x.Status = in.Status
				x.Notes = strings.TrimSpace(in.Notes)
				if in.Status == "resolved" || in.Status == "rejected" {
					x.ResolvedAt = nowISO()
				}
				db.TakedownRequests[i] = x
				out = x
				if in.Status == "resolved" {
					if p, ok := db.CustomCoachProfiles[x.ProfileID]; ok {
						p.Status = "removed"
						p.UpdatedAt = nowISO()
						db.CustomCoachProfiles[p.ID] = p
					}
				}
				u := db.Users[cu.User.ID]
				s.audit(db, &u, "coaching.takedown_update", id, clientIP(r), map[string]any{"status": in.Status})
				return nil
			}
		}
		return errors.New("takedown request not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}
