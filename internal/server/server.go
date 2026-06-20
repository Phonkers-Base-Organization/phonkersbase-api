package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/PhonkersBase/base-api2/internal/config"
	"github.com/PhonkersBase/base-api2/internal/handlers"
	"github.com/PhonkersBase/base-api2/internal/middlewares"
	"github.com/PhonkersBase/base-api2/internal/migrations"
	"github.com/PhonkersBase/base-api2/internal/repository"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	initCtx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	dbpool, err := initDB(initCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer dbpool.Close()

	if err := runMigrations(initCtx, dbpool); err != nil {
		return err
	}
	log.Info().Msg("database migrations applied")

	h := handlers.NewHandler(
		repository.NewArtistRepo(dbpool),
		repository.NewLabelRepo(dbpool),
		repository.NewEvidenceSourceRepo(dbpool),
		repository.NewUserRepo(dbpool),
		repository.NewSuggestionRepo(dbpool),
		repository.NewFeedbackRepo(dbpool),
		cfg.JWTSecret,
	)

	router := gin.New()
	router.Use(gin.Recovery())

	// No reverse proxy sits in front of this service directly, so don't trust
	// X-Forwarded-For for ClientIP() (used for rate limiting) — trusting it by
	// default would let clients spoof their IP and bypass rate limits.
	if err := router.SetTrustedProxies(nil); err != nil {
		return err
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	router.Use(logger("/", "/health", "/metrics"))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	v1 := router.Group("/api/v1")
	v1.Use(middlewares.RateLimitRequests())

	// Public routes
	v1.GET("/artist/all", h.GetArtists)
	v1.GET("/label/all", h.GetLabels)
	v1.GET("/source/all", h.GetSources)
	v1.POST("/auth/login", h.Login)
	v1.POST("/suggestion", h.CreateSuggestion)
	v1.POST("/feedback", h.CreateFeedback)

	// Protected routes (any valid token)
	protected := v1.Group("")
	protected.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		protected.POST("/auth/logout", h.Logout)
		protected.GET("/auth/me", h.GetMe)

		protected.GET("/artist/admin/all", h.GetAdminArtists)
		protected.POST("/artist", h.CreateArtist)
		protected.PUT("/artist/:id", h.UpdateArtist)
		protected.DELETE("/artist/:id", h.DeleteArtist)
		protected.GET("/artist/stats", h.GetArtistStats)


		protected.POST("/label", h.CreateLabel)
		protected.PUT("/label/:id", h.UpdateLabel)
		protected.DELETE("/label/:id", h.DeleteLabel)

		protected.GET("/suggestion", h.GetSuggestions)
		protected.PATCH("/suggestion/:id/status", h.UpdateSuggestionStatus)

		protected.GET("/feedback", h.GetFeedbacks)
		protected.DELETE("/feedback/:id", h.DeleteFeedback)
	}

	// Admin-only routes
	admin := v1.Group("")
	admin.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	admin.Use(middlewares.RequireRole("ADMIN"))
	{
		admin.GET("/auth", h.GetUsers)
		admin.POST("/auth/register", h.RegisterUser)
		admin.DELETE("/auth/:id", h.DeleteUser)
		admin.PATCH("/auth/:id/role", h.UpdateUserRole)

		admin.POST("/source", h.CreateSource)
		admin.PUT("/source/:id", h.UpdateSource)
		admin.DELETE("/source/:id", h.DeleteSource)
	}

	errCh := make(chan error, 1)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Msgf("starting server on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	select {
	case sig := <-sigCh:
		log.Info().Msgf("received signal: %s, shutting down server", sig)
	case err := <-errCh:
		return err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Info().Msg("server gracefully stopped")

	return nil
}

func initDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

func runMigrations(ctx context.Context, db *pgxpool.Pool) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	m, err := migrate.NewMigrator(ctx, conn.Conn(), "public.schema_version")
	if err != nil {
		return err
	}
	if err := m.LoadMigrations(migrations.FS); err != nil {
		return err
	}
	return m.Migrate(ctx)
}

func logger(skipPaths ...string) gin.HandlerFunc {
	skip := make(map[string]struct{}, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		c.Next()

		if _, ok := skip[c.Request.URL.Path]; ok {
			return
		}

		latency := time.Since(start)
		status := c.Writer.Status()

		event := log.Info()
		if status >= 500 {
			event = log.Error()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Msg("request")
	}
}
