package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// Logger provides structured ND-JSON logging
type Logger struct {
	output   io.Writer
	minLevel LogLevel
}

// LogEntry represents a single log entry in JSON format
// Additional fields are added directly to the JSON at the top level
type LogEntry struct {
	Time    string   `json:"time"`
	Level   LogLevel `json:"level"`
	Message string   `json:"message"`
}

// NewLogger creates a new logger that writes to stdout with Info level
func NewLogger() *Logger {
	return &Logger{
		output:   os.Stdout,
		minLevel: LogLevelInfo,
	}
}

// NewLoggerWithLevel creates a new logger with a specific minimum log level
func NewLoggerWithLevel(minLevel LogLevel) *Logger {
	return &Logger{
		output:   os.Stdout,
		minLevel: minLevel,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.minLevel = level
}

// shouldLog returns true if the given level should be logged
func (l *Logger) shouldLog(level LogLevel) bool {
	levelOrder := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}
	return levelOrder[level] >= levelOrder[l.minLevel]
}

// log outputs a log entry as ND-JSON
func (l *Logger) log(level LogLevel, message string, fields map[string]any) {
	// Check if this level should be logged
	if !l.shouldLog(level) {
		return
	}

	// Manually build JSON with guaranteed field order
	jsonStr := "{"

	// Add level, message first (in that specific order)
	levelJSON, _ := json.Marshal(level)
	messageJSON, _ := json.Marshal(message)

	jsonStr += fmt.Sprintf("\"level\":%s,\"message\":%s", levelJSON, messageJSON)

	// Add additional fields, checking for conflicts with reserved fields
	reservedFields := map[string]bool{
		"level":   true,
		"message": true,
	}

	for key, value := range fields {
		// If field conflicts with reserved field, prefix with underscore
		fieldName := key
		if reservedFields[key] {
			fieldName = "_" + key
		}

		valueJSON, err := json.Marshal(value)
		if err != nil {
			// Skip fields that can't be marshaled
			continue
		}

		jsonStr += fmt.Sprintf(",\"%s\":%s", fieldName, valueJSON)
	}

	jsonStr += "}\n"

	// Explicitly ignore the error as we can't do much about it in a logger
	_, _ = l.output.Write([]byte(jsonStr))
}

// Debug logs a debug-level message
func (l *Logger) Debug(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelDebug, message, f)
}

// Info logs an info-level message
func (l *Logger) Info(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelInfo, message, f)
}

// Warn logs a warning-level message
func (l *Logger) Warn(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelWarn, message, f)
}

// Error logs an error-level message
func (l *Logger) Error(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelError, message, f)
}

// Fatal logs a fatal-level message and exits with code 1
func (l *Logger) Fatal(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LogLevelFatal, message, f)
	os.Exit(1)
}

// Fatalf logs a fatal-level formatted message and exits with code 1
func (l *Logger) Fatalf(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.log(LogLevelFatal, message, nil)
	os.Exit(1)
}
