package main

import (
	"CurrencyControl/internal/app"
	"CurrencyControl/internal/logger"
)

// @title Currency Control API
// @version 1.0
// @description API сервера для системы валютного контроля.
// @host localhost:8088
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Session-Token
// @description Введите Session-Token (полученный при /auth/login)

func main() {
	log := logger.GetLogger()

	if err := app.Run(); err != nil {
		log.Error().Err(err).Msg("application stopped")
	}
}
