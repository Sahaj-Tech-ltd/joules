package meal

type MealResponse struct {
	ID        string             `json:"id"`
	Timestamp string             `json:"timestamp"`
	MealType  string             `json:"meal_type"`
	PhotoPath *string            `json:"photo_path"`
	Note      *string            `json:"note"`
	Foods     []FoodItemResponse `json:"foods"`
}

type FoodItemResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Calories    int32   `json:"calories"`
	ProteinG    float64 `json:"protein_g"`
	CarbsG      float64 `json:"carbs_g"`
	FatG        float64 `json:"fat_g"`
	FiberG      float64 `json:"fiber_g"`
	ServingSize *string `json:"serving_size"`
	Source      string  `json:"source"`
}

type CreateMealRequest struct {
	MealType    string       `json:"meal_type"`
	Photo       string       `json:"photo"`
	Note        string       `json:"note"`
	PortionHint string       `json:"portion_hint"`
	Foods       []ManualFood `json:"foods"`
}

type ManualFood struct {
	Name        string  `json:"name"`
	Calories    float64 `json:"calories"`
	ProteinG    float64 `json:"protein_g"`
	CarbsG      float64 `json:"carbs_g"`
	FatG        float64 `json:"fat_g"`
	FiberG      float64 `json:"fiber_g"`
	ServingSize string  `json:"serving_size"`
}

type DailyLogResponse struct {
	Meals         []MealResponse `json:"meals"`
	TotalCalories int32          `json:"total_calories"`
	TotalProtein  float64        `json:"total_protein"`
	TotalCarbs    float64        `json:"total_carbs"`
	TotalFat      float64        `json:"total_fat"`
	TotalFiber    float64        `json:"total_fiber"`
}

// LogMealFromRecipeRequest is the body for POST /api/meals/from-recipe/{recipeId}.
type LogMealFromRecipeRequest struct {
	MealType string `json:"meal_type"`
}
