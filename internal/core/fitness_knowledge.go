package core

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// FitnessKnowledgeChunk is a compact, locally bundled coaching reference.
// It is intentionally written in original language and is not a substitute for
// licensed textbooks, medical care, or current primary-source research.
type FitnessKnowledgeChunk struct {
	ID       string   `json:"id"`
	Domain   string   `json:"domain"`
	Title    string   `json:"title"`
	Keywords []string `json:"keywords"`
	Content  string   `json:"content"`
}

var fitnessKnowledgeVault = []FitnessKnowledgeChunk{
	{ID: "hypertrophy-volume", Domain: "Hypertrophy", Title: "Volume and recoverability", Keywords: []string{"hypertrophy", "volume", "sets", "muscle growth", "recoverable"}, Content: "Use enough challenging weekly sets to create progress, then adjust from performance and recovery rather than chasing a universal set count. Add volume only when technique, effort, sleep, soreness, and progression remain stable."},
	{ID: "hypertrophy-proximity", Domain: "Hypertrophy", Title: "Proximity to failure", Keywords: []string{"failure", "rir", "rpe", "hypertrophy", "effort"}, Content: "Most hypertrophy work can be productive with roughly 0–4 reps in reserve. Compounds usually carry a higher fatigue and technique cost near failure than stable isolation exercises."},
	{ID: "hypertrophy-length", Domain: "Hypertrophy", Title: "Range of motion and long-muscle training", Keywords: []string{"range of motion", "stretch", "lengthened", "rom", "hypertrophy"}, Content: "Favor controlled, pain-free range of motion that loads the target muscle. Lengthened positions can be useful, but exercise selection and depth must respect anatomy, stability, and symptoms."},
	{ID: "hypertrophy-frequency", Domain: "Hypertrophy", Title: "Training frequency", Keywords: []string{"frequency", "twice per week", "split", "hypertrophy"}, Content: "Frequency mainly distributes weekly work. Train a muscle often enough that set quality stays high and soreness does not interfere with the next exposure; many people do well with two exposures per week."},
	{ID: "hypertrophy-exercise-selection", Domain: "Hypertrophy", Title: "Exercise selection", Keywords: []string{"exercise selection", "stimulus", "stability", "hypertrophy"}, Content: "Choose movements that fit the lifter, load the intended tissue, allow stable progression, and do not create disproportionate joint or systemic fatigue. A good exercise is repeatable, measurable, and tolerable."},
	{ID: "strength-specificity", Domain: "Strength", Title: "Specificity", Keywords: []string{"strength", "specificity", "squat", "bench", "deadlift", "skill"}, Content: "Strength is partly skill. Practice the competition or target lift often enough to maintain technique, while using variations and accessories to build weak ranges and muscle mass."},
	{ID: "strength-intensity", Domain: "Strength", Title: "Heavy exposure and fatigue", Keywords: []string{"heavy", "one rep max", "intensity", "strength", "fatigue"}, Content: "Heavy work improves familiarity with high force, but frequent grinders can degrade technique and recovery. Use most work below maximal effort and reserve true max attempts for testing or competition."},
	{ID: "strength-periodization", Domain: "Strength", Title: "Periodization", Keywords: []string{"periodization", "block", "strength", "peaking"}, Content: "Organize training so volume, intensity, specificity, and fatigue change with the goal. General blocks build capacity; specific blocks emphasize target lifts; peaking reduces fatigue while retaining intensity."},
	{ID: "progression-double", Domain: "Programming", Title: "Double progression", Keywords: []string{"progressive overload", "double progression", "reps", "weight"}, Content: "Keep a load until all prescribed sets reach the top of a rep range with target technique and effort, then add the smallest practical load and return toward the lower end of the range."},
	{ID: "progression-autoregulation", Domain: "Programming", Title: "Autoregulation", Keywords: []string{"autoregulation", "rpe", "rir", "readiness", "load"}, Content: "Use RPE, reps in reserve, bar speed, symptoms, and warm-up performance to adjust load or volume. Autoregulation should constrain decisions, not justify random training."},
	{ID: "progression-plateau", Domain: "Programming", Title: "Plateau diagnosis", Keywords: []string{"plateau", "stuck", "not progressing", "stall"}, Content: "Before adding complexity, check adherence, exercise consistency, calorie intake, protein, sleep, technique, range of motion, rest intervals, and whether the target is realistic. Change one major variable at a time."},
	{ID: "deload", Domain: "Programming", Title: "Deloads", Keywords: []string{"deload", "recovery week", "fatigue", "volume reduction"}, Content: "A deload reduces fatigue while preserving movement familiarity. Common options include cutting hard sets by roughly one-third to one-half, lowering load, increasing reps in reserve, or combining these."},
	{ID: "exercise-order", Domain: "Programming", Title: "Exercise order", Keywords: []string{"exercise order", "priority", "workout sequence"}, Content: "Place the highest-priority, most technically demanding, or most fatigue-sensitive work early. Later exercises can be more stable and local when systemic fatigue is higher."},
	{ID: "rest-intervals", Domain: "Programming", Title: "Rest intervals", Keywords: []string{"rest", "rest interval", "between sets"}, Content: "Rest long enough to preserve the intended performance. Compounds and strength work often need several minutes; isolation work may need less. Short rest is a tool, not a requirement for muscle growth."},
	{ID: "warmup", Domain: "Programming", Title: "Warm-ups", Keywords: []string{"warm up", "warmup", "ramp sets", "mobility"}, Content: "A warm-up should raise readiness, rehearse the movement, and expose symptoms without creating fatigue. Use a brief general warm-up when helpful, then progressively heavier movement-specific sets."},
	{ID: "technique", Domain: "Technique", Title: "Technique and consistency", Keywords: []string{"form", "technique", "tempo", "control"}, Content: "Technique should be safe, repeatable, and suitable for the goal. Consistent setup and range of motion make progression interpretable. Small individual variation is normal."},
	{ID: "tempo", Domain: "Technique", Title: "Tempo", Keywords: []string{"tempo", "eccentric", "pause", "rep speed"}, Content: "Control the eccentric enough to maintain position and target the intended tissue. Pauses and slower tempos can improve control or increase difficulty, but excessively slow reps can limit load and total productive work."},
	{ID: "bar-path", Domain: "Technique", Title: "Bar path", Keywords: []string{"bar path", "bench path", "squat path", "deadlift path"}, Content: "Efficient bar paths keep the system balanced over the base of support while respecting joint structure. Video should be interpreted with camera angle, anthropometry, load, and intent in mind."},
	{ID: "fat-loss-energy", Domain: "Nutrition", Title: "Energy deficit", Keywords: []string{"fat loss", "calorie deficit", "cut", "lose weight"}, Content: "Fat loss requires a sustained energy deficit. Use a moderate rate that preserves training performance, protein intake, sleep, and adherence. Adjust from multi-week weight trends rather than single weigh-ins."},
	{ID: "muscle-gain-energy", Domain: "Nutrition", Title: "Energy surplus", Keywords: []string{"bulk", "muscle gain", "surplus", "gain weight"}, Content: "A small consistent surplus can support muscle gain while limiting unnecessary fat gain. Faster scale gain is not automatically faster muscle gain, especially for experienced lifters."},
	{ID: "protein", Domain: "Nutrition", Title: "Protein", Keywords: []string{"protein", "muscle", "grams", "meal"}, Content: "Daily protein matters more than perfect timing. A practical range for many resistance-trained adults is roughly 1.6–2.2 g per kg per day, with higher relative intakes sometimes useful during aggressive dieting. Distribute protein across several meals."},
	{ID: "carbohydrates", Domain: "Nutrition", Title: "Carbohydrates and training", Keywords: []string{"carbs", "carbohydrate", "glycogen", "performance"}, Content: "Carbohydrates support high-volume and high-intensity training by replenishing glycogen. Place enough around demanding sessions to support performance while fitting total calories and food preferences."},
	{ID: "dietary-fat", Domain: "Nutrition", Title: "Dietary fat", Keywords: []string{"fat", "dietary fat", "hormones"}, Content: "Dietary fat supports essential physiology and food enjoyment. Avoid chronically pushing it extremely low; choose mostly unsaturated sources while allowing flexibility within calorie goals."},
	{ID: "fiber", Domain: "Nutrition", Title: "Fiber and food quality", Keywords: []string{"fiber", "vegetables", "fruit", "digestion"}, Content: "Build meals around adequate fiber, fruits, vegetables, legumes, whole grains, and minimally processed protein sources while adjusting for gastrointestinal tolerance and training timing."},
	{ID: "hydration", Domain: "Nutrition", Title: "Hydration and sodium", Keywords: []string{"hydration", "water", "sodium", "electrolytes", "sweat"}, Content: "Hydration needs vary with body size, climate, sweat rate, diet, and training. Replace fluids and sodium proportionally during long, hot, or high-sweat sessions rather than following one fixed number."},
	{ID: "meal-timing", Domain: "Nutrition", Title: "Meal timing", Keywords: []string{"meal timing", "pre workout", "post workout", "anabolic window"}, Content: "Total intake dominates, but meals containing protein and carbohydrate in the hours around training can support performance and recovery. Avoid rigid timing rules that reduce adherence."},
	{ID: "supp-creatine", Domain: "Supplements", Title: "Creatine monohydrate", Keywords: []string{"creatine", "monohydrate", "supplement"}, Content: "Creatine monohydrate is one of the best-supported performance supplements. A simple maintenance approach is 3–5 g daily. It may increase scale weight through intracellular water. People with medical concerns should ask a clinician."},
	{ID: "supp-caffeine", Domain: "Supplements", Title: "Caffeine", Keywords: []string{"caffeine", "pre workout", "stimulant"}, Content: "Caffeine can improve alertness and performance, but tolerance, anxiety, sleep disruption, heart symptoms, and total daily intake matter. Use the lowest effective dose and protect sleep."},
	{ID: "supp-beta-alanine", Domain: "Supplements", Title: "Beta-alanine", Keywords: []string{"beta alanine", "supplement", "endurance"}, Content: "Beta-alanine can help repeated high-intensity efforts lasting roughly one to several minutes by increasing muscle carnosine. Benefits require consistent dosing and are not an acute stimulant effect."},
	{ID: "supp-protein-powder", Domain: "Supplements", Title: "Protein powder", Keywords: []string{"whey", "protein powder", "casein"}, Content: "Protein powder is a convenient food supplement, not mandatory. Choose a product that fits digestion, budget, dietary restrictions, and third-party testing needs."},
	{ID: "supp-evidence", Domain: "Supplements", Title: "Supplement evaluation", Keywords: []string{"supplement", "evidence", "test booster", "fat burner"}, Content: "Evaluate supplements by plausible mechanism, human outcome evidence, dose, safety, quality control, interactions, and cost. Be skeptical of proprietary blends, dramatic hormone claims, and products replacing fundamentals."},
	{ID: "sleep", Domain: "Recovery", Title: "Sleep", Keywords: []string{"sleep", "recovery", "fatigue"}, Content: "Sleep supports performance, appetite regulation, learning, and recovery. Protect a consistent schedule, adequate duration, morning light, and a wind-down routine before adding more recovery products."},
	{ID: "soreness", Domain: "Recovery", Title: "Soreness", Keywords: []string{"sore", "doms", "muscle soreness"}, Content: "Soreness is an imperfect signal. Mild soreness can coexist with productive training; severe soreness that changes movement may justify reducing load, range, or volume. Progress does not require chasing soreness."},
	{ID: "stress", Domain: "Recovery", Title: "Life stress", Keywords: []string{"stress", "recovery", "work", "school"}, Content: "Training stress and life stress share recovery resources. During difficult periods, preserve key lifts and habits while temporarily reducing optional volume, complexity, or conditioning."},
	{ID: "pain-rules", Domain: "Pain and Injury", Title: "Pain response", Keywords: []string{"pain", "injury", "hurt", "joint"}, Content: "Do not diagnose from chat or video alone. Stop sharp, worsening, or neurologic symptoms; use pain-free ranges and alternatives; and seek qualified assessment for trauma, major swelling, weakness, numbness, persistent symptoms, chest pain, fainting, or severe breathing difficulty."},
	{ID: "pain-substitution", Domain: "Pain and Injury", Title: "Exercise substitution", Keywords: []string{"substitute", "pain-free", "regression", "injury"}, Content: "Substitute by preserving the training function: movement pattern, target tissue, range, stability, and loading intent. Change grip, implement, range, tempo, support, or exercise before abandoning the entire pattern."},
	{ID: "cardio-zones", Domain: "Cardio", Title: "Aerobic intensity", Keywords: []string{"zone 2", "cardio", "aerobic", "heart rate"}, Content: "Easy-to-moderate aerobic work should feel sustainable and conversational. Heart-rate zones are estimates; use pace, breathing, perceived effort, and device trends together."},
	{ID: "cardio-intervals", Domain: "Cardio", Title: "Intervals", Keywords: []string{"interval", "hiit", "sprint", "vo2"}, Content: "Intervals efficiently train high-intensity capacity but create more fatigue. Match work and recovery durations to the energy system and avoid stacking hard intervals near demanding lower-body sessions without a reason."},
	{ID: "concurrent", Domain: "Cardio", Title: "Concurrent training", Keywords: []string{"cardio and lifting", "concurrent", "interference"}, Content: "Strength and endurance can coexist. Manage interference by separating hard sessions when possible, controlling endurance volume, and prioritizing the quality most important to the goal."},
	{ID: "body-composition", Domain: "Body Composition", Title: "Measurement error", Keywords: []string{"body fat", "scale", "measurement", "composition"}, Content: "Body-fat estimates, smart scales, photos, and circumference measurements all have error. Use standardized conditions and trends across multiple measures rather than treating one reading as exact."},
	{ID: "scale-trends", Domain: "Body Composition", Title: "Scale trends", Keywords: []string{"weight fluctuation", "scale", "water weight"}, Content: "Daily body weight changes with glycogen, sodium, hydration, food mass, and digestion. Compare rolling averages under similar conditions before changing calories."},
	{ID: "beginners", Domain: "Populations", Title: "Beginners", Keywords: []string{"beginner", "new lifter", "novice"}, Content: "Beginners benefit from simple full-body or upper/lower plans, repeated movement practice, conservative progression, and learning effort. More variety is rarely better than consistent execution."},
	{ID: "advanced", Domain: "Populations", Title: "Advanced lifters", Keywords: []string{"advanced", "experienced", "elite"}, Content: "Advanced lifters usually progress more slowly and need more precise fatigue management, exercise selection, and specialization. Small gains are meaningful; frequent wholesale program changes obscure cause and effect."},
	{ID: "youth", Domain: "Populations", Title: "Youth resistance training", Keywords: []string{"teen", "youth", "adolescent", "18"}, Content: "Youth resistance training can be appropriate with qualified supervision, sound technique, suitable loading, and gradual progression. Avoid supplement or drug claims that exploit developmental concerns."},
	{ID: "older-adults", Domain: "Populations", Title: "Older adults", Keywords: []string{"older adult", "senior", "aging"}, Content: "Resistance, power, balance, and aerobic training can support function with age. Progress from current capacity, account for medical conditions and medications, and emphasize consistency and fall-risk reduction."},
	{ID: "female-training", Domain: "Populations", Title: "Female training considerations", Keywords: []string{"female", "women", "menstrual", "cycle"}, Content: "Most programming principles apply across sexes. Individual symptoms, iron status, pregnancy status, energy availability, and cycle-related changes may affect training, but rigid phase-based rules are not universally necessary."},
	{ID: "energy-availability", Domain: "Nutrition", Title: "Low energy availability", Keywords: []string{"low energy availability", "red-s", "amenorrhea", "underfueling"}, Content: "Persistent underfueling can impair performance, bone health, hormones, mood, and recovery. Concerning signs warrant reduced training stress and assessment by qualified medical and nutrition professionals."},
	{ID: "power", Domain: "Athletic Performance", Title: "Power development", Keywords: []string{"power", "jump", "throw", "explosive"}, Content: "Train power with high intent, low fatigue, full recovery, and technically sound movements. Stop sets when speed or jump height clearly drops."},
	{ID: "speed", Domain: "Athletic Performance", Title: "Speed training", Keywords: []string{"speed", "sprint", "acceleration"}, Content: "Speed is best trained fresh with high-quality repetitions, long recovery, and manageable volume. Sprint exposure should progress gradually because tissue demands are high."},
	{ID: "mobility", Domain: "Mobility", Title: "Mobility", Keywords: []string{"mobility", "flexibility", "stretching"}, Content: "Mobility is usable range of motion. Improve it with appropriately loaded movement, targeted stretching, and repeated practice. More range is not always better if it lacks control or relevance."},
	{ID: "calisthenics", Domain: "Training Styles", Title: "Calisthenics progression", Keywords: []string{"calisthenics", "pull up", "push up", "bodyweight"}, Content: "Progress bodyweight training by leverage, range of motion, tempo, pauses, external load, density, and skill complexity. Keep progression measurable and preserve joint tolerance."},
	{ID: "powerlifting", Domain: "Training Styles", Title: "Powerlifting", Keywords: []string{"powerlifting", "squat bench deadlift", "meet"}, Content: "Powerlifting programming balances competition-lift practice, hypertrophy, technical work, fatigue management, and peaking. Meet preparation also includes commands, attempts, equipment, and weight-class strategy."},
	{ID: "bodybuilding", Domain: "Training Styles", Title: "Bodybuilding", Keywords: []string{"bodybuilding", "posing", "symmetry", "stage"}, Content: "Bodybuilding prioritizes muscular development, symmetry, presentation, and body composition. Exercise selection and volume can be more muscle-specific, while contest preparation requires careful health oversight."},
	{ID: "nutrition-adherence", Domain: "Nutrition", Title: "Adherence", Keywords: []string{"adherence", "diet", "consistency", "meal plan"}, Content: "The best diet is nutritionally adequate, goal-aligned, affordable, culturally acceptable, and repeatable. Build flexible defaults and leave room for social eating instead of relying on perfection."},
	{ID: "food-safety", Domain: "Nutrition", Title: "Food safety", Keywords: []string{"food safety", "meal prep", "storage"}, Content: "Meal plans should account for safe cooking, refrigeration, storage time, allergies, and cross-contamination. Fitness goals never override food-safety basics."},
	{ID: "behavior-goals", Domain: "Behavior", Title: "Behavior design", Keywords: []string{"habit", "motivation", "discipline", "consistency"}, Content: "Make desired behavior easy to start, tie it to a stable cue, track a small number of meaningful actions, and recover quickly after missed days. Environment often beats willpower."},
	{ID: "evidence-hierarchy", Domain: "Evidence", Title: "Evidence hierarchy", Keywords: []string{"study", "research", "evidence", "citation"}, Content: "Interpret evidence by study design, population, intervention, comparator, outcome, effect size, uncertainty, replication, and real-world applicability. A mechanistic explanation alone does not prove a meaningful outcome."},
	{ID: "source-literacy", Domain: "Evidence", Title: "Source literacy", Keywords: []string{"source", "link", "citation", "quote"}, Content: "Use primary research and authoritative guidance when possible. Distinguish exact quotations from paraphrases, include links when requested, state when advice is general knowledge, and never invent a citation or quote."},
	{ID: "drug-safety", Domain: "Safety", Title: "Performance-enhancing drugs", Keywords: []string{"steroids", "sarms", "gear", "ped", "testosterone cycle"}, Content: "Performance-enhancing drugs carry cardiovascular, endocrine, fertility, psychiatric, hepatic, and legal risks. FormForge can discuss general risk and encourage medical care, but should not provide unsupervised cycle design, sourcing, or concealment guidance."},
	{ID: "eating-disorder", Domain: "Safety", Title: "Disordered eating", Keywords: []string{"binge", "purge", "eating disorder", "starve"}, Content: "Rigid dieting, purging, severe restriction, compulsive exercise, or intense fear around food may require professional support. The coach should prioritize safety and avoid reinforcing destructive targets."},
}

func knowledgeWords(s string) map[string]bool {
	out := map[string]bool{}
	var b strings.Builder
	flush := func() {
		if b.Len() >= 3 {
			out[b.String()] = true
		}
		b.Reset()
	}
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func fitnessKnowledgeSearch(query string, limit int) []FitnessKnowledgeChunk {
	if limit <= 0 {
		limit = 6
	}
	q := knowledgeWords(query)
	type scored struct {
		chunk FitnessKnowledgeChunk
		score int
	}
	items := make([]scored, 0, len(fitnessKnowledgeVault))
	for _, chunk := range fitnessKnowledgeVault {
		score := 0
		for word := range q {
			if strings.Contains(strings.ToLower(chunk.Title), word) {
				score += 5
			}
			if strings.Contains(strings.ToLower(chunk.Domain), word) {
				score += 3
			}
			for _, keyword := range chunk.Keywords {
				if strings.Contains(strings.ToLower(keyword), word) || strings.Contains(word, strings.ToLower(keyword)) {
					score += 4
				}
			}
			if strings.Contains(strings.ToLower(chunk.Content), word) {
				score++
			}
		}
		if score > 0 {
			items = append(items, scored{chunk: chunk, score: score})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].chunk.ID < items[j].chunk.ID
		}
		return items[i].score > items[j].score
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]FitnessKnowledgeChunk, 0, len(items))
	for _, item := range items {
		out = append(out, item.chunk)
	}
	return out
}

func fitnessKnowledgeContext(query string, limit int) string {
	chunks := fitnessKnowledgeSearch(query, limit)
	if len(chunks) == 0 {
		chunks = fitnessKnowledgeVault[:minKnowledge(limit, len(fitnessKnowledgeVault))]
	}
	var b strings.Builder
	for _, chunk := range chunks {
		fmt.Fprintf(&b, "[%s — %s] %s\n", chunk.Domain, chunk.Title, chunk.Content)
	}
	return strings.TrimSpace(b.String())
}

func minKnowledge(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fitnessKnowledgeDomains() []string {
	seen := map[string]bool{}
	var out []string
	for _, chunk := range fitnessKnowledgeVault {
		if !seen[chunk.Domain] {
			seen[chunk.Domain] = true
			out = append(out, chunk.Domain)
		}
	}
	sort.Strings(out)
	return out
}
