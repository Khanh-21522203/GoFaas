package logging

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// SimpleLogger implements Logger using standard library log
type SimpleLogger struct {
	fields []Field
}

// NewSimpleLogger creates a new simple logger
func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{
		fields: make([]Field, 0),
	}
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string, fields ...Field) {
	l.log("DEBUG", msg, fields...)
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string, fields ...Field) {
	l.log("INFO", msg, fields...)
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(msg string, fields ...Field) {
	l.log("WARN", msg, fields...)
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, fields ...Field) {
	l.log("ERROR", msg, fields...)
}

// WithFields returns a new logger with additional fields
func (l *SimpleLogger) WithFields(fields ...Field) Logger {
	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &SimpleLogger{
		fields: newFields,
	}
}

func (l *SimpleLogger) log(level, msg string, fields ...Field) {
	timestamp := time.Now().Format(time.RFC3339)

	// Combine logger fields with message fields
	allFields := append(l.fields, fields...)

	// Format fields
	fieldStrs := make([]string, len(allFields))
	for i, f := range allFields {
		fieldStrs[i] = fmt.Sprintf("%s=%v", f.Key, f.Value)
	}

	fieldsStr := ""
	if len(fieldStrs) > 0 {
		fieldsStr = " " + strings.Join(fieldStrs, " ")
	}

	log.Printf("[%s] %s: %s%s", timestamp, level, msg, fieldsStr)
}
