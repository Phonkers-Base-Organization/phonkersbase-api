package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	DatabaseURL string `validate:"required,uri"`
}

func Load() (*Config, error) {
	config := &Config{
		DatabaseURL: os.Getenv("DB_URL"),
	}

	// Validate the configuration
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
