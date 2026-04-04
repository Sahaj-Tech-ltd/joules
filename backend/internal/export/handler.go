package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"

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

func getUserID(r *http.Request) (string, error) {
	userID, ok := r.Context().Value(auth.ContextUserID).(string)
	if !ok {
		return "", fmt.Errorf("unauthorized")
	}
	return userID, nil
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
	userID, err := getUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
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
		rows, wErr := h.pool.Query(ctx,
			`SELECT date, COALESCE(SUM(amount_ml),0)::bigint as total FROM water_logs
			 WHERE user_id = $1 AND date BETWEEN $2 AND $3
			 GROUP BY date ORDER BY date`, userID, fromDate, toDate)
		if wErr == nil {
			defer rows.Close()
			for rows.Next() {
				var d time.Time
				var total int64
				if rows.Scan(&d, &total) == nil {
					cw.Write([]string{d.Format("2006-01-02"), fmt.Sprintf("%d", total)})
				}
			}
		}

	case "exercise":
		cw.Write([]string{"Date", "Name", "Duration (min)", "Calories Burned"})
		rows, eErr := h.pool.Query(ctx,
			`SELECT (timestamp AT TIME ZONE 'UTC')::date, name, duration_min, calories_burned
			 FROM exercises WHERE user_id = $1 AND timestamp >= $2 AND timestamp < $3 + interval '1 day'
			 ORDER BY timestamp`, userID, fromDate, toDate)
		if eErr == nil {
			defer rows.Close()
			for rows.Next() {
				var d time.Time
				var name string
				var dur, cal int32
				if rows.Scan(&d, &name, &dur, &cal) == nil {
					cw.Write([]string{d.Format("2006-01-02"), name, fmt.Sprintf("%d", dur), fmt.Sprintf("%d", cal)})
				}
			}
		}

	default:
		cw.Write([]string{"Date", "Calories", "Protein (g)", "Carbs (g)", "Fat (g)", "Fiber (g)", "Water (ml)", "Calories Burned"})
		rows, mErr := h.pool.Query(ctx,
			`SELECT d::date,
			        COALESCE(m.cal, 0)::int, COALESCE(m.prot, 0)::float8, COALESCE(m.carb, 0)::float8,
			        COALESCE(m.fat, 0)::float8, COALESCE(m.fib, 0)::float8,
			        COALESCE(w.water, 0)::int, COALESCE(e.burned, 0)::int
			 FROM generate_series($2, $3, '1 day'::interval) d
			 LEFT JOIN (SELECT (timestamp AT TIME ZONE 'UTC')::date as dd,
			               SUM(fi.calories)::int as cal, SUM(fi.protein_g)::float8 as prot,
			               SUM(fi.carbs_g)::float8 as carb, SUM(fi.fat_g)::float8 as fat, SUM(fi.fiber_g)::float8 as fib
			           FROM meals JOIN food_items fi ON fi.meal_id = meals.id
			           WHERE meals.user_id = $1 GROUP BY dd) m ON m.dd = d::date
			 LEFT JOIN (SELECT date, SUM(amount_ml)::int as water FROM water_logs
			           WHERE user_id = $1 GROUP BY date) w ON w.date = d::date
			 LEFT JOIN (SELECT (timestamp AT TIME ZONE 'UTC')::date as dd, SUM(calories_burned)::int as burned
			           FROM exercises WHERE user_id = $1 GROUP BY dd) e ON e.dd = d::date
			 ORDER BY d`, userID, fromDate, toDate)
		if mErr == nil {
			defer rows.Close()
			for rows.Next() {
				var d time.Time
				var cal, water, burned int
				var prot, carb, fat, fib float64
				if rows.Scan(&d, &cal, &prot, &carb, &fat, &fib, &water, &burned) == nil {
					cw.Write([]string{
						d.Format("2006-01-02"),
						fmt.Sprintf("%d", cal), fmt.Sprintf("%.1f", prot),
						fmt.Sprintf("%.1f", carb), fmt.Sprintf("%.1f", fat),
						fmt.Sprintf("%.1f", fib), fmt.Sprintf("%d", water),
						fmt.Sprintf("%d", burned),
					})
				}
			}
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
	userID, err := getUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
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
		dbRows, wErr := h.pool.Query(ctx,
			`SELECT date, COALESCE(SUM(amount_ml),0)::bigint as total FROM water_logs
			 WHERE user_id = $1 AND date BETWEEN $2 AND $3
			 GROUP BY date ORDER BY date`, userID, fromDate, toDate)
		if wErr == nil {
			defer dbRows.Close()
			for dbRows.Next() {
				var d time.Time
				var total int64
				if dbRows.Scan(&d, &total) == nil {
					rows = append(rows, waterRow{
						Date: d.Format("2006-01-02"),
						Ml:   total,
					})
				}
			}
		}
		json.NewEncoder(w).Encode(rows)

	case "exercise":
		var rows []exerciseRow
		dbRows, eErr := h.pool.Query(ctx,
			`SELECT (timestamp AT TIME ZONE 'UTC')::date, name, duration_min, calories_burned
			 FROM exercises WHERE user_id = $1 AND timestamp >= $2 AND timestamp < $3 + interval '1 day'
			 ORDER BY timestamp`, userID, fromDate, toDate)
		if eErr == nil {
			defer dbRows.Close()
			for dbRows.Next() {
				var d time.Time
				var name string
				var dur, cal int32
				if dbRows.Scan(&d, &name, &dur, &cal) == nil {
					rows = append(rows, exerciseRow{
						Date:        d.Format("2006-01-02"),
						Name:        name,
						DurationMin: dur,
						Calories:    cal,
					})
				}
			}
		}
		json.NewEncoder(w).Encode(rows)

	default:
		var rows []dayRow
		dbRows, mErr := h.pool.Query(ctx,
			`SELECT d::date,
			        COALESCE(m.cal, 0)::int, COALESCE(m.prot, 0)::float8, COALESCE(m.carb, 0)::float8,
			        COALESCE(m.fat, 0)::float8, COALESCE(m.fib, 0)::float8,
			        COALESCE(w.water, 0)::int, COALESCE(e.burned, 0)::int
			 FROM generate_series($2, $3, '1 day'::interval) d
			 LEFT JOIN (SELECT (timestamp AT TIME ZONE 'UTC')::date as dd,
			               SUM(fi.calories)::int as cal, SUM(fi.protein_g)::float8 as prot,
			               SUM(fi.carbs_g)::float8 as carb, SUM(fi.fat_g)::float8 as fat, SUM(fi.fiber_g)::float8 as fib
			           FROM meals JOIN food_items fi ON fi.meal_id = meals.id
			           WHERE meals.user_id = $1 GROUP BY dd) m ON m.dd = d::date
			 LEFT JOIN (SELECT date, SUM(amount_ml)::int as water FROM water_logs
			           WHERE user_id = $1 GROUP BY date) w ON w.date = d::date
			 LEFT JOIN (SELECT (timestamp AT TIME ZONE 'UTC')::date as dd, SUM(calories_burned)::int as burned
			           FROM exercises WHERE user_id = $1 GROUP BY dd) e ON e.dd = d::date
			 ORDER BY d`, userID, fromDate, toDate)
		if mErr == nil {
			defer dbRows.Close()
			for dbRows.Next() {
				var d time.Time
				var cal, water, burned int
				var prot, carb, fat, fib float64
				if dbRows.Scan(&d, &cal, &prot, &carb, &fat, &fib, &water, &burned) == nil {
					rows = append(rows, dayRow{
						Date:     d.Format("2006-01-02"),
						Calories: int32(cal),
						Protein:  prot,
						Carbs:    carb,
						Fat:      fat,
						Fiber:    fib,
						Water:    int32(water),
						Burned:   int32(burned),
					})
				}
			}
		}
		json.NewEncoder(w).Encode(rows)
	}
}
