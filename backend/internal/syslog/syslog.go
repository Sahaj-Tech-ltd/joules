package syslog

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool
var logCh chan logEntry

type logEntry struct {
	level    string
	category string
	message  string
	details  []byte
}

func Init(p *pgxpool.Pool) {
	pool = p
	logCh = make(chan logEntry, 1000)
	go drainLoop()
}

func drainLoop() {
	for entry := range logCh {
		if pool == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool.Exec(ctx,
			"INSERT INTO system_logs (level, category, message, details) VALUES ($1, $2, $3, $4)",
			entry.level, entry.category, entry.message, entry.details,
		)
		cancel()
	}
}

func Log(level, category, message string, details map[string]any) {
	if logCh == nil {
		return
	}
	var detailsJSON []byte
	if len(details) > 0 {
		detailsJSON, _ = json.Marshal(details)
	}
	select {
	case logCh <- logEntry{level: level, category: category, message: message, details: detailsJSON}:
	default:
	}
}

func Info(category, message string, details map[string]any)  { Log("info", category, message, details) }
func Warn(category, message string, details map[string]any)  { Log("warn", category, message, details) }
func Error(category, message string, details map[string]any) { Log("error", category, message, details) }
