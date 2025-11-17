package logger

import "go.uber.org/zap"

// toZap converts []Field to []zap.Field
func toZap(fields ...Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}
	out := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		out = append(out, zap.Any(f.Key, f.Value))
	}
	return out
}
