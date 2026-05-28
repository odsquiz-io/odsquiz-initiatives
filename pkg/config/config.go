// pkg/config/config.go: handles loading and managing application configuration from environment variables.
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration settings.
type Config struct {
	Port       string // Server port
	JWTSecret  string // JWT Secret
	DBHost     string // Database host
	DBPort     string // Database port
	DBUser     string // Database user
	DBPassword string // Database password
	DBName     string // Database name
	DBSSLMode  string // Database SSL mode
}

// Load reads configuration from environment variables and validates required fields.
func Load() (*Config, error) {
	// Load environment variables from .env file if present
	_ = godotenv.Load()

	// Create config struct with values from environment
	cfg := &Config{
		Port:       os.Getenv("PORT"),
		JWTSecret:  os.Getenv("JWTSecret"),
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
	}

	// Validate that required database fields are set
	if cfg.DBHost == "" || cfg.DBPort == "" || cfg.DBUser == "" || cfg.DBName == "" {
		return nil, fmt.Errorf("missing required database environment variables")
	}

	return cfg, nil
}
