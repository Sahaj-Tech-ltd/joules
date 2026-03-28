// Code generated manually — matches sqlc output conventions.

package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type NutritionCache struct {
	ID          string
	Query       string
	Name        string
	Calories    int32
	ProteinG    pgtype.Numeric
	CarbsG      pgtype.Numeric
	FatG        pgtype.Numeric
	FiberG      pgtype.Numeric
	ServingSize string
	Source      string
	CreatedAt   pgtype.Timestamptz
}

type UpsertNutritionCacheParams struct {
	Query       string
	Name        string
	Calories    int32
	ProteinG    pgtype.Numeric
	CarbsG      pgtype.Numeric
	FatG        pgtype.Numeric
	FiberG      pgtype.Numeric
	ServingSize string
	Source      string
}

const getNutritionCache = `SELECT id, query, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source, created_at
FROM nutrition_cache WHERE lower(query) = lower($1) LIMIT 1`

func (q *Queries) GetNutritionCache(ctx context.Context, query string) (NutritionCache, error) {
	row := q.db.QueryRow(ctx, getNutritionCache, query)
	var n NutritionCache
	err := row.Scan(
		&n.ID, &n.Query, &n.Name, &n.Calories,
		&n.ProteinG, &n.CarbsG, &n.FatG, &n.FiberG,
		&n.ServingSize, &n.Source, &n.CreatedAt,
	)
	return n, err
}

const upsertNutritionCache = `INSERT INTO nutrition_cache (query, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (lower(query)) DO UPDATE SET
    name = EXCLUDED.name, calories = EXCLUDED.calories,
    protein_g = EXCLUDED.protein_g, carbs_g = EXCLUDED.carbs_g,
    fat_g = EXCLUDED.fat_g, fiber_g = EXCLUDED.fiber_g,
    serving_size = EXCLUDED.serving_size, source = EXCLUDED.source,
    created_at = NOW()
RETURNING id, query, name, calories, protein_g, carbs_g, fat_g, fiber_g, serving_size, source, created_at`

func (q *Queries) UpsertNutritionCache(ctx context.Context, p UpsertNutritionCacheParams) (NutritionCache, error) {
	row := q.db.QueryRow(ctx, upsertNutritionCache,
		p.Query, p.Name, p.Calories,
		p.ProteinG, p.CarbsG, p.FatG, p.FiberG,
		p.ServingSize, p.Source,
	)
	var n NutritionCache
	err := row.Scan(
		&n.ID, &n.Query, &n.Name, &n.Calories,
		&n.ProteinG, &n.CarbsG, &n.FatG, &n.FiberG,
		&n.ServingSize, &n.Source, &n.CreatedAt,
	)
	return n, err
}
