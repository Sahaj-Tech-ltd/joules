package coach

// context_manager.go
//
// LCM-inspired context management for Joule's AI coach.
// Drop this file into backend/internal/coach/ alongside handler.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"joules/internal/admin"
	"joules/internal/ai"
)

const (
	defaultRawWindowSize = 20
	summaryTargetChars   = 3200
)

// ── DB types ──────────────────────────────────────────────────────────────────

type coachSummary struct {
	ID           string
	UserID       string
	SummaryText  string
	MessageCount int
	CoveredFrom  time.Time
	CoveredTo    time.Time
	CreatedAt    time.Time
}

type rawCoachMessage struct {
	ID        string
	Role      string
	Content   string
	CreatedAt time.Time
}

// ── Schema init ───────────────────────────────────────────────────────────────

// EnsureSummaryTable creates the coach_summaries table and adds the
// covered_by_summary_id column to coach_messages if they don't exist.
// Call once at startup in main.go.
func (h *Handler) EnsureSummaryTable(ctx context.Context) error {
	_, err := h.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS coach_summaries (
			id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			summary_text  TEXT NOT NULL,
			message_count INT  NOT NULL DEFAULT 0,
			covered_from  TIMESTAMPTZ NOT NULL,
			covered_to    TIMESTAMPTZ NOT NULL,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_coach_summaries_user_id
			ON coach_summaries(user_id, covered_from);

		ALTER TABLE coach_messages
			ADD COLUMN IF NOT EXISTS covered_by_summary_id UUID
			REFERENCES coach_summaries(id) ON DELETE SET NULL;
		CREATE INDEX IF NOT EXISTS idx_coach_messages_covered
			ON coach_messages(user_id, covered_by_summary_id);
	`)
	return err
}

// ── DB queries ────────────────────────────────────────────────────────────────

func (h *Handler) loadSummaries(ctx context.Context, userID string) ([]coachSummary, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, user_id, summary_text, message_count, covered_from, covered_to, created_at
		FROM coach_summaries
		WHERE user_id = $1
		ORDER BY covered_from ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []coachSummary
	for rows.Next() {
		var s coachSummary
		if err := rows.Scan(&s.ID, &s.UserID, &s.SummaryText, &s.MessageCount,
			&s.CoveredFrom, &s.CoveredTo, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (h *Handler) loadUncoveredMessages(ctx context.Context, userID string) ([]rawCoachMessage, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, role, content, created_at
		FROM coach_messages
		WHERE user_id = $1
		  AND covered_by_summary_id IS NULL
		ORDER BY created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rawCoachMessage
	for rows.Next() {
		var m rawCoachMessage
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// ── Three-level summarization ─────────────────────────────────────────────────

// summarizeMessages compresses a batch of messages into a summary string.
// Escalates through 3 levels to guarantee output is smaller than input.
//
//	Level 1 — preserve detail (full prose summary)
//	Level 2 — aggressive bullet points, half token target
//	Level 3 — deterministic truncation, no LLM involved
func (h *Handler) summarizeMessages(ctx context.Context, msgs []rawCoachMessage) (string, int) {
	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(strings.ToUpper(m.Role))
		sb.WriteString(": ")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	transcript := sb.String()
	originalLen := len(transcript)

	// Level 1 — detailed prose
	l1Template := admin.GetSettingDefault(h.pool, ctx, "prompt_compact_l1", admin.DefaultPrompts["prompt_compact_l1"])
	level1Prompt := fmt.Sprintf(l1Template, summaryTargetChars, transcript)

	if summary, err := h.ai.Chat(level1Prompt, nil); err == nil && len(summary) < originalLen {
		slog.Info("compaction: level 1 success",
			"original_chars", originalLen, "summary_chars", len(summary))
		return summary, 1
	}

	// Level 2 — aggressive bullets
	l2Template := admin.GetSettingDefault(h.pool, ctx, "prompt_compact_l2", admin.DefaultPrompts["prompt_compact_l2"])
	level2Prompt := fmt.Sprintf(l2Template, summaryTargetChars/2, transcript)

	if summary, err := h.ai.Chat(level2Prompt, nil); err == nil && len(summary) < originalLen {
		slog.Info("compaction: level 2 success",
			"original_chars", originalLen, "summary_chars", len(summary))
		return summary, 2
	}

	// Level 3 — deterministic fallback
	slog.Warn("compaction: falling back to level 3 (deterministic truncation)")
	cap := summaryTargetChars
	if len(transcript) < cap {
		cap = len(transcript)
	}
	fallback := "[Earlier conversation — truncated for context management]\n" + transcript[:cap]
	return fallback, 3
}

// ── Compaction ────────────────────────────────────────────────────────────────

// compactIfNeeded summarizes the oldest uncovered messages when the total
// exceeds rawWindowSize. Safe to call on every turn — no-op when not needed.
func (h *Handler) compactIfNeeded(ctx context.Context, userID string) error {
	coachCfg := admin.GetCoachConfig(h.pool, ctx)
	rawWindow := coachCfg.ContextWindowSize

	uncovered, err := h.loadUncoveredMessages(ctx, userID)
	if err != nil {
		return fmt.Errorf("load uncovered messages: %w", err)
	}

	if len(uncovered) <= rawWindow {
		return nil
	}

	toSummarize := uncovered[:len(uncovered)-rawWindow]
	slog.Info("coach: compacting", "user_id", userID, "count", len(toSummarize))

	summaryText, level := h.summarizeMessages(ctx, toSummarize)

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var summaryID string
	err = tx.QueryRow(ctx, `
		INSERT INTO coach_summaries
			(user_id, summary_text, message_count, covered_from, covered_to)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`,
		userID,
		summaryText,
		len(toSummarize),
		toSummarize[0].CreatedAt,
		toSummarize[len(toSummarize)-1].CreatedAt,
	).Scan(&summaryID)
	if err != nil {
		return fmt.Errorf("insert summary: %w", err)
	}

	ids := make([]string, len(toSummarize))
	for i, m := range toSummarize {
		ids[i] = m.ID
	}
	_, err = tx.Exec(ctx, `
		UPDATE coach_messages
		SET covered_by_summary_id = $1
		WHERE id = ANY($2::uuid[])
	`, summaryID, ids)
	if err != nil {
		return fmt.Errorf("mark messages covered: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit compaction transaction: %w", err)
	}

	slog.Info("coach: compaction complete",
		"user_id", userID,
		"summary_id", summaryID,
		"level", level,
		"messages_covered", len(toSummarize),
		"summary_chars", len(summaryText),
	)
	return nil
}

// ── Active context builder ────────────────────────────────────────────────────

// buildActiveContext assembles the message slice sent to the model each turn.
// It replaces the raw history slice in SendMessage.
//
// Output structure:
//
//	[user: all summaries combined]      ← only if summaries exist
//	[assistant: "Got it..."]            ← only if summaries exist
//	[raw message 1]                     ← uncovered messages, oldest first
//	...
//	[raw message N]                     ← most recent
func (h *Handler) buildActiveContext(ctx context.Context, userID string) ([]ai.ChatMessage, error) {
	if err := h.compactIfNeeded(ctx, userID); err != nil {
		// Non-fatal — log and keep going with raw history
		slog.Warn("coach: compaction failed, continuing without it", "error", err)
	}

	summaries, err := h.loadSummaries(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load summaries: %w", err)
	}

	uncovered, err := h.loadUncoveredMessages(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load uncovered messages: %w", err)
	}

	var messages []ai.ChatMessage

	if len(summaries) > 0 {
		var sb strings.Builder
		sb.WriteString("[COACHING HISTORY — summarized]\n\n")
		for i, s := range summaries {
			sb.WriteString(fmt.Sprintf(
				"--- Summary %d (%s → %s, %d messages) ---\n%s\n\n",
				i+1,
				s.CoveredFrom.Format("Jan 2"),
				s.CoveredTo.Format("Jan 2"),
				s.MessageCount,
				s.SummaryText,
			))
		}
		messages = append(messages,
			ai.ChatMessage{Role: "user", Content: sb.String()},
			ai.ChatMessage{Role: "assistant", Content: "Got it — I have your coaching history in mind."},
		)
	}

	for _, m := range uncovered {
		messages = append(messages, ai.ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	slog.Info("coach: context built",
		"summaries", len(summaries),
		"raw_messages", len(uncovered),
	)
	return messages, nil
}

// ── History search ────────────────────────────────────────────────────────────

// GrepHistory searches ALL messages for a user — including summarized ones.
// lcm_grep equivalent. Add a "search_my_history" agent tool later if needed.
func (h *Handler) GrepHistory(ctx context.Context, userID, pattern string) ([]rawCoachMessage, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, role, content, created_at
		FROM coach_messages
		WHERE user_id = $1
		  AND content ILIKE '%' || $2 || '%'
		ORDER BY created_at ASC
		LIMIT 20
	`, userID, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rawCoachMessage
	for rows.Next() {
		var m rawCoachMessage
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// ── search_my_history tool (for agentTools + executeTool) ─────────────────────

func searchHistoryTool() ai.Tool {
	return ai.Tool{
		Name:        "search_my_history",
		Description: "Search the user's full coaching history for a keyword or topic, including older conversations that have been summarized. Use this when the user references something from earlier conversations or you need to recall past advice.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Keyword or phrase to search for in the user's history",
				},
			},
			"required": []string{"query"},
		},
	}
}

func (h *Handler) executeSearchHistory(ctx context.Context, userID string, argsJSON string) string {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("error: invalid arguments: %v", err)
	}
	if args.Query == "" {
		return "error: query is required"
	}
	msgs, err := h.GrepHistory(ctx, userID, args.Query)
	if err != nil || len(msgs) == 0 {
		return "No relevant history found."
	}
	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			m.CreatedAt.Format("Jan 2"), m.Role, m.Content))
	}
	return sb.String()
}
