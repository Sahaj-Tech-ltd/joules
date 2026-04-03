package coach

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MemoryEntry struct {
	ID        string
	UserID    string
	Category  string
	Content   string
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func SaveMemory(ctx context.Context, pool *pgxpool.Pool, userID, category, content, source string) error {
	var count int
	err := pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM coach_memory WHERE user_id = $1 AND category = $2 AND content = $3",
		userID, category, content,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = pool.Exec(ctx,
		"INSERT INTO coach_memory (user_id, category, content, source) VALUES ($1, $2, $3, $4)",
		userID, category, content, source,
	)
	return err
}

func SearchMemory(ctx context.Context, pool *pgxpool.Pool, userID, query string) ([]MemoryEntry, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, user_id, category, content, source, created_at, updated_at
		 FROM coach_memory
		 WHERE user_id = $1 AND content ILIKE '%' || $2 || '%'
		 ORDER BY category, created_at DESC`,
		userID, query,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var m MemoryEntry
		if err := rows.Scan(&m.ID, &m.UserID, &m.Category, &m.Content, &m.Source, &m.CreatedAt, &m.UpdatedAt); err != nil {
			continue
		}
		entries = append(entries, m)
	}
	return entries, nil
}

func LoadAllMemories(ctx context.Context, pool *pgxpool.Pool, userID string) ([]MemoryEntry, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, user_id, category, content, source, created_at, updated_at
		 FROM coach_memory
		 WHERE user_id = $1
		 ORDER BY category, created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var m MemoryEntry
		if err := rows.Scan(&m.ID, &m.UserID, &m.Category, &m.Content, &m.Source, &m.CreatedAt, &m.UpdatedAt); err != nil {
			continue
		}
		entries = append(entries, m)
	}
	return entries, nil
}

func DeleteMemory(ctx context.Context, pool *pgxpool.Pool, userID, memoryID string) error {
	_, err := pool.Exec(ctx,
		"DELETE FROM coach_memory WHERE id = $1 AND user_id = $2",
		memoryID, userID,
	)
	return err
}

func PruneMemories(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	var count int
	err := pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM coach_memory WHERE user_id = $1", userID,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count <= 50 {
		return nil
	}
	toDelete := count - 50
	_, err = pool.Exec(ctx,
		`DELETE FROM coach_memory WHERE user_id = $1 AND id IN (
			SELECT id FROM coach_memory
			WHERE user_id = $1 AND category NOT IN ('allergy', 'health_condition', 'goal')
			ORDER BY created_at ASC
			LIMIT $2
		)`,
		userID, toDelete,
	)
	return err
}
