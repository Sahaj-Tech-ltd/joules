package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"joules/internal/auth"
	"joules/internal/db/sqlc"
)

type Handler struct {
	q *sqlc.Queries
}

func NewHandler(q *sqlc.Queries) *Handler {
	return &Handler{q: q}
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

func parseDateOrDefault(s string, defaultDate time.Time) time.Time {
	if s == "" {
		return defaultDate
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return defaultDate
	}
	return t
}

func clampDateRange(fromDate, toDate time.Time) (time.Time, time.Time) {
	maxRange := 730 * 24 * time.Hour
	if toDate.Sub(fromDate) > maxRange {
		fromDate = toDate.AddDate(0, 0, -730)
	}
	return fromDate, toDate
}

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r)
	exportType := r.URL.Query().Get("type")
	if exportType == "" {
		exportType = "meals"
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	fromDate := parseDateOrDefault(r.URL.Query().Get("from"), today.AddDate(0, 0, -30))
	toDate := parseDateOrDefault(r.URL.Query().Get("to"), today)
	fromDate, toDate = clampDateRange(fromDate, toDate)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="joule-%s-%s-to-%s.csv"`, exportType, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	switch exportType {
	case "weight":
		weights, err := h.q.GetWeightHistory(ctx, sqlc.GetWeightHistoryParams{
			UserID: userID,
			Date:   fromDate,
			Date_2: toDate,
		})
		if err != nil {
			slog.Error("export weight csv", "error", err)
			return
		}
		cw.Write([]string{"Date", "Weight (kg)"})
		for _, wt := range weights {
			cw.Write([]string{wt.Date.Format("2006-01-02"), fmt.Sprintf("%.2f", numericToFloat(wt.WeightKg))})
		}

	case "water":
		cw.Write([]string{"Date", "Water (ml)"})
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			total, err := h.q.GetWaterByDate(ctx, sqlc.GetWaterByDateParams{UserID: userID, Date: d})
			if err != nil {
				continue
			}
			cw.Write([]string{d.Format("2006-01-02"), fmt.Sprintf("%d", total)})
		}

	case "exercise":
		cw.Write([]string{"Date", "Name", "Duration (min)", "Calories Burned"})
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			exercises, err := h.q.GetExercisesByDate(ctx, sqlc.GetExercisesByDateParams{UserID: userID, Timestamp: d})
			if err != nil {
				continue
			}
			for _, ex := range exercises {
				cw.Write([]string{d.Format("2006-01-02"), ex.Name, fmt.Sprintf("%d", ex.DurationMin), fmt.Sprintf("%d", ex.CaloriesBurned)})
			}
		}

	default:
		cw.Write([]string{"Date", "Calories", "Protein (g)", "Carbs (g)", "Fat (g)", "Fiber (g)", "Water (ml)", "Calories Burned"})
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{UserID: userID, Timestamp: d, Column3: "UTC"})
			if err != nil {
				continue
			}
			cw.Write([]string{
				d.Format("2006-01-02"),
				fmt.Sprintf("%d", summary.TotalCalories),
				fmt.Sprintf("%.1f", summary.TotalProtein),
				fmt.Sprintf("%.1f", summary.TotalCarbs),
				fmt.Sprintf("%.1f", summary.TotalFat),
				fmt.Sprintf("%.1f", summary.TotalFiber),
				fmt.Sprintf("%d", summary.TotalWaterMl),
				fmt.Sprintf("%d", summary.TotalBurned),
			})
		}
	}
}

type dayRow struct {
	Date     string  `json:"date"`
	Calories int32   `json:"calories"`
	Protein  float64 `json:"protein_g"`
	Carbs    float64 `json:"carbs_g"`
	Fat      float64 `json:"fat_g"`
	Fiber    float64 `json:"fiber_g"`
	Water    int32   `json:"water_ml"`
	Burned   int32   `json:"calories_burned"`
}

type weightRow struct {
	Date string  `json:"date"`
	Kg   float64 `json:"weight_kg"`
}

type waterRow struct {
	Date string `json:"date"`
	Ml   int64  `json:"water_ml"`
}

type exerciseRow struct {
	Date        string `json:"date"`
	Name        string `json:"name"`
	DurationMin int32  `json:"duration_min"`
	Calories    int32  `json:"calories_burned"`
}

func (h *Handler) ExportJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r)
	exportType := r.URL.Query().Get("type")
	if exportType == "" {
		exportType = "meals"
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	fromDate := parseDateOrDefault(r.URL.Query().Get("from"), today.AddDate(0, 0, -30))
	toDate := parseDateOrDefault(r.URL.Query().Get("to"), today)
	fromDate, toDate = clampDateRange(fromDate, toDate)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="joule-%s-%s-to-%s.json"`, exportType, fromDate.Format("2006-01-02"), toDate.Format("2006-01-02")))

	switch exportType {
	case "weight":
		weights, err := h.q.GetWeightHistory(ctx, sqlc.GetWeightHistoryParams{
			UserID: userID,
			Date:   fromDate,
			Date_2: toDate,
		})
		if err != nil {
			slog.Error("export weight json", "error", err)
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}
		rows := make([]weightRow, 0, len(weights))
		for _, wt := range weights {
			rows = append(rows, weightRow{
				Date: wt.Date.Format("2006-01-02"),
				Kg:   numericToFloat(wt.WeightKg),
			})
		}
		json.NewEncoder(w).Encode(rows)

	case "water":
		var rows []waterRow
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			total, err := h.q.GetWaterByDate(ctx, sqlc.GetWaterByDateParams{UserID: userID, Date: d})
			if err != nil {
				continue
			}
			rows = append(rows, waterRow{
				Date: d.Format("2006-01-02"),
				Ml:   int64(total),
			})
		}
		json.NewEncoder(w).Encode(rows)

	case "exercise":
		var rows []exerciseRow
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			exercises, err := h.q.GetExercisesByDate(ctx, sqlc.GetExercisesByDateParams{UserID: userID, Timestamp: d})
			if err != nil {
				continue
			}
			for _, ex := range exercises {
				rows = append(rows, exerciseRow{
					Date:        d.Format("2006-01-02"),
					Name:        ex.Name,
					DurationMin: ex.DurationMin,
					Calories:    ex.CaloriesBurned,
				})
			}
		}
		json.NewEncoder(w).Encode(rows)

	default:
		var rows []dayRow
		for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
			summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{UserID: userID, Timestamp: d, Column3: "UTC"})
			if err != nil {
				continue
			}
			rows = append(rows, dayRow{
				Date:     d.Format("2006-01-02"),
				Calories: summary.TotalCalories,
				Protein:  summary.TotalProtein,
				Carbs:    summary.TotalCarbs,
				Fat:      summary.TotalFat,
				Fiber:    summary.TotalFiber,
				Water:    summary.TotalWaterMl,
				Burned:   summary.TotalBurned,
			})
		}
		json.NewEncoder(w).Encode(rows)
	}
}
