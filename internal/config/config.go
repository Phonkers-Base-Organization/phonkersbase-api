package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	DatabaseURL          string `validate:"required,uri"`
	JWTSecret            string `validate:"required,min=32"`
	CORSOrigins          []string
	Port                 string `validate:"required,numeric"`
	OTELServiceName      string `validate:"required"`
	OTELExporterEndpoint string `validate:"omitempty,uri"`
}

func Load() (*Config, error) {
	config := &Config{
		DatabaseURL:          os.Getenv("DB_URL"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		CORSOrigins:          parseCORSOrigins(os.Getenv("CORS_ORIGIN")),
		Port:                 envOrDefault("PORT", "8080"),
		OTELServiceName:      envOrDefault("OTEL_SERVICE_NAME", "pb-api2"),
		OTELExporterEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}

	validate := validator.New()
	err := validate.Struct(config)
	if err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, fmt.Sprintf("field '%s' failed validation: %s", err.Field(), err.Tag()))
		}
		return nil, fmt.Errorf("configuration validation failed: %v", validationErrors)
	}

	return config, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseCORSOrigins(raw string) []string {
	if raw == "" {
		return []string{"http://localhost:3000"}
	}

	var origins []string
	for _, origin := range strings.Split(raw, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}

	return origins
}
