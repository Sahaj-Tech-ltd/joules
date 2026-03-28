package user

import "math"

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
	"calorie_deficit":        {40, 30, 30},
	"keto":                   {5, 25, 70},
	"intermittent_fasting":   {40, 30, 30},
	"paleo":                  {25, 35, 40},
	"mediterranean":          {45, 25, 30},
	"balanced":               {50, 25, 25},
}

func CalculateTDEE(sex string, age int, weightKg, heightCm float64, activityLevel, objective, dietPlan string) (bmr, tdee, calorieTarget float64, proteinG, carbsG, fatG int) {
	bmr = 10*weightKg + 6.25*heightCm - 5*float64(age)
	if sex == "male" {
		bmr += 5
	} else {
		bmr -= 161
	}
	bmr = math.Max(bmr, 0)

	actMul := activityMultipliers[activityLevel]
	if actMul == 0 {
		actMul = 1.2
	}
	tdee = bmr * actMul

	objMul := objectiveMultipliers[objective]
	if objMul == 0 {
		objMul = 1.0
	}
	calorieTarget = tdee * objMul
	calorieTarget = math.Max(calorieTarget, 1200)

	split, ok := macroSplits[dietPlan]
	if !ok {
		split = macroSplits["calorie_deficit"]
	}

	proteinG = int(math.Round((calorieTarget * split[1] / 100) / 4))
	carbsG = int(math.Round((calorieTarget * split[0] / 100) / 4))
	fatG = int(math.Round((calorieTarget * split[2] / 100) / 9))

	return
}
