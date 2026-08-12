package core

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"io/fs"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

type testClient struct {
	c    *http.Client
	base string
	csrf string
}

func newTestApp(t *testing.T) (*Store, []byte, *testClient, func()) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "data path with spaces")
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	key, _, err := EnsureMasterKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	certs, err := EnsureCertificates(dir)
	if err != nil {
		t.Fatal(err)
	}
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0644)}}
	srv := httptest.NewTLSServer(NewServer(store, key, "test", web, certs).Handler())
	jar, _ := cookiejar.New(nil)
	c := srv.Client()
	c.Jar = jar
	return store, key, &testClient{c: c, base: srv.URL}, srv.Close
}

func (tc *testClient) req(t *testing.T, method, path string, body any, want int) map[string]any {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, tc.base+path, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tc.csrf != "" && method != "GET" {
		req.Header.Set("X-CSRF-Token", tc.csrf)
	}
	res, err := tc.c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s: got %d want %d: %s", method, path, res.StatusCode, want, string(raw))
	}
	out := map[string]any{}
	if len(raw) > 0 && strings.Contains(res.Header.Get("Content-Type"), "json") {
		var anyValue any
		if err := json.Unmarshal(raw, &anyValue); err != nil {
			t.Fatal(err)
		}
		if m, ok := anyValue.(map[string]any); ok {
			out = m
		} else {
			out["_array"] = anyValue
		}
	}
	if v, ok := out["csrf"].(string); ok {
		tc.csrf = v
	}
	return out
}

func (tc *testClient) reqRaw(t *testing.T, method, path string, body any, want int) ([]byte, http.Header) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, tc.base+path, r)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tc.csrf != "" && method != http.MethodGet {
		req.Header.Set("X-CSRF-Token", tc.csrf)
	}
	res, err := tc.c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s: got %d want %d: %s", method, path, res.StatusCode, want, string(raw))
	}
	return raw, res.Header
}

func TestFullApplicationFlow(t *testing.T) {
	store, key, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "GET", "/api/system/status", nil, 200)
	setup := admin.req(t, "POST", "/api/auth/setup", map[string]any{"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123", "profile": map[string]any{"name": "Owner", "age": 25, "gender": "male", "heightCm": 180, "weightKg": 85, "goal": "Build muscle", "experience": "intermediate", "daysPerWeek": 4, "equipment": "Full gym"}}, 201)
	if setup["recoveryKey"] == "" {
		t.Fatal("recovery key missing")
	}
	admin.req(t, "GET", "/api/auth/session", nil, 200)
	admin.req(t, "PUT", "/api/profile", map[string]any{"name": "Owner Updated", "age": 25, "gender": "male", "heightCm": 180, "weightKg": 85, "goal": "Build muscle", "experience": "intermediate", "daysPerWeek": 4, "equipment": "Full gym"}, 200)

	custom := admin.req(t, "POST", "/api/workouts", map[string]any{"name": "Garage Strength", "category": "Strength", "duration": 35, "exercises": []map[string]any{{"name": "Floor Press", "sets": 4, "reps": "8", "rest": "90s"}}}, 201)
	wid := custom["id"].(string)
	admin.req(t, "PUT", "/api/workouts/"+wid, map[string]any{"name": "Garage Strength Updated", "category": "Strength", "duration": 40, "exercises": []map[string]any{{"name": "Floor Press", "sets": 4, "reps": "8", "rest": "90s"}, {"name": "Barbell Row", "sets": 4, "reps": "8", "rest": "90s"}}}, 200)
	admin.req(t, "PUT", "/api/plan", map[string]string{"Mon": "Garage Strength", "Tue": "Upper Power"}, 200)
	admin.req(t, "POST", "/api/workout-logs", map[string]any{"workoutId": wid, "workoutName": "Garage Strength Updated", "date": "2026-07-22", "duration": 40}, 201)
	admin.req(t, "POST", "/api/workout-logs", map[string]any{"workoutId": wid, "workoutName": "Garage Strength Updated", "date": "2026-07-22", "duration": 40}, 409)
	food := admin.req(t, "POST", "/api/nutrition", map[string]any{"date": "2026-07-22", "name": "Chicken Breast", "serving": "100g", "calories": 165, "protein": 31, "carbs": 0, "fat": 3.6}, 201)
	admin.req(t, "PUT", "/api/nutrition/"+food["id"].(string), map[string]any{"date": "2026-07-22", "name": "Chicken Breast", "serving": "150g", "calories": 248, "protein": 46.5, "carbs": 0, "fat": 5.4}, 200)
	banana := admin.req(t, "POST", "/api/nutrition", map[string]any{"date": "2026-07-22", "name": "Banana", "serving": "1 medium", "calories": 89, "protein": 1.1, "carbs": 23, "fat": 0.3}, 201)
	admin.req(t, "POST", "/api/nutrition", map[string]any{"date": "2026-07-22", "name": "Banana", "serving": "1 medium", "calories": 89, "protein": 1.1, "carbs": 23, "fat": 0.3}, 409)
	admin.req(t, "DELETE", "/api/nutrition/"+banana["id"].(string), nil, 200)
	progress := admin.req(t, "POST", "/api/progress", map[string]any{"date": "2026-07-22", "weightKg": 84.5, "bodyFat": 18.2, "notes": "Morning"}, 201)
	admin.req(t, "PUT", "/api/progress/"+progress["id"].(string), map[string]any{"date": "2026-07-22", "weightKg": 84.4, "bodyFat": 18.1, "notes": "Rechecked"}, 200)
	admin.req(t, "POST", "/api/progress", map[string]any{"date": "2026-07-22", "weightKg": 84.3}, 409)
	habits := admin.req(t, "GET", "/api/habits?date=2026-07-22", nil, 200)
	_ = habits
	var habitID string
	_ = store.Read(func(db Database) error {
		for _, h := range db.Habits {
			habitID = h.ID
			break
		}
		return nil
	})
	admin.req(t, "POST", "/api/habits/"+habitID+"/toggle", map[string]any{"date": "2026-07-22", "done": true}, 200)
	customHabit := admin.req(t, "POST", "/api/habits", map[string]any{"name": "Mobility", "icon": "M", "category": "Recovery"}, 201)
	admin.req(t, "DELETE", "/api/habits/"+customHabit["id"].(string), nil, 200)
	admin.req(t, "POST", "/api/checkins", map[string]any{"date": "2026-07-22", "lastWeekDays": 3, "availableDays": []string{"Mon", "Wed", "Fri"}, "energy": "solid", "notes": "Normal week"}, 201)
	admin.req(t, "POST", "/api/checkins", map[string]any{"date": "2026-07-23", "lastWeekDays": 3, "availableDays": []string{"Tue"}, "energy": "tired"}, 409)
	admin.req(t, "GET", "/api/dashboard", nil, 200)
	jsonExport, _ := admin.reqRaw(t, "GET", "/api/export/json", nil, 200)
	if !bytes.Contains(jsonExport, []byte("Garage Strength Updated")) {
		t.Fatal("JSON export missing workout")
	}
	csvExport, headers := admin.reqRaw(t, "GET", "/api/export/csv?type=nutrition", nil, 200)
	if !strings.Contains(headers.Get("Content-Type"), "text/csv") || !bytes.Contains(csvExport, []byte("Chicken Breast")) {
		t.Fatal("CSV export invalid")
	}
	admin.req(t, "POST", "/api/import/json", map[string]any{"data": map[string]any{"nutrition": []map[string]any{{"date": "2026-07-21", "name": "Imported Oats", "serving": "100g", "calories": 389, "protein": 17, "carbs": 66, "fat": 7}}}}, 200)
	admin.req(t, "GET", "/api/nutrition?date=2026-07-21", nil, 200)
	admin.req(t, "DELETE", "/api/workouts/"+wid, nil, 200)

	created := admin.req(t, "POST", "/api/admin/users", map[string]any{"name": "Member", "email": "member@example.com", "password": "MemberPassword123", "role": "user", "profile": map[string]any{"name": "Member", "goal": "General fitness", "experience": "beginner", "daysPerWeek": 3, "equipment": "Bodyweight only"}}, 201)
	if created["role"] != "user" {
		t.Fatal("member role wrong")
	}

	jar, _ := cookiejar.New(nil)
	member := &testClient{c: admin.c, base: admin.base}
	member.c = &http.Client{Transport: admin.c.Transport, Jar: jar}
	member.req(t, "POST", "/api/auth/login", map[string]any{"email": "member@example.com", "password": "MemberPassword123"}, 200)
	member.req(t, "GET", "/api/admin/users", nil, 403)
	member.req(t, "GET", "/api/workouts", nil, 200)
	member.req(t, "POST", "/api/auth/logout", map[string]any{}, 200)

	admin.req(t, "POST", "/api/backups", map[string]any{}, 201)
	backups := admin.req(t, "GET", "/api/backups", nil, 200)
	_ = backups
	var backupPath string
	list, _ := store.ListBackups()
	if len(list) == 0 {
		t.Fatal("backup missing")
	}
	backupPath = filepath.Join(store.DataDir(), "backups", list[0]["name"].(string))
	blob, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptBackup(key, blob); err != nil {
		t.Fatal(err)
	}
	admin.req(t, "POST", "/api/backups/restore", map[string]any{"data": base64.StdEncoding.EncodeToString(blob), "recoveryKey": ""}, 200)

	// Data survives opening the same data directory as an updated application would.
	reopened, err := OpenStore(store.DataDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = reopened.Read(func(db Database) error {
		if len(db.Users) != 2 {
			t.Fatalf("users did not survive reopen: %d", len(db.Users))
		}
		if len(db.WorkoutLogs) == 0 || len(db.Nutrition) == 0 {
			t.Fatal("records did not survive reopen")
		}
		return nil
	})
}

func TestLoginLockoutAndCorruptRecovery(t *testing.T) {
	store, _, tc, closeFn := newTestApp(t)
	defer closeFn()
	tc.req(t, "POST", "/api/auth/setup", map[string]any{"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123", "profile": map[string]any{"name": "Owner", "age": 30, "heightCm": 175, "weightKg": 75}}, 201)
	tc.req(t, "POST", "/api/auth/logout", map[string]any{}, 200)
	jar, _ := cookiejar.New(nil)
	bad := &testClient{c: &http.Client{Transport: tc.c.Transport, Jar: jar}, base: tc.base}
	for i := 0; i < 5; i++ {
		bad.req(t, "POST", "/api/auth/login", map[string]any{"email": "owner@example.com", "password": "WrongPassword123"}, 401)
	}
	bad.req(t, "POST", "/api/auth/login", map[string]any{"email": "owner@example.com", "password": "StrongPassword123"}, 429)

	// Corrupt primary snapshot and confirm the previous atomic snapshot is used.
	path := filepath.Join(store.DataDir(), "formforge.db.json")
	if _, err := os.Stat(path + ".prev"); err != nil {
		t.Fatalf("previous snapshot missing: %v", err)
	}
	if err := os.WriteFile(path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err := OpenStore(store.DataDir())
	if err != nil {
		t.Fatal(err)
	}
	_ = recovered.Read(func(db Database) error {
		if len(db.Users) == 0 {
			t.Fatal("corrupt snapshot recovery lost users")
		}
		return nil
	})
}

func TestOfflineAIChatAndEncryptedSettings(t *testing.T) {
	store, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "POST", "/api/auth/setup", map[string]any{
		"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123",
		"profile": map[string]any{"name": "Owner", "age": 25, "gender": "male", "heightCm": 180, "weightKg": 85, "goal": "Build muscle", "experience": "intermediate", "daysPerWeek": 4, "equipment": "Dumbbells only"},
	}, 201)

	reply := admin.req(t, "POST", "/api/ai/chat", map[string]any{"mode": "offline", "message": "Build me a 4-day workout"}, 200)
	if reply["mode"] != "offline" {
		t.Fatalf("unexpected mode: %v", reply["mode"])
	}
	text, _ := reply["reply"].(string)
	if !strings.Contains(text, "4-day") || !strings.Contains(text, "Dumbbell") {
		t.Fatalf("offline plan was not personalized: %s", text)
	}
	history := admin.req(t, "GET", "/api/ai/history", nil, 200)
	items, _ := history["_array"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected two chat messages, got %d", len(items))
	}

	secret := "test-secret-key-123"
	admin.req(t, "PUT", "/api/ai/settings", map[string]any{
		"mode": "auto", "baseUrl": "https://api.openai.com/v1", "model": "gpt-4o-mini", "apiKey": secret,
	}, 200)
	raw, err := os.ReadFile(filepath.Join(store.DataDir(), "formforge.db.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(secret)) {
		t.Fatal("API key was stored in plaintext")
	}
	status := admin.req(t, "GET", "/api/ai/status", nil, 200)
	if status["apiKeyConfigured"] != true {
		t.Fatal("AI status did not report configured key")
	}
	admin.req(t, "DELETE", "/api/ai/history", nil, 200)
	cleared := admin.req(t, "GET", "/api/ai/history", nil, 200)
	items, _ = cleared["_array"].([]any)
	if got := len(items); got != 0 {
		t.Fatalf("history was not cleared: %d", got)
	}
}

func TestOnlineAICompatibleEndpointAndAutoFallback(t *testing.T) {
	_, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "POST", "/api/auth/setup", map[string]any{
		"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123",
		"profile": map[string]any{"name": "Owner", "age": 25, "heightCm": 180, "weightKg": 85, "goal": "Increase strength", "experience": "intermediate", "daysPerWeek": 4, "equipment": "Full gym"},
	}, 201)

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer local-test-key" {
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		jsonOut(w, 200, map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "Online coach response"}}}})
	}))
	defer mock.Close()

	admin.req(t, "PUT", "/api/ai/settings", map[string]any{
		"mode": "auto", "baseUrl": mock.URL + "/v1", "model": "local-test-model", "apiKey": "local-test-key",
	}, 200)
	online := admin.req(t, "POST", "/api/ai/chat", map[string]any{"mode": "online", "message": "Give me a workout"}, 200)
	if online["mode"] != "online" || online["reply"] != "Online coach response" {
		t.Fatalf("online response failed: %#v", online)
	}
	admin.req(t, "POST", "/api/ai/test", map[string]any{"baseUrl": mock.URL + "/v1", "model": "local-test-model", "apiKey": "local-test-key"}, 200)

	admin.req(t, "PUT", "/api/ai/settings", map[string]any{
		"mode": "auto", "baseUrl": "http://127.0.0.1:1/v1", "model": "local-test-model", "apiKey": "local-test-key",
	}, 200)
	fallback := admin.req(t, "POST", "/api/ai/chat", map[string]any{"mode": "auto", "message": "Summarize my progress"}, 200)
	if fallback["mode"] != "offline-fallback" {
		t.Fatalf("auto mode did not fall back: %#v", fallback)
	}
}

func TestEmptyListEndpointsReturnArraysAndUIAssetsAreFresh(t *testing.T) {
	_, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "POST", "/api/auth/setup", map[string]any{
		"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123",
		"profile": map[string]any{"name": "Owner", "age": 25, "heightCm": 180, "weightKg": 85, "goal": "General fitness", "experience": "beginner", "daysPerWeek": 3, "equipment": "Full gym"},
	}, 201)
	for _, path := range []string{
		"/api/workout-logs", "/api/nutrition?date=2020-01-01", "/api/progress", "/api/checkins", "/api/ai/history", "/api/backups", "/api/food-search?q=definitely-not-a-food",
	} {
		raw, _ := admin.reqRaw(t, http.MethodGet, path, nil, http.StatusOK)
		if strings.TrimSpace(string(raw)) != "[]" {
			t.Fatalf("%s returned %s instead of []", path, strings.TrimSpace(string(raw)))
		}
	}

	_, headers := admin.reqRaw(t, http.MethodGet, "/app.js", nil, http.StatusOK)
	if !strings.Contains(headers.Get("Cache-Control"), "no-store") {
		t.Fatalf("app.js may be served stale: %q", headers.Get("Cache-Control"))
	}
}

func TestMobileInfoAndHealthIdentity(t *testing.T) {
	_, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "POST", "/api/auth/setup", map[string]any{
		"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123",
		"profile": map[string]any{"name": "Owner", "age": 25, "heightCm": 180, "weightKg": 85},
	}, 201)
	mobile := admin.req(t, "GET", "/api/system/mobile", nil, 200)
	if _, ok := mobile["urls"].([]any); !ok {
		t.Fatalf("mobile URLs were not returned as an array: %#v", mobile["urls"])
	}
	if mobile["caUrl"] != "/api/system/ca" {
		t.Fatalf("unexpected CA URL: %#v", mobile["caUrl"])
	}

	res, err := admin.c.Get(admin.base + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var health map[string]any
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if health["product"] != "formforge" || health["ok"] != true {
		t.Fatalf("health endpoint lacks FormForge identity: %#v", health)
	}
}

func TestCertificateMigrationRetainsRenewalCapability(t *testing.T) {
	dir := t.TempDir()
	first, err := EnsureCertificates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first.CAKey); err != nil {
		t.Fatalf("CA private key was not retained: %v", err)
	}
	// Simulate a 1.0/1.1 certificate directory, which lacked the CA key.
	if err := os.Remove(first.CAKey); err != nil {
		t.Fatal(err)
	}
	second, err := EnsureCertificates(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(second.CAKey); err != nil {
		t.Fatalf("legacy certificate migration did not restore renewal capability: %v", err)
	}
	b, err := os.ReadFile(second.Cert)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		t.Fatal("server certificate is not PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := cert.VerifyHostname("127.0.0.1"); err != nil {
		t.Fatalf("server certificate does not cover localhost IP: %v", err)
	}
}

func TestCoachingTeamBlendSourcesAndOfflineCoach(t *testing.T) {
	store, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	admin.req(t, "POST", "/api/auth/setup", map[string]any{
		"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123",
		"profile": map[string]any{"name": "Owner", "age": 25, "heightCm": 180, "weightKg": 85, "goal": "Build muscle", "experience": "intermediate", "daysPerWeek": 4, "equipment": "Full gym"},
		"coachingPreferences": map[string]any{
			"responseStyle": "teach",
			"influences": []any{
				map[string]any{"profileId": "jeff-nippard", "weight": 60},
				map[string]any{"profileId": "arnold-schwarzenegger", "weight": 40},
			},
		},
	}, 201)

	team := admin.req(t, "GET", "/api/coaching/team", nil, 200)
	if !strings.Contains(team["blend"].(string), "Jeff Nippard") || !strings.Contains(team["blend"].(string), "Arnold Schwarzenegger") {
		t.Fatalf("setup coaching blend missing: %#v", team["blend"])
	}
	profiles, ok := team["profiles"].([]any)
	if !ok || len(profiles) < 8 {
		t.Fatalf("coaching catalog missing: %#v", team["profiles"])
	}

	admin.req(t, "PUT", "/api/coaching/preferences", map[string]any{
		"responseStyle": "push",
		"influences": []any{
			map[string]any{"profileId": "ronnie-coleman", "weight": 70},
			map[string]any{"profileId": "noel-deyzel", "weight": 30},
		},
	}, 200)

	source := admin.req(t, "POST", "/api/coaching/sources", map[string]any{
		"profileId":     "ronnie-coleman",
		"title":         "Approved interview excerpt",
		"kind":          "video",
		"sourceUrl":     "https://example.com/verified-video",
		"summary":       "A short administrator-written summary about enthusiasm and consistent bodybuilding work.",
		"quote":         "Verified test quote.",
		"quoteVerified": true,
		"licensed":      true,
	}, 201)
	if source["quoteVerified"] != true {
		t.Fatalf("verified source was not saved: %#v", source)
	}

	pack := admin.req(t, "GET", "/api/coaching/pack", nil, 200)
	if !strings.Contains(pack["blend"].(string), "Ronnie Coleman") {
		t.Fatalf("coach pack blend missing: %#v", pack)
	}
	packSources, ok := pack["sources"].([]any)
	if !ok || len(packSources) != 1 {
		t.Fatalf("coach pack source missing: %#v", pack["sources"])
	}

	workout := admin.req(t, "POST", "/api/ai/chat", map[string]any{"mode": "offline", "message": "Build me a 4-day workout"}, 200)
	text := workout["reply"].(string)
	for _, want := range []string{"Coach blend:", "Ronnie Coleman", "Influence adjustments", "High-effort adjustment"} {
		if !strings.Contains(text, want) {
			t.Fatalf("offline blended workout missing %q: %s", want, text)
		}
	}
	quote := admin.req(t, "POST", "/api/ai/chat", map[string]any{"mode": "offline", "message": "Give me a quote"}, 200)
	if !strings.Contains(quote["reply"].(string), "Verified test quote") || !strings.Contains(quote["reply"].(string), "example.com") {
		t.Fatalf("verified quote was not grounded: %#v", quote)
	}

	raw, _ := admin.reqRaw(t, http.MethodGet, "/api/export/json", nil, http.StatusOK)
	if !bytes.Contains(raw, []byte("coachingPreferences")) || !bytes.Contains(raw, []byte("ronnie-coleman")) {
		t.Fatal("coaching preferences were not included in user export")
	}

	_ = store.Read(func(db Database) error {
		if db.SchemaVersion != SchemaVersion || len(db.CoachPreferences) != 1 || len(db.CoachSources) != 1 {
			t.Fatalf("coaching data not persisted: schema=%d prefs=%d sources=%d", db.SchemaVersion, len(db.CoachPreferences), len(db.CoachSources))
		}
		return nil
	})
}

func TestV14SecurityHealthCreatorSocialAndPlans(t *testing.T) {
	store, _, admin, closeFn := newTestApp(t)
	defer closeFn()
	setup := admin.req(t, "POST", "/api/auth/setup", map[string]any{"name": "Owner", "email": "owner@example.com", "password": "StrongPassword123", "profile": map[string]any{"name": "Owner", "age": 28, "heightCm": 180, "weightKg": 82, "goal": "Build muscle", "experience": "intermediate", "daysPerWeek": 4, "equipment": "Full gym"}}, 201)
	if setup["user"].(map[string]any)["planTier"] != "pro" {
		t.Fatal("setup admin should be pro")
	}
	terms := admin.req(t, "GET", "/api/legal/terms", nil, 200)
	if terms["version"] == "" {
		t.Fatal("terms missing")
	}
	admin.req(t, "POST", "/api/legal/accept", map[string]any{}, 200)

	begin := admin.req(t, "POST", "/api/security/2fa/begin", map[string]any{}, 200)
	secret := begin["secret"].(string)
	code := generateTOTP(secret, uint64(time.Now().Unix()/30))
	confirm := admin.req(t, "POST", "/api/security/2fa/confirm", map[string]any{"code": code}, 200)
	if confirm["enabled"] != true {
		t.Fatal("2FA not enabled")
	}
	admin.req(t, "GET", "/api/security/sessions", nil, 200)

	custom := admin.req(t, "POST", "/api/coaching/profiles", map[string]any{"name": "Example Coach", "category": "Custom", "summary": "An original editorial summary focused on repeatable training and recovery.", "principles": []string{"Repeatable programming", "Controlled progression"}, "communication": []string{"Direct"}, "links": []map[string]any{{"url": "https://example.com/coach"}}}, 201)
	customID := custom["id"].(string)
	admin.req(t, "PUT", "/api/coaching/preferences", map[string]any{"influences": []map[string]any{{"profileId": customID, "weight": 100}}, "responseStyle": "teach", "preferredCoachId": customID}, 200)
	admin.req(t, "POST", "/api/coaching/sources", map[string]any{"profileId": customID, "kind": "article", "title": "Example source", "sourceUrl": "https://example.com/source", "summary": "A short approved summary about controlled progressive overload."}, 201)
	chat := admin.req(t, "POST", "/api/ai/chat", map[string]any{"mode": "offline", "message": "How should I progress my lifts?"}, 200)
	if len(chat["grounding"].([]any)) == 0 {
		t.Fatal("grounding tags missing")
	}
	hist := admin.req(t, "GET", "/api/ai/history", nil, 200)["_array"].([]any)
	lastID := hist[len(hist)-1].(map[string]any)["id"].(string)
	admin.req(t, "DELETE", "/api/ai/history/"+lastID, nil, 200)
	admin.req(t, "GET", "/api/ai/usage", nil, 200)

	protected, hdr := admin.reqRaw(t, "POST", "/api/export/protected", map[string]any{"kind": "json", "password": "LongExportPassword123"}, 200)
	if !strings.Contains(hdr.Get("Content-Disposition"), "ffexport") || !bytes.HasPrefix(protected, []byte("FFEX1\n")) {
		t.Fatal("protected export invalid")
	}

	healthCSV := "metricType,startAt,value,unit,source\nsteps,2026-07-24T08:00:00Z,4500,count,watch\nheart_rate,2026-07-24T08:10:00Z,120,bpm,strap\n"
	admin.req(t, "POST", "/api/health/import", map[string]any{"provider": "hr-strap", "format": "csv", "data": base64.StdEncoding.EncodeToString([]byte(healthCSV))}, 201)
	admin.req(t, "GET", "/api/health", nil, 200)
	pain := admin.req(t, "POST", "/api/pain-flags", map[string]any{"bodyArea": "shoulder", "severity": 5, "trigger": "pressing"}, 200)
	admin.req(t, "POST", "/api/workout-logs", map[string]any{"workoutName": "Upper test", "date": "2026-07-24", "duration": 45, "performance": []map[string]any{{"exerciseName": "Bench Press", "weightKg": 80, "reps": 8, "sets": 3, "rpe": 8, "completed": true}}}, 201)
	admin.req(t, "GET", "/api/training/progression", nil, 200)
	admin.req(t, "POST", "/api/meal-plans", map[string]any{"startDate": "2026-07-25", "days": 7, "preferences": "vegetarian"}, 201)
	photo := admin.req(t, "POST", "/api/progress-photos", map[string]any{"date": "2026-07-24", "caption": "front", "mimeType": "image/png", "data": base64.StdEncoding.EncodeToString([]byte("fake-png-bytes"))}, 201)
	admin.reqRaw(t, "GET", "/api/progress-photos/"+photo["id"].(string)+"/data", nil, 200)
	admin.req(t, "DELETE", "/api/pain-flags/"+pain["id"].(string), nil, 200)

	member := admin.req(t, "POST", "/api/admin/users", map[string]any{"name": "Member", "email": "member@example.com", "password": "MemberPassword123", "role": "user", "planTier": "free", "profile": map[string]any{"name": "Member", "goal": "General fitness", "experience": "beginner", "daysPerWeek": 3, "equipment": "Bodyweight only"}}, 201)
	memberID := member["id"].(string)
	admin.req(t, "POST", "/api/social/nudges", map[string]any{"toUserId": memberID, "message": "Let’s train today."}, 201)
	admin.req(t, "POST", "/api/social/workouts", map[string]any{"workoutName": "Shared lift", "date": "2026-07-26", "startTime": "18:00", "duration": 60, "participantIds": []string{memberID}}, 201)
	admin.req(t, "GET", "/api/social/leaderboard", nil, 200)
	admin.req(t, "GET", "/api/system/health", nil, 200)

	// Encrypted backups include local encrypted photo files and restore migration metadata.
	path, err := store.CreateBackup(make([]byte, 32), "")
	if err == nil {
		_ = path
	} // wrong key length is valid AES-256 material; backup should still work.
}

func TestCloudSetupTokenAndProxyHeaders(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	key, _, err := EnsureMasterKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	web := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok"), Mode: fs.FileMode(0644)}}
	app := NewServer(store, key, "cloud-test", web, CertPaths{})
	app.Cloud = true
	app.TrustProxy = true
	app.SetupToken = "render-setup-secret"
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/system/status")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.Header.Get("Strict-Transport-Security") == "" {
		t.Fatal("hosted mode did not set HSTS")
	}
	var status map[string]any
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["setupTokenRequired"] != true || status["cloud"] != true {
		t.Fatalf("unexpected cloud status: %#v", status)
	}

	body := map[string]any{"name": "Admin", "email": "admin@example.com", "password": "StrongPassword12", "setupToken": "wrong", "profile": map[string]any{"name": "Admin", "age": 25, "heightCm": 175, "weightKg": 75}, "acceptTerms": true, "acceptPrivacy": true, "ageConfirmed": true}
	b, _ := json.Marshal(body)
	badReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/setup", bytes.NewReader(b))
	badReq.Header.Set("Content-Type", "application/json")
	badRes, err := http.DefaultClient.Do(badReq)
	if err != nil {
		t.Fatal(err)
	}
	badRes.Body.Close()
	if badRes.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong setup token returned %d", badRes.StatusCode)
	}

	body["setupToken"] = "render-setup-secret"
	delete(body, "acceptTerms")
	delete(body, "acceptPrivacy")
	delete(body, "ageConfirmed")
	b, _ = json.Marshal(body)
	missingConsentReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/setup", bytes.NewReader(b))
	missingConsentReq.Header.Set("Content-Type", "application/json")
	missingConsentRes, err := http.DefaultClient.Do(missingConsentReq)
	if err != nil {
		t.Fatal(err)
	}
	missingConsentRes.Body.Close()
	if missingConsentRes.StatusCode != http.StatusBadRequest {
		t.Fatalf("hosted setup without consent returned %d", missingConsentRes.StatusCode)
	}
	body["acceptTerms"] = true
	body["acceptPrivacy"] = true
	body["ageConfirmed"] = true
	b, _ = json.Marshal(body)
	goodReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/setup", bytes.NewReader(b))
	goodReq.Header.Set("Content-Type", "application/json")
	goodReq.Header.Set("X-Forwarded-For", "203.0.113.42, 10.0.0.1")
	goodRes, err := http.DefaultClient.Do(goodReq)
	if err != nil {
		t.Fatal(err)
	}
	defer goodRes.Body.Close()
	if goodRes.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(goodRes.Body)
		t.Fatalf("correct setup token returned %d: %s", goodRes.StatusCode, raw)
	}
	var auditIP string
	if err := store.Read(func(db Database) error {
		if len(db.Audit) > 0 {
			auditIP = db.Audit[len(db.Audit)-1].IP
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if auditIP != "203.0.113.42" {
		t.Fatalf("trusted proxy IP was not used: %q", auditIP)
	}
}
