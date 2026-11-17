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
	postgresDatabase, err := envRequired("POSTGRES_DATABASE")
	if err != nil {
		return err
	}
	adServer, err := envRequired("AD_SERVER")
	if err != nil {
		return err
	}
	adDomain, err := envRequired("AD_DOMAIN")
	if err != nil {
		return err
	}
	adSearchBase, err := envRequired("AD_SEARCH_BASE")
	if err != nil {
		return err
	}
	absEndpoint, err := envRequired("ABS_ENDPOINT")
	if err != nil {
		return err
	}
	jwtSecret, err := envRequired("JWT_SECRET")
	if err != nil {
		return err
	}
	AppSettings = Configs{
		AppParams: AppParams{
			ServerURL:  envString("SERVER_URL", "localhost"),
			ServerName: envString("SERVER_NAME", "GlobalServer"),
			PortRun:    envString("SERVER_PORT", "8088"),
		},
		PostgresParams: PostgresParams{
			Host:     envString("POSTGRES_HOST", "localhost"),
			Port:     envString("POSTGRES_PORT", "5432"),
			User:     postgresUser,
			Password: postgresPassword,
			Database: postgresDatabase,
		},
		ADParams: ADParams{
			Server:     adServer,
			Domain:     adDomain,
			SearchBase: adSearchBase,
		},
		ABSParams: ABSParams{
			Endpoint: absEndpoint,
		},
		AuthParams: AuthParams{
			AccessTokenTtlMinutes: envInt("ACCESS_TOKEN_TTL_MINUTES", 20),
			RefreshTokenTtlDays:   envInt("REFRESH_TOKEN_TTL_DAYS", 1),
			JwtSecret:             jwtSecret,
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

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil || result <= 0 {
		return fallback
	}
	return result
}
