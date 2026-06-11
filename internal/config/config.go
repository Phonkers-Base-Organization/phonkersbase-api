package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	DatabaseURL string `validate:"required,uri"`
	JWTSecret   string `validate:"required,min=32"`
	CORSOrigin  string
}

func Load() (*Config, error) {
	config := &Config{
		DatabaseURL:         os.Getenv("DB_URL"),
		JWTSecret:  os.Getenv("JWT_SECRET"),
		CORSOrigin: os.Getenv("CORS_ORIGIN"),
	}

	if config.CORSOrigin == "" {
		config.CORSOrigin = "http://localhost:3000"
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
