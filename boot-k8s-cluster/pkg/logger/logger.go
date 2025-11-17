package logger

import "context"

// Field represents a key/value pair for logging (agnóstico del backend).
type Field struct {
	Key   string
	Value any
}

// Level is a lightweight level type for configuration.
type Level string

const (
	Debug Level = "debug"
	Info  Level = "info"
	Warn  Level = "warn"
	Error Level = "error"
)

// Logger is the application-facing logger interface.
// Methods accept variadic Fields (optional) so you can call e.g. Info("msg") or Info("msg", String("k","v")).
type Logger interface {
	With(fields ...Field) Logger

	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

	// WithContext returns a logger enriched with values from context (optional).
	WithContext(ctx context.Context) Logger

	Sync() error
}

// Config contains options to initialize a logger.
type Config struct {
	Environment string // "dev" or "prod"
	Level       Level  // "debug","info","warn","error"
	Format      string // "console" or "json"
	Color       bool   // colorize console output
}

// Option for functional options pattern.
type Option func(*Config)
