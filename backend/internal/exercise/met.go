package exercise

import (
	"math"
	"strings"
)

var metDatabase = map[string]float64{
	"walking": 3.5, "walking (slow)": 2.5, "walking (moderate)": 3.5, "walking (brisk)": 4.3,
	"walking (very brisk)": 5.0, "hiking": 6.0, "nordic walking": 5.9,
	"running": 8.0, "running (slow)": 6.0, "running (moderate)": 8.3, "running (fast)": 11.0,
	"jogging": 7.0, "sprint": 12.0, "trail running": 8.5,
	"cycling": 6.8, "cycling (leisure)": 4.0, "cycling (moderate)": 6.8, "cycling (vigorous)": 10.0,
	"cycling (stationary)": 6.8, "spinning": 8.5,
	"swimming": 6.0, "swimming (moderate)": 6.0, "swimming (vigorous)": 9.8,
	"swimming (laps)": 7.0, "water aerobics": 5.5,
	"weight training": 6.0, "strength training": 6.0, "weight lifting": 6.0,
	"resistance training": 5.0, "bodyweight exercises": 3.8, "crossfit": 8.0,
	"yoga": 3.0, "yoga (hatha)": 2.5, "yoga (vinyasa)": 4.0, "yoga (power)": 4.5,
	"pilates": 3.0, "stretching": 2.3,
	"hiit": 8.0, "jump rope": 10.0, "burpees": 8.0, "cardio": 6.0,
	"elliptical": 5.0, "stair climbing": 9.0, "rowing": 7.0,
	"basketball": 6.5, "soccer": 7.0, "tennis": 7.3, "badminton": 5.5,
	"volleyball": 4.0, "table tennis": 4.0, "racquetball": 7.0,
	"boxing": 7.8, "martial arts": 5.0, "judo": 6.0, "wrestling": 6.0,
	"dancing": 5.0, "zumba": 7.0, "dance (aerobic)": 6.5, "dance (ballet)": 5.0,
	"gardening": 3.8, "housework": 3.3, "playing with kids": 4.0,
	"skating": 5.5, "skating (ice)": 7.0, "skiing": 7.0, "snowboarding": 5.3,
	"golf (walking)": 4.3, "golf (cart)": 2.5, "rock climbing": 8.0,
}

func FindMET(name string) float64 {
	if met, ok := metDatabase[name]; ok {
		return met
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	if met, ok := metDatabase[lower]; ok {
		return met
	}
	return 5.0
}

func CalculateCalories(met float64, weightKg float64, durationMin int32) int32 {
	hours := float64(durationMin) / 60.0
	cal := met * weightKg * hours
	return int32(math.Round(cal))
}
