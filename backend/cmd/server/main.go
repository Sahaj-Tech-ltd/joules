package main

import (
	"context"
	"embed"
	"encoding/json"
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

	"github.com/jackc/pgx/v5/pgxpool"

	"joules/internal/achievement"
	"joules/internal/admin"
	"joules/internal/ai"
	"joules/internal/auth"
	"joules/internal/coach"
	"joules/internal/config"
	"joules/internal/dashboard"
	"joules/internal/db"
	"joules/internal/db/sqlc"
	"joules/internal/exercise"
	"joules/internal/export"
	"joules/internal/fasting"
	"joules/internal/favorites"
	"joules/internal/foodmemory"
	"joules/internal/foods"
	"joules/internal/groups"
	"joules/internal/habits"
	"joules/internal/identity"
	"joules/internal/intentions"
	"joules/internal/meal"
	"joules/internal/notify"
	"joules/internal/recipe"
	"joules/internal/steps"
	syslog "joules/internal/syslog"
	"joules/internal/user"
	"joules/internal/water"
	"joules/internal/weight"
)

//go:embed all:dist
var frontendFS embed.FS

type gateResponse struct {
	Error string `json:"error,omitempty"`
}

func writeGateJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func featureGate(pool *pgxpool.Pool, feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !admin.IsFeatureEnabled(pool, r.Context(), feature) {
				writeGateJSON(w, http.StatusNotFound, gateResponse{Error: "this feature is not available"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func maxBodySize(max int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, max)
			next.ServeHTTP(w, r)
		})
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

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
	setupToken := authSvc.EnsureAdminUser()
	userHandler := user.NewHandler(queries, pool, cfg)

	var aiPrompts map[string]string
	if promptRows, err := pool.Query(context.Background(),
		"SELECT key, value FROM app_settings WHERE key LIKE 'prompt_%'"); err == nil {
		aiPrompts = make(map[string]string)
		for promptRows.Next() {
			var k, v string
			if promptRows.Scan(&k, &v) == nil {
				aiPrompts[k] = v
			}
		}
		promptRows.Close()
	}

	aiClient := ai.NewReloadingClient(pool, ai.Config{
		Provider:          cfg.AIProvider,
		OpenAIKey:         cfg.OpenAIKey,
		OpenAIBaseURL:     cfg.OpenAIBaseURL,
		Model:             cfg.AIModel,
		VisionModel:       cfg.VisionModel,
		OCRModel:          cfg.OCRModel,
		RoutingModel:      cfg.RoutingModel,
		ClassifierModel:   cfg.ClassifierModel,
		Prompts:           aiPrompts,
		VisionAPIKey:      cfg.VisionAPIKey,
		VisionBaseURL:     cfg.VisionBaseURL,
		OCRAPIKey:         cfg.OCRAPIKey,
		OCRBaseURL:        cfg.OCRBaseURL,
		ClassifierAPIKey:  cfg.ClassifierAPIKey,
		ClassifierBaseURL: cfg.ClassifierBaseURL,
	})
	mealHandler := meal.NewHandler(queries, aiClient, cfg.UploadDir, cfg, pool)
	foodsHandler := foods.NewHandler(pool, aiClient)
	recipeHandler := recipe.NewHandler(pool)
	dashHandler := dashboard.NewHandler(queries, pool)
	weightHandler := weight.NewHandler(queries, pool)
	waterHandler := water.NewHandler(queries)
	exerciseHandler := exercise.NewHandler(queries, pool)
	coachHandler := coach.NewHandler(queries, aiClient, pool, cfg)
	if err := coachHandler.EnsureSummaryTable(context.Background()); err != nil {
		slog.Error("coach summary table init", "error", err)
		os.Exit(1)
	}
	achievementHandler := achievement.NewHandler(queries)
	exportHandler := export.NewHandler(queries, pool)
	fastingHandler := fasting.NewHandler(queries)
	srv := &http.Server{}
	adminHandler := admin.NewHandler(pool, cfg.RequireApproval, cfg, aiClient, srv)
	notifySvc := notify.NewService(queries, pool, cfg)
	notifyHandler := notify.NewHandler(queries, pool, notifySvc, cfg)
	stepsHandler := steps.NewHandler(queries, pool, cfg)
	groupsHandler := groups.NewHandler(queries, pool)
	habitsHandler := habits.NewHandler(queries, pool)
	favoritesHandler := favorites.NewHandler(queries, pool)
	identityHandler := identity.NewHandler(queries, pool, aiClient)
	intentionsHandler := intentions.NewHandler(queries, pool)
	foodMemoryHandler := foodmemory.NewHandler(queries, pool)

	// Start notification scheduler in background
	schedCtx, schedCancel := context.WithCancel(context.Background())
	defer schedCancel()
	go notifySvc.StartScheduler(schedCtx)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)
	r.Use(chicors.Handler(chicors.Options{
		AllowedOrigins:   []string{cfg.AppURL, "http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Route("/api", func(r chi.Router) {
		r.Use(maxBodySize(10 << 20))
		// Public banner endpoint — no auth required
		r.Get("/banners", adminHandler.GetBanners)
		r.Get("/features", adminHandler.GetPublicFeatures)

		r.Route("/foods", func(r chi.Router) {
			r.Get("/search", foodsHandler.Search)
			r.Get("/barcode/{upc}", foodsHandler.GetByBarcode)
			r.Group(func(r chi.Router) {
				r.Use(auth.JWTMiddleware(cfg.JWTSecret))
				r.Post("/barcode-scan", foodsHandler.BarcodeScan)
			})
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", authHandler.Signup)
			r.Post("/verify", authHandler.Verify)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/setup-complete", authHandler.SetupComplete)

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
			r.Get("/coach-notes", userHandler.GetCoachNotes)
			r.Put("/coach-notes", userHandler.UpdateCoachNotes)
			r.Get("/coach-memories", userHandler.GetCoachMemories)
			r.Delete("/coach-memories/{id}", userHandler.DeleteCoachMemory)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Use(auth.AdminMiddleware(pool))
			r.Get("/users", adminHandler.GetUsers)
			r.Get("/users/{id}/view", adminHandler.GetUserView)
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
			r.Get("/foods/stats", adminHandler.GetFoodsStats)
			r.Get("/prompts", adminHandler.GetPrompts)
			r.Put("/prompts", adminHandler.UpdatePrompts)
			r.Get("/features", adminHandler.GetFeatures)
			r.Put("/features", adminHandler.UpdateFeatures)
			r.Get("/coach-config", adminHandler.GetCoachConfig)
			r.Put("/coach-config", adminHandler.UpdateCoachConfig)
			r.Get("/tdee-config", adminHandler.GetTDEEConfig)
			r.Put("/tdee-config", adminHandler.UpdateTDEEConfig)
			r.Post("/models", adminHandler.GetModels)
			r.Post("/ai/test", adminHandler.TestAI)
			r.Get("/healthcheck", adminHandler.GetHealthcheck)
			r.Post("/foods/import", adminHandler.ImportFoods)
			r.Get("/nutrition-cache", adminHandler.GetNutritionCache)
			r.Delete("/nutrition-cache/{id}", adminHandler.DeleteNutritionCacheEntry)
			r.Post("/nutrition-cache/clear", adminHandler.ClearNutritionCache)
		})

		r.Route("/meals", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Post("/identify", mealHandler.IdentifyFood)
			r.Post("/", mealHandler.CreateMeal)
			r.Get("/", mealHandler.GetMealsByDate)
			r.Get("/recent", mealHandler.GetRecentMeals)
			r.Post("/carry-forward", mealHandler.CarryForward)
			r.Post("/from-recipe/{recipeId}", mealHandler.LogMealFromRecipe)
			r.Delete("/{id}", mealHandler.DeleteMeal)
			r.Put("/{mealId}/foods/{foodId}", mealHandler.UpdateFoodItem)
			r.Delete("/{mealId}/foods/{foodId}", mealHandler.DeleteFoodItemHandler)
		})

		r.Route("/recipes", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Use(featureGate(pool, "recipes"))
			r.Get("/", recipeHandler.List)
			r.Post("/", recipeHandler.Create)
			r.Delete("/{id}", recipeHandler.Delete)
		})

		r.Route("/dashboard", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/summary", dashHandler.GetSummary)
			r.Post("/cheat-day", dashHandler.MarkCheatDay)
			r.Delete("/cheat-day", dashHandler.UnmarkCheatDay)
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
			r.Use(featureGate(pool, "coach"))
			r.Get("/tips", coachHandler.GetTips)
			r.Get("/chat", coachHandler.GetChatHistory)
			r.Post("/chat", coachHandler.SendMessage)
			r.Get("/reminders", coachHandler.GetRemindersAPI)
			r.Put("/reminders/{id}", coachHandler.ToggleReminderAPI)
			r.Delete("/reminders/{id}", coachHandler.DeleteReminderAPI)
		})

		r.Route("/fasting", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Use(featureGate(pool, "fasting"))
			r.Get("/status", fastingHandler.GetStatus)
			r.Post("/start", fastingHandler.StartFast)
			r.Post("/break", fastingHandler.BreakFast)
			r.Put("/window", fastingHandler.UpdateWindow)
		})

		r.Route("/achievements", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Use(featureGate(pool, "achievements"))
			r.Get("/", achievementHandler.GetAchievements)
			r.Post("/check", achievementHandler.CheckAchievements)
		})

		r.Route("/favorites", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/", favoritesHandler.GetFavorites)
			r.Get("/top", favoritesHandler.GetTopFavorites)
			r.Post("/", favoritesHandler.AddFavorite)
			r.Delete("/{id}", favoritesHandler.RemoveFavorite)
			r.Post("/{id}/use", favoritesHandler.LogFromFavorite)
		})

		r.Route("/export", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Use(featureGate(pool, "export"))
			r.Get("/csv", exportHandler.ExportCSV)
			r.Get("/json", exportHandler.ExportJSON)
		})

		r.Route("/steps", func(r chi.Router) {
			// Google OAuth callback — no JWT (redirect from Google)
			r.Get("/google/callback", stepsHandler.GoogleCallback)

			r.Group(func(r chi.Router) {
				r.Use(auth.JWTMiddleware(cfg.JWTSecret))
				r.Use(featureGate(pool, "steps"))
				r.Get("/", stepsHandler.GetSteps)
				r.Post("/", stepsHandler.LogSteps)
				r.Get("/history", stepsHandler.GetStepsHistory)
				r.Get("/google/status", stepsHandler.GoogleStatus)
				r.Get("/google/connect", stepsHandler.GoogleConnect)
				r.Post("/google/sync", stepsHandler.GoogleSync)
			})
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
				r.Post("/register-expo", notifyHandler.RegisterExpoPush)
			})
		})

		r.Route("/groups", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Use(featureGate(pool, "groups"))
			r.Get("/", groupsHandler.ListMyGroups)
			r.Post("/", groupsHandler.CreateGroup)
			r.Get("/discover", groupsHandler.DiscoverGroups)
			r.Post("/join", groupsHandler.JoinGroup)
			r.Get("/{id}", groupsHandler.GetGroup)
			r.Post("/{id}/leave", groupsHandler.LeaveGroup)
			r.Delete("/{id}", groupsHandler.DeleteGroup)
			r.Get("/{id}/leaderboard", groupsHandler.GetLeaderboard)
			r.Get("/{id}/challenges", groupsHandler.ListChallenges)
			r.Post("/{id}/challenges", groupsHandler.CreateChallenge)
		})

		r.Route("/habits", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/summary", habitsHandler.GetSummary)
			r.Get("/phase", habitsHandler.GetPhase)
			r.Post("/checkin", habitsHandler.Checkin)
		})

		r.Route("/identity", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/quote", identityHandler.GetQuote)
		})

		r.Route("/intentions", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/", intentionsHandler.List)
			r.Post("/", intentionsHandler.Create)
			r.Put("/{id}", intentionsHandler.Update)
			r.Delete("/{id}", intentionsHandler.Delete)
		})

		r.Route("/food-memory", func(r chi.Router) {
			r.Use(auth.JWTMiddleware(cfg.JWTSecret))
			r.Get("/", foodMemoryHandler.List)
		})
	})

	r.Handle("/uploads/*", addUploadSecurityHeaders(http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir)))))

	serveFrontend(r)

	addr := fmt.Sprintf(":%s", cfg.Port)
	printStartupBanner(cfg.AppURL, cfg.AdminEmail, setupToken)
	slog.Info("starting server", "addr", addr)
	srv.Addr = addr
	srv.Handler = r
	srv.ReadTimeout = 30 * time.Second
	srv.WriteTimeout = 90 * time.Second
	srv.IdleTimeout = 120 * time.Second
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func printStartupBanner(appURL, adminEmail, setupToken string) {
	url := appURL
	if url == "" {
		url = "http://localhost:8687"
	}
	const width = 52
	line := strings.Repeat("─", width)
	pad := func(s string) string {
		r := []rune(s)
		if len(r) >= width {
			return s
		}
		p := width - len(r)
		return strings.Repeat(" ", p/2) + s + strings.Repeat(" ", p-p/2)
	}
	fmt.Printf("\n  ╭%s╮\n", line)
	fmt.Printf("  │%s│\n", pad(""))
	fmt.Printf("  │%s│\n", pad("✦  Joules is ready  ✦"))
	fmt.Printf("  │%s│\n", pad(""))
	fmt.Printf("  │%s│\n", pad(url))
	if setupToken != "" {
		fmt.Printf("  │%s│\n", pad(""))
		fmt.Printf("  │%s│\n", pad("─── First-run setup ───"))
		fmt.Printf("  │%s│\n", pad("Admin: "+adminEmail))
		fmt.Printf("  │%s│\n", pad(""))
		fmt.Printf("  │%s│\n", pad("Open this URL to set your password:"))
		setupURL := url + "/setup/" + setupToken
		if len([]rune(setupURL)) < width {
			fmt.Printf("  │%s│\n", pad(setupURL))
		} else {
			// URL too long to center — just left-align with indent
			fmt.Printf("  │  %-*s│\n", width-2, setupURL)
		}
	}
	fmt.Printf("  │%s│\n", pad(""))
	fmt.Printf("  ╰%s╯\n\n", line)
}

func addUploadSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src * data:; style-src 'unsafe-inline'")
		next.ServeHTTP(w, r)
	})
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
