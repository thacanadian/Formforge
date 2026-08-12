package core

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func encryptPasswordExport(password string, plain []byte) ([]byte, error) {
	if len(password) < 10 {
		return nil, errors.New("export password must be at least 10 characters")
	}
	salt := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, salt)
	key := pbkdf2SHA256([]byte(password), salt, 300000, 32)
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
	sum := sha256.Sum256(plain)
	ct := gcm.Seal(nil, nonce, plain, []byte("FORMFORGE-EXPORT-V1"))
	out := append([]byte("FFEX1\n"), salt...)
	out = append(out, nonce...)
	out = append(out, sum[:]...)
	out = append(out, ct...)
	return out, nil
}
func decryptPasswordExport(password string, blob []byte) ([]byte, error) {
	if len(blob) < 6+16+12+32 || string(blob[:6]) != "FFEX1\n" {
		return nil, errors.New("not a password-protected FormForge export")
	}
	salt := blob[6:22]
	key := pbkdf2SHA256([]byte(password), salt, 300000, 32)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	ns := gcm.NonceSize()
	nonce := blob[22 : 22+ns]
	expected := blob[22+ns : 22+ns+32]
	plain, err := gcm.Open(nil, nonce, blob[22+ns+32:], []byte("FORMFORGE-EXPORT-V1"))
	if err != nil {
		return nil, errors.New("export password is incorrect or the file is damaged")
	}
	sum := sha256.Sum256(plain)
	if !equalBytes(sum[:], expected) {
		return nil, errors.New("export integrity check failed")
	}
	return plain, nil
}

func userExportPayload(db Database, userID, version string) map[string]any {
	payload := map[string]any{"version": version, "exportedAt": nowISO(), "profile": db.Profiles[userID], "plan": db.WeeklyPlans[userID], "coachingPreferences": preferencesFor(db, userID)}
	workouts := []Workout{}
	for _, x := range db.Workouts {
		if x.OwnerID == userID {
			workouts = append(workouts, x)
		}
	}
	payload["customWorkouts"] = workouts
	wl := []WorkoutLog{}
	for _, x := range db.WorkoutLogs {
		if x.UserID == userID {
			wl = append(wl, x)
		}
	}
	payload["workoutLogs"] = wl
	n := []NutritionEntry{}
	for _, x := range db.Nutrition {
		if x.UserID == userID {
			n = append(n, x)
		}
	}
	payload["nutrition"] = n
	h := []Habit{}
	for _, x := range db.Habits {
		if x.UserID == userID {
			h = append(h, x)
		}
	}
	payload["habits"] = h
	hl := []HabitLog{}
	for _, x := range db.HabitLogs {
		if x.UserID == userID {
			hl = append(hl, x)
		}
	}
	payload["habitLogs"] = hl
	pr := []ProgressEntry{}
	for _, x := range db.Progress {
		if x.UserID == userID {
			pr = append(pr, x)
		}
	}
	payload["progress"] = pr
	ci := []CheckIn{}
	for _, x := range db.CheckIns {
		if x.UserID == userID {
			ci = append(ci, x)
		}
	}
	payload["checkIns"] = ci
	ch := []ChatMessage{}
	for _, x := range db.ChatMessages {
		if x.UserID == userID {
			ch = append(ch, x)
		}
	}
	payload["chatMessages"] = ch
	pf := []PainFlag{}
	for _, x := range db.PainFlags {
		if x.UserID == userID {
			pf = append(pf, x)
		}
	}
	payload["painFlags"] = pf
	hm := []HealthMetric{}
	for _, x := range db.HealthMetrics {
		if x.UserID == userID {
			hm = append(hm, x)
		}
	}
	payload["healthMetrics"] = hm
	mp := []MealPlan{}
	for _, x := range db.MealPlans {
		if x.UserID == userID {
			mp = append(mp, x)
		}
	}
	payload["mealPlans"] = mp
	return payload
}

func buildUserCSV(db Database, userID, kind string) ([]byte, error) {
	var b bytes.Buffer
	cw := csv.NewWriter(&b)
	switch kind {
	case "nutrition":
		_ = cw.Write([]string{"date", "name", "serving", "calories", "protein", "carbs", "fat"})
		for _, x := range db.Nutrition {
			if x.UserID == userID {
				_ = cw.Write([]string{x.Date, x.Name, x.Serving, fmt.Sprint(x.Calories), fmt.Sprint(x.Protein), fmt.Sprint(x.Carbs), fmt.Sprint(x.Fat)})
			}
		}
	case "progress":
		_ = cw.Write([]string{"date", "weightKg", "bodyFat", "notes"})
		for _, x := range db.Progress {
			if x.UserID == userID {
				_ = cw.Write([]string{x.Date, fmt.Sprint(x.WeightKG), fmt.Sprint(x.BodyFat), x.Notes})
			}
		}
	case "workouts":
		_ = cw.Write([]string{"date", "workout", "duration", "notes"})
		for _, x := range db.WorkoutLogs {
			if x.UserID == userID {
				_ = cw.Write([]string{x.Date, x.WorkoutName, strconv.Itoa(x.Duration), x.Notes})
			}
		}
	case "health":
		_ = cw.Write([]string{"provider", "metricType", "startAt", "endAt", "value", "unit", "source"})
		for _, x := range db.HealthMetrics {
			if x.UserID == userID {
				_ = cw.Write([]string{x.Provider, x.MetricType, x.StartAt, x.EndAt, fmt.Sprint(x.Value), x.Unit, x.Source})
			}
		}
	default:
		return nil, errors.New("unsupported CSV export type")
	}
	cw.Flush()
	return b.Bytes(), cw.Error()
}

func (s *Server) protectedExport(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct {
		Kind     string `json:"kind"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var plain []byte
	var err error
	_ = s.Store.Read(func(db Database) error {
		if in.Kind == "json" || in.Kind == "" {
			plain, err = json.MarshalIndent(userExportPayload(db, cu.User.ID, s.Version), "", "  ")
		} else {
			plain, err = buildUserCSV(db, cu.User.ID, strings.TrimPrefix(in.Kind, "csv:"))
		}
		return nil
	})
	if err != nil {
		jsonError(w, 400, "export_failed", err.Error())
		return
	}
	blob, err := encryptPasswordExport(in.Password, plain)
	if err != nil {
		jsonError(w, 400, "export_failed", err.Error())
		return
	}
	u := cu.User
	_ = s.Store.Update(func(db *Database) error {
		s.audit(db, &u, "data.export_protected", in.Kind, clientIP(r), map[string]any{"bytes": len(plain)})
		return nil
	})
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="FormForge-protected.ffexport"`)
	_, _ = w.Write(blob)
}
func (s *Server) protectedImport(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct {
		Password string `json:"password"`
		Data     string `json:"data"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	blob, err := base64.StdEncoding.DecodeString(in.Data)
	if err != nil {
		jsonError(w, 400, "invalid_file", "Protected export data is invalid.")
		return
	}
	plain, err := decryptPasswordExport(in.Password, blob)
	if err != nil {
		jsonError(w, 400, "decrypt_failed", err.Error())
		return
	}
	var probe map[string]json.RawMessage
	if json.Unmarshal(plain, &probe) != nil {
		jsonError(w, 400, "unsupported_import", "This protected export is CSV and cannot be imported as FormForge JSON.")
		return
	}
	r2 := r.Clone(r.Context())
	body, _ := json.Marshal(map[string]any{"data": json.RawMessage(plain)})
	r2.Body = io.NopCloser(bytes.NewReader(body))
	s.importJSON(w, r2, cu)
}
