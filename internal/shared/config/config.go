package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	RunMigrations       bool
	LogLevel            string
	UserServiceAddress  string
	PartyServiceAddress string
}

func Load() (Config, error) {
	envPath, err := findEnvFile(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	return LoadFile(envPath)
}

func LoadFile(path string) (Config, error) {
	values := map[string]string{}

	if path != "" {
		var err error
		values, err = godotenv.Read(path)
		if err != nil {
			return Config{}, fmt.Errorf("read env file: %w", err)
		}
	}

	runMigrations, err := envBool(values, "RUN_MIGRATIONS", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:            envString(values, "HTTP_ADDR", ":8080"),
		DatabaseURL:         envString(values, "DATABASE_URL", ""),
		RunMigrations:       runMigrations,
		LogLevel:            envString(values, "LOG_LEVEL", "info"),
		PartyServiceAddress: envString(values, "PARTY_SERVICE_ADDR", ""),
		UserServiceAddress:  envString(values, "USER_SERVICE_ADDR", ""),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
}

func findEnvFile(name string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func envString(values map[string]string, name string, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	if value := values[name]; value != "" {
		return value
	}
	return defaultValue
}

func envBool(values map[string]string, name string, defaultValue bool) (bool, error) {
	value := envString(values, name, "")
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s as bool: %w", name, err)
	}
	return parsed, nil
}
