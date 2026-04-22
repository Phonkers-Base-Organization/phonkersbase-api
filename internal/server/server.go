package server

import (
	"context"
	"os"
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
	if err := runMigrations(initCtx, dbpool); err != nil {
		return err
	}
	log.Info().Msg("database migrations applied")

	h := handlers.NewHandler(
		repository.NewArtistRepo(dbpool),
		repository.NewCountryRepo(dbpool),
		repository.NewLabelRepo(dbpool),
	)

	router := gin.Default()
	v1 := router.Group("/api/v1")
	v1.Use(middlewares.RateLimitRequests())
	{
		v1.GET("/artist/all", h.GetArtists)
		v1.GET("/country/all", h.GetCountries)
		v1.GET("/label/all", h.GetLabels)
		v1.POST("/app/dev/force-sync", h.ForceSync)
	}

	log.Info().Str("port", port).Msg("server listening")
	return router.Run(":" + port)
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
