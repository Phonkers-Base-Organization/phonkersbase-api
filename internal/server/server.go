package server

import (
	"context"
	"os"
	"os/signal"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"

	"github.com/PhonkersBase/base-api2/internal/config"
	"github.com/PhonkersBase/base-api2/internal/handlers"
	"github.com/PhonkersBase/base-api2/internal/migrations"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/PhonkersBase/base-api2/internal/middlewares"
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
		repository.NewCountryRepo(dbpool),
		repository.NewLabelRepo(dbpool),
	)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger("/", "/health", "/metrics"))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	v1 := router.Group("/api/v1")
	v1.Use(middlewares.RateLimitRequests())
	{
		v1.GET("/artist/all", h.GetArtists)
		v1.GET("/country/all", h.GetCountries)
		v1.GET("/label/all", h.GetLabels)
		v1.POST("/app/dev/force-sync", h.ForceSync)
	}
	errCh := make(chan error, 1)
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
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