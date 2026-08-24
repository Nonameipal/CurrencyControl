package main

import (
	"CurrencyControl/internal/app"
	"CurrencyControl/internal/logger"
)

func main() {
	log := logger.GetLogger()

	if err := app.Run(); err != nil {
		log.Error().Err(err).Msg("application stopped")
	}
}
