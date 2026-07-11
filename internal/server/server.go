package server

import (
	"context"
	"errors"
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
	"github.com/PhonkersBase/base-api2/internal/metrics"
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
	log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger().
		Hook(metrics.ErrorHook{}).
		Hook(metrics.CallerHook{})

	initCtx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	metricsShutdown, err := metrics.Init(initCtx, cfg.OTELServiceName, cfg.OTELExporterEndpoint)
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsShutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to shut down metrics exporter")
		}
	}()

	dbpool, err := initDB(initCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer dbpool.Close()
	err = runMigrations(initCtx, dbpool)
	if err != nil {
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
		repository.NewOrganisationRepo(dbpool),
		repository.NewChangeHistoryRepo(dbpool),
		cfg.JWTSecret,
	)

	router := gin.New()
	router.Use(gin.Recovery())

	if err := router.SetTrustedProxies([]string{"172.19.0.0/16"}); err != nil {
		return err
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	router.Use(requestTimeout(30 * time.Second))
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
	v1.GET("/organisation/all", h.GetOrganisations)
	v1.GET("/organisation/labels", h.GetLabelOrganisations)
	v1.GET("/organisation/distributors", h.GetDistributorOrganisations)
	v1.GET("/organisation/cults", h.GetCultOrganisations)
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
		protected.DELETE("/suggestion/:id", h.DeleteSuggestion)

		protected.GET("/feedback", h.GetFeedbacks)
		protected.DELETE("/feedback/:id", h.DeleteFeedback)

		protected.GET("/history", h.GetHistory)
		protected.GET("/history/editors", h.GetHistoryEditors)
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

		admin.POST("/organisation", h.CreateOrganisation)
		admin.PUT("/organisation/:id", h.UpdateOrganisation)
		admin.DELETE("/organisation/:id", h.DeleteOrganisation)

	}

	errCh := make(chan error, 1)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second, // above the 30s request context timeout
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info().Msgf("starting server on port %s", cfg.Port)
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
	cfg.ConnConfig.Tracer = metrics.NewDBTracer()
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

func requestTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
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

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		metrics.RecordHTTPRequest(c.Request.Context(), c.Request.Method, route, status, latency)

		event := log.Info()
		if status >= 500 && !errors.Is(c.Request.Context().Err(), context.Canceled) {
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
