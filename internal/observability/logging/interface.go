package logging

// Logger defines logging interface
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
}

// Field represents a log field
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new field (convenience function)
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}
