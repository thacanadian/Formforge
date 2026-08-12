package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type aiChatRequest struct {
	Message string `json:"message"`
	Mode    string `json:"mode"`
}

type aiChatResponse struct {
	Reply           string        `json:"reply"`
	Mode            string        `json:"mode"`
	OnlineAvailable bool          `json:"onlineAvailable"`
	FallbackReason  string        `json:"fallbackReason,omitempty"`
	Grounding       []AIGrounding `json:"grounding"`
	Tokens          int           `json:"tokens,omitempty"`
	CostMicros      int64         `json:"costMicros,omitempty"`
}

type aiSettingsInput struct {
	Mode     string `json:"mode"`
	BaseURL  string `json:"baseUrl"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
	ClearKey bool   `json:"clearKey"`
}

type aiSettingsOutput struct {
	Mode             string `json:"mode"`
	BaseURL          string `json:"baseUrl"`
	Model            string `json:"model"`
	APIKeyConfigured bool   `json:"apiKeyConfigured"`
	OfflineAvailable bool   `json:"offlineAvailable"`
}

func (s *Server) aiStatus(w http.ResponseWriter, cu *contextUser) {
	var out aiSettingsOutput
	_ = s.Store.Read(func(db Database) error {
		out = publicAISettings(db.Settings)
		return nil
	})
	jsonOut(w, 200, out)
}

func publicAISettings(x Settings) aiSettingsOutput {
	mode := strings.ToLower(strings.TrimSpace(x.AIMode))
	if mode != "offline" && mode != "online" && mode != "auto" {
		mode = "auto"
	}
	base := strings.TrimSpace(x.AIBaseURL)
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(x.AIModel)
	if model == "" {
		model = "gpt-4o-mini"
	}
	return aiSettingsOutput{Mode: mode, BaseURL: base, Model: model, APIKeyConfigured: strings.TrimSpace(x.AIAPIKeyEncrypted) != "", OfflineAvailable: true}
}

func (s *Server) getAISettings(w http.ResponseWriter, cu *contextUser) {
	if cu.User.Role != "admin" {
		jsonError(w, 403, "forbidden", "Administrator access is required.")
		return
	}
	s.aiStatus(w, cu)
}

func (s *Server) putAISettings(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if cu.User.Role != "admin" {
		jsonError(w, 403, "forbidden", "Administrator access is required.")
		return
	}
	var in aiSettingsInput
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	in.Mode = strings.ToLower(strings.TrimSpace(in.Mode))
	if in.Mode != "offline" && in.Mode != "online" && in.Mode != "auto" {
		jsonError(w, 400, "invalid_input", "AI mode must be auto, online, or offline.")
		return
	}
	base, err := normalizeAIBaseURL(in.BaseURL)
	if err != nil {
		jsonError(w, 400, "invalid_input", err.Error())
		return
	}
	model := strings.TrimSpace(in.Model)
	if len(model) < 2 || len(model) > 120 {
		jsonError(w, 400, "invalid_input", "Enter a valid model name.")
		return
	}
	var encrypted string
	if strings.TrimSpace(in.APIKey) != "" {
		if len(strings.TrimSpace(in.APIKey)) > 500 {
			jsonError(w, 400, "invalid_input", "The API key is too long.")
			return
		}
		encrypted, err = EncryptSecret(s.MasterKey, strings.TrimSpace(in.APIKey))
		if err != nil {
			jsonError(w, 500, "encryption_failed", "The API key could not be encrypted.")
			return
		}
	}
	err = s.Store.Update(func(db *Database) error {
		db.Settings.AIMode = in.Mode
		db.Settings.AIBaseURL = base
		db.Settings.AIModel = model
		if in.ClearKey {
			db.Settings.AIAPIKeyEncrypted = ""
		} else if encrypted != "" {
			db.Settings.AIAPIKeyEncrypted = encrypted
		}
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "ai.settings.update", "system", clientIP(r), map[string]any{"mode": in.Mode, "model": model, "keyChanged": encrypted != "" || in.ClearKey})
		return nil
	})
	if err != nil {
		jsonError(w, 500, "save_failed", err.Error())
		return
	}
	s.getAISettings(w, cu)
}

func normalizeAIBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "https://api.openai.com/v1"
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return "", errors.New("Enter a valid AI service URL.")
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme == "https" {
		if host != "api.openai.com" {
			return "", errors.New("For security, public online AI must use https://api.openai.com/v1. Loopback URLs are allowed for a local OpenAI-compatible service.")
		}
	} else if u.Scheme == "http" {
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", errors.New("Unencrypted AI URLs are allowed only on this computer (localhost or 127.0.0.1).")
		}
	} else {
		return "", errors.New("AI service URL must use HTTPS, or HTTP on localhost.")
	}
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

func (s *Server) testAI(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	if cu.User.Role != "admin" {
		jsonError(w, 403, "forbidden", "Administrator access is required.")
		return
	}
	var in aiSettingsInput
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	var settings Settings
	_ = s.Store.Read(func(db Database) error { settings = db.Settings; return nil })
	if strings.TrimSpace(in.BaseURL) != "" {
		base, err := normalizeAIBaseURL(in.BaseURL)
		if err != nil {
			jsonError(w, 400, "invalid_input", err.Error())
			return
		}
		settings.AIBaseURL = base
	}
	if strings.TrimSpace(in.Model) != "" {
		settings.AIModel = strings.TrimSpace(in.Model)
	}
	key := strings.TrimSpace(in.APIKey)
	if key == "" {
		var err error
		key, err = DecryptSecret(s.MasterKey, settings.AIAPIKeyEncrypted)
		if err != nil {
			jsonError(w, 500, "decrypt_failed", "The saved API key could not be decrypted.")
			return
		}
	}
	if key == "" {
		jsonError(w, 400, "missing_key", "Enter or save an API key first.")
		return
	}
	result, err := s.callOnlineAI(r, settings, key, []onlineMessage{{Role: "user", Content: "Reply with exactly: FormForge online AI is connected."}})
	if err != nil {
		jsonError(w, 502, "online_ai_failed", err.Error())
		return
	}
	jsonOut(w, 200, map[string]any{"ok": true, "reply": result.Text})
}

func (s *Server) listAIHistory(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	limit := 80
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= 200 {
		limit = n
	}
	out := []ChatMessage{}
	_ = s.Store.Read(func(db Database) error {
		for _, m := range db.ChatMessages {
			if m.UserID == cu.User.ID {
				out = append(out, m)
			}
		}
		if len(out) > limit {
			out = out[len(out)-limit:]
		}
		return nil
	})
	jsonOut(w, 200, out)
}

func (s *Server) clearAIHistory(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	_ = s.Store.Update(func(db *Database) error {
		n := db.ChatMessages[:0]
		for _, m := range db.ChatMessages {
			if m.UserID != cu.User.ID {
				n = append(n, m)
			}
		}
		db.ChatMessages = n
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "ai.history.clear", cu.User.ID, clientIP(r), nil)
		return nil
	})
	jsonOut(w, 200, map[string]any{"ok": true})
}

func (s *Server) chatAI(w http.ResponseWriter, r *http.Request, cu *contextUser) {
	var in aiChatRequest
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "invalid_json", err.Error())
		return
	}
	in.Message = strings.TrimSpace(in.Message)
	if in.Message == "" || len(in.Message) > 4000 {
		jsonError(w, 400, "invalid_input", "Enter a message between 1 and 4,000 characters.")
		return
	}
	requestedMode := strings.ToLower(strings.TrimSpace(in.Mode))
	if requestedMode != "offline" && requestedMode != "online" && requestedMode != "auto" && requestedMode != "" {
		jsonError(w, 400, "invalid_input", "AI mode must be auto, online, or offline.")
		return
	}
	var db Database
	_ = s.Store.Read(func(x Database) error { db = x; return nil })
	settings := db.Settings
	mode := requestedMode
	if mode == "" {
		mode = publicAISettings(settings).Mode
	}
	userMessage := ChatMessage{ID: RandomID("chat_"), UserID: cu.User.ID, Role: "user", Content: in.Message, Mode: mode, At: nowISO()}
	var reply, usedMode, fallback string
	var tokenCount int
	var costMicros int64
	grounding := aiGroundingFor(db, cu.User.ID)
	webResearch := ""
	if wantsSourceLinks(in.Message) && settings.AgentAllowWeb && strings.TrimSpace(settings.AgentSearchURL) != "" {
		if research, steps, err := agentWebResearch(settings.AgentSearchURL, in.Message); err == nil {
			webResearch = research
			if webGrounding := groundingFromAgentSteps(steps); len(webGrounding) > 0 {
				grounding = webGrounding
			}
		}
	}
	key, keyErr := DecryptSecret(s.MasterKey, settings.AIAPIKeyEncrypted)
	onlineConfigured := keyErr == nil && strings.TrimSpace(key) != ""
	if mode == "online" || (mode == "auto" && onlineConfigured) {
		messages := s.onlineContext(db, cu.User.ID, in.Message)
		if webResearch != "" {
			messages[0].Content += "\n\nCURRENT PUBLIC WEB RESEARCH FOR THIS REQUEST:\n" + webResearch + "\nUse these sources carefully. Include clickable full URLs when the user requested links. Quote only short passages that are visibly present in the supplied source material; otherwise paraphrase and say that it is a paraphrase."
		}
		estInput := 0
		for _, m := range messages {
			estInput += estimateTokens(m.Content)
		}
		budgetErr := onlineBudgetError(db, cu.User, estInput)
		if budgetErr != nil {
			if mode == "online" {
				jsonError(w, 402, "online_ai_limit", budgetErr.Error())
				return
			}
			fallback = budgetErr.Error()
			reply, usedMode = s.offlineCoach(db, cu.User.ID, in.Message), "offline-fallback"
		} else {
			result, err := s.callOnlineAI(r, settings, key, messages)
			if err == nil {
				reply, usedMode = result.Text, "online"
				tokenCount = result.InputTokens + result.OutputTokens
				costMicros = result.CostMicros
			} else if mode == "online" {
				jsonError(w, 502, "online_ai_failed", err.Error()+" Switch to Auto or Offline to keep chatting without internet.")
				return
			} else {
				fallback = err.Error()
				reply, usedMode = s.offlineCoach(db, cu.User.ID, in.Message), "offline-fallback"
			}
		}
	} else {
		reply, usedMode = s.offlineCoach(db, cu.User.ID, in.Message), "offline"
		if mode == "auto" && !onlineConfigured {
			fallback = "Online AI is not configured, so the built-in offline coach answered."
		}
	}
	assistantMessage := ChatMessage{ID: RandomID("chat_"), UserID: cu.User.ID, Role: "assistant", Content: reply, Mode: usedMode, Grounding: grounding, Tokens: tokenCount, CostMicros: costMicros, At: nowISO()}
	if err := s.Store.Update(func(db *Database) error {
		db.ChatMessages = append(db.ChatMessages, userMessage, assistantMessage)
		if usedMode == "online" {
			db.AIUsage = append(db.AIUsage, AIUsage{ID: RandomID("aiu_"), UserID: cu.User.ID, Date: todayISO(), Provider: "openai-compatible", Model: settings.AIModel, InputTokens: maxInt(1, tokenCount-estimateTokens(reply)), OutputTokens: estimateTokens(reply), CostMicros: costMicros, At: nowISO()})
		}
		if len(db.ChatMessages) > 4000 {
			db.ChatMessages = db.ChatMessages[len(db.ChatMessages)-4000:]
		}
		u := db.Users[cu.User.ID]
		s.audit(db, &u, "ai.chat", cu.User.ID, clientIP(r), map[string]any{"mode": usedMode})
		return nil
	}); err != nil {
		jsonError(w, 500, "save_failed", err.Error())
		return
	}
	jsonOut(w, 200, aiChatResponse{Reply: reply, Mode: usedMode, OnlineAvailable: onlineConfigured, FallbackReason: fallback, Grounding: grounding, Tokens: tokenCount, CostMicros: costMicros})
}

type onlineMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type onlineAIResult struct {
	Text         string
	InputTokens  int
	OutputTokens int
	CostMicros   int64
}

func (s *Server) onlineContext(db Database, userID, message string) []onlineMessage {
	p := db.Profiles[userID]
	unitSystem := measurementSystemFor(db, userID)
	system := fmt.Sprintf(`You are FormForge Coach, a highly capable, source-aware fitness and nutrition coach inside a private fitness app. User profile: name=%s, age=%d, goal=%s, experience=%s, days/week=%d, equipment=%s, weight=%s, height=%s, calorie target=%d, protein target=%d g, preferred units=%s. Preserve every existing FormForge capability: personalized workouts, progression, pain-aware substitutions, recovery, meal planning, nutrition, health-data interpretation, creator-informed coaching, habit support, and progress summaries. Distinguish strong evidence, uncertain evidence, practical inference, and user preference. Never claim omniscience. Never invent a study, URL, quotation, creator endorsement, or personal experience. When the user requests sources or links, provide the relevant approved or researched URLs. Exact quotations must be short, clearly attributed, and either administrator-verified or visibly present in supplied source material; otherwise paraphrase. Do not diagnose injuries or diseases. For alarming pain, chest pain, fainting, severe shortness of breath, acute neurologic symptoms, or major trauma, tell the user to stop and seek appropriate medical care. Give actionable workouts with exercises, sets, reps, rest, and substitutions when asked.`, p.Name, p.Age, p.Goal, p.Experience, p.DaysPerWeek, p.Equipment, formatWeightForUser(db, userID, p.WeightKG), formatHeightForUser(db, userID, p.HeightCM), p.CalorieGoal, p.ProteinGoal, unitSystem)
	system += "\n\nBUILT-IN FORGE KNOWLEDGE VAULT (original local reference notes):\n" + fitnessKnowledgeContext(message, 10)
	if links := approvedLinkBlock(db, userID, message); links != "" {
		system += "\n\n" + links
	}
	system += coachPromptContext(db, userID)
	out := []onlineMessage{{Role: "system", Content: system}}
	var recent []ChatMessage
	for _, m := range db.ChatMessages {
		if m.UserID == userID && (m.Role == "user" || m.Role == "assistant") {
			recent = append(recent, m)
		}
	}
	if len(recent) > 12 {
		recent = recent[len(recent)-12:]
	}
	for _, m := range recent {
		out = append(out, onlineMessage{Role: m.Role, Content: m.Content})
	}
	out = append(out, onlineMessage{Role: "user", Content: message})
	return out
}

func (s *Server) callOnlineAI(r *http.Request, settings Settings, key string, messages []onlineMessage) (onlineAIResult, error) {
	if strings.TrimSpace(key) == "" {
		return onlineAIResult{}, errors.New("Online AI is not configured with an API key.")
	}
	base, err := normalizeAIBaseURL(settings.AIBaseURL)
	if err != nil {
		return onlineAIResult{}, err
	}
	endpoint := strings.TrimRight(base, "/")
	if !strings.HasSuffix(endpoint, "/chat/completions") {
		endpoint += "/chat/completions"
	}
	model := strings.TrimSpace(settings.AIModel)
	if model == "" {
		model = "gpt-4o-mini"
	}
	payload := map[string]any{"model": model, "messages": messages, "temperature": 0.35, "max_tokens": 1000}
	body, _ := json.Marshal(payload)
	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return onlineAIResult{}, errors.New("Online AI request could not be prepared.")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "FormForge/"+s.Version)
	client := &http.Client{Timeout: 35 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return onlineAIResult{}, errors.New("Online AI could not be reached. Check the internet connection, firewall, proxy, and API settings.")
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return onlineAIResult{}, errors.New("Online AI response could not be read.")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := strings.TrimSpace(e.Error.Message)
		if msg == "" {
			msg = fmt.Sprintf("Online AI returned HTTP %d.", resp.StatusCode)
		}
		return onlineAIResult{}, errors.New(msg)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return onlineAIResult{}, errors.New("Online AI returned an unexpected response.")
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	if text == "" {
		return onlineAIResult{}, errors.New("Online AI returned an empty response.")
	}
	inputTokens, outputTokens := out.Usage.PromptTokens, out.Usage.CompletionTokens
	if inputTokens <= 0 {
		for _, m := range messages {
			inputTokens += estimateTokens(m.Content)
		}
	}
	if outputTokens <= 0 {
		outputTokens = estimateTokens(text)
	}
	return onlineAIResult{Text: text, InputTokens: inputTokens, OutputTokens: outputTokens, CostMicros: estimateOnlineCostMicros(model, inputTokens, outputTokens)}, nil
}

var dayCountPattern = regexp.MustCompile(`(?i)\b([2-6])\s*(?:day|days|x|times)\b`)

func (s *Server) offlineCoach(db Database, userID, message string) string {
	p := db.Profiles[userID]
	m := strings.ToLower(message)
	if containsAny(m, "chest pain", "fainted", "fainting", "can't breathe", "cannot breathe", "severe shortness of breath") {
		return "Stop training now. Those symptoms are not something an app should coach through. Seek urgent medical evaluation, and call emergency services if the symptoms are severe or happening now."
	}
	if containsAny(m, "pain", "injury", "hurt", "swollen", "torn") {
		return "Do not push through sharp, worsening, or joint pain. Stop the movement, avoid testing it repeatedly, and switch to pain-free activity. If there is major swelling, weakness, loss of motion, numbness, a pop with immediate pain, or symptoms that persist, get evaluated by a qualified clinician. I can help you build a temporary pain-free plan, but I cannot diagnose the injury."
	}
	if containsAny(m, "quote", "give me a quote", "what would my coach say") {
		return verifiedQuoteReply(db, userID)
	}
	if wantsSourceLinks(message) {
		knowledge := fitnessKnowledgeContext(message, 5)
		links := approvedSourceLinks(db, userID, message, 5)
		var b strings.Builder
		b.WriteString(coachToneLead(db, userID) + "\n\n")
		b.WriteString(knowledge)
		if len(links) > 0 {
			b.WriteString("\n\nApproved links:\n")
			for _, src := range links {
				fmt.Fprintf(&b, "• %s — %s\n", src.Title, src.SourceURL)
			}
		} else {
			b.WriteString("\n\nNo administrator-approved link matches this question yet. In Online or Agent mode, FormForge can search the public web through the configured search service and return source URLs. I will not invent one.")
		}
		b.WriteString("\n\nBasis: FormForge Knowledge Vault plus any approved links shown above.")
		return strings.TrimSpace(b.String())
	}
	if containsAny(m, "who are my coaches", "coach blend", "coaching team", "influences") {
		return fmt.Sprintf("Your current coaching blend is %s. I use those profiles to combine broad training principles and communication traits, but I remain FormForge Coach and do not impersonate or claim endorsement from any creator.\n\n%s", coachBlendSummary(db, userID), coachBasis(db, userID))
	}
	if containsAny(m, "workout", "routine", "program", "training plan", "train this week", "make me a plan", "give me a plan", "lighter week", "make this week lighter") {
		return s.offlineWorkoutPlan(db, userID, message)
	}
	if containsAny(m, "protein", "calorie", "macro", "carbs", "fat goal") {
		return fmt.Sprintf("%s\n\nYour saved daily targets are %d calories, %d g protein, %d g carbs, and %d g fat. For your %s goal, make protein the non-negotiable target, distribute it across 3–5 meals, and adjust calories only after at least two consistent weeks of scale and training data.\n\n%s", coachToneLead(db, userID), p.CalorieGoal, p.ProteinGoal, p.CarbGoal, p.FatGoal, strings.ToLower(p.Goal), coachBasis(db, userID))
	}
	if containsAny(m, "plateau", "progressive overload", "get stronger", "not progressing", "stuck") {
		return coachToneLead(db, userID) + "\n\nUse double progression: keep the same weight until every set reaches the top of the rep range with clean form and 1–2 reps left in reserve, then add the smallest practical amount of weight and return to the bottom of the range. If performance stalls for 2–3 weeks, check sleep, calories, exercise consistency, and total fatigue before adding more volume.\n\n" + coachBasis(db, userID)
	}
	if containsAny(m, "sleep", "recovery", "sore", "fatigue", "tired") {
		return coachToneLead(db, userID) + "\n\nFor recovery, protect sleep first, keep protein and hydration consistent, and avoid turning every session into a max-effort test. Mild muscle soreness is usually compatible with training, but reduce load or volume when soreness changes your technique. A deload week can use about 50–65% of normal hard sets while keeping movements familiar.\n\n" + coachBasis(db, userID)
	}
	if containsAny(m, "what did i", "my progress", "logged", "how am i doing", "today") {
		return offlineProgressSummary(db, userID)
	}
	if containsAny(m, "hello", "hey", "hi ", "hi", "what can you do") {
		return fmt.Sprintf("Hey %s. Your coaching blend is %s. I stay FormForge Coach, but I use those selected influences to shape the principles, level of explanation, and energy of the answer. I work offline and can build workouts around your %s goal, %s experience level, and %s equipment. Ask for something specific like “make me a 4-day muscle-building plan,” “replace cable rows,” or “summarize my progress.”", firstName(p.Name), coachBlendSummary(db, userID), strings.ToLower(p.Goal), p.Experience, p.Equipment)
	}
	if containsAny(m, "motivation", "don't want", "dont want", "skip") {
		return coachToneLead(db, userID) + "\n\nLower the activation cost: commit to the warm-up and the first two working sets only. Once you start, finish the planned session if your energy improves; if it does not, leave after a short quality session. Consistency beats waiting to feel motivated.\n\n" + coachBasis(db, userID)
	}
	knowledge := fitnessKnowledgeContext(message, 4)
	return fmt.Sprintf("%s\n\nBased on your saved profile—%s goal, %s level, %d training days, and %s equipment—the best next move is to keep one repeatable plan long enough to measure it.\n\nRelevant knowledge:\n%s\n\nAsk me for a workout, exercise substitution, progression rule, recovery adjustment, meal plan, nutrition target, verified quote, sources, or a summary of your logged data. I can answer all of those without leaving FormForge.\n\n%s", coachToneLead(db, userID), p.Goal, p.Experience, p.DaysPerWeek, p.Equipment, knowledge, coachBasis(db, userID))
}

func containsAny(s string, values ...string) bool {
	for _, v := range values {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}

func firstName(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return "there"
	}
	return fields[0]
}

func (s *Server) offlineWorkoutPlan(db Database, userID, request string) string {
	p := db.Profiles[userID]
	activePain := []PainFlag{}
	for _, f := range db.PainFlags {
		if f.UserID == userID && f.Active {
			activePain = append(activePain, f)
		}
	}
	equipment := p.Equipment
	lowerRequest := strings.ToLower(request)
	switch {
	case strings.Contains(lowerRequest, "bodyweight") || strings.Contains(lowerRequest, "no equipment"):
		equipment = "Bodyweight only"
	case strings.Contains(lowerRequest, "dumbbell"):
		equipment = "Dumbbells only"
	case strings.Contains(lowerRequest, "resistance band") || strings.Contains(lowerRequest, "bands"):
		equipment = "Resistance bands"
	case strings.Contains(lowerRequest, "home gym"):
		equipment = "Home gym"
	case strings.Contains(lowerRequest, "full gym"):
		equipment = "Full gym"
	}
	goal := p.Goal
	switch {
	case strings.Contains(lowerRequest, "build muscle") || strings.Contains(lowerRequest, "muscle-building") || strings.Contains(lowerRequest, "hypertrophy"):
		goal = "Build muscle"
	case strings.Contains(lowerRequest, "strength"):
		goal = "Increase strength"
	case strings.Contains(lowerRequest, "lose fat") || strings.Contains(lowerRequest, "fat loss") || strings.Contains(lowerRequest, "cut"):
		goal = "Lose fat"
	case strings.Contains(lowerRequest, "endurance") || strings.Contains(lowerRequest, "cardio"):
		goal = "Improve endurance"
	}
	days := p.DaysPerWeek
	if match := dayCountPattern.FindStringSubmatch(request); len(match) == 2 {
		if n, err := strconv.Atoi(match[1]); err == nil {
			days = n
		}
	}
	if days < 2 {
		days = 2
	}
	if days > 6 {
		days = 6
	}
	var available []string
	for i := len(db.CheckIns) - 1; i >= 0; i-- {
		if db.CheckIns[i].UserID == userID {
			available = append([]string(nil), db.CheckIns[i].AvailableDays...)
			break
		}
	}
	defaultDays := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	if len(available) < days {
		available = defaultDays
	}
	var library []Workout
	for _, w := range db.Workouts {
		if w.BuiltIn && w.Level == p.Experience {
			library = append(library, w)
		}
	}
	sort.Slice(library, func(i, j int) bool { return library[i].ID < library[j].ID })
	if len(library) == 0 {
		for _, w := range BuiltInWorkouts() {
			if w.Level == "beginner" {
				library = append(library, w)
			}
		}
		sort.Slice(library, func(i, j int) bool { return library[i].ID < library[j].ID })
	}
	energyNote := "Use 1–3 reps in reserve on most sets."
	for i := len(db.CheckIns) - 1; i >= 0; i-- {
		c := db.CheckIns[i]
		if c.UserID != userID {
			continue
		}
		switch c.Energy {
		case "tired":
			energyNote = "Your last check-in said tired: remove one working set from each exercise and stop 3 reps before failure."
		case "burnt":
			energyNote = "Your last check-in said burnt out: use a recovery week at about 60% of normal hard-set volume."
		case "crushed":
			energyNote = "Your last check-in said strong: keep the plan controlled and add load only when every rep is clean."
		}
		break
	}
	if strings.Contains(lowerRequest, "lighter") || strings.Contains(lowerRequest, "deload") || strings.Contains(lowerRequest, "recovery week") {
		energyNote = "Use a lighter recovery week: perform about 60% of normal hard-set volume and stop 3–4 reps before failure."
	}
	var b strings.Builder
	b.WriteString(coachToneLead(db, userID) + "\n\n")
	fmt.Fprintf(&b, "%d-day %s plan · %s · %s\nCoach blend: %s\n\n", days, goal, p.Experience, equipment, coachBlendSummary(db, userID))
	for i := 0; i < days; i++ {
		w := library[i%len(library)]
		day := available[i%len(available)]
		fmt.Fprintf(&b, "%s — %s (%d min)\n", day, w.Name, w.Duration)
		for _, ex := range w.Exercises {
			adapted := adaptExercise(ex, equipment)
			adapted.Name = painSubstitution(adapted.Name, activePain)
			sets := adapted.Sets
			if strings.Contains(strings.ToLower(energyNote), "remove one") && sets > 1 {
				sets--
			}
			if strings.Contains(strings.ToLower(energyNote), "60%") && sets > 2 {
				sets = maxInt(2, (sets*3)/5)
			}
			fmt.Fprintf(&b, "• %s — %d × %s, rest %s\n", adapted.Name, sets, adapted.Reps, adapted.Rest)
		}
		b.WriteString("\n")
	}
	b.WriteString("Progression: when all sets reach the top of the rep range with clean form, add the smallest practical load next time.\n")
	b.WriteString("Recovery adjustment: " + energyNote)
	if len(activePain) > 0 {
		b.WriteString("\nPain flags: substitutions use pain-free regressions only. Stop sharp or worsening pain and seek qualified medical evaluation; FormForge does not diagnose injuries.")
	}
	if notes := offlineInfluenceAdjustments(db, userID); len(notes) > 0 {
		b.WriteString("\n\nInfluence adjustments:\n• " + strings.Join(notes, "\n• "))
	}
	b.WriteString("\n\n" + coachBasis(db, userID))
	return strings.TrimSpace(b.String())
}

func adaptExercise(ex Exercise, equipment string) Exercise {
	e := ex
	mode := strings.ToLower(equipment)
	if strings.Contains(mode, "bodyweight") {
		replacements := map[string]string{
			"Goblet Squat": "Tempo Bodyweight Squat", "Back Squat": "Rear-Foot-Elevated Split Squat", "Hack Squat": "Reverse Lunge", "Leg Press": "Walking Lunge", "Romanian Deadlift": "Single-Leg Hip Hinge", "Bench Press": "Feet-Elevated Push-Up", "Barbell Bench Press": "Tempo Push-Up", "Dumbbell Press": "Push-Up", "Incline Dumbbell Press": "Feet-Elevated Push-Up", "Barbell Row": "Doorway Towel Row", "Cable Row": "Prone Y-T-W Raise", "Lat Pulldown": "Prone Lat Sweep", "Pull-Ups": "Doorway Towel Row", "Pull-Ups (Weighted)": "Slow Pull-Up or Doorway Row", "Overhead Press": "Pike Push-Up", "Cable Lateral Raise": "Lean-Away Isometric Lateral Raise", "Bicep Curl": "Towel Isometric Curl", "Tricep Pushdown": "Close-Grip Push-Up", "Skull Crusher": "Bodyweight Triceps Extension", "Leg Curl": "Sliding Hamstring Curl", "Hip Thrust": "Single-Leg Glute Bridge", "Calf Raise": "Single-Leg Calf Raise", "Deadlift": "Single-Leg Hip Hinge", "Face Pull": "Prone W Raise", "Hammer Curl": "Towel Isometric Curl", "Sprint Intervals": "Fast High-Knee Intervals",
		}
		if x := replacements[e.Name]; x != "" {
			e.Name = x
		}
	} else if strings.Contains(mode, "dumbbell") {
		replacements := map[string]string{
			"Back Squat": "Dumbbell Front Squat", "Hack Squat": "Dumbbell Bulgarian Split Squat", "Leg Press": "Dumbbell Reverse Lunge", "Bench Press": "Dumbbell Floor Press", "Barbell Bench Press": "Dumbbell Floor Press", "Barbell Row": "One-Arm Dumbbell Row", "Cable Row": "Chest-Supported Dumbbell Row", "Lat Pulldown": "Dumbbell Pullover", "Overhead Press": "Dumbbell Overhead Press", "Cable Lateral Raise": "Dumbbell Lateral Raise", "Tricep Pushdown": "Dumbbell Overhead Extension", "Face Pull": "Rear-Delt Dumbbell Row", "Deadlift": "Dumbbell Romanian Deadlift", "Leg Curl": "Dumbbell Leg Curl", "Calf Raise": "Dumbbell Calf Raise",
		}
		if x := replacements[e.Name]; x != "" {
			e.Name = x
		}
	}
	return e
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func offlineProgressSummary(db Database, userID string) string {
	date := todayISO()
	weekStart := startOfWeek(time.Now())
	var calories, protein float64
	workouts := 0
	for _, x := range db.Nutrition {
		if x.UserID == userID && x.Date == date {
			calories += x.Calories
			protein += x.Protein
		}
	}
	for _, x := range db.WorkoutLogs {
		if x.UserID != userID {
			continue
		}
		if t, err := time.Parse("2006-01-02", x.Date); err == nil && !t.Before(weekStart) {
			workouts++
		}
	}
	var entries []ProgressEntry
	for _, x := range db.Progress {
		if x.UserID == userID {
			entries = append(entries, x)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date > entries[j].Date })
	weight := "No bodyweight entry is logged yet."
	if len(entries) > 0 {
		weight = fmt.Sprintf("Your latest logged weight is %s on %s.", formatWeightForUser(db, userID, entries[0].WeightKG), entries[0].Date)
	}
	p := db.Profiles[userID]
	return fmt.Sprintf("Today you have logged %.0f calories and %.0f g protein against targets of %d calories and %d g protein. You have completed %d workout(s) this week. %s", calories, protein, p.CalorieGoal, p.ProteinGoal, workouts, weight)
}
