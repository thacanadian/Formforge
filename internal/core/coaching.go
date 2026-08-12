package core

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// CoachProfile is an editorial influence profile. A profile is not a claim that
// the named creator sponsors, endorses, or personally operates FormForge.
type CoachProfile struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Initials      string   `json:"initials"`
	Category      string   `json:"category"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	Principles    []string `json:"principles"`
	Communication []string `json:"communication"`
	SafetyNote    string   `json:"safetyNote"`
	SourceCount   int      `json:"sourceCount"`
}

func CoachCatalog() []CoachProfile {
	return []CoachProfile{
		{
			ID:       "jeff-nippard",
			Name:     "Jeff Nippard",
			Initials: "JN",
			Category: "Evidence-based hypertrophy",
			Status:   "editorial",
			Summary:  "Evidence-focused programming with clear explanations, measurable progression, and fatigue-aware volume.",
			Principles: []string{
				"Use progressive overload and repeatable exercise selection.",
				"Keep most working sets close enough to failure to be productive without turning every set into a max-effort test.",
				"Explain the reason for exercise, volume, and recovery choices.",
			},
			Communication: []string{"calm", "analytical", "plain-English rationale"},
			SafetyNote:    "FormForge summarizes broad public training themes. This is not an official Jeff Nippard product or endorsement.",
		},
		{
			ID:       "noel-deyzel",
			Name:     "Noel Deyzel",
			Initials: "ND",
			Category: "Supportive bodybuilding",
			Status:   "editorial",
			Summary:  "Encouraging bodybuilding guidance centered on consistency, fundamentals, confidence, and sustainable effort.",
			Principles: []string{
				"Prioritize consistent execution of basic movements.",
				"Use supportive accountability rather than shame.",
				"Make plans practical enough to follow during a normal week.",
			},
			Communication: []string{"supportive", "direct", "big-brother encouragement"},
			SafetyNote:    "FormForge does not imitate Noel Deyzel's exact voice or claim his approval.",
		},
		{
			ID:       "ronnie-coleman",
			Name:     "Ronnie Coleman",
			Initials: "RC",
			Category: "High-effort bodybuilding",
			Status:   "editorial",
			Summary:  "High-energy bodybuilding influence emphasizing hard work, heavy basics, and enthusiasm for training.",
			Principles: []string{
				"Build the session around proven compound and bodybuilding movements.",
				"Reserve the highest effort for controlled working sets and safe isolation work.",
				"Use intensity as motivation, not as permission to ignore technique or pain.",
			},
			Communication: []string{"energetic", "brief", "celebrates effort"},
			SafetyNote:    "FormForge will not reproduce catchphrases or prescribe unsafe maximal lifting merely to resemble a creator.",
		},
		{
			ID:       "arnold-schwarzenegger",
			Name:     "Arnold Schwarzenegger",
			Initials: "AS",
			Category: "Golden-era bodybuilding",
			Status:   "editorial",
			Summary:  "Classic bodybuilding influence emphasizing mind-muscle connection, exercise variety, symmetry, and focused effort.",
			Principles: []string{
				"Train for balanced development and visible weak points.",
				"Use controlled reps and intentional muscle focus.",
				"Add variety strategically without changing the whole program every week.",
			},
			Communication: []string{"confident", "aspirational", "simple imagery"},
			SafetyNote:    "This is an editorial training profile, not an official Arnold Schwarzenegger coaching product.",
		},
		{
			ID:       "mike-israetel",
			Name:     "Mike Israetel",
			Initials: "MI",
			Category: "Volume and fatigue management",
			Status:   "editorial",
			Summary:  "Structured hypertrophy influence emphasizing reps in reserve, recoverable volume, progression blocks, and deloads.",
			Principles: []string{
				"Match weekly volume to recovery and performance trends.",
				"Use reps in reserve to control effort across a training block.",
				"Reduce fatigue when performance and recovery consistently decline.",
			},
			Communication: []string{"technical", "direct", "uses clear training metrics"},
			SafetyNote:    "FormForge summarizes general concepts and does not claim affiliation with Renaissance Periodization or Mike Israetel.",
		},
		{
			ID:       "sam-sulek",
			Name:     "Sam Sulek",
			Initials: "SS",
			Category: "Conversational bodybuilding",
			Status:   "editorial",
			Summary:  "Straightforward gym-focused influence emphasizing consistency, simple sessions, and honest training reflection.",
			Principles: []string{
				"Keep the session understandable and focused on the target muscle.",
				"Reflect on execution and adjust the next workout based on performance.",
				"Do not copy another person's loads, drug use, diet extremes, or recovery capacity.",
			},
			Communication: []string{"conversational", "unpolished", "practical"},
			SafetyNote:    "FormForge does not imitate Sam Sulek's exact speech or endorse extreme practices.",
		},
		{
			ID:       "david-goggins",
			Name:     "David Goggins",
			Initials: "DG",
			Category: "Discipline and accountability",
			Status:   "editorial",
			Summary:  "Discipline-focused influence used for accountability and follow-through, not for overriding recovery or medical warning signs.",
			Principles: []string{
				"Act on the plan even when motivation is low.",
				"Use measurable commitments and honest self-review.",
				"Discipline does not mean training through injury, dangerous symptoms, or severe exhaustion.",
			},
			Communication: []string{"firm", "accountability-focused", "action-oriented"},
			SafetyNote:    "FormForge avoids abusive language, exact imitation, and unsafe endurance escalation.",
		},
		{
			ID:       "formforge-balanced",
			Name:     "FormForge Balanced",
			Initials: "FF",
			Category: "General fitness",
			Status:   "official",
			Summary:  "FormForge's built-in balanced coaching profile for clear programming, sustainable progress, and safety.",
			Principles: []string{
				"Choose the smallest effective plan that the user can repeat.",
				"Progress reps, load, or execution while protecting technique and recovery.",
				"Use logged data rather than one emotional day to judge progress.",
			},
			Communication: []string{"clear", "encouraging", "direct"},
			SafetyNote:    "Official FormForge profile.",
		},
	}
}

func coachCatalogMap() map[string]CoachProfile {
	out := map[string]CoachProfile{}
	for _, p := range ExpandedCoachCatalog() {
		out[p.ID] = p
	}
	return out
}

func normalizeCoachPreferences(in CoachPreferences, userID string) (CoachPreferences, error) {
	catalog := coachCatalogMap()
	style := strings.ToLower(strings.TrimSpace(in.ResponseStyle))
	switch style {
	case "balanced", "teach", "push", "simple", "recovery":
	default:
		style = "balanced"
	}
	seen := map[string]bool{}
	clean := make([]CoachSelection, 0, len(in.Influences))
	total := 0
	for _, item := range in.Influences {
		id := strings.ToLower(strings.TrimSpace(item.ProfileID))
		if _, ok := catalog[id]; !ok || seen[id] {
			continue
		}
		w := item.Weight
		if w < 1 {
			w = 1
		}
		if w > 100 {
			w = 100
		}
		seen[id] = true
		clean = append(clean, CoachSelection{ProfileID: id, Weight: w})
		total += w
		if len(clean) == 5 {
			break
		}
	}
	if len(clean) == 0 {
		clean = []CoachSelection{{ProfileID: "formforge-balanced", Weight: 100}}
		total = 100
	}
	// Store a normalized 100-point blend so the UI and prompts agree.
	running := 0
	for i := range clean {
		if i == len(clean)-1 {
			clean[i].Weight = 100 - running
			break
		}
		clean[i].Weight = int(float64(clean[i].Weight)/float64(total)*100 + 0.5)
		if clean[i].Weight < 1 {
			clean[i].Weight = 1
		}
		running += clean[i].Weight
	}
	if clean[len(clean)-1].Weight < 1 {
		clean[len(clean)-1].Weight = 1
		clean[0].Weight -= 1
	}
	sort.SliceStable(clean, func(i, j int) bool { return clean[i].Weight > clean[j].Weight })
	return CoachPreferences{UserID: userID, Influences: clean, ResponseStyle: style, UpdatedAt: nowISO()}, nil
}

func defaultCoachPreferences(userID string) CoachPreferences {
	p, _ := normalizeCoachPreferences(CoachPreferences{Influences: []CoachSelection{{ProfileID: "formforge-balanced", Weight: 100}}, ResponseStyle: "balanced"}, userID)
	return p
}

func preferencesFor(db Database, userID string) CoachPreferences {
	p, ok := db.CoachPreferences[userID]
	if !ok || len(p.Influences) == 0 {
		return defaultCoachPreferences(userID)
	}
	clean, err := normalizeCoachPreferencesForDB(p, userID, db)
	if err != nil {
		return defaultCoachPreferences(userID)
	}
	return clean
}

func selectedCoachProfiles(db Database, userID string) []CoachProfile {
	catalog := coachCatalogMapFor(db)
	prefs := preferencesFor(db, userID)
	counts := map[string]int{}
	for _, src := range db.CoachSources {
		counts[src.ProfileID]++
	}
	out := []CoachProfile{}
	for _, sel := range prefs.Influences {
		if p, ok := catalog[sel.ProfileID]; ok {
			p.SourceCount = counts[p.ID]
			out = append(out, p)
		}
	}
	return out
}

func coachBlendSummary(db Database, userID string) string {
	prefs := preferencesFor(db, userID)
	catalog := coachCatalogMapFor(db)
	parts := []string{}
	for _, sel := range prefs.Influences {
		if p, ok := catalog[sel.ProfileID]; ok {
			parts = append(parts, fmt.Sprintf("%d%% %s", sel.Weight, p.Name))
		}
	}
	return strings.Join(parts, " · ")
}

func approvedCoachSources(db Database, userID string) []CoachSource {
	selected := map[string]bool{}
	for _, sel := range preferencesFor(db, userID).Influences {
		selected[sel.ProfileID] = true
	}
	out := []CoachSource{}
	for _, src := range db.CoachSources {
		if selected[src.ProfileID] {
			out = append(out, src)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ProfileID != out[j].ProfileID {
			return out[i].ProfileID < out[j].ProfileID
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

func validateCoachSource(in CoachSource) (CoachSource, error) {
	in.ProfileID = strings.ToLower(strings.TrimSpace(in.ProfileID))
	if in.ProfileID == "" {
		return in, fmt.Errorf("choose a valid coaching profile")
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	in.SourceURL = strings.TrimSpace(in.SourceURL)
	in.Summary = strings.TrimSpace(in.Summary)
	in.Excerpt = strings.TrimSpace(in.Excerpt)
	in.Quote = strings.TrimSpace(in.Quote)
	if in.Title == "" || len(in.Title) > 180 {
		return in, fmt.Errorf("source title is required and must be under 180 characters")
	}
	switch in.Kind {
	case "article", "video", "transcript", "program", "research", "quote", "other":
	default:
		return in, fmt.Errorf("source kind is invalid")
	}
	if in.Summary == "" || len(in.Summary) > 3000 {
		return in, fmt.Errorf("an original summary is required and must be under 3,000 characters")
	}
	if len(in.Excerpt) > 1200 || len(in.Quote) > 500 {
		return in, fmt.Errorf("excerpt or quote is too long")
	}
	if in.SourceURL != "" {
		u, err := url.Parse(in.SourceURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return in, fmt.Errorf("source URL must be a valid HTTP or HTTPS address")
		}
	}
	if in.QuoteVerified && (in.Quote == "" || in.SourceURL == "") {
		return in, fmt.Errorf("a verified quote requires both exact quote text and a source URL")
	}
	return in, nil
}

func styleInstruction(style string) string {
	switch style {
	case "teach":
		return "Explain the reasoning and tradeoffs in plain language before the action steps."
	case "push":
		return "Be energetic and demanding while remaining respectful, safe, and specific."
	case "simple":
		return "Use short instructions, minimal theory, and a clear next action."
	case "recovery":
		return "Bias toward sustainable effort, recovery, fatigue reduction, and consistency."
	default:
		return "Balance concise teaching, encouragement, and direct action steps."
	}
}

func coachPromptContext(db Database, userID string) string {
	prefs := preferencesFor(db, userID)
	catalog := coachCatalogMap()
	var b strings.Builder
	b.WriteString("\nCOACHING TEAM CONFIGURATION:\n")
	for _, sel := range prefs.Influences {
		p, ok := catalog[sel.ProfileID]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "- %d%% %s (%s; %s): %s Principles: %s Communication guidance: %s.\n", sel.Weight, p.Name, p.Status, p.Category, p.Summary, strings.Join(p.Principles, " "), strings.Join(p.Communication, ", "))
	}
	fmt.Fprintf(&b, "Response mode: %s. %s\n", prefs.ResponseStyle, styleInstruction(prefs.ResponseStyle))
	b.WriteString("Identity and attribution rules: You are FormForge Coach, never the selected creator. Do not say or imply that a creator wrote, approved, endorsed, or personally delivered the answer. Do not imitate distinctive wording, catchphrases, or an exact voice. Use the selected profiles only to blend broad training principles and communication traits. Never invent a quotation. Use an exact quotation only when it appears below as VERIFIED QUOTE, and identify its source.\n")
	sources := approvedCoachSources(db, userID)
	if len(sources) == 0 {
		b.WriteString("Approved source library: none. Treat all creator profiles as FormForge editorial summaries and do not attribute specific claims or quotes to a creator.\n")
		return b.String()
	}
	b.WriteString("APPROVED SOURCE LIBRARY:\n")
	if len(sources) > 12 {
		sources = sources[:12]
	}
	for _, src := range sources {
		p := catalog[src.ProfileID]
		fmt.Fprintf(&b, "- %s — %s (%s, licensed=%t): %s", p.Name, src.Title, src.Kind, src.Licensed, src.Summary)
		if src.SourceURL != "" {
			fmt.Fprintf(&b, " Source URL: %s", src.SourceURL)
		}
		if src.Excerpt != "" {
			fmt.Fprintf(&b, " Approved excerpt: %q", src.Excerpt)
		}
		if src.QuoteVerified && src.Quote != "" {
			fmt.Fprintf(&b, " VERIFIED QUOTE: %q", src.Quote)
		}
		b.WriteString("\n")
	}
	b.WriteString("When source-grounding materially affects the answer, end with a short 'Basis' section naming the relevant approved source titles.\n")
	return b.String()
}

func coachToneLead(db Database, userID string) string {
	prefs := preferencesFor(db, userID)
	switch prefs.ResponseStyle {
	case "teach":
		return "Here is the reasoning first, then the plan."
	case "push":
		return "No waiting for perfect motivation. Execute this with control."
	case "simple":
		return "Keep it simple and complete the work below."
	case "recovery":
		return "The goal is productive training you can recover from."
	default:
		return "Here is the practical plan and why it fits."
	}
}

func coachBasis(db Database, userID string) string {
	blend := coachBlendSummary(db, userID)
	if blend == "" {
		blend = "FormForge Balanced"
	}
	parts := []string{"Coach blend: " + blend}
	sources := approvedCoachSources(db, userID)
	if len(sources) > 0 {
		names := []string{}
		for _, src := range sources {
			if len(names) == 3 {
				break
			}
			names = append(names, src.Title)
		}
		parts = append(parts, "Approved sources: "+strings.Join(names, "; "))
	} else {
		parts = append(parts, "Basis: FormForge editorial profiles; no creator endorsement or verified quote is implied")
	}
	return strings.Join(parts, "\n")
}

func verifiedQuoteReply(db Database, userID string) string {
	catalog := coachCatalogMap()
	for _, src := range approvedCoachSources(db, userID) {
		if !src.QuoteVerified || strings.TrimSpace(src.Quote) == "" {
			continue
		}
		name := catalog[src.ProfileID].Name
		return fmt.Sprintf("Verified quote from %s:\n\n\u201c%s\u201d\n\nSource: %s\n%s", name, src.Quote, src.SourceURL, coachBasis(db, userID))
	}
	return "I do not have a verified quotation in your approved source library yet. I will not invent or misattribute one. An administrator can add a short exact quote, its source URL, and mark it verified in Coaching Team."
}

func offlineInfluenceAdjustments(db Database, userID string) []string {
	selected := map[string]bool{}
	for _, sel := range preferencesFor(db, userID).Influences {
		selected[sel.ProfileID] = true
	}
	out := []string{}
	if selected["jeff-nippard"] {
		out = append(out, "Evidence-focused adjustment: keep exercise selection stable for at least 4–6 weeks and record reps in reserve.")
	}
	if selected["mike-israetel"] {
		out = append(out, "Fatigue-management adjustment: add sets only when performance, soreness, and recovery remain good; deload when those trends decline together.")
	}
	if selected["ronnie-coleman"] {
		out = append(out, "High-effort adjustment: push the final safe isolation set hard when technique is stable, but do not turn heavy compounds into uncontrolled grinders.")
	}
	if selected["arnold-schwarzenegger"] {
		out = append(out, "Golden-era adjustment: finish the target muscle with two controlled pump sets and focus on the contraction.")
	}
	if selected["noel-deyzel"] {
		out = append(out, "Consistency adjustment: complete the basic session before adding optional work; a repeatable week beats a perfect one-off workout.")
	}
	if selected["sam-sulek"] {
		out = append(out, "Conversational bodybuilding adjustment: keep each session focused on one clear target and write one sentence afterward about what worked.")
	}
	if selected["david-goggins"] {
		out = append(out, "Accountability adjustment: schedule the sessions now and log completion; discipline never overrides injury or dangerous symptoms.")
	}
	return out
}
