package cmd

import (
	"os"

	log "github.com/caarlos0/log"
)

// InitLogger wires the global caarlos0/log logger to stderr at info level.
func InitLogger() {
	log.Log = log.New(os.Stderr)
	log.SetLevel(log.InfoLevel)
}

// IsDebug reports whether the global logger is currently at debug level.
func IsDebug() bool {
	if logger, ok := log.Log.(*log.Logger); ok {
		return logger.Level <= log.DebugLevel
	}
	return false
}
