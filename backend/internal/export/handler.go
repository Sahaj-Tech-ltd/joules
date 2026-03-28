package export

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"joule/internal/auth"
	"joule/internal/db/sqlc"
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

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := getUserID(r)
	exportType := r.URL.Query().Get("type")
	if exportType == "" {
		exportType = "meals"
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dateStr := today.Format("2006-01-02")

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="joule-%s-%s.csv"`, exportType, dateStr))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	switch exportType {
	case "weight":
		weights, err := h.q.GetWeightHistory(ctx, sqlc.GetWeightHistoryParams{
			UserID: userID,
			Date:   today.AddDate(0, 0, -365),
			Date_2: today,
		})
		if err != nil {
			slog.Error("export weight", "error", err)
			return
		}
		cw.Write([]string{"Date", "Weight (kg)"})
		for _, wt := range weights {
			cw.Write([]string{wt.Date.Format("2006-01-02"), fmt.Sprintf("%.2f", numericToFloat(wt.WeightKg))})
		}

	case "water":
		cw.Write([]string{"Date", "Water (ml)"})
		for i := 0; i < 30; i++ {
			d := today.AddDate(0, 0, -i)
			total, err := h.q.GetWaterByDate(ctx, sqlc.GetWaterByDateParams{UserID: userID, Date: d})
			if err != nil {
				continue
			}
			cw.Write([]string{d.Format("2006-01-02"), fmt.Sprintf("%d", total)})
		}

	case "exercise":
		cw.Write([]string{"Date", "Name", "Duration (min)", "Calories Burned"})
		for i := 0; i < 30; i++ {
			d := today.AddDate(0, 0, -i)
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
		for i := 0; i < 30; i++ {
			d := today.AddDate(0, 0, -i)
			summary, err := h.q.GetDailySummary(ctx, sqlc.GetDailySummaryParams{UserID: userID, Timestamp: d})
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
