package logger

import (
	"context"
	"sync"
)

var (
	globalMu sync.RWMutex
	global   Logger = nil
)

// Init initializes the package-global logger. Call once from main.
//
// Example:
//
//	logger.Init(logger.WithEnvironment("dev"), logger.WithFormat("console"))
func Init(opts ...Option) {
	globalMu.Lock()
	defer globalMu.Unlock()

	global = NewZapLogger(opts...)
}

// L returns the package-global logger. If Init() was not called, it creates a default dev logger.
func L() Logger {
	globalMu.RLock()
	g := global
	globalMu.RUnlock()

	if g == nil {
		// Safe lazy-init with write-lock if necessary
		globalMu.Lock()
		if global == nil {
			global = NewZapLoggerDevelopment()
		}
		g = global
		globalMu.Unlock()
	}
	return g
}

// FromContext returns logger from context if present, otherwise returns package-global logger.
func FromContext(ctx context.Context) Logger {
	if ctx == nil {
		return L()
	}
	if v := ctx.Value(loggerContextKey{}); v != nil {
		if lg, ok := v.(Logger); ok && lg != nil {
			return lg
		}
	}
	return L()
}

// WithContext returns a new context carrying the provided logger.
func WithContext(ctx context.Context, lg Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerContextKey{}, lg)
}

// internal context key type to avoid collisions
type loggerContextKey struct{}
