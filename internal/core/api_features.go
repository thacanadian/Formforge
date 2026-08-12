package core

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ----- Wearable / health imports -----
type appleHealthExport struct {
	Records []struct {
		Type       string `xml:"type,attr"`
		SourceName string `xml:"sourceName,attr"`
		Unit       string `xml:"unit,attr"`
		Value      string `xml:"value,attr"`
		StartDate  string `xml:"startDate,attr"`
		EndDate    string `xml:"endDate,attr"`
	} `xml:"Record"`
}

func normalizeMetricType(x string) string {
	x = strings.ToLower(x)
	switch {
	case strings.Contains(x, "step"):
		return "steps"
	case strings.Contains(x, "heartrate") || strings.Contains(x, "heart_rate"):
		return "heart_rate"
	case strings.Contains(x, "restingheart"):
		return "resting_heart_rate"
	case strings.Contains(x, "sleep"):
		return "sleep"
	case strings.Contains(x, "activeenergy") || strings.Contains(x, "calorie"):
		return "active_calories"
	case strings.Contains(x, "weight") || strings.Contains(x, "bodymass"):
		return "weight"
	case strings.Contains(x, "workout"):
		return "workout"
	case strings.Contains(x, "recovery"):
		return "recovery"
	default:
		return strings.TrimSpace(strings.ReplaceAll(x, "hkquantitytypeidentifier", ""))
	}
}
func parseHealthImport(provider, format string, data []byte, userID string) ([]HealthMetric, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	allowed := map[string]bool{"apple-health": true, "google-fit": true, "health-connect": true, "garmin": true, "whoop": true, "oura": true, "hr-strap": true, "generic": true}
	if !allowed[provider] {
		return nil, errors.New("unsupported health provider")
	}
	out := []HealthMetric{}
	add := func(mt, start, end string, value float64, unit, source string) {
		if mt == "" || start == "" {
			return
		}
		out = append(out, HealthMetric{ID: RandomID("metric_"), UserID: userID, Provider: provider, MetricType: normalizeMetricType(mt), StartAt: start, EndAt: end, Value: value, Unit: unit, Source: source, ImportedAt: nowISO()})
	}
	switch strings.ToLower(format) {
	case "apple-xml", "xml":
		var x appleHealthExport
		if err := xml.Unmarshal(data, &x); err != nil {
			return nil, err
		}
		for _, r := range x.Records {
			v, _ := strconv.ParseFloat(r.Value, 64)
			add(r.Type, r.StartDate, r.EndDate, v, r.Unit, r.SourceName)
		}
	case "json":
		var raw any
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		var rows []map[string]any
		switch v := raw.(type) {
		case []any:
			for _, r := range v {
				if m, ok := r.(map[string]any); ok {
					rows = append(rows, m)
				}
			}
		case map[string]any:
			if a, ok := v["metrics"].([]any); ok {
				for _, r := range a {
					if m, ok := r.(map[string]any); ok {
						rows = append(rows, m)
					}
				}
			}
		}
		for _, m := range rows {
			val, _ := strconv.ParseFloat(fmt.Sprint(m["value"]), 64)
			start := fmt.Sprint(m["startAt"])
			if start == "<nil>" {
				start = fmt.Sprint(m["date"])
			}
			add(fmt.Sprint(m["metricType"]), start, fmt.Sprint(m["endAt"]), val, fmt.Sprint(m["unit"]), fmt.Sprint(m["source"]))
		}
	default:
		r := csv.NewReader(bytes.NewReader(data))
		rows, err := r.ReadAll()
		if err != nil {
			return nil, err
		}
		if len(rows) < 2 {
			return nil, errors.New("CSV needs a header and at least one row")
		}
		idx := map[string]int{}
		for i, h := range rows[0] {
			idx[strings.ToLower(strings.TrimSpace(h))] = i
		}
		cell := func(row []string, names ...string) string {
			for _, n := range names {
				if i, ok := idx[n]; ok && i < len(row) {
					return row[i]
				}
			}
			return ""
		}
		for _, row := range rows[1:] {
			val, _ := strconv.ParseFloat(cell(row, "value", "metricvalue", "amount"), 64)
			add(cell(row, "metrictype", "type", "metric"), cell(row, "startat", "date", "start", "timestamp"), cell(row, "endat", "end"), val, cell(row, "unit"), cell(row, "source", "device"))
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no supported health records were found")
	}
	if len(out) > 20000 {
		out = out[:20000]
	}
	return out, nil
}
func (s *Server) healthProviders(w http.ResponseWriter) {
	jsonOut(w, 200, []map[string]any{{"id": "apple-health", "name": "Apple Health", "modes": []string{"Apple Health export XML"}}, {"id": "google-fit", "name": "Google Fit", "modes": []string{"JSON/CSV export"}}, {"id": "health-connect", "name": "Google Health Connect", "modes": []string{"JSON/CSV export"}}, {"id": "garmin", "name": "Garmin", "modes": []string{"CSV/JSON export"}}, {"id": "whoop", "name": "WHOOP", "modes": []string{"CSV/JSON export"}}, {"id": "oura", "name": "Oura", "modes": []string{"CSV/JSON export"}}, {"id": "hr-strap", "name": "Heart-rate strap", "modes": []string{"CSV session export"}}})
}
func (s *Server) importHealth(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct{ Provider, Format, Data string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	blob, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		blob = []byte(in.Data)
	}
	metrics, err := parseHealthImport(in.Provider, in.Format, blob, cu.User.ID)
	if err != nil {
		jsonError(w, 400, "import_failed", err.Error())
		return
	}
	_ = s.Store.Update(func(db *Database) error {
		db.HealthMetrics = append(db.HealthMetrics, metrics...)
		now := nowISO()
		found := false
		for i, c := range db.WearableConnections {
			if c.UserID == cu.User.ID && c.Provider == strings.ToLower(in.Provider) {
				c.Status = "synced"
				c.LastSyncAt = now
				c.UpdatedAt = now
				db.WearableConnections[i] = c
				found = true
			}
		}
		if !found {
			db.WearableConnections = append(db.WearableConnections, WearableConnection{ID: RandomID("wear_"), UserID: cu.User.ID, Provider: strings.ToLower(in.Provider), Mode: "file-import", Status: "synced", LastSyncAt: now, CreatedAt: now, UpdatedAt: now})
		}
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "health.import", in.Provider, clientIP(r), map[string]any{"records": len(metrics)})
		return nil
	})
	jsonOut(w, 201, map[string]any{"imported": len(metrics)})
}
func (s *Server) listHealth(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	limit := 500
	if n, _ := strconv.Atoi(r.URL.Query().Get("limit")); n > 0 && n <= 5000 {
		limit = n
	}
	out := []HealthMetric{}
	connections := []WearableConnection{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.HealthMetrics {
			if x.UserID == cu.User.ID {
				out = append(out, x)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].StartAt > out[j].StartAt })
		if len(out) > limit {
			out = out[:limit]
		}
		for _, x := range db.WearableConnections {
			if x.UserID == cu.User.ID {
				connections = append(connections, x)
			}
		}
		return nil
	})
	jsonOut(w, 200, map[string]any{"metrics": out, "connections": connections})
}

// ----- Pain flags and progressive overload -----
func (s *Server) listPainFlags(w http.ResponseWriter, cu *contextUser) {
	out := []PainFlag{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.PainFlags {
			if x.UserID == cu.User.ID {
				out = append(out, x)
			}
		}
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) savePainFlag(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in PainFlag
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	in.BodyArea = strings.ToLower(strings.TrimSpace(in.BodyArea))
	if in.BodyArea == "" || in.Severity < 1 || in.Severity > 10 {
		jsonError(w, 400, "invalid_input", "Body area and severity from 1–10 are required.")
		return
	}
	now := nowISO()
	if in.ID == "" {
		in.ID = RandomID("pain_")
		in.CreatedAt = now
	}
	in.UserID = cu.User.ID
	in.Active = true
	in.UpdatedAt = now
	_ = s.Store.Update(func(db *Database) error {
		updated := false
		for i, x := range db.PainFlags {
			if x.ID == in.ID && x.UserID == cu.User.ID {
				in.CreatedAt = x.CreatedAt
				db.PainFlags[i] = in
				updated = true
			}
		}
		if !updated {
			db.PainFlags = append(db.PainFlags, in)
		}
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "pain_flag.save", in.ID, clientIP(r), map[string]any{"area": in.BodyArea, "severity": in.Severity})
		return nil
	})
	jsonOut(w, 200, in)
}
func (s *Server) deletePainFlag(w http.ResponseWriter, r *http.Request, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.PainFlags {
			if x.ID == id && x.UserID == cu.User.ID {
				db.PainFlags = append(db.PainFlags[:i], db.PainFlags[i+1:]...)
				return nil
			}
		}
		return errors.New("pain flag not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func painSubstitution(name string, flags []PainFlag) string {
	n := strings.ToLower(name)
	for _, f := range flags {
		if !f.Active {
			continue
		}
		a := f.BodyArea
		switch {
		case strings.Contains(a, "shoulder") && (strings.Contains(n, "press") || strings.Contains(n, "dip")):
			return "Neutral-grip dumbbell floor press (pain-free range)"
		case strings.Contains(a, "knee") && (strings.Contains(n, "squat") || strings.Contains(n, "lunge") || strings.Contains(n, "leg press")):
			return "Supported box squat or glute bridge (pain-free range)"
		case strings.Contains(a, "back") && (strings.Contains(n, "deadlift") || strings.Contains(n, "row") || strings.Contains(n, "squat")):
			return "Chest-supported row or split squat with light load"
		case strings.Contains(a, "elbow") && (strings.Contains(n, "curl") || strings.Contains(n, "tricep") || strings.Contains(n, "skull")):
			return "Neutral-grip cable/band arm movement"
		case strings.Contains(a, "wrist") && (strings.Contains(n, "press") || strings.Contains(n, "push-up")):
			return "Neutral-grip dumbbell press or push-up handles"
		}
	}
	return name
}
func (s *Server) progressionSuggestions(w http.ResponseWriter, cu *contextUser) {
	type rec struct {
		Exercise   string `json:"exercise"`
		Suggestion string `json:"suggestion"`
		Reason     string `json:"reason"`
	}
	by := map[string][]ExercisePerformance{}
	_ = s.Store.Read(func(db Database) error {
		logs := []WorkoutLog{}
		for _, x := range db.WorkoutLogs {
			if x.UserID == cu.User.ID {
				logs = append(logs, x)
			}
		}
		sort.Slice(logs, func(i, j int) bool { return logs[i].Date > logs[j].Date })
		for _, l := range logs {
			for _, p := range l.Performance {
				k := strings.ToLower(strings.TrimSpace(p.ExerciseName))
				if k != "" && len(by[k]) < 3 {
					by[k] = append(by[k], p)
				}
			}
		}
		return nil
	})
	out := []rec{}
	for k, v := range by {
		last := v[0]
		name := last.ExerciseName
		if name == "" {
			name = k
		}
		sug := "Repeat the load and add one clean rep."
		reason := "More data is needed before increasing load."
		if last.Completed && last.RPE > 0 && last.RPE <= 8 && last.Reps > 0 {
			sug = "Increase the load by the smallest practical amount (about 2–5%)."
			reason = "The latest set was completed with at least two reps in reserve."
		} else if last.RPE >= 9.5 {
			sug = "Keep the load or reduce it 2–5% until technique and reps stabilize."
			reason = "The latest effort was near-maximal."
		}
		if len(v) > 1 && last.Reps < v[1].Reps && last.WeightKG >= v[1].WeightKG {
			sug = "Do not add weight yet; recover and regain the previous reps."
			reason = "Reps declined at the same or higher load."
		}
		out = append(out, rec{name, sug, reason})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Exercise < out[j].Exercise })
	jsonOut(w, 200, out)
}

// ----- Meal plans -----
func generateMealPlan(p Profile, start string, days int, prefs string) MealPlan {
	if days < 1 {
		days = 7
	}
	if days > 14 {
		days = 14
	}
	base := []MealPlanItem{{"Breakfast", "Greek yogurt, oats, berries", "1 bowl", 520, 40, 65, 12}, {"Lunch", "Chicken rice bowl with vegetables", "1 bowl", 650, 50, 75, 15}, {"Snack", "Protein shake and fruit", "1 serving", 280, 28, 35, 4}, {"Dinner", "Salmon, potatoes, and vegetables", "1 plate", 700, 48, 65, 28}}
	vegetarian := strings.Contains(strings.ToLower(prefs), "vegetarian")
	if vegetarian {
		base[1] = MealPlanItem{"Lunch", "Tofu rice bowl with vegetables", "1 bowl", 620, 35, 80, 18}
		base[3] = MealPlanItem{"Dinner", "Lentil pasta with vegetables", "1 plate", 700, 38, 100, 18}
	}
	out := MealPlan{ID: RandomID("mealplan_"), UserID: p.UserID, StartDate: start, Preferences: prefs, CreatedAt: nowISO()}
	t, _ := time.Parse("2006-01-02", start)
	if t.IsZero() {
		t = time.Now()
	}
	for i := 0; i < days; i++ {
		items := append([]MealPlanItem(nil), base...)
		total := 0
		for _, x := range items {
			total += x.Calories
		}
		scale := 1.0
		if p.CalorieGoal > 0 {
			scale = float64(p.CalorieGoal) / float64(total)
		}
		for j := range items {
			items[j].Calories = int(float64(items[j].Calories)*scale + .5)
			items[j].Protein = int(float64(items[j].Protein)*scale + .5)
			items[j].Carbs = int(float64(items[j].Carbs)*scale + .5)
			items[j].Fat = int(float64(items[j].Fat)*scale + .5)
		}
		out.Days = append(out.Days, MealPlanDay{Date: t.AddDate(0, 0, i).Format("2006-01-02"), Items: items, Calories: p.CalorieGoal, Protein: p.ProteinGoal, Carbs: p.CarbGoal, Fat: p.FatGoal})
	}
	return out
}
func (s *Server) mealPlans(w http.ResponseWriter, cu *contextUser) {
	out := []MealPlan{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.MealPlans {
			if x.UserID == cu.User.ID {
				out = append(out, x)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createMealPlan(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct {
		StartDate   string `json:"startDate"`
		Days        int    `json:"days"`
		Preferences string `json:"preferences"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.StartDate == "" {
		in.StartDate = todayISO()
	}
	var plan MealPlan
	_ = s.Store.Update(func(db *Database) error {
		plan = generateMealPlan(db.Profiles[cu.User.ID], in.StartDate, in.Days, in.Preferences)
		db.MealPlans = append(db.MealPlans, plan)
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "meal_plan.create", plan.ID, clientIP(r), map[string]any{"days": len(plan.Days)})
		return nil
	})
	jsonOut(w, 201, plan)
}
func (s *Server) deleteMealPlan(w http.ResponseWriter, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.MealPlans {
			if x.ID == id && x.UserID == cu.User.ID {
				db.MealPlans = append(db.MealPlans[:i], db.MealPlans[i+1:]...)
				return nil
			}
		}
		return errors.New("meal plan not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}

// ----- Encrypted progress photos -----
func encryptLocalBlob(key, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonce)
	return append([]byte("FFPHOTO1\n"), append(nonce, gcm.Seal(nil, nonce, plain, []byte("FORMFORGE-PHOTO-V1"))...)...), nil
}
func decryptLocalBlob(key, blob []byte) ([]byte, error) {
	if len(blob) < 9+12 || string(blob[:9]) != "FFPHOTO1\n" {
		return nil, errors.New("invalid encrypted photo")
	}
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	ns := gcm.NonceSize()
	return gcm.Open(nil, blob[9:9+ns], blob[9+ns:], []byte("FORMFORGE-PHOTO-V1"))
}
func (s *Server) listPhotos(w http.ResponseWriter, cu *contextUser) {
	out := []ProgressPhoto{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.ProgressPhotos {
			if x.UserID == cu.User.ID {
				out = append(out, x)
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Date > out[j].Date })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) uploadPhoto(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct{ Date, Caption, MimeType, Data string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	if in.MimeType != "image/jpeg" && in.MimeType != "image/png" && in.MimeType != "image/webp" {
		jsonError(w, 400, "invalid_type", "Use JPEG, PNG, or WebP.")
		return
	}
	plain, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil || len(plain) == 0 || len(plain) > 8<<20 {
		jsonError(w, 400, "invalid_image", "Image must be valid and no larger than 8 MB.")
		return
	}
	enc, err := encryptLocalBlob(s.MasterKey, plain)
	if err != nil {
		jsonError(w, 500, "encrypt_failed", err.Error())
		return
	}
	id := RandomID("photo_")
	dir := filepath.Join(s.Store.DataDir(), "photos", cu.User.ID)
	_ = os.MkdirAll(dir, 0700)
	path := filepath.Join(dir, id+".ffphoto")
	if err := os.WriteFile(path, enc, 0600); err != nil {
		jsonError(w, 500, "save_failed", err.Error())
		return
	}
	x := ProgressPhoto{ID: id, UserID: cu.User.ID, Date: in.Date, Caption: strings.TrimSpace(in.Caption), EncryptedPath: path, MimeType: in.MimeType, Size: int64(len(plain)), CreatedAt: nowISO()}
	if x.Date == "" {
		x.Date = todayISO()
	}
	_ = s.Store.Update(func(db *Database) error {
		db.ProgressPhotos = append(db.ProgressPhotos, x)
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "progress_photo.create", id, clientIP(r), map[string]any{"size": len(plain)})
		return nil
	})
	jsonOut(w, 201, x)
}
func (s *Server) downloadPhoto(w http.ResponseWriter, cu *contextUser, id string) {
	var x ProgressPhoto
	found := false
	_ = s.Store.Read(func(db Database) error {
		for _, p := range db.ProgressPhotos {
			if p.ID == id && p.UserID == cu.User.ID {
				x = p
				found = true
			}
		}
		return nil
	})
	if !found {
		jsonError(w, 404, "not_found", "Photo not found.")
		return
	}
	blob, err := os.ReadFile(x.EncryptedPath)
	if err != nil {
		jsonError(w, 404, "missing_file", "Encrypted photo file is missing.")
		return
	}
	plain, err := decryptLocalBlob(s.MasterKey, blob)
	if err != nil {
		jsonError(w, 500, "decrypt_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", x.MimeType)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(plain)
}
func (s *Server) deletePhoto(w http.ResponseWriter, cu *contextUser, id string) {
	var path string
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.ProgressPhotos {
			if x.ID == id && x.UserID == cu.User.ID {
				path = x.EncryptedPath
				db.ProgressPhotos = append(db.ProgressPhotos[:i], db.ProgressPhotos[i+1:]...)
				return nil
			}
		}
		return errors.New("photo not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	_ = os.Remove(path)
	jsonOut(w, 200, map[string]any{"ok": true})
}

// ----- Social / household -----
func (s *Server) leaderboard(w http.ResponseWriter, cu *contextUser) {
	type row struct {
		UserID   string `json:"userId"`
		Name     string `json:"name"`
		Workouts int    `json:"workouts"`
		Streak   int    `json:"streak"`
		Habits   int    `json:"habits"`
	}
	out := []row{}
	week := startOfWeek(time.Now())
	_ = s.Store.Read(func(db Database) error {
		for id, u := range db.Users {
			if !u.Active || isBlocked(db, cu.User.ID, id) {
				continue
			}
			r := row{UserID: id, Name: u.Name, Streak: habitStreak(db, id)}
			for _, x := range db.WorkoutLogs {
				if x.UserID == id {
					if t, e := time.Parse("2006-01-02", x.Date); e == nil && !t.Before(week) {
						r.Workouts++
					}
				}
			}
			for _, x := range db.HabitLogs {
				if x.UserID == id && x.Done {
					if t, e := time.Parse("2006-01-02", x.Date); e == nil && !t.Before(week) {
						r.Habits++
					}
				}
			}
			out = append(out, r)
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].Workouts != out[j].Workouts {
			return out[i].Workouts > out[j].Workouts
		}
		return out[i].Streak > out[j].Streak
	})
	jsonOut(w, 200, out)
}
func (s *Server) listNudges(w http.ResponseWriter, cu *contextUser) {
	out := []map[string]any{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.Nudges {
			if x.ModerationStatus == "removed" || isBlocked(db, cu.User.ID, x.FromUserID) || isBlocked(db, cu.User.ID, x.ToUserID) || hiddenForReporter(db, cu.User.ID, "nudge", x.ID) {
				continue
			}
			if x.ToUserID == cu.User.ID || x.FromUserID == cu.User.ID {
				from := db.Users[x.FromUserID].Name
				to := db.Users[x.ToUserID].Name
				out = append(out, map[string]any{"id": x.ID, "fromUserId": x.FromUserID, "toUserId": x.ToUserID, "fromName": from, "toName": to, "message": x.Message, "createdAt": x.CreatedAt, "readAt": x.ReadAt})
			}
		}
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createNudge(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !s.communityGate(w, cu) {
		return
	}
	var in struct{ ToUserID, Message string }
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	in.Message = strings.TrimSpace(in.Message)
	if len(in.Message) > 280 || in.ToUserID == cu.User.ID {
		jsonError(w, 400, "invalid_input", "Choose another user and enter a message under 280 characters.")
		return
	}
	if err := moderateCommunityText(in.Message); err != nil {
		jsonError(w, 400, "content_rejected", err.Error())
		return
	}
	x := Nudge{ID: RandomID("nudge_"), FromUserID: cu.User.ID, ToUserID: in.ToUserID, Message: in.Message, CreatedAt: nowISO(), ModerationStatus: "visible"}
	err := s.Store.Update(func(db *Database) error {
		recipient, ok := db.Users[in.ToUserID]
		if !ok || !recipient.Active {
			return errors.New("recipient not found")
		}
		if isBlocked(*db, cu.User.ID, in.ToUserID) {
			return errors.New("messaging is unavailable between blocked users")
		}

		cutoff := time.Now().Add(-10 * time.Minute)
		count := 0
		for _, n := range db.Nudges {
			if n.FromUserID == cu.User.ID {
				if t, e := time.Parse(time.RFC3339, n.CreatedAt); e == nil && t.After(cutoff) {
					count++
				}
			}
		}
		if count >= 5 {
			return errors.New("rate limit reached; wait before sending more nudges")
		}
		db.Nudges = append(db.Nudges, x)
		actor := db.Users[cu.User.ID]
		s.audit(db, &actor, "social.nudge", in.ToUserID, clientIP(r), nil)
		return nil
	})
	if err != nil {
		code := 400
		if strings.Contains(err.Error(), "not found") {
			code = 404
		}
		if strings.Contains(err.Error(), "rate limit") {
			code = 429
		}
		jsonError(w, code, "nudge_failed", err.Error())
		return
	}
	jsonOut(w, 201, x)
}
func (s *Server) markNudge(w http.ResponseWriter, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.Nudges {
			if x.ID == id && x.ToUserID == cu.User.ID {
				x.ReadAt = nowISO()
				db.Nudges[i] = x
				return nil
			}
		}
		return errors.New("nudge not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (s *Server) listSharedWorkouts(w http.ResponseWriter, cu *contextUser) {
	out := []SharedWorkout{}
	_ = s.Store.Read(func(db Database) error {
		for _, x := range db.SharedWorkouts {
			if x.ModerationStatus == "removed" || isBlocked(db, cu.User.ID, x.CreatedBy) || hiddenForReporter(db, cu.User.ID, "shared_workout", x.ID) {
				continue
			}
			for _, id := range x.ParticipantIDs {
				if id == cu.User.ID {
					out = append(out, x)
					break
				}
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Date+out[i].StartTime < out[j].Date+out[j].StartTime })
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) createSharedWorkout(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if !s.communityGate(w, cu) {
		return
	}
	var in SharedWorkout
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	in.WorkoutName = strings.TrimSpace(in.WorkoutName)
	if in.Date == "" || in.StartTime == "" || in.WorkoutName == "" {
		jsonError(w, 400, "invalid_input", "Workout, date, and start time are required.")
		return
	}
	if len(in.WorkoutName) > 100 {
		jsonError(w, 400, "invalid_input", "Workout name must be under 100 characters.")
		return
	}
	if err := moderateCommunityText(in.WorkoutName); err != nil {
		jsonError(w, 400, "content_rejected", err.Error())
		return
	}
	err := s.Store.Update(func(db *Database) error {
		actor := db.Users[cu.User.ID]
		seen := map[string]bool{cu.User.ID: true}
		ids := []string{cu.User.ID}
		for _, id := range in.ParticipantIDs {
			if seen[id] || isBlocked(*db, cu.User.ID, id) {
				continue
			}
			if u, ok := db.Users[id]; ok && u.Active {
				ids = append(ids, id)
				seen[id] = true
			}
		}
		in.ID = RandomID("shared_")
		in.CreatedBy = cu.User.ID
		in.ParticipantIDs = ids
		in.Status = "scheduled"
		in.ModerationStatus = "visible"
		in.CreatedAt = nowISO()
		in.UpdatedAt = in.CreatedAt
		db.SharedWorkouts = append(db.SharedWorkouts, in)
		s.audit(db, &actor, "social.shared_workout_create", in.ID, clientIP(r), map[string]any{"participants": len(ids)})
		return nil
	})
	if err != nil {
		jsonError(w, 400, "create_failed", err.Error())
		return
	}
	jsonOut(w, 201, in)
}
func (s *Server) deleteSharedWorkout(w http.ResponseWriter, cu *contextUser, id string) {
	err := s.Store.Update(func(db *Database) error {
		for i, x := range db.SharedWorkouts {
			if x.ID == id && (x.CreatedBy == cu.User.ID || cu.User.Role == "admin") {
				db.SharedWorkouts = append(db.SharedWorkouts[:i], db.SharedWorkouts[i+1:]...)
				return nil
			}
		}
		return errors.New("shared workout not found")
	})
	if err != nil {
		jsonError(w, 404, "not_found", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true})
}
func (s *Server) socialUsers(w http.ResponseWriter, cu *contextUser) {
	out := []map[string]any{}
	_ = s.Store.Read(func(db Database) error {
		for _, u := range db.Users {
			if u.Active && !isBlocked(db, cu.User.ID, u.ID) && !communityRestricted(u) {
				out = append(out, map[string]any{"id": u.ID, "name": u.Name, "role": u.Role})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i]["name"].(string) < out[j]["name"].(string) })
		return nil
	})
	jsonOut(w, 200, out)
}
