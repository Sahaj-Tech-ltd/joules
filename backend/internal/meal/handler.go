package meal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/ai"
	"joules/internal/auth"
	"joules/internal/config"
	"joules/internal/db/sqlc"
	syslog "joules/internal/syslog"
)

type Handler struct {
	q         *sqlc.Queries
	ai        ai.Client
	uploadDir string
	cfg       *config.Config
	pool      *pgxpool.Pool
}

func NewHandler(q *sqlc.Queries, aiClient ai.Client, uploadDir string, cfg *config.Config, pool *pgxpool.Pool) *Handler {
	return &Handler{q: q, ai: aiClient, uploadDir: uploadDir, cfg: cfg, pool: pool}
}

type apiResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
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

func getUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value(auth.ContextUserID).(string)
	if !ok {
		return "", fmt.Errorf("unauthorized")
	}
	return userID, nil
}

func floatToNumeric(f float64) pgtype.Numeric {
	n := pgtype.Numeric{}
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

func stringPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

var validMealTypes = map[string]bool{
	"breakfast": true,
	"lunch":     true,
	"dinner":    true,
	"snack":     true,
}

func mealToResponse(m sqlc.Meal, foods []sqlc.FoodItem) MealResponse {
	resp := MealResponse{
		ID:        m.ID,
		Timestamp: m.Timestamp.Format(time.RFC3339),
		MealType:  m.MealType,
		PhotoPath: m.PhotoPath,
		Note:      m.Note,
	}
	for _, f := range foods {
		resp.Foods = append(resp.Foods, FoodItemResponse{
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
	return resp
}

// fetchMealsWithFoodsByDate fetches all meals and their food items for a given day
// in a single JOIN query instead of N+1 sequential queries.
func (h *Handler) fetchMealsWithFoodsByDate(ctx context.Context, userID string, date time.Time, tz string) ([]MealResponse, error) {
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

	return scanMealsWithFoods(rows)
}

// fetchRecentMealsWithFoods fetches the 20 most recent meals and their food items
// in a single JOIN query instead of N+1 sequential queries.
func (h *Handler) fetchRecentMealsWithFoods(ctx context.Context, userID string) ([]MealResponse, error) {
	const q = `
		SELECT
			m.id, m.timestamp, m.meal_type, m.photo_path, m.note,
			fi.id, fi.name, fi.calories, fi.protein_g, fi.carbs_g, fi.fat_g, fi.fiber_g, fi.serving_size, fi.source
		FROM (
			SELECT * FROM meals WHERE user_id = $1 ORDER BY created_at DESC LIMIT 20
		) m
		LEFT JOIN food_items fi ON fi.meal_id = m.id
		ORDER BY m.created_at DESC, fi.id ASC
	`

	rows, err := h.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMealsWithFoods(rows)
}

// scanMealsWithFoods scans JOINed meal+food rows into a []MealResponse,
// collapsing duplicate meal rows into a single MealResponse with multiple Foods.
func scanMealsWithFoods(rows pgx.Rows) ([]MealResponse, error) {
	var order []string
	mealMap := map[string]*MealResponse{}

	for rows.Next() {
		var (
			mID    string
			mTime  time.Time
			mType  string
			mPhoto *string
			mNote  *string
			fID    *string
			fName  *string
			fCal   *int32
			fProtG pgtype.Numeric
			fCarbG pgtype.Numeric
			fFatG  pgtype.Numeric
			fFibG  pgtype.Numeric
			fServ  *string
			fSrc   *string
		)

		if err := rows.Scan(
			&mID, &mTime, &mType, &mPhoto, &mNote,
			&fID, &fName, &fCal, &fProtG, &fCarbG, &fFatG, &fFibG, &fServ, &fSrc,
		); err != nil {
			return nil, err
		}

		if _, ok := mealMap[mID]; !ok {
			mealMap[mID] = &MealResponse{
				ID:        mID,
				Timestamp: mTime.Format(time.RFC3339),
				MealType:  mType,
				PhotoPath: mPhoto,
				Note:      mNote,
				Foods:     []FoodItemResponse{},
			}
			order = append(order, mID)
		}

		if fID != nil {
			cal := int32(0)
			if fCal != nil {
				cal = *fCal
			}
			src := ""
			if fSrc != nil {
				src = *fSrc
			}
			mealMap[mID].Foods = append(mealMap[mID].Foods, FoodItemResponse{
				ID:          *fID,
				Name:        *fName,
				Calories:    cal,
				ProteinG:    numericToFloat(fProtG),
				CarbsG:      numericToFloat(fCarbG),
				FatG:        numericToFloat(fFatG),
				FiberG:      numericToFloat(fFibG),
				ServingSize: fServ,
				Source:      src,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]MealResponse, 0, len(order))
	for _, id := range order {
		result = append(result, *mealMap[id])
	}
	return result, nil
}

func (h *Handler) IdentifyFood(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	var req struct {
		Photo       string `json:"photo"`
		PortionHint string `json:"portion_hint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Photo == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("photo is required"))
		return
	}

	imageBytes, _, err := decodePhotoData(req.Photo, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid photo: %w", err))
		return
	}

	foods, err := h.identifyFoodFromPhoto(imageBytes, req.PortionHint, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("food identification failed: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: foods})
}

func (h *Handler) CreateMeal(w http.ResponseWriter, r *http.Request) {
	var req CreateMealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if !validMealTypes[req.MealType] {
		writeError(w, http.StatusBadRequest, errors.New("meal_type must be breakfast, lunch, dinner, or snack"))
		return
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	timestamp := time.Now()

	var photoPath *string

	var aiFoods []ManualFood

	if req.Photo != "" {
		imageBytes, filename, err := decodePhotoData(req.Photo, userID)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid photo: %w", err))
			return
		}

		dir := fmt.Sprintf("%s/%s", h.uploadDir, userID)
		if err := os.MkdirAll(dir, 0755); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("create upload dir: %w", err))
			return
		}

		filePath := fmt.Sprintf("%s/%s", dir, filename)
		if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("save photo: %w", err))
			return
		}

		relPath := fmt.Sprintf("uploads/%s/%s", userID, filename)
		photoPath = &relPath

		if h.ai != nil {
			identified, err := h.identifyFoodFromPhoto(imageBytes, req.PortionHint, userID)
			if err != nil {
				slog.Error("ai food identification failed", "error", err)
				syslog.Error("ai", "Photo food identification failed", map[string]any{"user_id": userID, "error": err.Error(), "date": time.Now().Format("2006-01-02")})
			} else {
				syslog.Info("ai", "Photo food identification", map[string]any{"user_id": userID, "items_found": len(identified), "date": time.Now().Format("2006-01-02")})
				for _, food := range identified {
					aiFoods = append(aiFoods, ManualFood{
						Name:        food.Name,
						Calories:    food.Calories,
						ProteinG:    food.ProteinG,
						CarbsG:      food.CarbsG,
						FatG:        food.FatG,
						FiberG:      food.FiberG,
						ServingSize: food.ServingSize,
					})
				}
			}
		}
	}

	var note *string
	if req.Note != "" {
		note = &req.Note
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to begin transaction: %w", err))
		return
	}
	defer tx.Rollback(r.Context())

	q := h.q.WithTx(tx)

	meal, err := q.CreateMeal(r.Context(), sqlc.CreateMealParams{
		UserID:    userID,
		Timestamp: timestamp,
		MealType:  req.MealType,
		PhotoPath: photoPath,
		Note:      note,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create meal: %w", err))
		return
	}

	for i := range req.Foods {
		_, err := q.CreateFoodItem(r.Context(), sqlc.CreateFoodItemParams{
			MealID:      meal.ID,
			Name:        req.Foods[i].Name,
			Calories:    int32(req.Foods[i].Calories),
			ProteinG:    floatToNumeric(req.Foods[i].ProteinG),
			CarbsG:      floatToNumeric(req.Foods[i].CarbsG),
			FatG:        floatToNumeric(req.Foods[i].FatG),
			FiberG:      floatToNumeric(req.Foods[i].FiberG),
			ServingSize: stringPtr(req.Foods[i].ServingSize),
			Source:      "manual",
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("create food item: %w", err))
			return
		}
	}

	for i := range aiFoods {
		_, err := q.CreateFoodItem(r.Context(), sqlc.CreateFoodItemParams{
			MealID:      meal.ID,
			Name:        aiFoods[i].Name,
			Calories:    int32(aiFoods[i].Calories),
			ProteinG:    floatToNumeric(aiFoods[i].ProteinG),
			CarbsG:      floatToNumeric(aiFoods[i].CarbsG),
			FatG:        floatToNumeric(aiFoods[i].FatG),
			FiberG:      floatToNumeric(aiFoods[i].FiberG),
			ServingSize: stringPtr(aiFoods[i].ServingSize),
			Source:      "ai",
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("create food item: %w", err))
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %w", err))
		return
	}

	foods, err := h.q.GetFoodItemsByMeal(r.Context(), meal.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get food items: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: mealToResponse(meal, foods)})
}

func decodePhotoData(dataURL, userID string) ([]byte, string, error) {
	commaIdx := strings.Index(dataURL, ",")
	if commaIdx == -1 {
		return nil, "", errors.New("invalid data URL format")
	}

	header := dataURL[:commaIdx]
	b64Data := dataURL[commaIdx+1:]

	var ext string
	switch {
	case strings.Contains(header, "image/png"):
		ext = ".png"
	case strings.Contains(header, "image/webp"):
		ext = ".webp"
	default:
		ext = ".jpg"
	}

	imageBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return nil, "", fmt.Errorf("decode base64: %w", err)
	}

	filename := uuid.New().String() + ext
	return imageBytes, filename, nil
}

func (h *Handler) GetMealsByDate(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
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

	// Single JOIN query instead of N+1 sequential queries.
	mealResponses, err := h.fetchMealsWithFoodsByDate(r.Context(), userID, date, tz)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get meals: %w", err))
		return
	}

	summary, err := h.q.GetDailySummary(r.Context(), sqlc.GetDailySummaryParams{
		UserID:    userID,
		Timestamp: date,
		Column3:   tz,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get daily summary: %w", err))
		return
	}

	if mealResponses == nil {
		mealResponses = []MealResponse{}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: DailyLogResponse{
		Meals:         mealResponses,
		TotalCalories: summary.TotalCalories,
		TotalProtein:  summary.TotalProtein,
		TotalCarbs:    summary.TotalCarbs,
		TotalFat:      summary.TotalFat,
		TotalFiber:    summary.TotalFiber,
	}})
}

func (h *Handler) GetRecentMeals(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	// Single JOIN query instead of N+1 sequential queries.
	mealResponses, err := h.fetchRecentMealsWithFoods(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get recent meals: %w", err))
		return
	}

	if mealResponses == nil {
		mealResponses = []MealResponse{}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: mealResponses})
}

func (h *Handler) DeleteMeal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	if _, err := h.q.GetMealByID(r.Context(), sqlc.GetMealByIDParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("meal not found: %w", err))
		return
	}

	if err := h.q.DeleteMeal(r.Context(), sqlc.DeleteMealParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete meal: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "meal deleted"}})
}

func (h *Handler) UpdateFoodItem(w http.ResponseWriter, r *http.Request) {
	foodID := chi.URLParam(r, "foodId")
	mealID := chi.URLParam(r, "mealId")
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	if _, err := h.q.GetMealByID(r.Context(), sqlc.GetMealByIDParams{
		ID:     mealID,
		UserID: userID,
	}); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("meal not found: %w", err))
		return
	}

	var req ManualFood
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.q.UpdateFoodItem(r.Context(), sqlc.UpdateFoodItemParams{
		Name:        req.Name,
		Calories:    int32(req.Calories),
		ProteinG:    floatToNumeric(req.ProteinG),
		CarbsG:      floatToNumeric(req.CarbsG),
		FatG:        floatToNumeric(req.FatG),
		FiberG:      floatToNumeric(req.FiberG),
		ServingSize: stringPtr(req.ServingSize),
		ID:          foodID,
		UserID:      userID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("update food item: %w", err))
		return
	}

	foods, err := h.q.GetFoodItemsByMeal(r.Context(), mealID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get food items: %w", err))
		return
	}

	for _, f := range foods {
		if f.ID == foodID {
			writeJSON(w, http.StatusOK, apiResponse{Data: FoodItemResponse{
				ID:          f.ID,
				Name:        f.Name,
				Calories:    f.Calories,
				ProteinG:    numericToFloat(f.ProteinG),
				CarbsG:      numericToFloat(f.CarbsG),
				FatG:        numericToFloat(f.FatG),
				FiberG:      numericToFloat(f.FiberG),
				ServingSize: f.ServingSize,
				Source:      f.Source,
			}})
			return
		}
	}

	writeError(w, http.StatusNotFound, errors.New("food item not found"))
}

type carryForwardFood struct {
	Name           string  `json:"name"`
	Calories       int     `json:"calories"`
	ProteinG       float64 `json:"protein_g"`
	CarbsG         float64 `json:"carbs_g"`
	FatG           float64 `json:"fat_g"`
	FiberG         float64 `json:"fiber_g"`
	ServingSize    string  `json:"serving_size"`
	OriginalFoodID string  `json:"original_food_id"` // ID to delete from yesterday
}

type carryForwardRequest struct {
	MealType            string             `json:"meal_type"`
	Foods               []carryForwardFood `json:"foods"`
	RemoveFromYesterday bool               `json:"remove_from_yesterday"`
}

// CarryForward copies selected food items from yesterday into a new meal today,
// optionally removing the originals from yesterday's log.
func (h *Handler) CarryForward(w http.ResponseWriter, r *http.Request) {
	var req carryForwardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Foods) == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("no foods selected"))
		return
	}
	if !validMealTypes[req.MealType] {
		req.MealType = "snack"
	}

	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()
	noteText := "Leftovers from yesterday"

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to begin transaction: %w", err))
		return
	}
	defer tx.Rollback(ctx)

	q := h.q.WithTx(tx)

	meal, err := q.CreateMeal(ctx, sqlc.CreateMealParams{
		UserID:    userID,
		Timestamp: time.Now(),
		MealType:  req.MealType,
		Note:      &noteText,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create meal: %w", err))
		return
	}

	var createdFoods []sqlc.FoodItem
	for _, f := range req.Foods {
		fi, err := q.CreateFoodItem(ctx, sqlc.CreateFoodItemParams{
			MealID:      meal.ID,
			Name:        f.Name,
			Calories:    int32(f.Calories),
			ProteinG:    floatToNumeric(f.ProteinG),
			CarbsG:      floatToNumeric(f.CarbsG),
			FatG:        floatToNumeric(f.FatG),
			FiberG:      floatToNumeric(f.FiberG),
			ServingSize: stringPtr(f.ServingSize),
			Source:      "leftover",
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("create food item: %w", err))
			return
		}
		createdFoods = append(createdFoods, fi)

		// Optionally remove the original from yesterday
		if req.RemoveFromYesterday && f.OriginalFoodID != "" {
			if err := q.DeleteFoodItemByUser(ctx, sqlc.DeleteFoodItemByUserParams{
				ID:     f.OriginalFoodID,
				UserID: userID,
			}); err != nil {
				slog.Warn("carry-forward: could not remove original", "food_id", f.OriginalFoodID, "error", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: mealToResponse(meal, createdFoods)})
}

func (h *Handler) DeleteFoodItemHandler(w http.ResponseWriter, r *http.Request) {
	foodID := chi.URLParam(r, "foodId")
	mealID := chi.URLParam(r, "mealId")
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	if _, err := h.q.GetMealByID(r.Context(), sqlc.GetMealByIDParams{
		ID:     mealID,
		UserID: userID,
	}); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("meal not found: %w", err))
		return
	}

	if err := h.q.DeleteFoodItemByUser(r.Context(), sqlc.DeleteFoodItemByUserParams{
		ID:     foodID,
		UserID: userID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete food item: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "food item deleted"}})
}

// LogMealFromRecipe creates a meal from a saved recipe.
// POST /api/meals/from-recipe/{recipeId}
// Body: { "meal_type": "lunch" }
func (h *Handler) LogMealFromRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID := chi.URLParam(r, "recipeId")
	userID, err := getUserID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	ctx := r.Context()

	var req LogMealFromRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !validMealTypes[req.MealType] {
		req.MealType = "snack"
	}

	var recipeName string
	err = h.pool.QueryRow(ctx,
		`SELECT name FROM recipes WHERE id = $1 AND user_id = $2`, recipeID, userID,
	).Scan(&recipeName)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("recipe not found"))
		return
	}

	// Fetch recipe foods
	type recipeFood struct {
		name        string
		calories    int32
		proteinG    float64
		carbsG      float64
		fatG        float64
		fiberG      float64
		servingSize string
	}

	foodRows, err := h.pool.Query(ctx,
		`SELECT name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size
		 FROM recipe_foods WHERE recipe_id = $1 ORDER BY sort_order ASC, id ASC`,
		recipeID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("fetch recipe foods: %w", err))
		return
	}
	defer foodRows.Close()

	var recipeFoods []recipeFood
	for foodRows.Next() {
		var f recipeFood
		if err := foodRows.Scan(&f.name, &f.calories, &f.proteinG, &f.carbsG, &f.fatG, &f.fiberG, &f.servingSize); err != nil {
			continue
		}
		recipeFoods = append(recipeFoods, f)
	}
	if err := foodRows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("scan recipe foods: %w", err))
		return
	}

	// Create the meal
	noteText := fmt.Sprintf("From recipe: %s", recipeName)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to begin transaction: %w", err))
		return
	}
	defer tx.Rollback(ctx)

	q := h.q.WithTx(tx)

	meal, err := q.CreateMeal(ctx, sqlc.CreateMealParams{
		UserID:    userID,
		Timestamp: time.Now(),
		MealType:  req.MealType,
		Note:      &noteText,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create meal: %w", err))
		return
	}

	// Create food items from recipe
	var createdFoods []sqlc.FoodItem
	for _, f := range recipeFoods {
		fi, err := q.CreateFoodItem(ctx, sqlc.CreateFoodItemParams{
			MealID:      meal.ID,
			Name:        f.name,
			Calories:    f.calories,
			ProteinG:    floatToNumeric(f.proteinG),
			CarbsG:      floatToNumeric(f.carbsG),
			FatG:        floatToNumeric(f.fatG),
			FiberG:      floatToNumeric(f.fiberG),
			ServingSize: stringPtr(f.servingSize),
			Source:      "recipe",
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("create food item: %w", err))
			return
		}
		createdFoods = append(createdFoods, fi)
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to commit transaction: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: mealToResponse(meal, createdFoods)})
}

// identifyFoodFromPhoto uses a 4-tier routing strategy to identify food:
//
//	Tier 1 — Classify: cheap AI call to tag the image as food_photo, receipt, or nutrition_label.
//	Tier 2 — Text extraction: for receipts & nutrition labels, extract text via AI vision (cheap
//	         model), then parse with a text-only call — avoids expensive vision tokens for parsing.
//	Tier 3 — Tesseract OCR: when OCR_PROVIDER=tesseract, run local OCR then parse with cheap model.
//	Tier 4 — Full vision: send the image directly to the main vision model (most expensive).
//
// Falls through to the next tier on any error.
func (h *Handler) identifyFoodFromPhoto(imageData []byte, hint, userID string) ([]ai.IdentifiedFood, error) {
	// ── Tier 1: Classify ────────────────────────────────────────────────
	category := ""
	if classifyCategory, err := h.ai.ClassifyImage(imageData); err != nil {
		slog.Warn("image classification failed, skipping to OCR/vision", "error", err)
	} else {
		category = classifyCategory
		slog.Info("image classified", "user_id", userID, "category", category)
		syslog.Info("ai", "Image classified", map[string]any{
			"user_id":  userID,
			"category": category,
			"date":     time.Now().Format("2006-01-02"),
		})
	}

	// ── Tier 2: Text extraction for receipts & nutrition labels ─────────
	if category == "receipt" || category == "nutrition_label" {
		text, err := h.ai.ExtractTextFromImage(imageData)
		if err != nil {
			slog.Warn("ai text extraction failed, falling through", "category", category, "error", err)
		} else if len(text) > 20 {
			slog.Info("ai text extraction succeeded", "user_id", userID, "category", category, "text_len", len(text))
			syslog.Info("ai", "AI text extraction", map[string]any{
				"user_id":  userID,
				"category": category,
				"text_len": len(text),
				"date":     time.Now().Format("2006-01-02"),
			})
			foods, parseErr := h.ai.IdentifyFoodFromText(text, hint)
			if parseErr != nil {
				slog.Warn("text-based parsing failed, falling through", "error", parseErr)
			} else {
				return foods, nil
			}
		} else {
			slog.Info("ai text extraction returned little text, falling through", "user_id", userID, "text_len", len(text))
		}
	}

	// ── Tier 3: Tesseract OCR (local, when configured) ─────────────────
	useOCR := h.cfg != nil && h.cfg.OCRProvider == "tesseract" && ai.IsTesseractAvailable()
	if useOCR {
		prepared, err := ai.PrepareForOCR(imageData)
		if err != nil {
			slog.Warn("ocr image prepare failed, using original", "error", err)
			prepared = imageData
		}

		ocrText, err := ai.ExtractTextFromImage(prepared)
		if err != nil {
			slog.Warn("tesseract ocr failed, falling back to ai vision", "error", err)
		} else if len(ocrText) > 20 {
			slog.Info("tesseract ocr succeeded", "user_id", userID, "text_len", len(ocrText))
			syslog.Info("ai", "Tesseract OCR extracted text", map[string]any{
				"user_id":  userID,
				"text_len": len(ocrText),
				"date":     time.Now().Format("2006-01-02"),
			})
			foods, parseErr := h.ai.IdentifyFoodFromText(ocrText, hint)
			if parseErr != nil {
				slog.Warn("tesseract text parsing failed, falling through to vision", "error", parseErr)
			} else {
				return foods, nil
			}
		} else {
			slog.Info("tesseract returned little text, falling through to vision", "user_id", userID)
		}
	}

	// ── Tier 4: Full vision model ───────────────────────────────────────
	slog.Info("using full vision model for food identification", "user_id", userID, "category", category)
	return h.ai.IdentifyFood(imageData, hint)
}
