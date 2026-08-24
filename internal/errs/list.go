package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
	log = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger()
}

func GetLogger() *zerolog.Logger {
	return &log
}

func Error(err error, msg string, args ...interface{}) {
	log.Error().Err(err).Msgf(msg, args...)
}

func Info(msg string, args ...interface{}) {
	log.Info().Msgf(msg, args...)
}

func Debug(msg string, args ...interface{}) {
	log.Debug().Msgf(msg, args...)
}

func Fatal(err error, msg string, args ...interface{}) {
	log.Fatal().Err(err).Msgf(msg, args...)
}
