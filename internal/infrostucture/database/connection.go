package database

import (
	"context"
	"fmt"
	"os"

	"CurrencyControl/internal/configs"
	appLogger "CurrencyControl/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)
func InitConnection(ctx context.Context) (*pgxpool.Pool, error) {
	connectionConfigs := configs.AppSettings.PostgresParams
	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		connectionConfigs.User,
		connectionConfigs.Password,
		connectionConfigs.Host,
		connectionConfigs.Port,
		connectionConfigs.Database,
	)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	appLogger.GetLogger().Info().Msg("postgres connection established")

	migrationFiles := []string{

	}
	for _, file := range migrationFiles {
		content, err := os.ReadFile(file)
		if err == nil {
			_, err = pool.Exec(ctx, string(content))
			if err != nil {
				appLogger.GetLogger().Error().Err(err).Msgf("failed to execute migration %s", file)
			}
		}
	}

	return pool, nil
}

func CloseConnection(db *pgxpool.Pool) {
	if db != nil {
		db.Close()
	}
}
