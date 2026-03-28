package meal

import (
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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"joule/internal/ai"
	"joule/internal/auth"
	"joule/internal/config"
	"joule/internal/db/sqlc"
	syslog "joule/internal/syslog"
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
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

func getUserID(r *http.Request) string {
	return r.Context().Value(auth.ContextUserID).(string)
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

func (h *Handler) CreateMeal(w http.ResponseWriter, r *http.Request) {
	var req CreateMealRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if !validMealTypes[req.MealType] {
		writeError(w, http.StatusBadRequest, errors.New("meal_type must be breakfast, lunch, dinner, or snack"))
		return
	}

	userID := getUserID(r)
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
			identified, err := h.identifyFoodFromPhoto(imageBytes, req.PortionHint, getUserID(r))
			if err != nil {
				slog.Error("ai food identification failed", "error", err)
				syslog.Error("ai", "Photo food identification failed", map[string]any{"user_id": getUserID(r), "error": err.Error()})
			} else {
				syslog.Info("ai", "Photo food identification", map[string]any{"user_id": getUserID(r), "items_found": len(identified)})
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

	meal, err := h.q.CreateMeal(r.Context(), sqlc.CreateMealParams{
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
		_, err := h.q.CreateFoodItem(r.Context(), sqlc.CreateFoodItemParams{
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
		_, err := h.q.CreateFoodItem(r.Context(), sqlc.CreateFoodItemParams{
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

	meals, err := h.q.GetMealsByDate(r.Context(), sqlc.GetMealsByDateParams{
		UserID:    userID,
		Timestamp: date,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get meals: %w", err))
		return
	}

	var mealResponses []MealResponse
	for _, m := range meals {
		foods, err := h.q.GetFoodItemsByMeal(r.Context(), m.ID)
		if err != nil {
			slog.Error("get food items for meal", "meal_id", m.ID, "error", err)
			continue
		}
		mealResponses = append(mealResponses, mealToResponse(m, foods))
	}

	summary, err := h.q.GetDailySummary(r.Context(), sqlc.GetDailySummaryParams{
		UserID:    userID,
		Timestamp: date,
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
	userID := getUserID(r)

	meals, err := h.q.GetRecentMeals(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get recent meals: %w", err))
		return
	}

	var mealResponses []MealResponse
	for _, m := range meals {
		foods, err := h.q.GetFoodItemsByMeal(r.Context(), m.ID)
		if err != nil {
			slog.Error("get food items for meal", "meal_id", m.ID, "error", err)
			continue
		}
		mealResponses = append(mealResponses, mealToResponse(m, foods))
	}

	if mealResponses == nil {
		mealResponses = []MealResponse{}
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: mealResponses})
}

func (h *Handler) DeleteMeal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserID(r)

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
	userID := getUserID(r)

	if _, err := h.q.GetMealByID(r.Context(), sqlc.GetMealByIDParams{
		ID:     mealID,
		UserID: userID,
	}); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("meal not found: %w", err))
		return
	}

	var req ManualFood
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
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
	MealType             string             `json:"meal_type"`
	Foods                []carryForwardFood `json:"foods"`
	RemoveFromYesterday  bool               `json:"remove_from_yesterday"`
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

	userID := getUserID(r)
	ctx := r.Context()
	noteText := "Leftovers from yesterday"

	meal, err := h.q.CreateMeal(ctx, sqlc.CreateMealParams{
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
		fi, err := h.q.CreateFoodItem(ctx, sqlc.CreateFoodItemParams{
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
			slog.Warn("carry-forward: create food item failed", "error", err)
			continue
		}
		createdFoods = append(createdFoods, fi)

		// Optionally remove the original from yesterday
		if req.RemoveFromYesterday && f.OriginalFoodID != "" {
			h.q.DeleteFoodItem(ctx, f.OriginalFoodID)
		}
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: mealToResponse(meal, createdFoods)})
}

// LogMealFromRecipe creates a meal from a saved recipe.
// POST /api/meals/from-recipe/{recipeId}
// Body: { "meal_type": "lunch" }
func (h *Handler) LogMealFromRecipe(w http.ResponseWriter, r *http.Request) {
	recipeID := chi.URLParam(r, "recipeId")
	userID := getUserID(r)
	ctx := r.Context()

	var req LogMealFromRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !validMealTypes[req.MealType] {
		req.MealType = "snack"
	}

	// Fetch recipe (no ownership check — recipes are read-public within the app)
	var recipeName string
	err := h.pool.QueryRow(ctx,
		`SELECT name FROM recipes WHERE id = $1`, recipeID,
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
	meal, err := h.q.CreateMeal(ctx, sqlc.CreateMealParams{
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
		fi, err := h.q.CreateFoodItem(ctx, sqlc.CreateFoodItemParams{
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
			slog.Warn("from-recipe: create food item failed", "error", err)
			continue
		}
		createdFoods = append(createdFoods, fi)
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: mealToResponse(meal, createdFoods)})
}

// identifyFoodFromPhoto runs food identification on a photo.
// If OCR_PROVIDER=tesseract and Tesseract is available, it extracts text via
// Tesseract first, then uses a cheaper text-only AI call. This improves accuracy
// on nutrition labels and avoids expensive vision tokens.
// Falls back to standard AI vision if OCR is not configured or fails.
func (h *Handler) identifyFoodFromPhoto(imageData []byte, hint, userID string) ([]ai.IdentifiedFood, error) {
	useOCR := h.cfg != nil && h.cfg.OCRProvider == "tesseract" && ai.IsTesseractAvailable()

	if useOCR {
		// Preprocess image for better OCR accuracy (grayscale)
		prepared, err := ai.PrepareForOCR(imageData)
		if err != nil {
			slog.Warn("ocr image prepare failed, using original", "error", err)
			prepared = imageData
		}

		ocrText, err := ai.ExtractTextFromImage(prepared)
		if err != nil {
			slog.Warn("tesseract ocr failed, falling back to ai vision", "error", err)
			// Fall through to AI vision below
		} else if len(ocrText) > 20 {
			slog.Info("tesseract ocr succeeded", "user_id", userID, "text_len", len(ocrText))
			syslog.Info("ai", "Tesseract OCR extracted text", map[string]any{
				"user_id":  userID,
				"text_len": len(ocrText),
			})
			return h.ai.IdentifyFoodFromText(ocrText, hint)
		} else {
			slog.Info("tesseract returned little text, falling back to ai vision", "user_id", userID)
		}
	}

	// Default: send image directly to AI vision model
	return h.ai.IdentifyFood(imageData, hint)
}
