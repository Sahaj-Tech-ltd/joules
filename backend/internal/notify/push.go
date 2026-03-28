package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgxpool"

	"joule/internal/config"
	"joule/internal/db/sqlc"
)

// Payload is the JSON sent inside each push notification.
type Payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Tag   string `json:"tag,omitempty"` // deduplicates same-type notifications
}

// Service handles sending Web Push and ntfy notifications.
type Service struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewService(q *sqlc.Queries, pool *pgxpool.Pool, cfg *config.Config) *Service {
	return &Service{q: q, pool: pool, cfg: cfg}
}

// SendToUser sends a Web Push notification to all of the user's subscriptions.
// Dead subscriptions (410 Gone) are automatically pruned.
// If the user has an ntfy topic configured, it also sends via ntfy.
func (s *Service) SendToUser(ctx context.Context, userID string, p Payload) {
	if s.cfg.VAPIDPublicKey == "" || s.cfg.VAPIDPrivateKey == "" {
		slog.Warn("VAPID keys not configured, skipping web push", "user_id", userID)
		return
	}

	subs, err := s.q.GetPushSubscriptionsByUser(ctx, userID)
	if err != nil {
		slog.Error("notify: get subscriptions failed", "user_id", userID, "error", err)
		return
	}

	if len(subs) == 0 && s.cfg.NtfyBaseURL == "" {
		return
	}

	payloadBytes, err := json.Marshal(p)
	if err != nil {
		slog.Error("notify: marshal payload", "error", err)
		return
	}

	for _, sub := range subs {
		s.sendWebPush(ctx, sub, payloadBytes)
	}

	// ntfy: check if user has a topic configured
	prefs, err := s.q.GetNotificationPrefs(ctx, userID)
	if err == nil && prefs.NtfyTopic != "" && s.cfg.NtfyBaseURL != "" {
		s.sendNtfy(prefs.NtfyTopic, p)
	}
}

func (s *Service) sendWebPush(ctx context.Context, sub sqlc.PushSubscription, payload []byte) {
	webSub := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	resp, err := webpush.SendNotification(payload, webSub, &webpush.Options{
		VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
		VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
		Subscriber:      s.cfg.VAPIDContact,
		TTL:             3600, // 1 hour
	})
	if err != nil {
		slog.Error("notify: web push send failed", "endpoint", sub.Endpoint[:min(30, len(sub.Endpoint))], "error", err)
		return
	}
	defer resp.Body.Close()

	// 410 Gone = subscription is dead, prune it
	if resp.StatusCode == http.StatusGone {
		slog.Info("notify: pruning dead subscription", "endpoint", sub.Endpoint[:min(30, len(sub.Endpoint))])
		_ = s.q.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint)
		return
	}

	if resp.StatusCode >= 400 {
		slog.Warn("notify: web push unexpected status", "status", resp.StatusCode)
	}
}

// sendNtfy sends a notification via a self-hosted ntfy server.
func (s *Service) sendNtfy(topic string, p Payload) {
	url := fmt.Sprintf("%s/%s", s.cfg.NtfyBaseURL, topic)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(p.Body))
	if err != nil {
		slog.Error("ntfy: create request failed", "error", err)
		return
	}
	req.Header.Set("Title", p.Title)
	req.Header.Set("Content-Type", "text/plain")
	if s.cfg.NtfyToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.NtfyToken)
	}
	if p.URL != "" {
		req.Header.Set("Click", p.URL)
	}
	if p.Icon != "" {
		req.Header.Set("Icon", p.Icon)
	}
	if p.Tag != "" {
		req.Header.Set("Tags", p.Tag)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("ntfy: send failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("ntfy: unexpected status", "status", resp.StatusCode, "topic", topic)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
