package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARN"
	LogLevelError   LogLevel = "ERROR"
	LogLevelFatal   LogLevel = "FATAL"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	ProcessName string                 `json:"process_name,omitempty"`
	ProcessPID  int                    `json:"process_pid,omitempty"`
}

// StructuredLogger provides thread-safe structured logging
type StructuredLogger struct {
	mu            sync.RWMutex
	entries       []LogEntry
	writers       []io.Writer
	component     string
	allowedErrors map[string]bool
	emergencyMode bool
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(component string) *StructuredLogger {
	return &StructuredLogger{
		entries:       make([]LogEntry, 0),
		writers:       make([]io.Writer, 0),
		component:     component,
		allowedErrors: make(map[string]bool),
	}
}

// AddWriter adds an output writer (file, stdout, etc)
func (sl *StructuredLogger) AddWriter(w io.Writer) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.writers = append(sl.writers, w)
}

// SetAllowedErrors sets the map of allowed error patterns
func (sl *StructuredLogger) SetAllowedErrors(errors map[string]bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.allowedErrors = errors
}

// SetAllowedErrorsFromMap converts a map[string]string to map[string]bool
func (sl *StructuredLogger) SetAllowedErrorsFromMap(errors map[string]string) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.allowedErrors = make(map[string]bool)
	for k := range errors {
		sl.allowedErrors[k] = true
	}
}

// SetEmergencyMode toggles emergency mode
func (sl *StructuredLogger) SetEmergencyMode(enabled bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.emergencyMode = enabled
}

// Log adds a structured log entry
func (sl *StructuredLogger) Log(level LogLevel, message string, fields map[string]interface{}) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: sl.component,
		Message:   message,
		Fields:    fields,
	}

	// Add error field if present
	if err, ok := fields["error"]; ok {
		if e, ok := err.(error); ok {
			entry.Error = e.Error()
			delete(fields, "error")
		}
	}

	sl.mu.Lock()
	defer sl.mu.Unlock()

	sl.entries = append(sl.entries, entry)

	// Write to all writers
	for _, w := range sl.writers {
		sl.writeEntry(w, entry)
	}
}

// writeEntry writes a log entry to a writer
func (sl *StructuredLogger) writeEntry(w io.Writer, entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(w, "Failed to marshal log entry: %v\n", err)
		return
	}
	fmt.Fprintf(w, "%s\n", data)
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(message string, fields ...map[string]interface{}) {
	f := sl.mergeFields(fields...)
	sl.Log(LogLevelDebug, message, f)
}

// Info logs an info message
func (sl *StructuredLogger) Info(message string, fields ...map[string]interface{}) {
	f := sl.mergeFields(fields...)
	sl.Log(LogLevelInfo, message, f)
}

// Warning logs a warning message
func (sl *StructuredLogger) Warning(message string, fields ...map[string]interface{}) {
	f := sl.mergeFields(fields...)
	sl.Log(LogLevelWarning, message, f)
}

// Error logs an error message
func (sl *StructuredLogger) Error(message string, err error, fields ...map[string]interface{}) {
	f := sl.mergeFields(fields...)
	if err != nil {
		f["error"] = err
	}
	sl.Log(LogLevelError, message, f)
}

// Fatal logs a fatal message
func (sl *StructuredLogger) Fatal(message string, err error, fields ...map[string]interface{}) {
	f := sl.mergeFields(fields...)
	if err != nil {
		f["error"] = err
	}
	sl.Log(LogLevelFatal, message, f)
}

// mergeFields merges multiple field maps
func (sl *StructuredLogger) mergeFields(fields ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

// GetErrors returns all error entries, optionally filtering out allowed errors
func (sl *StructuredLogger) GetErrors(includeAllowed bool) []LogEntry {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	var errors []LogEntry
	for _, entry := range sl.entries {
		if entry.Level == LogLevelError || entry.Level == LogLevelFatal {
			if includeAllowed || !sl.isAllowedError(entry) {
				errors = append(errors, entry)
			}
		}
	}
	return errors
}

// isAllowedError checks if an error is in the allowed list
func (sl *StructuredLogger) isAllowedError(entry LogEntry) bool {
	// Check normal allowed errors
	for pattern := range sl.allowedErrors {
		if strings.Contains(entry.Message, pattern) || strings.Contains(entry.Error, pattern) {
			return true
		}
	}

	// Check emergency mode allowed errors
	if sl.emergencyMode {
		for pattern := range allowedErrorsDuringEmergencyMode {
			if strings.Contains(entry.Message, pattern) || strings.Contains(entry.Error, pattern) {
				return true
			}
		}
	}

	return false
}

// GetEntriesByLevel returns all entries of a specific level
func (sl *StructuredLogger) GetEntriesByLevel(level LogLevel) []LogEntry {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	var entries []LogEntry
	for _, entry := range sl.entries {
		if entry.Level == level {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetEntriesByTimeRange returns entries within a time range
func (sl *StructuredLogger) GetEntriesByTimeRange(start, end time.Time) []LogEntry {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	var entries []LogEntry
	for _, entry := range sl.entries {
		if entry.Timestamp.After(start) && entry.Timestamp.Before(end) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Summary returns a summary of log entries
func (sl *StructuredLogger) Summary() map[string]int {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	summary := map[string]int{
		"total":          len(sl.entries),
		"debug":          0,
		"info":           0,
		"warning":        0,
		"error":          0,
		"fatal":          0,
		"allowed_errors": 0,
	}

	for _, entry := range sl.entries {
		switch entry.Level {
		case LogLevelDebug:
			summary["debug"]++
		case LogLevelInfo:
			summary["info"]++
		case LogLevelWarning:
			summary["warning"]++
		case LogLevelError:
			summary["error"]++
			if sl.isAllowedError(entry) {
				summary["allowed_errors"]++
			}
		case LogLevelFatal:
			summary["fatal"]++
		}
	}

	return summary
}

// ProcessLogger wraps a logger for a specific process
type ProcessLogger struct {
	*StructuredLogger
	processName string
	processPID  int
}

// NewProcessLogger creates a logger for a specific process
func NewProcessLogger(logger *StructuredLogger, processName string, pid int) *ProcessLogger {
	return &ProcessLogger{
		StructuredLogger: logger,
		processName:      processName,
		processPID:       pid,
	}
}

// Log adds process information to the log entry
func (pl *ProcessLogger) Log(level LogLevel, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["process_name"] = pl.processName
	fields["process_pid"] = pl.processPID

	pl.StructuredLogger.Log(level, message, fields)
}

// Write implements io.Writer interface for process output capture
func (pl *ProcessLogger) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Determine log level based on content
		level := LogLevelInfo
		if strings.Contains(line, " ERR ") || strings.Contains(line, "[Error]") {
			level = LogLevelError
		} else if strings.Contains(line, " WARN ") || strings.Contains(line, "[Warning]") {
			level = LogLevelWarning
		} else if strings.Contains(line, " DEBUG ") || strings.Contains(line, "[Debug]") {
			level = LogLevelDebug
		}

		pl.Log(level, line, nil)
	}

	return len(p), nil
}

// Example usage in tests:
func ExampleStructuredLogging() {
	// Create main logger
	logger := NewStructuredLogger("protocolE2E")
	logger.SetAllowedErrorsFromMap(allowedErrors)

	// Add file writer
	// logFile, _ := os.Create("test.log")
	// logger.AddWriter(logFile)

	// Create process-specific loggers
	lavadLogger := NewProcessLogger(logger, "lavad", 12345)
	providerLogger := NewProcessLogger(logger, "provider1", 12346)

	// Use the loggers (this prevents "declared and not used" errors)
	_ = lavadLogger
	_ = providerLogger

	// Use in commands
	// cmd.Stdout = lavadLogger
	// cmd.Stderr = lavadLogger

	// Log test events
	logger.Info("Starting test phase", map[string]interface{}{
		"phase":   "initialization",
		"timeout": "5m",
	})

	// Log errors with context
	logger.Error("Provider failed to start", fmt.Errorf("connection refused"), map[string]interface{}{
		"provider": "provider1",
		"address":  "127.0.0.1:2221",
		"retry":    3,
	})

	// Get summary at end of test
	summary := logger.Summary()
	if summary["error"] > summary["allowed_errors"] {
		utils.LavaFormatError("Test failed with errors", nil,
			utils.LogAttr("total_errors", summary["error"]),
			utils.LogAttr("allowed_errors", summary["allowed_errors"]))
	}
}
