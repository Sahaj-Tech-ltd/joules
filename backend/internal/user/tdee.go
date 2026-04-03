package user

import (
	"math"

	"joules/internal/admin"
)

var activityMultipliers = map[string]float64{
	"sedentary":   1.2,
	"light":       1.375,
	"moderate":    1.55,
	"active":      1.725,
	"very_active": 1.9,
}

var objectiveMultipliers = map[string]float64{
	"cut_fat":      0.80,
	"feel_better":  0.90,
	"maintain":     1.0,
	"build_muscle": 1.10,
}

var macroSplits = map[string][3]float64{
	"calorie_deficit":      {40, 30, 30},
	"keto":                 {5, 25, 70},
	"intermittent_fasting": {40, 30, 30},
	"paleo":                {25, 35, 40},
	"mediterranean":        {45, 25, 30},
	"balanced":             {50, 25, 25},
}

func CalculateTDEE(sex string, age int, weightKg, heightCm float64, activityLevel, objective, dietPlan string, cfg admin.TDEEConfig) (bmr, tdee, calorieTarget float64, proteinG, carbsG, fatG int) {
	bmr = 10*weightKg + 6.25*heightCm - 5*float64(age)
	if sex == "male" {
		bmr += 5
	} else {
		bmr -= 161
	}
	bmr = math.Max(bmr, 0)

	actMul := cfg.ActivityMultipliers[activityLevel]
	if actMul == 0 {
		actMul = 1.2
	}
	tdee = bmr * actMul

	objMul := cfg.ObjectiveMultipliers[objective]
	if objMul == 0 {
		objMul = 1.0
	}
	calorieTarget = tdee * objMul

	minCal := float64(cfg.MinCalorieTarget)
	if minCal <= 0 {
		minCal = 1200
	}
	calorieTarget = math.Max(calorieTarget, minCal)

	split, ok := cfg.MacroSplits[dietPlan]
	if !ok {
		split = map[string]float64{"carbs": 40, "protein": 30, "fat": 30}
	}

	carbsPct := split["carbs"]
	proteinPct := split["protein"]
	fatPct := split["fat"]

	proteinG = int(math.Round((calorieTarget * proteinPct / 100) / 4))
	carbsG = int(math.Round((calorieTarget * carbsPct / 100) / 4))
	fatG = int(math.Round((calorieTarget * fatPct / 100) / 9))

	return
}
