package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

var themePresets = map[string]ThemePreferences{
	"forge":    {Preset: "forge", Accent: "#ff7a1a", Background: "#08090a", Surface: "#111315", Text: "#f2e7dc", Radius: 10, Density: "comfortable", MeasurementSystem: "imperial", NavigationMode: "focused"},
	"midnight": {Preset: "midnight", Accent: "#7c5cff", Background: "#080b17", Surface: "#12182b", Text: "#f4f6ff", Radius: 18, Density: "comfortable", MeasurementSystem: "imperial", NavigationMode: "focused"},
	"iron":     {Preset: "iron", Accent: "#d4af37", Background: "#111111", Surface: "#222222", Text: "#f5f1e6", Radius: 6, Density: "compact", MeasurementSystem: "imperial", NavigationMode: "focused"},
	"arctic":   {Preset: "arctic", Accent: "#0077ff", Background: "#f2f6fb", Surface: "#ffffff", Text: "#152033", Radius: 16, Density: "comfortable", MeasurementSystem: "imperial", NavigationMode: "focused"},
	"forest":   {Preset: "forest", Accent: "#43a047", Background: "#0d1711", Surface: "#17251b", Text: "#eff8f0", Radius: 12, Density: "comfortable", MeasurementSystem: "imperial", NavigationMode: "focused"},
}

func defaultThemeForUser(userID string) ThemePreferences {
	x := themePresets["forge"]
	x.UserID = userID
	return x
}

func normalizeThemePreferences(x ThemePreferences, userID string) (ThemePreferences, error) {
	measurement := strings.ToLower(strings.TrimSpace(x.MeasurementSystem))
	if measurement == "" {
		measurement = "imperial"
	}
	if measurement != "imperial" && measurement != "metric" {
		return x, fmt.Errorf("measurement system must be imperial or metric")
	}
	navigation := strings.ToLower(strings.TrimSpace(x.NavigationMode))
	if navigation == "" {
		navigation = "focused"
	}
	if navigation != "focused" && navigation != "full" {
		return x, fmt.Errorf("navigation mode must be focused or full")
	}
	if preset, ok := themePresets[x.Preset]; ok {
		// A preset changes only visual values. Units and navigation remain user choices.
		preset.MeasurementSystem = measurement
		preset.NavigationMode = navigation
		x = preset
	}
	if x.Accent == "" || x.Background == "" || x.Surface == "" || x.Text == "" {
		return x, fmt.Errorf("choose valid colors")
	}
	if x.Radius < 0 || x.Radius > 30 {
		return x, fmt.Errorf("corner radius must be between 0 and 30")
	}
	if x.Density != "compact" && x.Density != "comfortable" {
		x.Density = "comfortable"
	}
	x.MeasurementSystem = measurement
	x.NavigationMode = navigation
	x.UserID = userID
	x.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return x, nil
}

func (s *Server) getTheme(w http.ResponseWriter, cu *contextUser) {
	x := defaultThemeForUser(cu.User.ID)
	_ = s.Store.Read(func(db Database) error {
		if saved, ok := db.ThemePreferences[cu.User.ID]; ok {
			x = saved
		}
		return nil
	})
	if x.MeasurementSystem == "" {
		x.MeasurementSystem = "imperial"
	}
	if x.NavigationMode == "" {
		x.NavigationMode = "focused"
	}
	jsonOut(w, 200, map[string]any{"current": x, "presets": themePresets})
}
func (s *Server) putTheme(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var x ThemePreferences
	if readJSON(r, &x) != nil {
		jsonError(w, 400, "invalid_json", "Invalid appearance preferences.")
		return
	}
	normalized, err := normalizeThemePreferences(x, cu.User.ID)
	if err != nil {
		jsonError(w, 400, "invalid_preferences", err.Error())
		return
	}
	_ = s.Store.Update(func(db *Database) error { db.ThemePreferences[cu.User.ID] = normalized; return nil })
	jsonOut(w, 200, normalized)
}

func recoveryScore(db Database, uid string) map[string]any {
	score := 70.0
	reasons := []string{}
	cutoff := time.Now().Add(-48 * time.Hour)
	var sleep, hrv float64
	var n int
	for _, m := range db.HealthMetrics {
		if m.UserID != uid {
			continue
		}
		t, _ := time.Parse(time.RFC3339, m.StartAt)
		if t.Before(cutoff) {
			continue
		}
		switch strings.ToLower(m.MetricType) {
		case "sleep", "sleep_hours":
			sleep += m.Value
			n++
		case "hrv":
			hrv = m.Value
		}
	}
	if n > 0 {
		sleep /= float64(n)
		if sleep >= 7.5 {
			score += 15
			reasons = append(reasons, "sleep target met")
		} else if sleep < 6 {
			score -= 20
			reasons = append(reasons, "low recent sleep")
		}
	}
	if hrv > 0 {
		if hrv >= 50 {
			score += 8
		} else if hrv < 25 {
			score -= 10
		}
	}
	for _, p := range db.PainFlags {
		if p.UserID == uid && p.Active {
			score -= float64(p.Severity * 3)
			reasons = append(reasons, "active pain flag")
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return map[string]any{"score": int(score + 0.5), "reasons": reasons, "status": map[bool]string{true: "ready", false: "recover"}[score >= 65]}
}
func (s *Server) getRecovery(w http.ResponseWriter, cu *contextUser) {
	var out map[string]any
	_ = s.Store.Read(func(db Database) error { out = recoveryScore(db, cu.User.ID); return nil })
	jsonOut(w, 200, out)
}

func (s *Server) listMemories(w http.ResponseWriter, cu *contextUser) {
	out := []AgentMemory{}
	_ = s.Store.Read(func(db Database) error {
		for _, m := range db.AgentMemories {
			if m.UserID == cu.User.ID {
				out = append(out, m)
			}
		}
		return nil
	})
	jsonOut(w, 200, out)
}
func (s *Server) saveMemory(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct{ Kind, Fact string }
	if readJSON(r, &in) != nil || strings.TrimSpace(in.Fact) == "" {
		jsonError(w, 400, "invalid", "Fact required.")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	m := AgentMemory{ID: RandomID("v15"), UserID: cu.User.ID, Kind: strings.TrimSpace(in.Kind), Fact: strings.TrimSpace(in.Fact), Confidence: 1, CreatedAt: now, UpdatedAt: now}
	_ = s.Store.Update(func(db *Database) error { db.AgentMemories = append(db.AgentMemories, m); return nil })
	jsonOut(w, 201, m)
}
func (s *Server) deleteMemory(w http.ResponseWriter, cu *contextUser, id string) {
	_ = s.Store.Update(func(db *Database) error {
		out := db.AgentMemories[:0]
		for _, m := range db.AgentMemories {
			if !(m.ID == id && m.UserID == cu.User.ID) {
				out = append(out, m)
			}
		}
		db.AgentMemories = out
		return nil
	})
	jsonOut(w, 200, map[string]bool{"ok": true})
}

func (s *Server) grocery(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in struct{ MealPlanID, Store string }
	_ = readJSON(r, &in)
	items := []string{}
	_ = s.Store.Read(func(db Database) error {
		for _, p := range db.MealPlans {
			if p.UserID == cu.User.ID && (in.MealPlanID == "" || p.ID == in.MealPlanID) {
				seen := map[string]bool{}
				for _, d := range p.Days {
					for _, it := range d.Items {
						k := strings.TrimSpace(it.Name)
						if k != "" && !seen[k] {
							seen[k] = true
							items = append(items, k+" — "+it.Serving)
						}
					}
				}
				break
			}
		}
		return nil
	})
	sort.Strings(items)
	g := GroceryList{ID: RandomID("v15"), UserID: cu.User.ID, MealPlanID: in.MealPlanID, Store: in.Store, Items: items, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	_ = s.Store.Update(func(db *Database) error { db.GroceryLists = append(db.GroceryLists, g); return nil })
	jsonOut(w, 201, g)
}

func (s *Server) marketplace(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if r.Method == "GET" {
		out := []MarketplaceItem{}
		_ = s.Store.Read(func(db Database) error {
			for _, x := range db.MarketplaceItems {
				if x.Published || x.OwnerID == cu.User.ID || cu.User.Role == "admin" {
					out = append(out, x)
				}
			}
			return nil
		})
		jsonOut(w, 200, out)
		return
	}
	var in MarketplaceItem
	if readJSON(r, &in) != nil || strings.TrimSpace(in.Name) == "" {
		jsonError(w, 400, "invalid", "Name required.")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	in.ID = RandomID("v15")
	in.OwnerID = cu.User.ID
	in.CreatedAt = now
	in.UpdatedAt = now
	if cu.User.Role != "admin" {
		in.Official = false
	}
	_ = s.Store.Update(func(db *Database) error { db.MarketplaceItems = append(db.MarketplaceItems, in); return nil })
	jsonOut(w, 201, in)
}

func (s *Server) agentSettings(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if cu.User.Role != "admin" {
		jsonError(w, 403, "forbidden", "Admin required.")
		return
	}
	if r.Method == "GET" {
		var x Settings
		_ = s.Store.Read(func(db Database) error { x = db.Settings; return nil })
		jsonOut(w, 200, map[string]any{"enabled": x.AgentEnabled, "baseUrl": x.AgentBaseURL, "model": x.AgentModel, "searchUrl": x.AgentSearchURL, "allowWeb": x.AgentAllowWeb, "maxSteps": x.AgentMaxSteps})
		return
	}
	var in struct {
		Enabled, AllowWeb         bool
		BaseURL, Model, SearchURL string
		MaxSteps                  int
	}
	if readJSON(r, &in) != nil {
		jsonError(w, 400, "invalid", "Invalid settings.")
		return
	}
	if in.MaxSteps < 1 || in.MaxSteps > 20 {
		in.MaxSteps = 8
	}
	_ = s.Store.Update(func(db *Database) error {
		db.Settings.AgentEnabled = in.Enabled
		db.Settings.AgentAllowWeb = in.AllowWeb
		db.Settings.AgentBaseURL = strings.TrimRight(in.BaseURL, "/")
		db.Settings.AgentModel = in.Model
		db.Settings.AgentSearchURL = in.SearchURL
		db.Settings.AgentMaxSteps = in.MaxSteps
		return nil
	})
	jsonOut(w, 200, map[string]bool{"ok": true})
}
func (s *Server) agentTasks(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if r.Method == "GET" {
		out := []AgentTask{}
		_ = s.Store.Read(func(db Database) error {
			for _, t := range db.AgentTasks {
				if t.UserID == cu.User.ID || cu.User.Role == "admin" {
					out = append(out, t)
				}
			}
			return nil
		})
		jsonOut(w, 200, out)
		return
	}
	var in struct {
		Goal, Schedule string
		MaxSteps       int
	}
	if readJSON(r, &in) != nil || strings.TrimSpace(in.Goal) == "" {
		jsonError(w, 400, "invalid", "Goal required.")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if in.MaxSteps < 1 || in.MaxSteps > 20 {
		in.MaxSteps = 8
	}
	t := AgentTask{ID: RandomID("v15"), UserID: cu.User.ID, Goal: strings.TrimSpace(in.Goal), Status: "queued", Schedule: in.Schedule, MaxSteps: in.MaxSteps, CreatedAt: now, UpdatedAt: now}
	_ = s.Store.Update(func(db *Database) error { db.AgentTasks = append(db.AgentTasks, t); return nil })
	go s.runAgentTask(t.ID)
	jsonOut(w, 202, t)
}
func (s *Server) runAgentTask(id string) {
	var task AgentTask
	var settings Settings
	var profile Profile
	var memories []AgentMemory
	_ = s.Store.Read(func(db Database) error {
		settings = db.Settings
		for _, t := range db.AgentTasks {
			if t.ID == id {
				task = t
			}
		}
		profile = db.Profiles[task.UserID]
		for _, m := range db.AgentMemories {
			if m.UserID == task.UserID {
				memories = append(memories, m)
			}
		}
		return nil
	})
	if !settings.AgentEnabled {
		_ = s.finishAgent(id, "", fmt.Errorf("agent is disabled"))
		return
	}
	steps := []AgentStep{{At: time.Now().UTC().Format(time.RFC3339), Tool: "context", Input: task.Goal, Output: fmt.Sprintf("Profile goal: %s; equipment: %s; memories: %d", profile.Goal, profile.Equipment, len(memories))}}
	research := ""
	if settings.AgentAllowWeb && strings.TrimSpace(settings.AgentSearchURL) != "" {
		sources, searchSteps, searchErr := agentWebResearch(settings.AgentSearchURL, task.Goal)
		steps = append(steps, searchSteps...)
		if searchErr != nil {
			steps = append(steps, AgentStep{At: time.Now().UTC().Format(time.RFC3339), Tool: "web_error", Input: task.Goal, Output: searchErr.Error()})
		} else {
			research = sources
		}
	}
	var fullDB Database
	_ = s.Store.Read(func(db Database) error { fullDB = db; return nil })
	knowledge := fitnessKnowledgeContext(task.Goal, 12)
	approvedLinks := approvedLinkBlock(fullDB, task.UserID, task.Goal)
	prompt := fmt.Sprintf("You are FormForge Agent, an autonomous fitness research and planning agent running entirely inside FormForge. Goal: %s. User goal: %s. Equipment: %s. Preferred units: %s. User-controlled memories: %v.\n\nBUILT-IN FITNESS KNOWLEDGE VAULT:\n%s\n\nAPPROVED SOURCE LINKS:\n%s\n\nCURRENT PUBLIC RESEARCH:\n%s\n\nReturn a useful result. Preserve all existing FormForge capabilities. Cite full source URLs supplied in the research when the user requests sources or when a claim depends on current external information. Exact quotations must be short and either administrator-verified or visibly present in the supplied source text; otherwise paraphrase. Mark unsupported advice as general knowledge. Never impersonate creators, invent citations, fabricate quotes, or publish a creator profile without administrator review.", task.Goal, profile.Goal, profile.Equipment, measurementSystemFor(fullDB, task.UserID), memories, knowledge, approvedLinks, research)
	key, _ := DecryptSecret(s.MasterKey, settings.AIAPIKeyEncrypted)
	if key == "" {
		key = "local-agent"
	}
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/", nil)
	res, err := s.callOnlineAI(req, Settings{AIBaseURL: settings.AgentBaseURL, AIModel: settings.AgentModel}, key, []onlineMessage{{Role: "user", Content: prompt}})
	if err != nil {
		_ = s.finishAgentWithSteps(id, "", steps, err)
		return
	}
	steps = append(steps, AgentStep{At: time.Now().UTC().Format(time.RFC3339), Tool: "local_model", Input: truncateV15(prompt, 1000), Output: res.Text})
	_ = s.finishAgentWithSteps(id, res.Text, steps, nil)
}

func (s *Server) finishAgent(id, result string, err error) error {
	return s.finishAgentWithSteps(id, result, nil, err)
}
func (s *Server) finishAgentWithSteps(id, result string, steps []AgentStep, runErr error) error {
	return s.Store.Update(func(db *Database) error {
		for i := range db.AgentTasks {
			if db.AgentTasks[i].ID == id {
				db.AgentTasks[i].Status = "completed"
				if runErr != nil {
					db.AgentTasks[i].Status = "failed"
					db.AgentTasks[i].Error = runErr.Error()
				}
				db.AgentTasks[i].Result = result
				if steps != nil {
					db.AgentTasks[i].Steps = steps
				}
				db.AgentTasks[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			}
		}
		return nil
	})
}

func raw(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

func agentWebResearch(searchBase, query string) (string, []AgentStep, error) {
	base, err := url.Parse(strings.TrimSpace(searchBase))
	if err != nil || base.Host == "" {
		return "", nil, fmt.Errorf("invalid search endpoint")
	}
	if base.Scheme != "https" && base.Scheme != "http" {
		return "", nil, fmt.Errorf("search endpoint must use HTTP or HTTPS")
	}
	q := base.Query()
	q.Set("q", query)
	q.Set("format", "json")
	base.RawQuery = q.Encode()
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return http.ErrUseLastResponse
		}
		return nil
	}}
	resp, err := client.Get(base.String())
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("search returned HTTP %d", resp.StatusCode)
	}
	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", nil, err
	}
	var out struct {
		Results []searxResult `json:"results"`
	}
	if json.Unmarshal(rawBody, &out) != nil {
		return "", nil, fmt.Errorf("unexpected search response")
	}
	steps := []AgentStep{{At: time.Now().UTC().Format(time.RFC3339), Tool: "web_search", Input: query, Output: fmt.Sprintf("%d results", len(out.Results))}}
	var b strings.Builder
	for i, x := range out.Results {
		if i >= 5 {
			break
		}
		if !safePublicAgentURL(x.URL) {
			continue
		}
		text := strings.TrimSpace(x.Content)
		if page, err := agentFetchPage(client, x.URL); err == nil && page != "" {
			text = page
		}
		if len(text) > 3500 {
			text = text[:3500]
		}
		fmt.Fprintf(&b, "SOURCE %d: %s\nURL: %s\n%s\n\n", i+1, x.Title, x.URL, text)
		steps = append(steps, AgentStep{At: time.Now().UTC().Format(time.RFC3339), Tool: "web_fetch", Input: x.Title, Output: truncateV15(text, 500), SourceURL: x.URL})
	}
	return b.String(), steps, nil
}

func safePublicAgentURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return false
	}
	h := u.Hostname()
	if strings.EqualFold(h, "localhost") {
		return false
	}
	if ip := net.ParseIP(h); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast()) {
		return false
	}
	return true
}

func agentFetchPage(client *http.Client, rawURL string) (string, error) {
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/") && !strings.Contains(ct, "json") {
		return "", fmt.Errorf("unsupported content")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 512<<10))
	if err != nil {
		return "", err
	}
	text := string(b)
	text = strings.NewReplacer("<script", "\n<script", "<style", "\n<style", "<", " <").Replace(text)
	var clean strings.Builder
	inside := false
	for _, r := range text {
		if r == '<' {
			inside = true
			continue
		}
		if r == '>' {
			inside = false
			clean.WriteRune(' ')
			continue
		}
		if !inside {
			clean.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(clean.String()), " "), nil
}

func truncateV15(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
