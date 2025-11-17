package logger

// Helper constructors for common field types.
// These make call sites readable: logger.Info("ok", String("user", "leo"), Int("n", 7))

func String(key, v string) Field          { return Field{Key: key, Value: v} }
func Int(key string, v int) Field         { return Field{Key: key, Value: v} }
func Int64(key string, v int64) Field     { return Field{Key: key, Value: v} }
func Bool(key string, v bool) Field       { return Field{Key: key, Value: v} }
func Float64(key string, v float64) Field { return Field{Key: key, Value: v} }
func Any(key string, v any) Field         { return Field{Key: key, Value: v} }
func Err(err error) Field                 { return Field{Key: "error", Value: err} }
