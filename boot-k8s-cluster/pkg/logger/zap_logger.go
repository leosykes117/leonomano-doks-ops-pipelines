package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a Logger built on zap, configurable via functional options.
func NewZapLogger(opts ...Option) Logger {
	// defaults
	cfg := &Config{
		Environment: "dev",
		Level:       Info,
		Format:      "console",
		Color:       true,
	}

	for _, o := range opts {
		o(cfg)
	}

	// Build zap encoder config
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.Color {
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoder = zapcore.NewJSONEncoder(encCfg)
	} else {
		encoder = zapcore.NewConsoleEncoder(encCfg)
	}

	// Level
	var lvl zapcore.Level
	switch cfg.Level {
	case Debug:
		lvl = zapcore.DebugLevel
	case Info:
		lvl = zapcore.InfoLevel
	case Warn:
		lvl = zapcore.WarnLevel
	case Error:
		lvl = zapcore.ErrorLevel
	default:
		lvl = zapcore.InfoLevel
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lvl)

	// Add caller (so logs include file:line) and skip enough frames to reach user's caller.
	baseLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))

	return &zapLogger{
		z: baseLogger,
	}
}

// WithOption helpers to construct config via options:
func WithEnvironment(env string) Option {
	return func(c *Config) { c.Environment = env }
}
func WithLevel(l Level) Option {
	return func(c *Config) { c.Level = l }
}
func WithFormat(format string) Option {
	return func(c *Config) { c.Format = format }
}
func WithColor(color bool) Option {
	return func(c *Config) { c.Color = color }
}

// zapLogger implements Logger
type zapLogger struct {
	z *zap.Logger
}

func (z *zapLogger) With(fields ...Field) Logger {
	zapFields := toZap(fields...)
	return &zapLogger{z: z.z.With(zapFields...)}
}

func (z *zapLogger) Debug(msg string, fields ...Field) {
	//z.z.Sugar().With(toAnyMap(fields...)...).Debug(msg)
	zapFields := toZap(fields...)
	//z.z.With(zapFields...).Debug(msg)
	z.z.Debug(msg, zapFields...)
}

func (z *zapLogger) Info(msg string, fields ...Field) {
	//z.z.Sugar().With(toAnyMap(fields...)...).Info(msg)
	zapFields := toZap(fields...)
	//z.z.With(zapFields...).Info(msg)
	z.z.Info(msg, zapFields...)
}

func (z *zapLogger) Warn(msg string, fields ...Field) {
	//z.z.Sugar().With(toAnyMap(fields...)...).Warn(msg)
	zapFields := toZap(fields...)
	//z.z.With(zapFields...).Warn(msg)
	z.z.Warn(msg, zapFields...)
}

func (z *zapLogger) Error(msg string, fields ...Field) {
	//z.z.Sugar().With(toAnyMap(fields...)...).Error(msg)
	zapFields := toZap(fields...)
	//z.z.With(zapFields...).Error(msg)
	z.z.Error(msg, zapFields...)
}

func (z *zapLogger) WithContext(ctx context.Context) Logger {
	// Example: if context has trace-id etc, you can extract and attach here.
	// This implementation simply returns same logger (no-op) — extend as you need.
	return z
}

// Close or Sync if needed
func (z *zapLogger) Sync() error {
	return z.z.Sync()
}

// Optional: shortcut to create a standard dev logger
func NewZapLoggerDevelopment() Logger {
	return NewZapLogger(WithEnvironment("dev"), WithFormat("console"), WithLevel(Debug), WithColor(true))
}

// Optional: standard production logger
func NewZapLoggerProduction() Logger {
	return NewZapLogger(WithEnvironment("prod"), WithFormat("json"), WithLevel(Info), WithColor(false))
}
