package favorites

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
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

type favoriteRequest struct {
	Name        string  `json:"name"`
	Calories    float64 `json:"calories"`
	ProteinG    float64 `json:"protein_g"`
	CarbsG      float64 `json:"carbs_g"`
	FatG        float64 `json:"fat_g"`
	FiberG      float64 `json:"fiber_g"`
	ServingSize string  `json:"serving_size"`
	Source      string  `json:"source"`
}

type favoriteResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Calories    int32   `json:"calories"`
	ProteinG    float64 `json:"protein_g"`
	CarbsG      float64 `json:"carbs_g"`
	FatG        float64 `json:"fat_g"`
	FiberG      float64 `json:"fiber_g"`
	ServingSize string  `json:"serving_size"`
	Source      string  `json:"source"`
	UseCount    int32   `json:"use_count"`
}

func toResponse(f sqlc.FoodFavorite) favoriteResponse {
	return favoriteResponse{
		ID:          f.ID,
		Name:        f.Name,
		Calories:    f.Calories,
		ProteinG:    numericToFloat(f.ProteinG),
		CarbsG:      numericToFloat(f.CarbsG),
		FatG:        numericToFloat(f.FatG),
		FiberG:      numericToFloat(f.FiberG),
		ServingSize: f.ServingSize,
		Source:      f.Source,
		UseCount:    f.UseCount,
	}
}

func (h *Handler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req favoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if req.Source == "" {
		req.Source = "manual"
	}

	fav, err := h.q.AddFoodFavorite(r.Context(), sqlc.AddFoodFavoriteParams{
		UserID:      userID,
		Name:        req.Name,
		Calories:    int32(req.Calories),
		ProteinG:    floatToNumeric(req.ProteinG),
		CarbsG:      floatToNumeric(req.CarbsG),
		FatG:        floatToNumeric(req.FatG),
		FiberG:      floatToNumeric(req.FiberG),
		ServingSize: req.ServingSize,
		Source:      req.Source,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("add favorite: %w", err))
		return
	}

	writeJSON(w, http.StatusCreated, apiResponse{Data: toResponse(fav)})
}

func (h *Handler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserID(r)

	if err := h.q.RemoveFoodFavorite(r.Context(), sqlc.RemoveFoodFavoriteParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("remove favorite: %w", err))
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "favorite removed"}})
}

func (h *Handler) GetFavorites(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	favs, err := h.q.GetFoodFavorites(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get favorites: %w", err))
		return
	}

	resp := make([]favoriteResponse, 0, len(favs))
	for _, f := range favs {
		resp = append(resp, toResponse(f))
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) GetTopFavorites(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	limit := int32(5)

	favs, err := h.q.GetTopFavorites(r.Context(), sqlc.GetTopFavoritesParams{
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("get top favorites: %w", err))
		return
	}

	resp := make([]favoriteResponse, 0, len(favs))
	for _, f := range favs {
		resp = append(resp, toResponse(f))
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: resp})
}

func (h *Handler) LogFromFavorite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserID(r)

	_ = h.q.IncrementFavoriteUseCount(r.Context(), sqlc.IncrementFavoriteUseCountParams{
		ID:     id,
		UserID: userID,
	})

	writeJSON(w, http.StatusOK, apiResponse{Data: map[string]string{"message": "use count incremented"}})
}
