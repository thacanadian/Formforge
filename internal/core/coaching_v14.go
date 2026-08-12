package core

import "strings"

func AdditionalCoachCatalog() []CoachProfile {
	return []CoachProfile{
		{ID: "layne-norton", Name: "Layne Norton", Initials: "LN", Category: "Evidence-based nutrition and strength", Status: "editorial", Summary: "Evidence-focused nutrition, flexible dieting, strength progression, and myth correction.", Principles: []string{"Use measurable nutrition targets without moralizing food.", "Prioritize resistance training and adequate protein.", "Change the plan when trend data—not a single day—supports it."}, Communication: []string{"direct", "evidence-focused", "myth-resistant"}, SafetyNote: "Editorial profile; no endorsement or identity imitation."},
		{ID: "eric-helms", Name: "Eric Helms", Initials: "EH", Category: "Natural bodybuilding programming", Status: "editorial", Summary: "Sustainable natural bodybuilding with hierarchy-based nutrition and autoregulated programming.", Principles: []string{"Put adherence and energy balance before minor details.", "Use RPE/RIR to autoregulate training.", "Match volume to experience and recovery."}, Communication: []string{"measured", "educational", "practical"}, SafetyNote: "Editorial profile; no endorsement."},
		{ID: "jeff-cavaliere", Name: "Jeff Cavaliere", Initials: "JC", Category: "Athletic training and exercise mechanics", Status: "editorial", Summary: "Exercise-mechanics influence emphasizing athletic movement, joint awareness, and practical substitutions.", Principles: []string{"Choose exercises that can be performed pain-free.", "Train movement quality and neglected stabilizers.", "Use substitutions when equipment or joints limit a movement."}, Communication: []string{"instructional", "mechanics-focused", "direct"}, SafetyNote: "Editorial profile; not an official ATHLEAN-X product."},
		{ID: "jeremy-ethier", Name: "Jeremy Ethier", Initials: "JE", Category: "Evidence-based beginner fitness", Status: "editorial", Summary: "Accessible evidence-informed training, nutrition, and body-composition education.", Principles: []string{"Keep beginner plans simple and repeatable.", "Use clear exercise cues and progression rules.", "Support fat loss with sustainable calorie control."}, Communication: []string{"clear", "structured", "beginner-friendly"}, SafetyNote: "Editorial profile; no endorsement."},
		{ID: "chris-bumstead", Name: "Chris Bumstead", Initials: "CB", Category: "Classic physique bodybuilding", Status: "editorial", Summary: "Classic-physique influence emphasizing balanced development, consistency, and controlled bodybuilding work.", Principles: []string{"Build symmetry across the full physique.", "Use controlled reps and stable exercises.", "Keep consistency higher than novelty."}, Communication: []string{"calm", "confident", "bodybuilding-focused"}, SafetyNote: "Editorial profile; no endorsement or exact imitation."},
		{ID: "jay-cutler", Name: "Jay Cutler", Initials: "JC", Category: "High-volume professional bodybuilding", Status: "editorial", Summary: "Methodical bodybuilding influence centered on routine, volume tolerance, and repeatable execution.", Principles: []string{"Use repeatable routines and track performance.", "Distribute volume across stable exercises.", "Do not copy professional drug use or recovery capacity."}, Communication: []string{"steady", "practical", "workmanlike"}, SafetyNote: "Editorial profile; not medical or contest-prep advice."},
		{ID: "dorian-yates", Name: "Dorian Yates", Initials: "DY", Category: "High-intensity bodybuilding", Status: "editorial", Summary: "Low-volume, high-intensity bodybuilding influence with careful warm-ups and focused working sets.", Principles: []string{"Use fewer true working sets when effort is very high.", "Warm up without exhausting the target muscle.", "Never use intensity to justify unsafe form or pain."}, Communication: []string{"focused", "intense", "minimalist"}, SafetyNote: "Editorial profile; unsafe maximal effort is not copied."},
		{ID: "tom-platz", Name: "Tom Platz", Initials: "TP", Category: "Leg training intensity", Status: "editorial", Summary: "Leg-training enthusiasm, deep effort, and strong mind-muscle focus moderated by FormForge safety limits.", Principles: []string{"Give lower-body training deliberate attention.", "Use controlled range of motion appropriate to the user.", "High effort belongs only where technique remains stable."}, Communication: []string{"enthusiastic", "visual", "intense"}, SafetyNote: "Editorial profile; no unsafe pain-through-training guidance."},
		{ID: "ben-patrick", Name: "Ben Patrick", Initials: "BP", Category: "Knee and athletic resilience", Status: "editorial", Summary: "Progressive range-of-motion and lower-body resilience influence using gradual, pain-free regressions.", Principles: []string{"Regress movements until they are controlled and pain-free.", "Progress range and load gradually.", "Do not claim exercise replaces medical evaluation."}, Communication: []string{"optimistic", "progression-focused", "accessible"}, SafetyNote: "Editorial profile; not injury treatment."},
		{ID: "megsquats", Name: "MegSquats", Initials: "MS", Category: "Strength training accessibility", Status: "editorial", Summary: "Inclusive strength coaching influence emphasizing confidence, fundamentals, and sustainable progression.", Principles: []string{"Teach barbell fundamentals without gatekeeping.", "Progress from the user's actual starting point.", "Build confidence through measurable wins."}, Communication: []string{"supportive", "inclusive", "direct"}, SafetyNote: "Editorial profile; no endorsement."},
		{ID: "sohee-lee", Name: "Sohee Lee", Initials: "SL", Category: "Sustainable nutrition and behavior", Status: "editorial", Summary: "Behavior-focused fitness and nutrition influence emphasizing adherence, flexibility, and realistic expectations.", Principles: []string{"Design habits around real-life constraints.", "Avoid all-or-nothing food rules.", "Use compassionate accountability."}, Communication: []string{"empathetic", "practical", "behavior-focused"}, SafetyNote: "Editorial profile; no endorsement."},
		{ID: "greg-doucette", Name: "Greg Doucette", Initials: "GD", Category: "Body-composition accountability", Status: "editorial", Summary: "High-energy accountability influence focused on calorie awareness and consistent effort, without exact voice imitation.", Principles: []string{"Make calorie intake visible and measurable.", "Train harder than last time only when recovery allows.", "Avoid extreme deficits and abusive messaging."}, Communication: []string{"energetic", "blunt", "accountability-focused"}, SafetyNote: "Editorial profile; exact speech and insults are excluded."},
		{ID: "will-tennyson", Name: "Will Tennyson", Initials: "WT", Category: "Balanced lifestyle fitness", Status: "editorial", Summary: "Relatable fitness influence blending training consistency, food flexibility, and enjoyment.", Principles: []string{"Keep fitness compatible with normal life.", "Use humor without weakening the action plan.", "Balance performance and enjoyment."}, Communication: []string{"friendly", "light", "practical"}, SafetyNote: "Editorial profile; no endorsement."},
		{ID: "hybrid-calisthenics", Name: "Hybrid Calisthenics", Initials: "HC", Category: "Calisthenics progressions", Status: "editorial", Summary: "Accessible bodyweight progressions emphasizing kindness, patience, and scalable movement practice.", Principles: []string{"Use regressions that the user can perform cleanly.", "Progress one variable at a time.", "Avoid shame around starting level."}, Communication: []string{"kind", "calm", "beginner-friendly"}, SafetyNote: "Editorial profile; no endorsement."},
		{ID: "alan-thrall", Name: "Alan Thrall", Initials: "AT", Category: "Barbell fundamentals", Status: "editorial", Summary: "Straightforward barbell technique and strength-programming influence.", Principles: []string{"Practice stable barbell technique.", "Use sensible novice progression before complexity.", "Treat cues as tools, not universal laws."}, Communication: []string{"plainspoken", "technical", "practical"}, SafetyNote: "Editorial profile; no endorsement."},
		{ID: "bret-contreras", Name: "Bret Contreras", Initials: "BC", Category: "Glute training", Status: "editorial", Summary: "Glute-development influence emphasizing exercise variety, progressive loading, and targeted programming.", Principles: []string{"Combine shortened- and lengthened-position glute work.", "Progress load and reps with stable technique.", "Balance specialization with full-body training."}, Communication: []string{"technical", "specialized", "direct"}, SafetyNote: "Editorial profile; no endorsement."},
	}
}

func ExpandedCoachCatalog() []CoachProfile {
	return append(CoachCatalog(), AdditionalCoachCatalog()...)
}

func coachProfileFromCustom(p CustomCoachProfile, sourceCount int) CoachProfile {
	return CoachProfile{ID: p.ID, Name: p.Name, Initials: p.Initials, Category: p.Category, Status: p.Status, Summary: p.Summary, Principles: append([]string(nil), p.Principles...), Communication: append([]string(nil), p.Communication...), SafetyNote: p.SafetyNote, SourceCount: sourceCount}
}

func coachCatalogMapFor(db Database) map[string]CoachProfile {
	out := map[string]CoachProfile{}
	for _, p := range ExpandedCoachCatalog() {
		out[p.ID] = p
	}
	counts := map[string]int{}
	for _, s := range db.CoachSources {
		counts[s.ProfileID]++
	}
	for id, p := range db.CustomCoachProfiles {
		if p.Status != "removed" {
			out[id] = coachProfileFromCustom(p, counts[id])
		}
	}
	return out
}

func normalizeCoachPreferencesForDB(in CoachPreferences, userID string, db Database) (CoachPreferences, error) {
	catalog := coachCatalogMapFor(db)
	style := strings.ToLower(strings.TrimSpace(in.ResponseStyle))
	switch style {
	case "balanced", "teach", "push", "simple", "recovery":
	default:
		style = "balanced"
	}
	seen := map[string]bool{}
	clean := []CoachSelection{}
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
	running := 0
	for i := range clean {
		if i == len(clean)-1 {
			clean[i].Weight = 100 - running
			break
		}
		clean[i].Weight = int(float64(clean[i].Weight)/float64(total)*100 + .5)
		if clean[i].Weight < 1 {
			clean[i].Weight = 1
		}
		running += clean[i].Weight
	}
	preferred := strings.ToLower(strings.TrimSpace(in.PreferredCoachID))
	if _, ok := catalog[preferred]; !ok {
		preferred = clean[0].ProfileID
	}
	return CoachPreferences{UserID: userID, Influences: clean, ResponseStyle: style, PreferredCoachID: preferred, UpdatedAt: nowISO()}, nil
}
