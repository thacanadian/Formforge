package core

func BuiltInWorkouts() map[string]Workout {
	w := []Workout{
		{ID: "b1", Name: "Full Body Strength A", Level: "beginner", Category: "Strength", Duration: 45, BuiltIn: true, Why: "Starting with full-body sessions 3x/week lets every muscle group recover while still being trained frequently — critical for learning to recruit muscle fibers in the first months.", Exercises: []Exercise{
			{Name: "Goblet Squat", Sets: 3, Reps: "10", Rest: "90s", Why: "Teaches the squat pattern safely with weight in front keeping you upright."},
			{Name: "Push-Up", Sets: 3, Reps: "8-12", Rest: "60s", Why: "Foundational push pattern for chest, shoulders and triceps."},
			{Name: "Dumbbell Row", Sets: 3, Reps: "10 each", Rest: "60s", Why: "Counterbalances pushing work and improves posture."},
			{Name: "Plank", Sets: 3, Reps: "30s", Rest: "60s", Why: "Core stability transfers to every other lift."}}},
		{ID: "b2", Name: "Full Body Strength B", Level: "beginner", Category: "Strength", Duration: 45, BuiltIn: true, Why: "Alternating A and B sessions trains similar muscles from different angles while keeping recovery manageable.", Exercises: []Exercise{
			{Name: "Romanian Deadlift", Sets: 3, Reps: "10", Rest: "90s", Why: "Teaches the hip hinge for back health and athletic power."},
			{Name: "Dumbbell Press", Sets: 3, Reps: "10", Rest: "90s", Why: "Builds balanced horizontal pressing strength."},
			{Name: "Lat Pulldown", Sets: 3, Reps: "10", Rest: "60s", Why: "Builds the lats and supports posture."},
			{Name: "Dead Bug", Sets: 3, Reps: "8 each", Rest: "60s", Why: "Trains deep core stability in a back-friendly position."}}},
		{ID: "b3", Name: "Cardio Foundation", Level: "beginner", Category: "Cardio", Duration: 30, BuiltIn: true, Why: "Zone 2 cardio builds an aerobic base, supports recovery, and improves heart efficiency.", Exercises: []Exercise{{Name: "Brisk Walk or Light Jog", Sets: 1, Reps: "30 min", Rest: "N/A", Why: "Keep a conversational pace."}}},
		{ID: "i1", Name: "Upper Power", Level: "intermediate", Category: "Strength", Duration: 60, BuiltIn: true, Why: "Upper/lower splits allow more weekly volume. Power days use heavier loads to build strength.", Exercises: []Exercise{
			{Name: "Bench Press", Sets: 4, Reps: "4-6", Rest: "3min", Why: "Heavy compound pressing recruits high-threshold motor units."},
			{Name: "Barbell Row", Sets: 4, Reps: "4-6", Rest: "3min", Why: "Balances pressing and builds upper-back thickness."},
			{Name: "Overhead Press", Sets: 3, Reps: "6-8", Rest: "2min", Why: "Develops deltoids and overhead stability."},
			{Name: "Pull-Ups", Sets: 3, Reps: "Max", Rest: "2min", Why: "Builds lats and pulling strength."}}},
		{ID: "i2", Name: "Lower Power", Level: "intermediate", Category: "Strength", Duration: 60, BuiltIn: true, Why: "A dedicated lower session gives the legs enough focused volume to progress.", Exercises: []Exercise{
			{Name: "Back Squat", Sets: 4, Reps: "4-6", Rest: "3min", Why: "Trains quads, glutes, hamstrings and core together."},
			{Name: "Romanian Deadlift", Sets: 3, Reps: "6-8", Rest: "2min", Why: "Balances quad-dominant work with hip-dominant strength."},
			{Name: "Leg Press", Sets: 3, Reps: "10-12", Rest: "90s", Why: "Adds quad volume with less spinal loading."},
			{Name: "Calf Raise", Sets: 4, Reps: "15-20", Rest: "60s", Why: "Provides direct calf volume."}}},
		{ID: "i3", Name: "Upper Hypertrophy", Level: "intermediate", Category: "Strength", Duration: 60, BuiltIn: true, Why: "Moderate loads and higher reps increase muscle-building volume and time under tension.", Exercises: []Exercise{
			{Name: "Incline Dumbbell Press", Sets: 4, Reps: "10-12", Rest: "90s", Why: "Emphasizes upper chest."},
			{Name: "Cable Row", Sets: 4, Reps: "12-15", Rest: "90s", Why: "Maintains tension through the rowing motion."},
			{Name: "Lateral Raise", Sets: 4, Reps: "15-20", Rest: "60s", Why: "Builds side-delt width."},
			{Name: "Bicep Curl", Sets: 3, Reps: "12-15", Rest: "60s", Why: "Adds direct elbow-flexor volume."},
			{Name: "Tricep Pushdown", Sets: 3, Reps: "12-15", Rest: "60s", Why: "Adds direct triceps volume."}}},
		{ID: "i4", Name: "Lower Hypertrophy", Level: "intermediate", Category: "Strength", Duration: 55, BuiltIn: true, Why: "A second lower day uses different angles and rep ranges for balanced development.", Exercises: []Exercise{
			{Name: "Hack Squat", Sets: 4, Reps: "10-12", Rest: "2min", Why: "Quad-focused squatting with lower technique demand."},
			{Name: "Walking Lunge", Sets: 3, Reps: "12 each", Rest: "90s", Why: "Unilateral work helps address side-to-side imbalances."},
			{Name: "Leg Curl", Sets: 4, Reps: "12-15", Rest: "90s", Why: "Direct hamstring work balances the quads."},
			{Name: "Hip Thrust", Sets: 3, Reps: "15", Rest: "90s", Why: "Directly trains hip extension and glutes."}}},
		{ID: "i5", Name: "HIIT Cardio", Level: "intermediate", Category: "Cardio", Duration: 30, BuiltIn: true, Why: "Intervals improve high-end cardiovascular capacity in a time-efficient format.", Exercises: []Exercise{{Name: "Sprint Intervals", Sets: 8, Reps: "30s on / 90s off", Rest: "N/A", Why: "Alternate hard efforts with full recovery."}}},
		{ID: "a1", Name: "Push (Chest/Shoulders/Tri)", Level: "advanced", Category: "Strength", Duration: 70, BuiltIn: true, Exercises: []Exercise{
			{Name: "Barbell Bench Press", Sets: 5, Reps: "5", Rest: "3min"}, {Name: "Overhead Press", Sets: 4, Reps: "6-8", Rest: "2min"}, {Name: "Incline Dumbbell Press", Sets: 3, Reps: "10-12", Rest: "90s"}, {Name: "Cable Lateral Raise", Sets: 4, Reps: "15", Rest: "60s"}, {Name: "Skull Crusher", Sets: 3, Reps: "10-12", Rest: "60s"}}},
		{ID: "a2", Name: "Pull (Back/Biceps)", Level: "advanced", Category: "Strength", Duration: 65, BuiltIn: true, Exercises: []Exercise{
			{Name: "Deadlift", Sets: 4, Reps: "3-5", Rest: "4min"}, {Name: "Pull-Ups (Weighted)", Sets: 4, Reps: "6-8", Rest: "2min"}, {Name: "Cable Row", Sets: 4, Reps: "10-12", Rest: "90s"}, {Name: "Face Pull", Sets: 3, Reps: "15-20", Rest: "60s"}, {Name: "Hammer Curl", Sets: 3, Reps: "12", Rest: "60s"}}},
		{ID: "a3", Name: "Legs", Level: "advanced", Category: "Strength", Duration: 75, BuiltIn: true, Exercises: []Exercise{
			{Name: "Back Squat", Sets: 5, Reps: "5", Rest: "4min"}, {Name: "Romanian Deadlift", Sets: 4, Reps: "8", Rest: "2min"}, {Name: "Leg Press", Sets: 4, Reps: "12-15", Rest: "2min"}, {Name: "Leg Curl", Sets: 3, Reps: "12-15", Rest: "90s"}, {Name: "Calf Raise", Sets: 5, Reps: "15-20", Rest: "60s"}}},
	}
	out := map[string]Workout{}
	for _, x := range w {
		out[x.ID] = x
	}
	return out
}

func DefaultHabits(userID string) []Habit {
	return []Habit{
		{ID: RandomID("habit_"), UserID: userID, Name: "Morning workout", Icon: "🏋️", Category: "Training"},
		{ID: RandomID("habit_"), UserID: userID, Name: "Hit protein goal", Icon: "💪", Category: "Nutrition"},
		{ID: RandomID("habit_"), UserID: userID, Name: "Sleep 7-8 hours", Icon: "😴", Category: "Recovery"},
		{ID: RandomID("habit_"), UserID: userID, Name: "Drink 3L water", Icon: "💧", Category: "Health"},
		{ID: RandomID("habit_"), UserID: userID, Name: "Stretch / Mobility work", Icon: "🧘", Category: "Recovery"},
	}
}

var CalorieDB = map[string]NutritionEntry{
	"banana":         {Name: "Banana", Calories: 89, Protein: 1.1, Carbs: 23, Fat: .3, Serving: "1 medium (118g)"},
	"chicken breast": {Name: "Chicken Breast", Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Serving: "100g"},
	"rice":           {Name: "White Rice", Calories: 130, Protein: 2.7, Carbs: 28, Fat: .3, Serving: "100g cooked"},
	"egg":            {Name: "Egg", Calories: 72, Protein: 6, Carbs: .4, Fat: 5, Serving: "1 large"},
	"oats":           {Name: "Oats", Calories: 389, Protein: 17, Carbs: 66, Fat: 7, Serving: "100g dry"},
	"greek yogurt":   {Name: "Greek Yogurt", Calories: 59, Protein: 10, Carbs: 3.6, Fat: .4, Serving: "100g"},
	"broccoli":       {Name: "Broccoli", Calories: 34, Protein: 2.8, Carbs: 7, Fat: .4, Serving: "100g"},
	"salmon":         {Name: "Salmon", Calories: 208, Protein: 20, Carbs: 0, Fat: 13, Serving: "100g"},
	"apple":          {Name: "Apple", Calories: 52, Protein: .3, Carbs: 14, Fat: .2, Serving: "1 medium (182g)"},
	"protein shake":  {Name: "Protein Shake", Calories: 120, Protein: 25, Carbs: 5, Fat: 2, Serving: "1 scoop"},
}
