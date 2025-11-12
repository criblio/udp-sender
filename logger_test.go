package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestLogger_OutputFormat(t *testing.T) {
	// Create a logger that writes to a buffer for testing
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Test Info log
	logger.Info("Test info message", map[string]any{
		"key1": "value1",
		"key2": 42,
	})

	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("Log output should end with newline")
	}

	// Parse the JSON as a map since fields are now at top level
	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify core fields
	if logMap["level"] != "info" {
		t.Errorf("Expected level 'info', got '%s'", logMap["level"])
	}
	if logMap["message"] != "Test info message" {
		t.Errorf("Expected message 'Test info message', got '%s'", logMap["message"])
	}
	// Verify additional fields are at top level
	if logMap["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", logMap["key1"])
	}
	if logMap["key2"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected key2=42, got '%v'", logMap["key2"])
	}
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(*Logger, string, map[string]any)
		expected LogLevel
	}{
		{
			name: "Debug",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Debug(msg, fields)
			},
			expected: LogLevelDebug,
		},
		{
			name: "Info",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Info(msg, fields)
			},
			expected: LogLevelInfo,
		},
		{
			name: "Warn",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Warn(msg, fields)
			},
			expected: LogLevelWarn,
		},
		{
			name: "Error",
			logFunc: func(l *Logger, msg string, fields map[string]any) {
				l.Error(msg, fields)
			},
			expected: LogLevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{
				output:   &buf,
				minLevel: LogLevelDebug,
			}

			tt.logFunc(logger, "test message", nil)

			var logMap map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
				t.Fatalf("Failed to parse JSON output: %v", err)
			}

			if logMap["level"] != string(tt.expected) {
				t.Errorf("Expected level '%s', got '%s'", tt.expected, logMap["level"])
			}
		})
	}
}

func TestLogger_WithoutFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	logger.Info("Simple message")

	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if logMap["message"] != "Simple message" {
		t.Errorf("Expected message 'Simple message', got '%s'", logMap["message"])
	}

	// Should only have the core fields (level, message)
	expectedFieldCount := 2
	if len(logMap) != expectedFieldCount {
		t.Errorf("Expected %d fields (level, message), got %d: %v", expectedFieldCount, len(logMap), logMap)
	}
}

func TestLogger_MultipleLogEntries(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Write multiple log entries
	logger.Info("First message")
	logger.Warn("Second message")
	logger.Error("Third message")

	// Split by newlines and verify each is valid JSON
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		var logMap map[string]any
		if err := json.Unmarshal([]byte(line), &logMap); err != nil {
			t.Errorf("Failed to parse line %d as JSON: %v", i+1, err)
		}
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		minLevel  LogLevel
		logLevel  LogLevel
		shouldLog bool
	}{
		{"Debug logs at debug level", LogLevelDebug, LogLevelDebug, true},
		{"Info logs at debug level", LogLevelDebug, LogLevelInfo, true},
		{"Debug blocked at info level", LogLevelInfo, LogLevelDebug, false},
		{"Info logs at info level", LogLevelInfo, LogLevelInfo, true},
		{"Error logs at info level", LogLevelInfo, LogLevelError, true},
		{"Debug blocked at error level", LogLevelError, LogLevelDebug, false},
		{"Info blocked at error level", LogLevelError, LogLevelInfo, false},
		{"Error logs at error level", LogLevelError, LogLevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{
				output:   &buf,
				minLevel: tt.minLevel,
			}

			// Log based on the test level
			switch tt.logLevel {
			case LogLevelDebug:
				logger.Debug("Test message")
			case LogLevelInfo:
				logger.Info("Test message")
			case LogLevelWarn:
				logger.Warn("Test message")
			case LogLevelError:
				logger.Error("Test message")
			}

			logged := buf.Len() > 0
			if logged != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v, but logged=%v", tt.shouldLog, logged)
			}
		})
	}
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelInfo,
	}

	// Debug should be filtered
	logger.Debug("Debug message")
	if buf.Len() > 0 {
		t.Error("Debug message should be filtered at Info level")
	}

	// Change to debug level
	logger.SetLevel(LogLevelDebug)
	logger.Debug("Debug message after level change")
	if buf.Len() == 0 {
		t.Error("Debug message should be logged after setting level to Debug")
	}
}

func TestLogger_FieldConflicts(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Test that conflicting fields are prefixed with underscore
	logger.Info("Test message", map[string]any{
		"time":    "custom_time",
		"level":   "custom_level",
		"message": "custom_message",
		"safe":    "safe_value",
	})

	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Core fields should have their proper values
	if logMap["message"] != "Test message" {
		t.Errorf("Expected message='Test message', got '%v'", logMap["message"])
	}
	if logMap["level"] != "info" {
		t.Errorf("Expected level='info', got '%v'", logMap["level"])
	}

	// Conflicting fields should be prefixed with underscore
	if logMap["_level"] != "custom_level" {
		t.Errorf("Expected _level='custom_level', got '%v'", logMap["_level"])
	}
	if logMap["_message"] != "custom_message" {
		t.Errorf("Expected _message='custom_message', got '%v'", logMap["_message"])
	}
	// 'time' is no longer a reserved field and should appear as-is
	if logMap["time"] != "custom_time" {
		t.Errorf("Expected time='custom_time', got '%v'", logMap["time"])
	}

	// Non-conflicting fields should be added as-is
	if logMap["safe"] != "safe_value" {
		t.Errorf("Expected safe='safe_value', got '%v'", logMap["safe"])
	}
}

func TestLogger_FieldOrder(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Log with additional fields
	logger.Info("Test message", map[string]any{
		"zebra": "last",
		"alpha": "first",
		"extra": 123,
	})

	jsonStr := strings.TrimSpace(buf.String())

	// Find positions of the core fields
	levelPos := strings.Index(jsonStr, `"level":`)
	messagePos := strings.Index(jsonStr, `"message":`)

	// Verify they exist
	if levelPos == -1 || messagePos == -1 {
		t.Fatalf("Missing core fields in JSON output: %s", jsonStr)
	}

	// Verify they appear in the correct order: level < message
	if levelPos >= messagePos {
		t.Errorf("Fields not in correct order. Positions: level=%d, message=%d. JSON: %s",
			levelPos, messagePos, jsonStr)
	}

	// Verify 'level' is the first field (position 1, after opening brace)
	if levelPos != 1 {
		t.Errorf("level should be the first field, but found at position %d. JSON: %s", levelPos, jsonStr)
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger()

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Verify default level is Info
	if logger.minLevel != LogLevelInfo {
		t.Errorf("Expected default level to be Info, got %s", logger.minLevel)
	}

	// Verify output is os.Stderr (can't compare directly, but check it's not nil)
	if logger.output == nil {
		t.Error("Expected output to be set, got nil")
	}
}

func TestNewLoggerWithLevel(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
	}{
		{"Debug level", LogLevelDebug},
		{"Info level", LogLevelInfo},
		{"Warn level", LogLevelWarn},
		{"Error level", LogLevelError},
		{"Fatal level", LogLevelFatal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLoggerWithLevel(tt.level)

			if logger == nil {
				t.Fatal("NewLoggerWithLevel() returned nil")
			}

			if logger.minLevel != tt.level {
				t.Errorf("Expected level to be %s, got %s", tt.level, logger.minLevel)
			}

			if logger.output == nil {
				t.Error("Expected output to be set, got nil")
			}
		})
	}
}

func TestLogger_UnmarshalableFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// Create a channel, which cannot be marshaled to JSON
	ch := make(chan int)

	logger.Info("Test message", map[string]any{
		"valid_field":   "should appear",
		"invalid_field": ch,
		"another_valid": 42,
	})

	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify valid fields are present
	if logMap["valid_field"] != "should appear" {
		t.Errorf("Expected valid_field to be present, got %v", logMap["valid_field"])
	}
	if logMap["another_valid"] != float64(42) {
		t.Errorf("Expected another_valid to be present, got %v", logMap["another_valid"])
	}

	// Verify invalid field was skipped
	if _, exists := logMap["invalid_field"]; exists {
		t.Error("Expected invalid_field to be skipped")
	}
}

func TestLogger_FatalLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		output:   &buf,
		minLevel: LogLevelDebug,
	}

	// We can't actually test Fatal/Fatalf without them calling os.Exit()
	// But we can test that Fatal level is filtered correctly using the log() method directly
	logger.log(LogLevelFatal, "Fatal message", map[string]any{"error": "critical"})

	var logMap map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &logMap); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if logMap["level"] != "fatal" {
		t.Errorf("Expected level 'fatal', got '%s'", logMap["level"])
	}
	if logMap["message"] != "Fatal message" {
		t.Errorf("Expected message 'Fatal message', got '%s'", logMap["message"])
	}
	if logMap["error"] != "critical" {
		t.Errorf("Expected error field to be 'critical', got '%v'", logMap["error"])
	}
}

// TestLogger_Fatal tests the Fatal method using subprocess pattern
func TestLogger_Fatal(t *testing.T) {
	if os.Getenv("TEST_FATAL") == "1" {
		// This is the subprocess that will call Fatal
		logger := &Logger{
			output:   os.Stdout,
			minLevel: LogLevelDebug,
		}
		logger.Fatal("Fatal error occurred", map[string]any{
			"code": 500,
		})
		return
	}

	// This is the parent test process
	cmd := exec.Command(os.Args[0], "-test.run=TestLogger_Fatal")
	cmd.Env = append(os.Environ(), "TEST_FATAL=1")

	// Capture stdout
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()

	// Fatal should cause exit with code 1
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		// Expected: process exited with non-zero status

		// Verify the log output was written
		output := stdout.String()
		if !strings.Contains(output, "Fatal error occurred") {
			t.Errorf("Expected error message in output, got: %s", output)
		}
		if !strings.Contains(output, "fatal") {
			t.Errorf("Expected 'fatal' level in output, got: %s", output)
		}
		if !strings.Contains(output, "500") {
			t.Errorf("Expected code field in output, got: %s", output)
		}
	} else {
		t.Errorf("Expected process to exit with error, got: %v", err)
	}
}

// TestLogger_Fatalf tests the Fatalf method using subprocess pattern
func TestLogger_Fatalf(t *testing.T) {
	if os.Getenv("TEST_FATALF") == "1" {
		// This is the subprocess that will call Fatalf
		logger := &Logger{
			output:   os.Stdout,
			minLevel: LogLevelDebug,
		}
		logger.Fatalf("Fatal error: %s (code: %d)", "connection failed", 500)
		return
	}

	// This is the parent test process
	cmd := exec.Command(os.Args[0], "-test.run=TestLogger_Fatalf")
	cmd.Env = append(os.Environ(), "TEST_FATALF=1")

	// Capture stdout
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()

	// Fatalf should cause exit with code 1
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		// Expected: process exited with non-zero status

		// Verify the log output was written
		output := stdout.String()
		if !strings.Contains(output, "Fatal error: connection failed (code: 500)") {
			t.Errorf("Expected formatted error message in output, got: %s", output)
		}
		if !strings.Contains(output, "fatal") {
			t.Errorf("Expected 'fatal' level in output, got: %s", output)
		}
	} else {
		t.Errorf("Expected process to exit with error, got: %v", err)
	}
}
