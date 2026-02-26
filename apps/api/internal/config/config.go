package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	Primary  PrimaryConfig  `validate:"required"`
	Server   ServerConfig   `validate:"required"`
	Database DatabaseConfig `validate:"required"`
	Auth     AuthConfig     `validate:"required"`
	Email    EmailConfig    `validate:"required"`
}

type PrimaryConfig struct {
	Env string `validate:"required,oneof=development staging production"`
}

type ServerConfig struct {
	Port          int           `validate:"required,min=1"`
	ReadTimeout   time.Duration `validate:"required"`
	WriteTimeout  time.Duration `validate:"required"`
	IdleTimeout   time.Duration `validate:"required"`
	AllowedOrigin string        `validate:"required"`
}

type DatabaseConfig struct {
	URL             string        `validate:"required"`
	MaxOpenConns    int           `validate:"required,min=1"`
	MaxIdleConns    int           `validate:"required,min=0"`
	ConnMaxLifetime time.Duration `validate:"required"`
	ConnMaxIdleTime time.Duration `validate:"required"`
}

type AuthConfig struct {
	JWTSecret string `validate:"required,min=32"`
}

type EmailConfig struct {
	ResendAPIKey string `validate:"required"`
	FromEmail    string `validate:"required,email"`
}

func Load() (*Config, error) {
	// Support running from either apps/api or repo root.
	_ = godotenv.Load(".env", "apps/api/.env")

	cfg := &Config{
		Primary: PrimaryConfig{
			Env: getEnv("APP_ENV", "development"),
		},
		Server: ServerConfig{
			Port:          getEnvAsInt("PORT", 8080),
			ReadTimeout:   getEnvAsDuration("READ_TIMEOUT", 10*time.Second),
			WriteTimeout:  getEnvAsDuration("WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:   getEnvAsDuration("IDLE_TIMEOUT", 60*time.Second),
			AllowedOrigin: getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", time.Hour),
			ConnMaxIdleTime: getEnvAsDuration("DB_CONN_MAX_IDLE_TIME", 30*time.Minute),
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", ""),
		},
		Email: EmailConfig{
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
			FromEmail:    getEnv("FROM_EMAIL", "onboarding@resend.dev"),
		},
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		panic(fmt.Sprintf("invalid integer for %s", key))
	}
	return val
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		panic(fmt.Sprintf("invalid duration for %s", key))
	}
	return val
}
