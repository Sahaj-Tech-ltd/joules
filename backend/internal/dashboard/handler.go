package dashboard

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"joule/internal/db/sqlc"
)

type contextKey string

type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
}

type apiResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type SummaryResponse struct {
	Date          string     `json:"date"`
	TotalCalories int32      `json:"total_calories"`
	TotalProtein  float64    `json:"total_protein"`
	TotalCarbs    float64    `json:"total_carbs"`
	TotalFat      float64    `json:"total_fat"`
	TotalFiber    float64    `json:"total_fiber"`
	TotalBurned   int32      `json:"total_burned"`
	TotalWaterMl  int32      `json:"total_water_ml"`
	Meals         []MealItem `json:"meals"`
}

type MealItem struct {
	ID        string         `json:"id"`
	Timestamp string         `json:"timestamp"`
	MealType  string         `json:"meal_type"`
	PhotoPath *string        `json:"photo_path"`
	Note      *string        `json:"note"`
	Foods     []FoodItemResp `json:"foods"`
}

type FoodItemResp struct {
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	slog.Error("request error", "status", status, "error", err)
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(contextKey("userID")).(string)
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	dateStr := r.URL.Query().Get("date")
	var date time.Time
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date format: %w", err))
			return
		}
	} else {
		date = time.Now()
	}

	summary, err := h.q.GetDailySummary(r.Context(), sqlc.GetDailySummaryParams{
		UserID:    userID,
		Timestamp: date,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get daily summary: %w", err))
		return
	}

	meals, err := h.q.GetMealsByDate(r.Context(), sqlc.GetMealsByDateParams{
		UserID:    userID,
		Timestamp: date,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get meals: %w", err))
		return
	}

	var mealItems []MealItem
	for _, m := range meals {
		foods, err := h.q.GetFoodItemsByMeal(r.Context(), m.ID)
		if err != nil {
			slog.Error("get food items for meal", "meal_id", m.ID, "error", err)
			continue
		}

		var foodItems []FoodItemResp
		for _, f := range foods {
			foodItems = append(foodItems, FoodItemResp{
				ID:          f.ID,
				Name:        f.Name,
				Calories:    f.Calories,
				ProteinG:    numericToFloat(f.ProteinG),
				CarbsG:      numericToFloat(f.CarbsG),
				FatG:        numericToFloat(f.FatG),
				FiberG:      numericToFloat(f.FiberG),
				ServingSize: f.ServingSize,
				Source:      f.Source,
			})
		}

		mealItems = append(mealItems, MealItem{
			ID:        m.ID,
			Timestamp: m.Timestamp.Format(time.RFC3339),
			MealType:  m.MealType,
			PhotoPath: m.PhotoPath,
			Note:      m.Note,
			Foods:     foodItems,
		})
	}

	if mealItems == nil {
		mealItems = []MealItem{}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: SummaryResponse{
		Date:          date.Format("2006-01-02"),
		TotalCalories: summary.TotalCalories,
		TotalProtein:  summary.TotalProtein,
		TotalCarbs:    summary.TotalCarbs,
		TotalFat:      summary.TotalFat,
		TotalFiber:    summary.TotalFiber,
		TotalBurned:   summary.TotalBurned,
		TotalWaterMl:  summary.TotalWaterMl,
		Meals:         mealItems,
	}})
}
