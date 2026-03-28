package syslog

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

// Init registers the DB pool used for logging. Call once at startup.
func Init(p *pgxpool.Pool) {
	pool = p
}

// Log writes an event to system_logs asynchronously (fire-and-forget).
func Log(level, category, message string, details map[string]any) {
	if pool == nil {
		return
	}
	var detailsJSON []byte
	if len(details) > 0 {
		detailsJSON, _ = json.Marshal(details)
	}
	go func() {
		pool.Exec(context.Background(),
			"INSERT INTO system_logs (level, category, message, details) VALUES ($1, $2, $3, $4)",
			level, category, message, detailsJSON,
		)
	}()
}

func Info(category, message string, details map[string]any)  { Log("info", category, message, details) }
func Warn(category, message string, details map[string]any)  { Log("warn", category, message, details) }
func Error(category, message string, details map[string]any) { Log("error", category, message, details) }
