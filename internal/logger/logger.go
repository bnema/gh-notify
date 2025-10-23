package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init initializes the global logger with pretty console output
func Init(verbose bool) {
	// Configure zerolog for human-readable output
	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
		NoColor:    false,
	}

	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	// Set log level based on verbose flag
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// Info logs an info message
func Info() *zerolog.Event {
	return log.Info()
}

// Debug logs a debug message
func Debug() *zerolog.Event {
	return log.Debug()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs an error message
func Error() *zerolog.Event {
	return log.Error()
}

// WithDuration creates a log entry with duration tracking
func WithDuration(start time.Time) *zerolog.Event {
	return log.Debug().Dur("duration", time.Since(start))
}
