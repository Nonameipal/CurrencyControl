package configs

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var AppSettings Configs

func Load() error {
	_ = godotenv.Load(".env")

	postgresUser, err := envRequired("POSTGRES_USER")
	if err != nil {
		return err
	}
	postgresPassword, err := envRequired("POSTGRES_PASSWORD")
	if err != nil {
		return err
	}

	AppSettings = Configs{
		AppParams: AppParams{
			ServerURL:  envString("SERVER_URL", "localhost"),
			ServerName: envString("SERVER_NAME", "GlobalServer"),
			PortRun:    envString("SERVER_PORT", "8088"),
			GinMode:    envString("GIN_MODE", "debug"),
		},
		PostgresParams: PostgresParams{
			Host:     envString("POSTGRES_HOST", "localhost"),
			Port:     envString("POSTGRES_PORT", "5432"),
			User:     postgresUser,
			Password: postgresPassword,
			Database: envString("POSTGRES_DATABASE", "currency_control"),
		},
	}

	return nil
}

func envString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envRequired(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("%s is empty", key)
	}
	return value, nil
}
