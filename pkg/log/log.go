// Package log provides structured logging for TerraCi CLI
package log

import (
	"os"

	"github.com/caarlos0/log"
)

// Level represents log severity
type Level = log.Level

// Log levels
const (
	DebugLevel = log.DebugLevel
	InfoLevel  = log.InfoLevel
	WarnLevel  = log.WarnLevel
	ErrorLevel = log.ErrorLevel
	FatalLevel = log.FatalLevel
)

// currentLevel holds the current log level for IsDebug check
var currentLevel = InfoLevel

// SetLevel sets the global log level
func SetLevel(level Level) {
	currentLevel = level
	log.SetLevel(level)
}

// SetLevelFromString sets the log level from a string
// Supported values: debug, info, warn, error, fatal
func SetLevelFromString(level string) error {
	l, err := log.ParseLevel(level)
	if err != nil {
		return err
	}
	currentLevel = l
	log.SetLevel(l)
	return nil
}

// Debug logs a debug message
func Debug(msg string) {
	log.Debug(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...any) {
	log.Debugf(format, args...)
}

// Info logs an info message
func Info(msg string) {
	log.Info(msg)
}

// Infof logs a formatted info message
func Infof(format string, args ...any) {
	log.Infof(format, args...)
}

// Warn logs a warning message
func Warn(msg string) {
	log.Warn(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...any) {
	log.Warnf(format, args...)
}

// Error logs an error message
func Error(msg string) {
	log.Error(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...any) {
	log.Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	log.Fatal(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...any) {
	log.Fatalf(format, args...)
}

// WithField returns an entry with the given field
func WithField(key string, value any) *log.Entry {
	return log.WithField(key, value)
}

// WithError returns an entry with the given error
func WithError(err error) *log.Entry {
	return log.WithError(err)
}

// IncreasePadding increases log output indentation
func IncreasePadding() {
	log.IncreasePadding()
}

// DecreasePadding decreases log output indentation
func DecreasePadding() {
	log.DecreasePadding()
}

// ResetPadding resets log output indentation
func ResetPadding() {
	log.ResetPadding()
}

// IsDebug returns true if debug level is enabled
func IsDebug() bool {
	return currentLevel <= DebugLevel
}

// Init initializes the logger with default settings
func Init() {
	log.Log = log.New(os.Stderr)
	log.SetLevel(InfoLevel)
}
