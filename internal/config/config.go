package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	DatabaseURL string `validate:"required,uri"`
	JWTSecret   string `validate:"required,min=32"`
	CORSOrigins []string
}

func Load() (*Config, error) {
	config := &Config{
		DatabaseURL: os.Getenv("DB_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		CORSOrigins: parseCORSOrigins(os.Getenv("CORS_ORIGIN")),
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
