package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/auth"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewHandler(q *sqlc.Queries, pool *pgxpool.Pool) *Handler {
	return &Handler{q: q, pool: pool}
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
	TotalSteps    int32      `json:"total_steps"`
	Meals         []MealItem `json:"meals"`
	IsCheatDay    bool       `json:"is_cheat_day"`
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
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
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
	ctx := r.Context()
	tz := r.Header.Get("X-Timezone")
	if tz == "" {
		tz = "UTC"
	}

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

	// Run summary and meals+foods queries in parallel — eliminates N+1 and cuts wall time in half.
	type summaryResult struct {
		row sqlc.GetDailySummaryRow
		err error
	}
	type mealsResult struct {
		items []MealItem
		err   error
	}

	sumCh := make(chan summaryResult, 1)
	mealsCh := make(chan mealsResult, 1)

	go func() {
		s, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{
			UserID:    userID,
			Timestamp: date,
			Column3:   tz,
		})
		sumCh <- summaryResult{s, err}
	}()

	go func() {
		items, err := h.fetchMealsWithFoods(ctx, userID, date, tz)
		mealsCh <- mealsResult{items, err}
	}()

	sr := <-sumCh
	mr := <-mealsCh

	if sr.err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get daily summary: %w", sr.err))
		return
	}
	if mr.err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get meals: %w", mr.err))
		return
	}

	isCheatDay, _ := h.q.IsCheatDay(ctx, sqlc.IsCheatDayParams{
		UserID: userID,
		Date:   date,
	})

	var totalSteps int32
	_ = h.pool.QueryRow(ctx,
		"SELECT step_count FROM step_logs WHERE user_id = $1 AND date = $2",
		userID, date,
	).Scan(&totalSteps)

	mealItems := mr.items
	if mealItems == nil {
		mealItems = []MealItem{}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: SummaryResponse{
		Date:          date.Format("2006-01-02"),
		TotalCalories: sr.row.TotalCalories,
		TotalProtein:  sr.row.TotalProtein,
		TotalCarbs:    sr.row.TotalCarbs,
		TotalFat:      sr.row.TotalFat,
		TotalFiber:    sr.row.TotalFiber,
		TotalBurned:   sr.row.TotalBurned,
		TotalWaterMl:  sr.row.TotalWaterMl,
		TotalSteps:    totalSteps,
		Meals:         mealItems,
		IsCheatDay:    isCheatDay,
	}})
}

// fetchMealsWithFoods fetches all meals and their food items for a given day
// in a single JOIN query instead of N+1 sequential queries.
func (h *Handler) fetchMealsWithFoods(ctx context.Context, userID string, date time.Time, tz string) ([]MealItem, error) {
	const q = `
		SELECT
			m.id, m.timestamp, m.meal_type, m.photo_path, m.note,
			fi.id, fi.name, fi.calories, fi.protein_g, fi.carbs_g, fi.fat_g, fi.fiber_g, fi.serving_size, fi.source
		FROM meals m
		LEFT JOIN food_items fi ON fi.meal_id = m.id
		WHERE m.user_id = $1 AND (m.timestamp AT TIME ZONE COALESCE($3::text, 'UTC'))::date = $2
		ORDER BY m.timestamp ASC, fi.id ASC
	`

	rows, err := h.pool.Query(ctx, q, userID, date, tz)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var order []string
	mealMap := map[string]*MealItem{}

	for rows.Next() {
		var (
			mID      string
			mTime    time.Time
			mType    string
			mPhoto   *string
			mNote    *string
			fID      *string
			fName    *string
			fCal     *int32
			fProtG   pgtype.Numeric
			fCarbsG  pgtype.Numeric
			fFatG    pgtype.Numeric
			fFiberG  pgtype.Numeric
			fServing *string
			fSource  *string
		)

		if err := rows.Scan(
			&mID, &mTime, &mType, &mPhoto, &mNote,
			&fID, &fName, &fCal, &fProtG, &fCarbsG, &fFatG, &fFiberG, &fServing, &fSource,
		); err != nil {
			return nil, err
		}

		if _, ok := mealMap[mID]; !ok {
			mealMap[mID] = &MealItem{
				ID:        mID,
				Timestamp: mTime.Format(time.RFC3339),
				MealType:  mType,
				PhotoPath: mPhoto,
				Note:      mNote,
				Foods:     []FoodItemResp{},
			}
			order = append(order, mID)
		}

		if fID != nil {
			cal := int32(0)
			if fCal != nil {
				cal = *fCal
			}
			mealMap[mID].Foods = append(mealMap[mID].Foods, FoodItemResp{
				ID:          *fID,
				Name:        *fName,
				Calories:    cal,
				ProteinG:    numericToFloat(fProtG),
				CarbsG:      numericToFloat(fCarbsG),
				FatG:        numericToFloat(fFatG),
				FiberG:      numericToFloat(fFiberG),
				ServingSize: fServing,
				Source:      *fSource,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]MealItem, 0, len(order))
	for _, id := range order {
		result = append(result, *mealMap[id])
	}
	return result, nil
}

func (h *Handler) MarkCheatDay(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Date string `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date format: %w", err))
		return
	}
	if err := h.q.MarkCheatDay(r.Context(), sqlc.MarkCheatDayParams{UserID: userID, Date: date}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("mark cheat day: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"is_cheat_day": true}})
}

func (h *Handler) UnmarkCheatDay(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid date format: %w", err))
		return
	}
	if err := h.q.UnmarkCheatDay(r.Context(), sqlc.UnmarkCheatDayParams{UserID: userID, Date: date}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("unmark cheat day: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]bool{"is_cheat_day": false}})
}
