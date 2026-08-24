package app
import (
	"CurrencyControl/internal/logger"
	"CurrencyControl/internal/configs"
	"CurrencyControl/internal/infrostucture/database"
	"context"
	"net/http"

)
func Run() error {
	log := logger.GetLogger()
	if err := configs.Load(); err != nil {
		return err
	}

	ctx := context.Background()
	db, err := database.InitConnection(ctx)
	if err != nil {
		return err
	}
	defer database.CloseConnection(db)

	
	log.Info().Str("address", server.Addr).Msg("server listening")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}