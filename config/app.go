package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// AppConfig groups the settings needed to start the API and its worker.
type AppConfig struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	UserCount UserCountConfig
}

type ServerConfig struct {
	Addr            string
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	URI                string
	Name               string
	Collection         string
	StartupTimeout     time.Duration
	CleanupTimeout     time.Duration
	HealthcheckTimeout time.Duration
}

// The fallback keeps bcrypt work similar when a login email has no account.
const defaultMissingUserHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

type AuthConfig struct {
	JWTSecret           string
	MissingUserHash     string
	AccessTokenLifetime time.Duration
}

// UserCountConfig controls the frequency and deadline of the count query.
type UserCountConfig struct {
	Interval time.Duration
	Timeout  time.Duration
}

// Load reads application settings from the environment and an optional .env file.
func Load() (AppConfig, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		// A dotenv parser error can contain values from the file.
		return AppConfig{}, errors.New("cannot load .env: check file syntax and permissions")
	}

	cfg := AppConfig{
		Server: ServerConfig{Addr: envOrDefault("SERVER_ADDR", "127.0.0.1:8080")},
		Database: DatabaseConfig{
			URI:        os.Getenv("MONGO_URI"),
			Name:       envOrDefault("MONGO_DATABASE", "user_management"),
			Collection: envOrDefault("MONGO_COLLECTION", "users"),
		},
		Auth: AuthConfig{
			JWTSecret:       os.Getenv("JWT_SECRET"),
			MissingUserHash: envOrDefault("MISSING_USER_HASH", defaultMissingUserHash),
		},
	}

	for _, setting := range []struct {
		key      string
		target   *time.Duration
		fallback time.Duration
	}{
		{"SERVER_SHUTDOWN_TIMEOUT", &cfg.Server.ShutdownTimeout, 5 * time.Second},
		{"MONGO_STARTUP_TIMEOUT", &cfg.Database.StartupTimeout, 10 * time.Second},
		{"MONGO_CLEANUP_TIMEOUT", &cfg.Database.CleanupTimeout, 5 * time.Second},
		{"MONGO_HEALTHCHECK_TIMEOUT", &cfg.Database.HealthcheckTimeout, 2 * time.Second},
		{"JWT_ACCESS_TOKEN_LIFETIME", &cfg.Auth.AccessTokenLifetime, time.Hour},
		{"USER_COUNT_INTERVAL", &cfg.UserCount.Interval, 10 * time.Second},
		{"USER_COUNT_TIMEOUT", &cfg.UserCount.Timeout, 5 * time.Second},
	} {
		value, err := time.ParseDuration(envOrDefault(setting.key, setting.fallback.String()))
		if err != nil || value <= 0 {
			return AppConfig{}, fmt.Errorf("%s must be a positive duration (for example, 5s or 1h)", setting.key)
		}
		*setting.target = value
	}
	if cfg.Auth.AccessTokenLifetime < time.Second {
		return AppConfig{}, errors.New("JWT_ACCESS_TOKEN_LIFETIME must be at least 1s")
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
