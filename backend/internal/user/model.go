package user

type OnboardingRequest struct {
	Name              string  `json:"name"`
	Age               int     `json:"age"`
	Sex               string  `json:"sex"`
	HeightCm          float64 `json:"height_cm"`
	WeightKg          float64 `json:"weight_kg"`
	TargetWeightKg    float64 `json:"target_weight_kg"`
	ActivityLevel     string  `json:"activity_level"`
	Objective         string  `json:"objective"`
	DietPlan          string  `json:"diet_plan"`
	FastingWindow     *string `json:"fasting_window"`
	EatingWindowStart *string `json:"eating_window_start"` // "HH:MM", only for intermittent_fasting
}

type OnboardingResponse struct {
	BMR                float64 `json:"bmr"`
	TDEE               float64 `json:"tdee"`
	CalorieTarget      int32   `json:"calorie_target"`
	ProteinG           int32   `json:"protein_g"`
	CarbsG             int32   `json:"carbs_g"`
	FatG               int32   `json:"fat_g"`
	OnboardingComplete bool    `json:"onboarding_complete"`
}

type ProfileResponse struct {
	Name               string   `json:"name"`
	Age                *int32   `json:"age"`
	Sex                *string  `json:"sex"`
	HeightCm           *float64 `json:"height_cm"`
	WeightKg           *float64 `json:"weight_kg"`
	TargetWeightKg     *float64 `json:"target_weight_kg"`
	ActivityLevel      *string  `json:"activity_level"`
	OnboardingComplete bool     `json:"onboarding_complete"`
	IsAdmin            bool     `json:"is_admin"`
	AvatarURL          *string  `json:"avatar_url"`
}

type UpdateProfileRequest struct {
	Name           string  `json:"name"`
	Age            int     `json:"age"`
	Sex            string  `json:"sex"`
	HeightCm       float64 `json:"height_cm"`
	WeightKg       float64 `json:"weight_kg"`
	TargetWeightKg float64 `json:"target_weight_kg"`
	ActivityLevel  string  `json:"activity_level"`
}

type UpdateGoalsRequest struct {
	Objective          string  `json:"objective"`
	DietPlan           string  `json:"diet_plan"`
	FastingWindow      *string `json:"fasting_window"`
	ManualOverride     bool    `json:"manual_override"`
	DailyCalorieTarget int32   `json:"daily_calorie_target"`
	DailyProteinG      int32   `json:"daily_protein_g"`
	DailyCarbsG        int32   `json:"daily_carbs_g"`
	DailyFatG          int32   `json:"daily_fat_g"`
}

type GoalsResponse struct {
	Objective          string  `json:"objective"`
	DietPlan           string  `json:"diet_plan"`
	FastingWindow      *string `json:"fasting_window"`
	DailyCalorieTarget int32   `json:"daily_calorie_target"`
	DailyProteinG      int32   `json:"daily_protein_g"`
	DailyCarbsG        int32   `json:"daily_carbs_g"`
	DailyFatG          int32   `json:"daily_fat_g"`
	EatingWindowStart  *string `json:"eating_window_start"`
	FastingStreak      *int32  `json:"fasting_streak"`
}
