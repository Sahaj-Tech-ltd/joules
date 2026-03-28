package recipe

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"joule/internal/auth"
)

// Handler handles recipe CRUD for users.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler creates a new recipe Handler.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
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
	slog.Error("recipe request error", "status", status, "error", err)
	writeJSON(w, status, apiResponse{Error: err.Error()})
}

// RecipeFood represents a single food item within a recipe.
type RecipeFood struct {
	Name        string  `json:"name"`
	Calories    int     `json:"calories"`
	ProteinG    float64 `json:"protein_g"`
	CarbsG      float64 `json:"carbs_g"`
	FatG        float64 `json:"fat_g"`
	FiberG      float64 `json:"fiber_g"`
	ServingSize string  `json:"serving_size"`
}

// Recipe is the full recipe response with embedded foods.
type Recipe struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Foods       []RecipeFood `json:"foods"`
	CreatedAt   time.Time    `json:"created_at"`
}

// CreateRecipeRequest is the body for POST /api/recipes.
type CreateRecipeRequest struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Foods       []RecipeFood `json:"foods"`
}

// List handles GET /api/recipes — returns all recipes for the authenticated user.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextUserID).(string)
	ctx := r.Context()

	rows, err := h.pool.Query(ctx,
		`SELECT id, name, description, created_at FROM recipes WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("query recipes: %w", err))
		return
	}
	defer rows.Close()

	type recipeRow struct {
		id          string
		name        string
		description string
		createdAt   time.Time
	}

	var recipeRows []recipeRow
	var recipeIDs []string
	for rows.Next() {
		var rr recipeRow
		if err := rows.Scan(&rr.id, &rr.name, &rr.description, &rr.createdAt); err != nil {
			continue
		}
		recipeRows = append(recipeRows, rr)
		recipeIDs = append(recipeIDs, rr.id)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("scan recipes: %w", err))
		return
	}

	// Build recipe map and load foods
	recipeMap := map[string]*Recipe{}
	var recipeOrder []string
	for _, rr := range recipeRows {
		recipeMap[rr.id] = &Recipe{
			ID:          rr.id,
			Name:        rr.name,
			Description: rr.description,
			Foods:       []RecipeFood{},
			CreatedAt:   rr.createdAt,
		}
		recipeOrder = append(recipeOrder, rr.id)
	}

	if len(recipeIDs) > 0 {
		foodRows, err := h.pool.Query(ctx,
			`SELECT recipe_id, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size
			 FROM recipe_foods
			 WHERE recipe_id = ANY($1)
			 ORDER BY sort_order ASC, id ASC`,
			recipeIDs,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("query recipe foods: %w", err))
			return
		}
		defer foodRows.Close()

		for foodRows.Next() {
			var recipeID string
			var rf RecipeFood
			if err := foodRows.Scan(&recipeID, &rf.Name, &rf.Calories, &rf.ProteinG, &rf.CarbsG, &rf.FatG, &rf.FiberG, &rf.ServingSize); err != nil {
				continue
			}
			if rec, ok := recipeMap[recipeID]; ok {
				rec.Foods = append(rec.Foods, rf)
			}
		}
	}

	results := make([]Recipe, 0, len(recipeOrder))
	for _, id := range recipeOrder {
		results = append(results, *recipeMap[id])
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: results})
}

// Create handles POST /api/recipes — creates a new recipe for the authenticated user.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextUserID).(string)
	ctx := r.Context()

	var req CreateRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("name is required"))
		return
	}

	// Insert recipe
	var recipeID string
	var createdAt time.Time
	err := h.pool.QueryRow(ctx,
		`INSERT INTO recipes (user_id, name, description) VALUES ($1, $2, $3) RETURNING id, created_at`,
		userID, req.Name, req.Description,
	).Scan(&recipeID, &createdAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("create recipe: %w", err))
		return
	}

	// Insert recipe foods
	for i, f := range req.Foods {
		_, err := h.pool.Exec(ctx,
			`INSERT INTO recipe_foods (recipe_id, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			recipeID, f.Name, f.Calories,
			fmt.Sprintf("%.2f", f.ProteinG),
			fmt.Sprintf("%.2f", f.CarbsG),
			fmt.Sprintf("%.2f", f.FatG),
			fmt.Sprintf("%.2f", f.FiberG),
			f.ServingSize, i,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("create recipe food: %w", err))
			return
		}
	}

	foods := make([]RecipeFood, len(req.Foods))
	copy(foods, req.Foods)

	resp := Recipe{
		ID:          recipeID,
		Name:        req.Name,
		Description: req.Description,
		Foods:       foods,
		CreatedAt:   createdAt,
	}
	writeJSON(w, http.StatusCreated, apiResponse{Data: resp})
}

// Delete handles DELETE /api/recipes/{id} — deletes a recipe owned by the user.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.ContextUserID).(string)
	recipeID := chi.URLParam(r, "id")
	ctx := r.Context()

	// Verify ownership
	var ownerID string
	err := h.pool.QueryRow(ctx,
		`SELECT user_id FROM recipes WHERE id = $1`, recipeID,
	).Scan(&ownerID)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("recipe not found"))
		return
	}
	if ownerID != userID {
		writeError(w, http.StatusForbidden, fmt.Errorf("not your recipe"))
		return
	}

	_, err = h.pool.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, recipeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("delete recipe: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "recipe deleted"}})
}
