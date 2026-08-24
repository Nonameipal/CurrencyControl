package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
	log = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()
}

func GetLogger() *zerolog.Logger {
	return &log
}
