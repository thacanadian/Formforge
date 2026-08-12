package core

import (
	"encoding/json"
	"time"
)

const SchemaVersion = 9

type User struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	Email                    string   `json:"email"`
	PasswordHash             string   `json:"passwordHash"`
	Role                     string   `json:"role"`
	Active                   bool     `json:"active"`
	FailedAttempts           int      `json:"failedAttempts"`
	LockedUntil              string   `json:"lockedUntil,omitempty"`
	CreatedAt                string   `json:"createdAt"`
	UpdatedAt                string   `json:"updatedAt"`
	TOTPEnabled              bool     `json:"totpEnabled,omitempty"`
	TOTPSecretEncrypted      string   `json:"totpSecretEncrypted,omitempty"`
	TOTPRecoveryHashes       []string `json:"totpRecoveryHashes,omitempty"`
	PlanTier                 string   `json:"planTier,omitempty"`
	TermsAcceptedVersion     string   `json:"termsAcceptedVersion,omitempty"`
	TermsAcceptedAt          string   `json:"termsAcceptedAt,omitempty"`
	PrivacyAcceptedVersion   string   `json:"privacyAcceptedVersion,omitempty"`
	PrivacyAcceptedAt        string   `json:"privacyAcceptedAt,omitempty"`
	CommunityAcceptedVersion string   `json:"communityAcceptedVersion,omitempty"`
	CommunityAcceptedAt      string   `json:"communityAcceptedAt,omitempty"`
	AgeConfirmedAt           string   `json:"ageConfirmedAt,omitempty"`
	CommunityBanned          bool     `json:"communityBanned,omitempty"`
	CommunitySuspendedUntil  string   `json:"communitySuspendedUntil,omitempty"`
	CommunityBanReason       string   `json:"communityBanReason,omitempty"`
}

type Profile struct {
	UserID       string  `json:"userId"`
	Name         string  `json:"name"`
	Age          int     `json:"age"`
	Gender       string  `json:"gender"`
	HeightCM     float64 `json:"heightCm"`
	WeightKG     float64 `json:"weightKg"`
	GoalWeightKG float64 `json:"goalWeightKg"`
	Goal         string  `json:"goal"`
	Experience   string  `json:"experience"`
	DaysPerWeek  int     `json:"daysPerWeek"`
	Equipment    string  `json:"equipment"`
	CalorieGoal  int     `json:"calorieGoal"`
	ProteinGoal  int     `json:"proteinGoal"`
	CarbGoal     int     `json:"carbGoal"`
	FatGoal      int     `json:"fatGoal"`
	UpdatedAt    string  `json:"updatedAt"`
}

type Exercise struct {
	Name string `json:"name"`
	Sets int    `json:"sets"`
	Reps string `json:"reps"`
	Rest string `json:"rest"`
	Why  string `json:"why,omitempty"`
}
type Workout struct {
	ID        string     `json:"id"`
	OwnerID   string     `json:"ownerId,omitempty"`
	Name      string     `json:"name"`
	Level     string     `json:"level"`
	Category  string     `json:"category"`
	Duration  int        `json:"duration"`
	Why       string     `json:"why,omitempty"`
	Exercises []Exercise `json:"exercises"`
	BuiltIn   bool       `json:"builtIn"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}
type ExercisePerformance struct {
	ExerciseName string  `json:"exerciseName"`
	WeightKG     float64 `json:"weightKg,omitempty"`
	Reps         int     `json:"reps,omitempty"`
	Sets         int     `json:"sets,omitempty"`
	RPE          float64 `json:"rpe,omitempty"`
	Completed    bool    `json:"completed"`
}
type WorkoutLog struct {
	ID          string                `json:"id"`
	UserID      string                `json:"userId"`
	WorkoutID   string                `json:"workoutId,omitempty"`
	WorkoutName string                `json:"workoutName"`
	Date        string                `json:"date"`
	Duration    int                   `json:"duration"`
	Notes       string                `json:"notes,omitempty"`
	Performance []ExercisePerformance `json:"performance,omitempty"`
	CompletedAt string                `json:"completedAt"`
}

type NutritionEntry struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	Date      string  `json:"date"`
	Name      string  `json:"name"`
	Serving   string  `json:"serving,omitempty"`
	Calories  float64 `json:"calories"`
	Protein   float64 `json:"protein"`
	Carbs     float64 `json:"carbs"`
	Fat       float64 `json:"fat"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}
type Habit struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Category  string `json:"category"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}
type HabitLog struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	HabitID   string `json:"habitId"`
	Date      string `json:"date"`
	Done      bool   `json:"done"`
	UpdatedAt string `json:"updatedAt"`
}
type ProgressEntry struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	Date      string  `json:"date"`
	WeightKG  float64 `json:"weightKg"`
	BodyFat   float64 `json:"bodyFat,omitempty"`
	Notes     string  `json:"notes,omitempty"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}
type CheckIn struct {
	ID             string   `json:"id"`
	UserID         string   `json:"userId"`
	Date           string   `json:"date"`
	LastWeekDays   int      `json:"lastWeekDays"`
	AvailableDays  []string `json:"availableDays"`
	Energy         string   `json:"energy"`
	Notes          string   `json:"notes,omitempty"`
	Recommendation string   `json:"recommendation"`
	CreatedAt      string   `json:"createdAt"`
}

type AIGrounding struct {
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	SourceID string `json:"sourceId,omitempty"`
	URL      string `json:"url,omitempty"`
}
type ChatMessage struct {
	ID         string        `json:"id"`
	UserID     string        `json:"userId"`
	Role       string        `json:"role"`
	Content    string        `json:"content"`
	Mode       string        `json:"mode"`
	Grounding  []AIGrounding `json:"grounding,omitempty"`
	Tokens     int           `json:"tokens,omitempty"`
	CostMicros int64         `json:"costMicros,omitempty"`
	At         string        `json:"at"`
}

type CoachLink struct {
	Platform     string `json:"platform"`
	URL          string `json:"url"`
	Handle       string `json:"handle,omitempty"`
	Title        string `json:"title,omitempty"`
	Provider     string `json:"provider,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	AddedAt      string `json:"addedAt"`
}
type CustomCoachProfile struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Initials      string      `json:"initials"`
	Category      string      `json:"category"`
	Status        string      `json:"status"`
	Summary       string      `json:"summary"`
	Principles    []string    `json:"principles"`
	Communication []string    `json:"communication"`
	SafetyNote    string      `json:"safetyNote"`
	Links         []CoachLink `json:"links"`
	Licensed      bool        `json:"licensed"`
	AddedBy       string      `json:"addedBy"`
	CreatedAt     string      `json:"createdAt"`
	UpdatedAt     string      `json:"updatedAt"`
}
type CoachSelection struct {
	ProfileID string `json:"profileId"`
	Weight    int    `json:"weight"`
}
type CoachPreferences struct {
	UserID           string           `json:"userId"`
	Influences       []CoachSelection `json:"influences"`
	ResponseStyle    string           `json:"responseStyle"`
	PreferredCoachID string           `json:"preferredCoachId,omitempty"`
	UpdatedAt        string           `json:"updatedAt"`
}
type CoachSource struct {
	ID            string `json:"id"`
	ProfileID     string `json:"profileId"`
	Title         string `json:"title"`
	Kind          string `json:"kind"`
	SourceURL     string `json:"sourceUrl,omitempty"`
	Summary       string `json:"summary"`
	Excerpt       string `json:"excerpt,omitempty"`
	Quote         string `json:"quote,omitempty"`
	QuoteVerified bool   `json:"quoteVerified"`
	Licensed      bool   `json:"licensed"`
	AddedBy       string `json:"addedBy"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}
type TakedownRequest struct {
	ID            string `json:"id"`
	ProfileID     string `json:"profileId"`
	RequesterName string `json:"requesterName"`
	Contact       string `json:"contact"`
	Reason        string `json:"reason"`
	EvidenceURL   string `json:"evidenceUrl,omitempty"`
	Status        string `json:"status"`
	Notes         string `json:"notes,omitempty"`
	CreatedAt     string `json:"createdAt"`
	ResolvedAt    string `json:"resolvedAt,omitempty"`
}

type Session struct {
	TokenHash  string `json:"tokenHash"`
	UserID     string `json:"userId"`
	CSRF       string `json:"csrf"`
	ExpiresAt  string `json:"expiresAt"`
	CreatedAt  string `json:"createdAt"`
	LastSeenAt string `json:"lastSeenAt,omitempty"`
	IP         string `json:"ip"`
	UserAgent  string `json:"userAgent"`
	DeviceName string `json:"deviceName,omitempty"`
}
type LoginAttempt struct {
	Key         string `json:"key"`
	Count       int    `json:"count"`
	WindowStart string `json:"windowStart"`
	LockedUntil string `json:"lockedUntil,omitempty"`
}
type AuditEvent struct {
	ID        string         `json:"id"`
	At        string         `json:"at"`
	ActorID   string         `json:"actorId,omitempty"`
	ActorName string         `json:"actorName,omitempty"`
	Action    string         `json:"action"`
	Target    string         `json:"target,omitempty"`
	IP        string         `json:"ip,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

type AIUsage struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	Date         string `json:"date"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	CostMicros   int64  `json:"costMicros"`
	At           string `json:"at"`
}
type MigrationRecord struct {
	Version   int    `json:"version"`
	Name      string `json:"name"`
	AppliedAt string `json:"appliedAt"`
}
type PainFlag struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	BodyArea  string `json:"bodyArea"`
	Severity  int    `json:"severity"`
	Trigger   string `json:"trigger,omitempty"`
	Notes     string `json:"notes,omitempty"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}
type ProgressPhoto struct {
	ID            string `json:"id"`
	UserID        string `json:"userId"`
	Date          string `json:"date"`
	Caption       string `json:"caption,omitempty"`
	EncryptedPath string `json:"encryptedPath"`
	MimeType      string `json:"mimeType"`
	Size          int64  `json:"size"`
	CreatedAt     string `json:"createdAt"`
}
type MealPlanItem struct {
	Meal     string `json:"meal"`
	Name     string `json:"name"`
	Serving  string `json:"serving"`
	Calories int    `json:"calories"`
	Protein  int    `json:"protein"`
	Carbs    int    `json:"carbs"`
	Fat      int    `json:"fat"`
}
type MealPlanDay struct {
	Date     string         `json:"date"`
	Items    []MealPlanItem `json:"items"`
	Calories int            `json:"calories"`
	Protein  int            `json:"protein"`
	Carbs    int            `json:"carbs"`
	Fat      int            `json:"fat"`
}
type MealPlan struct {
	ID          string        `json:"id"`
	UserID      string        `json:"userId"`
	StartDate   string        `json:"startDate"`
	Days        []MealPlanDay `json:"days"`
	Preferences string        `json:"preferences,omitempty"`
	CreatedAt   string        `json:"createdAt"`
}
type WearableConnection struct {
	ID                  string `json:"id"`
	UserID              string `json:"userId"`
	Provider            string `json:"provider"`
	Mode                string `json:"mode"`
	Status              string `json:"status"`
	LastSyncAt          string `json:"lastSyncAt,omitempty"`
	EncryptedCredential string `json:"encryptedCredential,omitempty"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}
type HealthMetric struct {
	ID         string  `json:"id"`
	UserID     string  `json:"userId"`
	Provider   string  `json:"provider"`
	MetricType string  `json:"metricType"`
	StartAt    string  `json:"startAt"`
	EndAt      string  `json:"endAt,omitempty"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	Source     string  `json:"source,omitempty"`
	ImportedAt string  `json:"importedAt"`
}
type Nudge struct {
	ID               string `json:"id"`
	FromUserID       string `json:"fromUserId"`
	ToUserID         string `json:"toUserId"`
	Message          string `json:"message"`
	CreatedAt        string `json:"createdAt"`
	ReadAt           string `json:"readAt,omitempty"`
	ModerationStatus string `json:"moderationStatus,omitempty"`
	RemovedAt        string `json:"removedAt,omitempty"`
	RemovedBy        string `json:"removedBy,omitempty"`
	RemovalReason    string `json:"removalReason,omitempty"`
}
type SharedWorkout struct {
	ID               string   `json:"id"`
	WorkoutID        string   `json:"workoutId,omitempty"`
	WorkoutName      string   `json:"workoutName"`
	Date             string   `json:"date"`
	StartTime        string   `json:"startTime"`
	Duration         int      `json:"duration"`
	ParticipantIDs   []string `json:"participantIds"`
	CreatedBy        string   `json:"createdBy"`
	Status           string   `json:"status"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	ModerationStatus string   `json:"moderationStatus,omitempty"`
	RemovedAt        string   `json:"removedAt,omitempty"`
	RemovedBy        string   `json:"removedBy,omitempty"`
	RemovalReason    string   `json:"removalReason,omitempty"`
}

type UserBlock struct {
	ID        string `json:"id"`
	BlockerID string `json:"blockerId"`
	BlockedID string `json:"blockedId"`
	CreatedAt string `json:"createdAt"`
}

type ContentReport struct {
	ID           string `json:"id"`
	ReporterID   string `json:"reporterId"`
	TargetType   string `json:"targetType"`
	TargetID     string `json:"targetId"`
	TargetUserID string `json:"targetUserId,omitempty"`
	Category     string `json:"category"`
	Details      string `json:"details,omitempty"`
	Status       string `json:"status"`
	Resolution   string `json:"resolution,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	ReviewedBy   string `json:"reviewedBy,omitempty"`
	ReviewedAt   string `json:"reviewedAt,omitempty"`
}

type ModerationAction struct {
	ID           string `json:"id"`
	ModeratorID  string `json:"moderatorId"`
	Action       string `json:"action"`
	TargetType   string `json:"targetType"`
	TargetID     string `json:"targetId"`
	TargetUserID string `json:"targetUserId,omitempty"`
	Reason       string `json:"reason,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type BackgroundJob struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	RunAt     string `json:"runAt"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"lastError,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Settings struct {
	LANEnabled             bool   `json:"lanEnabled"`
	Port                   int    `json:"port"`
	BackupCopyPath         string `json:"backupCopyPath,omitempty"`
	LastAutoBackupAt       string `json:"lastAutoBackupAt,omitempty"`
	RecoveryHash           string `json:"recoveryHash,omitempty"`
	AIMode                 string `json:"aiMode,omitempty"`
	AIBaseURL              string `json:"aiBaseUrl,omitempty"`
	AIModel                string `json:"aiModel,omitempty"`
	AIAPIKeyEncrypted      string `json:"aiApiKeyEncrypted,omitempty"`
	AIDailyTokenCap        int    `json:"aiDailyTokenCap,omitempty"`
	AIDailyCostCapMicros   int64  `json:"aiDailyCostCapMicros,omitempty"`
	FreeDailyTokenCap      int    `json:"freeDailyTokenCap,omitempty"`
	FreeDailyCostCapMicros int64  `json:"freeDailyCostCapMicros,omitempty"`
	UpdateManifestURL      string `json:"updateManifestUrl,omitempty"`
	UpdatePublicKey        string `json:"updatePublicKey,omitempty"`
	CrashReportingEnabled  bool   `json:"crashReportingEnabled,omitempty"`
	CrashEndpoint          string `json:"crashEndpoint,omitempty"`
	BackupIntervalHours    int    `json:"backupIntervalHours,omitempty"`
	TakedownContact        string `json:"takedownContact,omitempty"`
	TermsVersion           string `json:"termsVersion,omitempty"`
	PrivacyVersion         string `json:"privacyVersion,omitempty"`
	CommunityVersion       string `json:"communityVersion,omitempty"`
	SubscriptionVersion    string `json:"subscriptionVersion,omitempty"`
	AgentEnabled           bool   `json:"agentEnabled,omitempty"`
	AgentBaseURL           string `json:"agentBaseUrl,omitempty"`
	AgentModel             string `json:"agentModel,omitempty"`
	AgentSearchURL         string `json:"agentSearchUrl,omitempty"`
	AgentMaxSteps          int    `json:"agentMaxSteps,omitempty"`
	AgentAllowWeb          bool   `json:"agentAllowWeb,omitempty"`
}

type ThemePreferences struct {
	UserID            string `json:"userId"`
	Preset            string `json:"preset"`
	Accent            string `json:"accent"`
	Background        string `json:"background"`
	Surface           string `json:"surface"`
	Text              string `json:"text"`
	Radius            int    `json:"radius"`
	Density           string `json:"density"`
	MeasurementSystem string `json:"measurementSystem"`
	NavigationMode    string `json:"navigationMode"`
	UpdatedAt         string `json:"updatedAt"`
}
type AgentTask struct {
	ID        string      `json:"id"`
	UserID    string      `json:"userId"`
	Goal      string      `json:"goal"`
	Status    string      `json:"status"`
	Schedule  string      `json:"schedule,omitempty"`
	MaxSteps  int         `json:"maxSteps"`
	Steps     []AgentStep `json:"steps,omitempty"`
	Result    string      `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
}
type AgentStep struct {
	At        string `json:"at"`
	Tool      string `json:"tool"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	SourceURL string `json:"sourceUrl,omitempty"`
}
type AgentMemory struct {
	ID         string  `json:"id"`
	UserID     string  `json:"userId"`
	Kind       string  `json:"kind"`
	Fact       string  `json:"fact"`
	Confidence float64 `json:"confidence"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}
type MarketplaceItem struct {
	ID          string          `json:"id"`
	OwnerID     string          `json:"ownerId"`
	Kind        string          `json:"kind"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Payload     json.RawMessage `json:"payload"`
	Official    bool            `json:"official"`
	PriceCents  int             `json:"priceCents"`
	Published   bool            `json:"published"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}
type VisionAnalysis struct {
	ID        string   `json:"id"`
	UserID    string   `json:"userId"`
	Kind      string   `json:"kind"`
	MediaPath string   `json:"mediaPath"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary,omitempty"`
	Findings  []string `json:"findings,omitempty"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}
type GroceryList struct {
	ID             string   `json:"id"`
	UserID         string   `json:"userId"`
	MealPlanID     string   `json:"mealPlanId,omitempty"`
	Store          string   `json:"store"`
	Items          []string `json:"items"`
	EstimatedCents int      `json:"estimatedCents,omitempty"`
	CreatedAt      string   `json:"createdAt"`
}

type Database struct {
	SchemaVersion       int                           `json:"schemaVersion"`
	Users               map[string]User               `json:"users"`
	Profiles            map[string]Profile            `json:"profiles"`
	Workouts            map[string]Workout            `json:"workouts"`
	WeeklyPlans         map[string]map[string]string  `json:"weeklyPlans"`
	WorkoutLogs         []WorkoutLog                  `json:"workoutLogs"`
	Nutrition           []NutritionEntry              `json:"nutrition"`
	Habits              []Habit                       `json:"habits"`
	HabitLogs           []HabitLog                    `json:"habitLogs"`
	Progress            []ProgressEntry               `json:"progress"`
	CheckIns            []CheckIn                     `json:"checkIns"`
	ChatMessages        []ChatMessage                 `json:"chatMessages"`
	CoachPreferences    map[string]CoachPreferences   `json:"coachPreferences"`
	CoachSources        []CoachSource                 `json:"coachSources"`
	CustomCoachProfiles map[string]CustomCoachProfile `json:"customCoachProfiles"`
	TakedownRequests    []TakedownRequest             `json:"takedownRequests"`
	AIUsage             []AIUsage                     `json:"aiUsage"`
	Migrations          []MigrationRecord             `json:"migrations"`
	PainFlags           []PainFlag                    `json:"painFlags"`
	ProgressPhotos      []ProgressPhoto               `json:"progressPhotos"`
	MealPlans           []MealPlan                    `json:"mealPlans"`
	WearableConnections []WearableConnection          `json:"wearableConnections"`
	HealthMetrics       []HealthMetric                `json:"healthMetrics"`
	Nudges              []Nudge                       `json:"nudges"`
	SharedWorkouts      []SharedWorkout               `json:"sharedWorkouts"`
	UserBlocks          []UserBlock                   `json:"userBlocks"`
	ContentReports      []ContentReport               `json:"contentReports"`
	ModerationActions   []ModerationAction            `json:"moderationActions"`
	Jobs                []BackgroundJob               `json:"jobs"`
	Sessions            map[string]Session            `json:"sessions"`
	LoginAttempts       map[string]LoginAttempt       `json:"loginAttempts"`
	Audit               []AuditEvent                  `json:"audit"`
	Settings            Settings                      `json:"settings"`
	ThemePreferences    map[string]ThemePreferences   `json:"themePreferences"`
	AgentTasks          []AgentTask                   `json:"agentTasks"`
	AgentMemories       []AgentMemory                 `json:"agentMemories"`
	MarketplaceItems    []MarketplaceItem             `json:"marketplaceItems"`
	VisionAnalyses      []VisionAnalysis              `json:"visionAnalyses"`
	GroceryLists        []GroceryList                 `json:"groceryLists"`
	UpdatedAt           string                        `json:"updatedAt"`
}

func NewDatabase() Database {
	now := time.Now().UTC().Format(time.RFC3339)
	return Database{SchemaVersion: SchemaVersion, Users: map[string]User{}, Profiles: map[string]Profile{}, Workouts: map[string]Workout{}, WeeklyPlans: map[string]map[string]string{}, WorkoutLogs: []WorkoutLog{}, Nutrition: []NutritionEntry{}, Habits: []Habit{}, HabitLogs: []HabitLog{}, Progress: []ProgressEntry{}, CheckIns: []CheckIn{}, ChatMessages: []ChatMessage{}, CoachPreferences: map[string]CoachPreferences{}, CoachSources: []CoachSource{}, CustomCoachProfiles: map[string]CustomCoachProfile{}, TakedownRequests: []TakedownRequest{}, AIUsage: []AIUsage{}, Migrations: []MigrationRecord{{Version: SchemaVersion, Name: "initial-current-schema", AppliedAt: now}}, PainFlags: []PainFlag{}, ProgressPhotos: []ProgressPhoto{}, MealPlans: []MealPlan{}, WearableConnections: []WearableConnection{}, HealthMetrics: []HealthMetric{}, Nudges: []Nudge{}, SharedWorkouts: []SharedWorkout{}, UserBlocks: []UserBlock{}, ContentReports: []ContentReport{}, ModerationActions: []ModerationAction{}, Jobs: []BackgroundJob{}, Sessions: map[string]Session{}, LoginAttempts: map[string]LoginAttempt{}, Audit: []AuditEvent{}, Settings: Settings{Port: 8443, AIMode: "auto", AIBaseURL: "https://api.openai.com/v1", AIModel: "gpt-4o-mini", AIDailyTokenCap: 50000, AIDailyCostCapMicros: 2500000, BackupIntervalHours: 24, TakedownContact: "admin@localhost", TermsVersion: "2.0", PrivacyVersion: "1.0", CommunityVersion: "1.0", SubscriptionVersion: "1.0"}, ThemePreferences: map[string]ThemePreferences{}, AgentTasks: []AgentTask{}, AgentMemories: []AgentMemory{}, MarketplaceItems: []MarketplaceItem{}, VisionAnalyses: []VisionAnalysis{}, GroceryLists: []GroceryList{}, UpdatedAt: now}
}
