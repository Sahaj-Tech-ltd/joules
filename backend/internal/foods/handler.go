package foods

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/ai"
)

// Handler handles food search and barcode lookups.
type Handler struct {
	pool       *pgxpool.Pool
	httpClient *http.Client
	aiClient   ai.Client
}

// NewHandler creates a new foods Handler.
func NewHandler(pool *pgxpool.Pool, aiClient ai.Client) *Handler {
	return &Handler{
		pool:       pool,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		aiClient:   aiClient,
	}
}

// FoodResult is the unified response type for both local DB and OFF results.
type FoodResult struct {
	ID          int64   `json:"id,omitempty"`
	Barcode     string  `json:"barcode,omitempty"`
	Name        string  `json:"name"`
	Brand       string  `json:"brand,omitempty"`
	Calories    int     `json:"calories"`
	ProteinG    float64 `json:"protein_g"`
	CarbsG      float64 `json:"carbs_g"`
	FatG        float64 `json:"fat_g"`
	FiberG      float64 `json:"fiber_g"`
	ServingSize string  `json:"serving_size"`
	Ingredients string  `json:"ingredients,omitempty"`
	Source      string  `json:"source"` // "local" or "openfoodfacts"
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
	slog.Error("foods request error", "status", status, "error", err)
	msg := err.Error()
	if status >= 500 {
		msg = "internal server error"
	}
	writeJSON(w, status, apiResponse{Error: msg})
}

// offNutrientsResponse maps the relevant nutriments fields from the OFF API.
type offNutrientsResponse struct {
	EnergyKcal100g    float64 `json:"energy-kcal_100g"`
	Proteins100g      float64 `json:"proteins_100g"`
	Carbohydrates100g float64 `json:"carbohydrates_100g"`
	Fat100g           float64 `json:"fat_100g"`
	Fiber100g         float64 `json:"fiber_100g"`
}

type offProduct struct {
	ProductName     string               `json:"product_name"`
	Brands          string               `json:"brands"`
	Nutriments      offNutrientsResponse `json:"nutriments"`
	ServingSize     string               `json:"serving_size"`
	IngredientsText string               `json:"ingredients_text"`
}

type offBarcodeResponse struct {
	Status  int        `json:"status"`
	Product offProduct `json:"product"`
}

type offSearchResponse struct {
	Products []offProduct `json:"products"`
}

// searchLocal queries the local foods_db using full-text and LIKE search.
func (h *Handler) searchLocal(r *http.Request, q string, limit int) ([]FoodResult, error) {
	const localSQL = `
		SELECT id, COALESCE(barcode,''), name, brand, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, ingredients
		FROM foods_db
		WHERE to_tsvector('english', name) @@ plainto_tsquery('english', $1)
		   OR lower(name) LIKE lower('%' || $1 || '%')
		ORDER BY
		    CASE WHEN lower(name) LIKE lower($1 || '%') THEN 0 ELSE 1 END,
		    calories DESC
		LIMIT $2`

	rows, err := h.pool.Query(r.Context(), localSQL, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FoodResult
	for rows.Next() {
		var f FoodResult
		if err := rows.Scan(
			&f.ID, &f.Barcode, &f.Name, &f.Brand,
			&f.Calories, &f.ProteinG, &f.CarbsG, &f.FatG, &f.FiberG,
			&f.ServingSize, &f.Ingredients,
		); err != nil {
			continue
		}
		f.Source = "local"
		results = append(results, f)
	}
	return results, rows.Err()
}

// cacheInDB inserts an OFF product into foods_db for future lookups.
// Uses context.Background() because it may be called from a goroutine after the request ends.
func (h *Handler) cacheInDB(ctx context.Context, f FoodResult) {
	const insertSQL = `
		INSERT INTO foods_db (barcode, name, brand, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, ingredients)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING`

	var barcode *string
	if f.Barcode != "" {
		barcode = &f.Barcode
	}
	_, err := h.pool.Exec(ctx, insertSQL,
		barcode, f.Name, f.Brand, f.Calories,
		fmt.Sprintf("%.2f", f.ProteinG),
		fmt.Sprintf("%.2f", f.CarbsG),
		fmt.Sprintf("%.2f", f.FatG),
		fmt.Sprintf("%.2f", f.FiberG),
		f.ServingSize, f.Ingredients,
	)
	if err != nil {
		slog.Warn("foods: failed to cache OFF result", "error", err)
	}
}

// productToFoodResult converts an OFF product to a FoodResult.
func productToFoodResult(p offProduct, barcode string) FoodResult {
	serving := p.ServingSize
	if serving == "" {
		serving = "100g"
	}
	return FoodResult{
		Barcode:     barcode,
		Name:        p.ProductName,
		Brand:       p.Brands,
		Calories:    int(p.Nutriments.EnergyKcal100g),
		ProteinG:    p.Nutriments.Proteins100g,
		CarbsG:      p.Nutriments.Carbohydrates100g,
		FatG:        p.Nutriments.Fat100g,
		FiberG:      p.Nutriments.Fiber100g,
		ServingSize: serving,
		Ingredients: p.IngredientsText,
		Source:      "openfoodfacts",
	}
}

// Search handles GET /api/foods/search?q=...&limit=20
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("q parameter is required"))
		return
	}

	limit := 20
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	localResults, err := h.searchLocal(r, q, limit)
	if err != nil {
		slog.Warn("foods: local search error", "error", err)
		localResults = []FoodResult{}
	}

	// Fall back to Open Food Facts if fewer than 3 local results
	if len(localResults) < 3 {
		offResults := h.searchOFF(r, q)
		for _, f := range offResults {
			localResults = append(localResults, f)
			fc := f // capture for goroutine
			go h.cacheInDB(context.Background(), fc)
		}
	}

	if localResults == nil {
		localResults = []FoodResult{}
	}
	writeJSON(w, http.StatusOK, apiResponse{Data: localResults})
}

// searchOFF queries the Open Food Facts search API.
func (h *Handler) searchOFF(r *http.Request, q string) []FoodResult {
	offURL := fmt.Sprintf(
		"https://world.openfoodfacts.org/cgi/search.pl?search_terms=%s&search_simple=1&action=process&json=1&page_size=10&fields=product_name,brands,nutriments,serving_size",
		url.QueryEscape(q),
	)

	resp, err := h.httpClient.Get(offURL)
	if err != nil {
		slog.Warn("foods: OFF search request failed", "error", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("foods: OFF search read body failed", "error", err)
		return nil
	}

	var offResp offSearchResponse
	if err := json.Unmarshal(body, &offResp); err != nil {
		slog.Warn("foods: OFF search unmarshal failed", "error", err)
		return nil
	}

	var results []FoodResult
	for _, p := range offResp.Products {
		if p.ProductName == "" {
			continue
		}
		results = append(results, productToFoodResult(p, ""))
	}
	return results
}

// GetByBarcode handles GET /api/foods/barcode/{upc}
func (h *Handler) GetByBarcode(w http.ResponseWriter, r *http.Request) {
	upc := chi.URLParam(r, "upc")
	if upc == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("upc is required"))
		return
	}

	// Check local DB first
	const localSQL = `
		SELECT id, COALESCE(barcode,''), name, brand, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, ingredients
		FROM foods_db
		WHERE barcode = $1
		LIMIT 1`

	var f FoodResult
	err := h.pool.QueryRow(r.Context(), localSQL, upc).Scan(
		&f.ID, &f.Barcode, &f.Name, &f.Brand,
		&f.Calories, &f.ProteinG, &f.CarbsG, &f.FatG, &f.FiberG,
		&f.ServingSize, &f.Ingredients,
	)
	if err == nil {
		f.Source = "local"
		writeJSON(w, http.StatusOK, apiResponse{Data: f})
		return
	}

	// Fall back to OFF API
	offURL := fmt.Sprintf("https://world.openfoodfacts.org/api/v2/product/%s.json", url.PathEscape(upc))
	resp, err := h.httpClient.Get(offURL)
	if err != nil {
		slog.Warn("foods: OFF barcode request failed", "error", err)
		writeError(w, http.StatusNotFound, fmt.Errorf("product not found"))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("product not found"))
		return
	}

	var offResp offBarcodeResponse
	if err := json.Unmarshal(body, &offResp); err != nil || offResp.Status != 1 {
		writeError(w, http.StatusNotFound, fmt.Errorf("product not found"))
		return
	}

	if offResp.Product.ProductName == "" {
		writeError(w, http.StatusNotFound, fmt.Errorf("product not found"))
		return
	}

	result := productToFoodResult(offResp.Product, upc)

	// Cache asynchronously
	go h.cacheInDB(context.Background(), result)

	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}

// barcodeScanRequest is the JSON body for POST /api/foods/barcode-scan.
type barcodeScanRequest struct {
	Photo string `json:"photo"`
}

// BarcodeScan handles POST /api/foods/barcode-scan
// Accepts a base64-encoded photo and uses AI to identify the food product.
func (h *Handler) BarcodeScan(w http.ResponseWriter, r *http.Request) {
	var req barcodeScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Photo == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("photo is required"))
		return
	}

	// Decode base64 data URL to raw bytes
	commaIdx := strings.Index(req.Photo, ",")
	if commaIdx == -1 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid data URL format"))
		return
	}
	b64Data := req.Photo[commaIdx+1:]
	imageBytes, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid base64 image data"))
		return
	}

	if h.aiClient == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("AI is not configured"))
		return
	}

	identified, err := h.aiClient.IdentifyFood(imageBytes, "Product barcode scan — identify the food product from the packaging")
	if err != nil {
		slog.Error("foods: barcode scan AI error", "error", err)
		writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to identify product"))
		return
	}

	if len(identified) == 0 {
		writeError(w, http.StatusNotFound, fmt.Errorf("could not identify product from image"))
		return
	}

	item := identified[0]
	result := FoodResult{
		Name:        item.Name,
		Calories:    int(item.Calories),
		ProteinG:    item.ProteinG,
		CarbsG:      item.CarbsG,
		FatG:        item.FatG,
		FiberG:      item.FiberG,
		ServingSize: item.ServingSize,
		Source:      "ai_scan",
	}

	writeJSON(w, http.StatusOK, apiResponse{Data: result})
}
