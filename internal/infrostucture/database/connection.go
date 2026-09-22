package database

import (
	"context"
	"fmt"

	"CurrencyControl/internal/configs"
	appLogger "CurrencyControl/internal/logger"
	"gorm.io/gorm"
)

func InitConnection(ctx context.Context) (*gorm.DB, error) {
	connectionConfigs := configs.AppSettings.PostgresParams
	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		connectionConfigs.User,
		connectionConfigs.Password,
		connectionConfigs.Host,
		connectionConfigs.Port,
		connectionConfigs.Database,
	)

	gormDB, err := InitGormDB(connStr)
	if err != nil {
		appLogger.GetLogger().Error().Err(err).Msg("failed to initialize database and run GORM auto-migrations")
		return nil, err
	}

	appLogger.GetLogger().Info().Msg("GORM migrations applied successfully")
	appLogger.GetLogger().Info().Msg("postgres connection established")

	return gormDB, nil
}

func CloseConnection(db *gorm.DB) {
	if db != nil {
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}
