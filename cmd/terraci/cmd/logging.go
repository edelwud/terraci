package cmd

import (
	"os"

	log "github.com/caarlos0/log"
)

// initLogger wires the global caarlos0/log logger to stderr at info level.
// Replaces the previous pkg/log.Init() helper.
func initLogger() {
	log.Log = log.New(os.Stderr)
	log.SetLevel(log.InfoLevel)
}

// setLogLevelFromString applies a textual level (debug/info/warn/error/fatal).
func setLogLevelFromString(level string) error {
	parsed, err := log.ParseLevel(level)
	if err != nil {
		return err
	}
	log.SetLevel(parsed)
	return nil
}

// isDebug reports whether the global logger is currently at debug level.
func isDebug() bool {
	if logger, ok := log.Log.(*log.Logger); ok {
		return logger.Level <= log.DebugLevel
	}
	return false
}
