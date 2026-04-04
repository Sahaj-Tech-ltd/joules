package auth

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type planContextKey string

const PlanKey planContextKey = "user_plan"

func GetPlan(ctx context.Context) string {
	if plan, ok := ctx.Value(PlanKey).(string); ok {
		return plan
	}
	return "free"
}

func PlanMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, _ := r.Context().Value(ContextUserID).(string)
			if userID == "" {
				next.ServeHTTP(w, r)
				return
			}
			var plan string
			err := pool.QueryRow(r.Context(),
				`SELECT COALESCE(plan, 'free') FROM users WHERE id = $1`, userID).Scan(&plan)
			if err != nil {
				plan = "free"
			}
			ctx := context.WithValue(r.Context(), PlanKey, plan)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
