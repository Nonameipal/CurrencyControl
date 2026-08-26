package database

import (
	"context"
	"fmt"

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

	return pool, nil
}

func CloseConnection(db *pgxpool.Pool) {
	if db != nil {
		db.Close()
	}
}