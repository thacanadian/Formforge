package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

func (s *Server) dashboard(w http.ResponseWriter, cu *contextUser) {
	date := todayISO()
	weekStart := startOfWeek(time.Now())
	out := map[string]any{}
	_ = s.Store.Read(func(db Database) error {
		p := db.Profiles[cu.User.ID]
		out["profile"] = p
		out["plan"] = db.WeeklyPlans[cu.User.ID]
		var calories, protein, carbs, fat float64
		var nutrition []NutritionEntry
		for _, e := range db.Nutrition {
			if e.UserID == cu.User.ID && e.Date == date {
				nutrition = append(nutrition, e)
				calories += e.Calories
				protein += e.Protein
				carbs += e.Carbs
				fat += e.Fat
			}
		}
		workouts := 0
		var recent []WorkoutLog
		for _, e := range db.WorkoutLogs {
			if e.UserID == cu.User.ID {
				recent = append(recent, e)
				if t, err := time.Parse("2006-01-02", e.Date); err == nil && !t.Before(weekStart) {
					workouts++
				}
			}
		}
		sort.Slice(recent, func(i, j int) bool { return recent[i].Date > recent[j].Date })
		if len(recent) > 5 {
			recent = recent[:5]
		}
		var habits []Habit
		done := 0
		for _, h := range db.Habits {
			if h.UserID == cu.User.ID {
				habits = append(habits, h)
			}
		}
		for _, h := range db.HabitLogs {
			if h.UserID == cu.User.ID && h.Date == date && h.Done {
				done++
			}
		}
		streak := habitStreak(db, cu.User.ID)
		out["today"] = map[string]any{"date": date, "nutrition": nutrition, "calories": calories, "protein": protein, "carbs": carbs, "fat": fat, "habitDone": done, "habitTotal": len(habits)}
		out["weeklyWorkouts"] = workouts
		out["streak"] = streak
		out["recentWorkouts"] = recent
		return nil
	})
	jsonOut(w, 200, out)
}

func startOfWeek(t time.Time) time.Time {
	d := int(t.Weekday())
	if d == 0 {
		d = 7
	}
	x := t.AddDate(0, 0, 1-d)
	return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, x.Location())
}
func habitStreak(db Database, userID string) int {
	streak := 0
	for i := 0; i < 365; i++ {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		any := false
		for _, h := range db.HabitLogs {
			if h.UserID == userID && h.Date == d && h.Done {
				any = true
				break
			}
		}
		if any {
			streak++
		} else if i > 0 {
			return streak
		}
	}
	return streak
}

func (s *Server) getProfile(w http.ResponseWriter, cu *contextUser) {
	_ = s.Store.Read(func(db Database) error { jsonOut(w, 200, db.Profiles[cu.User.ID]); return nil })
}
func (s *Server) putProfile(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var p Profile
	if err := readJSON(r, &p); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	p = normalizeProfile(p, cu.User)
	err := s.Store.Update(func(db *Database) error {
		db.Profiles[cu.User.ID] = p
		u := db.Users[cu.User.ID]
		if strings.TrimSpace(p.Name) != "" {
			u.Name = p.Name
			u.UpdatedAt = nowISO()
			db.Users[u.ID] = u
		}
		s.audit(db, &u, "profile.update", u.ID, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 500, "save_failed", err.Error())
		return
	}
	jsonOut(w, 200, p)
}

func (s *Server) listWorkouts(w http.ResponseWriter, cu *contextUser) {
	out := []Workout{}
	_ = s.Store.Read(func(db Database) error {
		p := db.Profiles[cu.User.ID]
		for _, x := range db.Workouts {
			if x.BuiltIn && (x.Level == p.Experience || p.Experience == "") {
				out = append(out, x)
			} else if x.OwnerID == cu.User.ID {
				out = append(out, x)
			}
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].BuiltIn != out[j].BuiltIn {
				return out[i].BuiltIn
			}
			return out[i].Name < out[j].Name
		})
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createWorkout(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var x Workout
	if err := readJSON(r, &x); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if strings.TrimSpace(x.Name) == "" || len(x.Exercises) == 0 {
		jsonError(w, 400, "invalid_input", "Workout name and at least one exercise are required.")
		return
	}
	x.ID = RandomID("workout_")
	x.OwnerID = cu.User.ID
	x.BuiltIn = false
	x.Level = "custom"
	x.CreatedAt = nowISO()
	x.UpdatedAt = x.CreatedAt
	_ = s.Store.Update(func(db *Database) error {
		db.Workouts[x.ID] = x
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "workout.create", x.ID, clientIP(r), map[string]any{"name": x.Name})
		return nil
	})
	jsonOut(w, 201, x)
}
func (s *Server) updateWorkout(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	var in Workout
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var out Workout
	err := s.Store.Update(func(db *Database) error {
		x, ok := db.Workouts[id]
		if !ok || x.BuiltIn || x.OwnerID != cu.User.ID {
			return errors.New("custom workout not found")
		}
		if strings.TrimSpace(in.Name) == "" || len(in.Exercises) == 0 {
			return errors.New("name and exercises are required")
		}
		x.Name = in.Name
		x.Category = in.Category
		x.Duration = in.Duration
		x.Why = in.Why
		x.Exercises = in.Exercises
		x.UpdatedAt = nowISO()
		db.Workouts[id] = x
		out = x
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "workout.update", id, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}
func (s *Server) deleteWorkout(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		x, ok := db.Workouts[id]
		if !ok || x.BuiltIn || x.OwnerID != cu.User.ID {
			return errors.New("custom workout not found")
		}
		delete(db.Workouts, id)
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "workout.delete", id, clientIP(r), nil)
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func (s *Server) getPlan(w http.ResponseWriter, cu *contextUser) {
	_ = s.Store.Read(func(db Database) error {
		p := db.WeeklyPlans[cu.User.ID]
		if p == nil {
			p = map[string]string{}
		}
		jsonOut(w, 200, p)
		return nil
	})
}
func (s *Server) putPlan(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var p map[string]string
	if err := readJSON(r, &p); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	allowed := map[string]bool{"Mon": true, "Tue": true, "Wed": true, "Thu": true, "Fri": true, "Sat": true, "Sun": true}
	clean := map[string]string{}
	for k, v := range p {
		if allowed[k] {
			clean[k] = strings.TrimSpace(v)
		}
	}
	_ = s.Store.Update(func(db *Database) error {
		db.WeeklyPlans[cu.User.ID] = clean
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "plan.update", cu.User.ID, clientIP(r), nil)
		return nil
	})
	jsonOut(w, 200, clean)
}

func (s *Server) listWorkoutLogs(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	out := []WorkoutLog{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.WorkoutLogs {
			if x.UserID != cu.User.ID {
				continue
			}
			if from != "" && x.Date < from {
				continue
			}
			if to != "" && x.Date > to {
				continue
			}
			out = append(out, x)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createWorkoutLog(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var x WorkoutLog
	if err := readJSON(r, &x); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if x.Date == "" {
		x.Date = todayISO()
	}
	if !validDate(x.Date) || strings.TrimSpace(x.WorkoutName) == "" {
		jsonError(w, 400, "invalid_input", "Valid date and workout name are required.")
		return
	}
	x.ID = RandomID("wlog_")
	x.UserID = cu.User.ID
	x.CompletedAt = nowISO()
	err := s.Store.Update(func(db *Database) error {
		for _, e := range db.WorkoutLogs {
			if e.UserID == x.UserID && e.Date == x.Date && e.WorkoutName == x.WorkoutName {
				return errors.New("that workout is already logged for this date")
			}
		}
		db.WorkoutLogs = append(db.WorkoutLogs, x)
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "workout_log.create", x.ID, clientIP(r), map[string]any{"name": x.WorkoutName, "date": x.Date})
		return nil
	})
	if err != nil {
		jsonError(w, 409, "duplicate", err.Error())
		return
	}
	jsonOut(w, 201, x)
}
func (s *Server) deleteWorkoutLog(w http.ResponseWriter, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		n := db.WorkoutLogs[:0]
		found := false
		for _, x := range db.WorkoutLogs {
			if x.ID == id && x.UserID == cu.User.ID {
				found = true
				continue
			}
			n = append(n, x)
		}
		db.WorkoutLogs = n
		if !found {
			return errors.New("workout log not found")
		}
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func (s *Server) listNutrition(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = todayISO()
	}
	out := []NutritionEntry{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.Nutrition {
			if x.UserID == cu.User.ID && x.Date == date {
				out = append(out, x)
			}
		}
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createNutrition(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var x NutritionEntry
	if err := readJSON(r, &x); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if x.Date == "" {
		x.Date = todayISO()
	}
	if !validDate(x.Date) || strings.TrimSpace(x.Name) == "" || x.Calories < 0 {
		jsonError(w, 400, "invalid_input", "Name, valid date, and non-negative nutrition values are required.")
		return
	}
	x.ID = RandomID("food_")
	x.UserID = cu.User.ID
	x.CreatedAt = nowISO()
	x.UpdatedAt = x.CreatedAt
	err := s.Store.Update(func(db *Database) error {
		for _, e := range db.Nutrition {
			if e.UserID == x.UserID && e.Date == x.Date && strings.EqualFold(e.Name, x.Name) && e.Serving == x.Serving && time.Since(parseTime(e.CreatedAt)) < time.Minute {
				return errors.New("possible duplicate food entry")
			}
		}
		db.Nutrition = append(db.Nutrition, x)
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "nutrition.create", x.ID, clientIP(r), map[string]any{"name": x.Name})
		return nil
	})
	if err != nil {
		jsonError(w, 409, "duplicate", err.Error())
		return
	}
	jsonOut(w, 201, x)
}
func parseTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }
func (s *Server) updateNutrition(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	var in NutritionEntry
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var out NutritionEntry
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.Nutrition {
			if x.ID == id && x.UserID == cu.User.ID {
				in.ID = x.ID
				in.UserID = x.UserID
				in.CreatedAt = x.CreatedAt
				in.UpdatedAt = nowISO()
				if in.Date == "" {
					in.Date = x.Date
				}
				db.Nutrition[i] = in
				out = in
				return nil
			}
		}
		return errors.New("nutrition entry not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}
func (s *Server) deleteNutrition(w http.ResponseWriter, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		n := db.Nutrition[:0]
		found := false
		for _, x := range db.Nutrition {
			if x.ID == id && x.UserID == cu.User.ID {
				found = true
				continue
			}
			n = append(n, x)
		}
		db.Nutrition = n
		if !found {
			return errors.New("nutrition entry not found")
		}
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (s *Server) foodSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	out := []NutritionEntry{}
	for k, v := range CalorieDB {
		if q == "" || strings.Contains(k, q) || strings.Contains(strings.ToLower(v.Name), q) {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	jsonOut(w, 200, out)
}

// foodLookup preserves the source application's name/barcode workflow while
// keeping third-party calls on the backend. Only the search term or barcode is
// sent to Open Food Facts; no FormForge account or customer data is included.
func (s *Server) foodLookup(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" || len(q) > 160 {
		jsonError(w, 400, "invalid_input", "Enter a food name or an 8-14 digit barcode.")
		return
	}
	client := &http.Client{Timeout: 8 * time.Second}
	var endpoint string
	isBarcode := true
	for _, ch := range q {
		if ch < '0' || ch > '9' {
			isBarcode = false
			break
		}
	}
	if isBarcode && len(q) >= 8 && len(q) <= 14 {
		endpoint = "https://world.openfoodfacts.org/api/v2/product/" + q + ".json"
	} else {
		endpoint = "https://world.openfoodfacts.org/cgi/search.pl?search_terms=" + url.QueryEscape(q) + "&search_simple=1&action=process&json=1&page_size=1"
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		jsonError(w, 500, "lookup_failed", "Food lookup could not be prepared.")
		return
	}
	req.Header.Set("User-Agent", "FormForge/"+s.Version+" (local fitness application)")
	resp, err := client.Do(req)
	if err != nil {
		jsonError(w, 502, "lookup_failed", "Food lookup is unavailable. Check the internet connection or enter nutrition manually.")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		jsonError(w, 502, "lookup_failed", "The food database returned an error.")
		return
	}
	var payload struct {
		Product  offProduct   `json:"product"`
		Products []offProduct `json:"products"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, resp.Body, 2<<20)).Decode(&payload); err != nil {
		jsonError(w, 502, "lookup_failed", "The food database response could not be read.")
		return
	}
	p := payload.Product
	if p.Name == "" && len(payload.Products) > 0 {
		p = payload.Products[0]
	}
	if strings.TrimSpace(p.Name) == "" {
		jsonError(w, 404, "not_found", "No matching food was found. Try another name or enter nutrition manually.")
		return
	}
	name := strings.TrimSpace(p.Name)
	if strings.TrimSpace(p.Brands) != "" {
		name += " · " + strings.TrimSpace(p.Brands)
	}
	out := NutritionEntry{Name: name, Serving: "100 g", Calories: p.Nutrients.Calories, Protein: p.Nutrients.Protein, Carbs: p.Nutrients.Carbs, Fat: p.Nutrients.Fat}
	jsonOut(w, 200, out)
}

type offProduct struct {
	Name      string `json:"product_name"`
	Brands    string `json:"brands"`
	Nutrients struct {
		Calories float64 `json:"energy-kcal_100g"`
		Protein  float64 `json:"proteins_100g"`
		Carbs    float64 `json:"carbohydrates_100g"`
		Fat      float64 `json:"fat_100g"`
	} `json:"nutriments"`
}

func (s *Server) listHabits(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = todayISO()
	}
	habits := []map[string]any{}
	_ = s.Store.Read(func(db Database) error {
		for _, h := range db.Habits {
			if h.UserID != cu.User.ID {
				continue
			}
			done := false
			for _, l := range db.HabitLogs {
				if l.UserID == cu.User.ID && l.HabitID == h.ID && l.Date == date {
					done = l.Done
					break
				}
			}
			b, _ := json.Marshal(h)
			var m map[string]any
			_ = json.Unmarshal(b, &m)
			m["done"] = done
			habits = append(habits, m)
		}
		return nil
	})
	jsonOut(w, 200, habits)
}
func (s *Server) createHabit(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var h Habit
	if err := readJSON(r, &h); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if strings.TrimSpace(h.Name) == "" {
		jsonError(w, 400, "invalid_input", "Habit name is required.")
		return
	}
	h.ID = RandomID("habit_")
	h.UserID = cu.User.ID
	h.CreatedAt = nowISO()
	h.UpdatedAt = h.CreatedAt
	_ = s.Store.Update(func(db *Database) error { db.Habits = append(db.Habits, h); return nil })
	jsonOut(w, 201, h)
}
func (s *Server) updateHabit(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	var in Habit
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var out Habit
	err := s.Store.Update(func(db *Database) error {
		for i, h := range db.Habits {
			if h.ID == id && h.UserID == cu.User.ID {
				h.Name = in.Name
				h.Icon = in.Icon
				h.Category = in.Category
				h.UpdatedAt = nowISO()
				db.Habits[i] = h
				out = h
				return nil
			}
		}
		return errors.New("habit not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}
func (s *Server) deleteHabit(w http.ResponseWriter, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		n := db.Habits[:0]
		found := false
		for _, h := range db.Habits {
			if h.ID == id && h.UserID == cu.User.ID {
				found = true
				continue
			}
			n = append(n, h)
		}
		db.Habits = n
		if !found {
			return errors.New("habit not found")
		}
		ln := db.HabitLogs[:0]
		for _, l := range db.HabitLogs {
			if l.HabitID != id {
				ln = append(ln, l)
			}
		}
		db.HabitLogs = ln
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (s *Server) toggleHabit(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	var in struct {
		Date string
		Done bool
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.Date == "" {
		in.Date = todayISO()
	}
	var out HabitLog
	err := s.Store.Update(func(db *Database) error {
		exists := false
		for _, h := range db.Habits {
			if h.ID == id && h.UserID == cu.User.ID {
				exists = true
			}
		}
		if !exists {
			return errors.New("habit not found")
		}
		for i, l := range db.HabitLogs {
			if l.UserID == cu.User.ID && l.HabitID == id && l.Date == in.Date {
				l.Done = in.Done
				l.UpdatedAt = nowISO()
				db.HabitLogs[i] = l
				out = l
				return nil
			}
		}
		out = HabitLog{ID: RandomID("hlog_"), UserID: cu.User.ID, HabitID: id, Date: in.Date, Done: in.Done, UpdatedAt: nowISO()}
		db.HabitLogs = append(db.HabitLogs, out)
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}

func (s *Server) listProgress(w http.ResponseWriter, cu *contextUser) {
	out := []ProgressEntry{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.Progress {
			if x.UserID == cu.User.ID {
				out = append(out, x)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createProgress(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var x ProgressEntry
	if err := readJSON(r, &x); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if x.Date == "" {
		x.Date = todayISO()
	}
	if x.WeightKG <= 0 || !validDate(x.Date) {
		jsonError(w, 400, "invalid_input", "Weight and a valid date are required.")
		return
	}
	x.ID = RandomID("progress_")
	x.UserID = cu.User.ID
	x.CreatedAt = nowISO()
	x.UpdatedAt = x.CreatedAt
	err := s.Store.Update(func(db *Database) error {
		for _, e := range db.Progress {
			if e.UserID == x.UserID && e.Date == x.Date {
				return errors.New("a progress entry already exists for this date")
			}
		}
		db.Progress = append(db.Progress, x)
		return nil
	})
	if err != nil {
		jsonError(w, 409, "duplicate", err.Error())
		return
	}
	jsonOut(w, 201, x)
}
func (s *Server) updateProgress(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	var in ProgressEntry
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var out ProgressEntry
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.Progress {
			if x.ID == id && x.UserID == cu.User.ID {
				in.ID = x.ID
				in.UserID = x.UserID
				in.CreatedAt = x.CreatedAt
				in.UpdatedAt = nowISO()
				db.Progress[i] = in
				out = in
				return nil
			}
		}
		return errors.New("progress entry not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, out)
}
func (s *Server) deleteProgress(w http.ResponseWriter, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		n := db.Progress[:0]
		found := false
		for _, x := range db.Progress {
			if x.ID == id && x.UserID == cu.User.ID {
				found = true
				continue
			}
			n = append(n, x)
		}
		db.Progress = n
		if !found {
			return errors.New("progress entry not found")
		}
		return nil
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

func (s *Server) listCheckins(w http.ResponseWriter, cu *contextUser) {
	out := []CheckIn{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.CheckIns {
			if x.UserID == cu.User.ID {
				out = append(out, x)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createCheckin(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var x CheckIn
	if err := readJSON(r, &x); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if x.Date == "" {
		x.Date = todayISO()
	}
	if len(x.AvailableDays) == 0 || x.Energy == "" {
		jsonError(w, 400, "invalid_input", "Available days and energy are required.")
		return
	}
	x.ID = RandomID("checkin_")
	x.UserID = cu.User.ID
	x.CreatedAt = nowISO()
	x.Recommendation = s.buildRecommendation(cu.User.ID, x)
	err := s.Store.Update(func(db *Database) error {
		y, wk := isoWeekForDate(x.Date)
		for _, previous := range db.CheckIns {
			py, pw := isoWeekForDate(previous.Date)
			if previous.UserID == x.UserID && y == py && wk == pw {
				return errors.New("a weekly check-in already exists for this week")
			}
		}
		db.CheckIns = append(db.CheckIns, x)
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "checkin.create", x.ID, clientIP(r), map[string]any{"energy": x.Energy, "days": len(x.AvailableDays)})
		return nil
	})
	if err != nil {
		jsonError(w, 409, "duplicate", err.Error())
		return
	}
	jsonOut(w, 201, x)
}
func isoWeekForDate(date string) (int, int) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0, 0
	}
	return t.ISOWeek()
}

func (s *Server) buildRecommendation(userID string, c CheckIn) string {
	var p Profile
	var names []string
	_ = s.Store.Read(func(db Database) error {
		p = db.Profiles[userID]
		for _, w := range db.Workouts {
			if w.BuiltIn && w.Level == p.Experience {
				names = append(names, w.Name)
			}
		}
		return nil
	})
	sort.Strings(names)
	if len(names) == 0 {
		names = []string{"Full Body Strength A", "Cardio Foundation"}
	}
	intensity := "normal training volume"
	switch c.Energy {
	case "crushed":
		intensity = "a hard but controlled week"
	case "tired":
		intensity = "a lighter week with one fewer hard set per exercise"
	case "burnt":
		intensity = "a recovery week at roughly 60% of normal volume"
	}
	var lines []string
	for i, d := range c.AvailableDays {
		lines = append(lines, fmt.Sprintf("%s — %s", d, names[i%len(names)]))
	}
	return fmt.Sprintf("You completed %d session(s) last week. Based on your %s energy rating, use %s.\n\nThis week:\n%s\n\nFocus: finish each session with clean technique and leave 1–3 reps in reserve.", c.LastWeekDays, c.Energy, intensity, strings.Join(lines, "\n"))
}
