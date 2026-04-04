package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/ai"
	"joules/internal/notify"
)

type JobFn func(ctx context.Context, pool *pgxpool.Pool, aiClient ai.Client, notifySvc *notify.Service)

type job struct {
	name     string
	schedule string
	fn       JobFn
}

type Scheduler struct {
	jobs   []job
	pool   *pgxpool.Pool
	ai     ai.Client
	notify *notify.Service
	mu     sync.Mutex
	cancel context.CancelFunc
}

func New(pool *pgxpool.Pool, aiClient ai.Client, notifySvc *notify.Service) *Scheduler {
	return &Scheduler{
		pool:   pool,
		ai:     aiClient,
		notify: notifySvc,
	}
}

func (s *Scheduler) AddJob(schedule string, fn JobFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job{
		name:     time.Now().Format(time.RFC3339),
		schedule: schedule,
		fn:       fn,
	})
}

func parseInterval(schedule string) time.Duration {
	switch schedule {
	case "@daily":
		return 24 * time.Hour
	case "@hourly":
		return 1 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func initialDelay(schedule string) time.Duration {
	now := time.Now().UTC()
	switch schedule {
	case "@daily":
		next := now.Truncate(24 * time.Hour).Add(25 * time.Hour)
		if d := next.Sub(now); d > 0 {
			return d
		}
		return 1 * time.Hour
	default:
		return 2 * time.Minute
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	jobs := make([]job, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.Unlock()

	schedCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	for _, j := range jobs {
		go s.runJob(schedCtx, j)
	}
	slog.Info("scheduler: started", "jobs", len(jobs))
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Scheduler) runJob(ctx context.Context, j job) {
	delay := initialDelay(j.schedule)
	interval := parseInterval(j.schedule)

	slog.Info("scheduler: scheduling job", "schedule", j.schedule, "initial_delay", delay, "interval", interval)

	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			slog.Info("scheduler: running job", "schedule", j.schedule)
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("scheduler: job panicked", "schedule", j.schedule, "panic", r)
					}
				}()
				j.fn(ctx, s.pool, s.ai, s.notify)
			}()
			timer.Reset(interval)
		case <-ctx.Done():
			slog.Info("scheduler: job stopped", "schedule", j.schedule)
			return
		}
	}
}
