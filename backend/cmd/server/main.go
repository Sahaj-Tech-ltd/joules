package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	chicors "github.com/go-chi/cors"

	"joule/internal/achievement"
	"joule/internal/admin"
	"joule/internal/ai"
	"joule/internal/auth"
	"joule/internal/coach"
	"joule/internal/config"
	"joule/internal/dashboard"
	"joule/internal/db"
	"joule/internal/db/sqlc"
	"joule/internal/exercise"
	"joule/internal/export"
	"joule/internal/meal"
	syslog "joule/internal/syslog"
	"joule/internal/user"
	"joule/internal/notify"
	"joule/internal/water"
	"joule/internal/weight"
)

//go:embed all:dist
var frontendFS embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := sqlc.New(pool)
	syslog.Init(pool)
	authSvc := auth.NewService(queries, pool, cfg)
	authHandler := auth.NewHandler(authSvc)
	authSvc.EnsureAdminUser()
	userHandler := user.NewHandler(queries, pool, cfg)

	aiClient, _ := ai.NewClient(ai.Config{
		Provider:      cfg.AIProvider,
		OpenAIKey:     cfg.OpenAIKey,
		OpenAIBaseURL: cfg.OpenAIBaseURL,
		AnthropicKey:  cfg.AnthropicKey,
		Model:         cfg.AIModel,
	})
	mealHandler := meal.NewHandler(queries, aiClient, cfg.UploadDir, cfg)
	dashHandler := dashboard.NewHandler(queries, pool)
	weightHandler := weight.NewHandler(queries)
	waterHandler := water.NewHandler(queries)
	exerciseHandler := exercise.NewHandler(queries)
	coachHandler := coach.NewHandler(queries, aiClient, pool, cfg)
	achievementHandler := achievement.NewHandler(queries)
	exportHandler := export.NewHandler(queries)
	adminHandler := admin.NewHandler(pool, cfg.RequireApproval, cfg)
	notifySvc := notify.NewService(queries, pool, cfg)
	notifyHandler := notify.NewHandler(queries, notifySvc, cfg)

	// Start notification scheduler in background
	schedCtx, schedCancel := context.WithCancel(context.Background())
	_ = schedCancel // cancelled on server shutdown via defer
	go notifySvc.StartScheduler(schedCtx)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(chicors.Handler(chicors.Options{
		AllowedOrigins:   []string{cfg.AppURL, "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Route("/api", func(r chi.Router) {
		// Public banner endpoint — no auth required
		r.Get("/banners", adminHandler.GetBanners)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", authHandler.Signup)
			r.Post("/verify", authHandler.Verify)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)

			r.Group(func(r chi.Router) {
				r.Use(auth.JWTMiddleware(cfg.JWTSecret))
				r.Get("/me", authHandler.Me)
				r.Put("/password", authHandler.ChangePassword)
			})
		})

		r.Route("/user", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Post("/onboarding", userHandler.CompleteOnboarding)
			r.Get("/profile", userHandler.GetProfile)
			r.Put("/profile", userHandler.UpdateProfile)
			r.Get("/goals", userHandler.GetGoals)
			r.Put("/goals", userHandler.UpdateGoals)
			r.Get("/preferences", userHandler.GetPreferences)
			r.Put("/preferences", userHandler.UpdatePreferences)
			r.Post("/avatar", userHandler.UploadAvatar)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Use(auth.AdminMiddleware(pool))
			r.Get("/users", adminHandler.GetUsers)
			r.Post("/users/{id}/approve", adminHandler.ApproveUser)
			r.Post("/users/{id}/unapprove", adminHandler.UnapproveUser)
			r.Post("/users/{id}/make-admin", adminHandler.MakeAdmin)
			r.Post("/users/{id}/remove-admin", adminHandler.RemoveAdmin)
			r.Delete("/users/{id}", adminHandler.DeleteUser)
			r.Get("/settings", adminHandler.GetSettings)
			r.Put("/settings", adminHandler.UpdateSettings)
			r.Get("/info", adminHandler.GetInfo)
			r.Post("/restart", adminHandler.RestartServer)
			r.Post("/users/{id}/verify", adminHandler.VerifyUser)
			r.Get("/banners", adminHandler.GetBanners)
			r.Post("/banners", adminHandler.CreateBanner)
			r.Delete("/banners/{id}", adminHandler.DeleteBanner)
			r.Get("/logs", adminHandler.GetLogs)
		})

		r.Route("/meals", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Post("/", mealHandler.CreateMeal)
			r.Get("/", mealHandler.GetMealsByDate)
			r.Get("/recent", mealHandler.GetRecentMeals)
			r.Post("/carry-forward", mealHandler.CarryForward)
			r.Delete("/{id}", mealHandler.DeleteMeal)
			r.Put("/{mealId}/foods/{foodId}", mealHandler.UpdateFoodItem)
		})

		r.Route("/dashboard", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/summary", dashHandler.GetSummary)
		})

		r.Route("/weight", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Post("/", weightHandler.LogWeight)
			r.Get("/", weightHandler.GetWeightHistory)
		})

		r.Route("/water", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Post("/", waterHandler.LogWater)
			r.Get("/", waterHandler.GetWaterByDate)
		})

		r.Route("/exercises", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Post("/", exerciseHandler.LogExercise)
			r.Get("/", exerciseHandler.GetExercisesByDate)
		})

		r.Route("/coach", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/tips", coachHandler.GetTips)
			r.Get("/chat", coachHandler.GetChatHistory)
			r.Post("/chat", coachHandler.SendMessage)
		})

		r.Route("/achievements", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/", achievementHandler.GetAchievements)
			r.Post("/check", achievementHandler.CheckAchievements)
		})

		r.Route("/export", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/csv", exportHandler.ExportCSV)
		})

		r.Route("/notifications", func(r chi.Router) {
			// VAPID public key is public (needed before subscription)
			r.Get("/vapid-public-key", notifyHandler.GetVAPIDPublicKey)
			// All other routes require auth
			r.Group(func(r chi.Router) {
				r.Use(auth.JWTMiddleware(cfg.JWTSecret))
				r.Post("/subscribe", notifyHandler.Subscribe)
				r.Post("/unsubscribe", notifyHandler.Unsubscribe)
				r.Get("/preferences", notifyHandler.GetPreferences)
				r.Put("/preferences", notifyHandler.SavePreferences)
				r.Post("/test", notifyHandler.SendTest)
			})
		})
	})

	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	serveFrontend(r)

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("starting server", "addr", addr)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second, // generous for AI endpoints
		IdleTimeout:  120 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func serveFrontend(r chi.Router) {
	distFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		slog.Warn("no embedded frontend found, running API-only mode")
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		path := strings.TrimPrefix(req.URL.Path, "/")

		if path != "" {
			if _, err := distFS.Open(path); err == nil {
				fileServer.ServeHTTP(w, req)
				return
			}
		}

		req.URL.Path = "/"
		fileServer.ServeHTTP(w, req)
	})
}
