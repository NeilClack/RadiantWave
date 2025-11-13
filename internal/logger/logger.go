// Package logger provides a simple, singleton utility for logging to both the database and standard output.
package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Logger is configured to write log messages to the database and console.
type Logger struct {
	// db holds the database connection for writing log entries
	db *gorm.DB
	// consoleLogger handles console output
	consoleLogger *log.Logger
}

var (
	instance *Logger
	once     sync.Once
)

// InitLogger initializes the singleton Logger instance.
// It must be called once at application startup before any calls to Get.
// It accepts a gorm.DB instance for writing logs to the database.
func InitLogger(db *gorm.DB) error {
	var err error
	once.Do(func() {
		if db == nil {
			err = fmt.Errorf("database connection cannot be nil")
			return
		}

		// Create console logger for stderr output
		consoleLogger := log.New(os.Stderr, "", 0)

		instance = &Logger{
			db:            db,
			consoleLogger: consoleLogger,
		}

		InfoF("Logger initialized with database backend")
	})
	return err
}

// Get returns the singleton Logger instance.
// It will panic if InitLogger has not been called first.
func Get() *Logger {
	if instance == nil {
		panic("Logger has not been initialized. Call InitLogger at application startup.")
	}
	return instance
}

// Close is now a no-op since we're using the database connection
// which is managed separately. Kept for backward compatibility.
func (l *Logger) Close() error {
	InfoF("Logger closed")
	return nil
}

// LogEntry represents a log entry in the database
// This mirrors the db.LogEntry struct to avoid circular imports
type LogEntry struct {
	Timestamp string
	Level     string
	Message   string
}

// output is the internal method that formats, writes to console, and saves to database.
func (l *Logger) output(level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now()

	// Format timestamp with milliseconds: 2006-01-02 15:04:05.000
	timestampStr := timestamp.Format("2006-01-02 15:04:05.000")

	// Format for console output
	logLine := fmt.Sprintf("%s [%s] %s", timestampStr, level, message)
	l.consoleLogger.Println(logLine)

	// Write to database asynchronously to avoid blocking
	go func() {
		entry := map[string]interface{}{
			"timestamp": timestampStr,
			"level":     level,
			"message":   message,
		}

		// Use Table() to avoid needing to import db package
		if err := l.db.Table("log_entries").Create(entry).Error; err != nil {
			// Log database errors to console only to avoid infinite recursion
			l.consoleLogger.Printf("%s [ERROR] Failed to write log to database: %v\n",
				time.Now().Format("2006-01-02 15:04:05.000"), err)
		}
	}()
}

// Debug logs a debug-level message
func Debug(message string) {
	Get().output("DEBUG", "%s", message)
}

// DebugF logs a formatted debug-level message
func DebugF(format string, args ...any) {
	Get().output("DEBUG", format, args...)
}

// Info logs an info-level message
func Info(message string) {
	Get().output("INFO", "%s", message)
}

// InfoF logs a formatted info-level message
func InfoF(format string, args ...any) {
	Get().output("INFO", format, args...)
}

// Warning logs a warning-level message
func Warning(message string) {
	Get().output("WARNING", "%s", message)
}

// WarningF logs a formatted warning-level message
func WarningF(format string, args ...any) {
	Get().output("WARNING", format, args...)
}

// Error logs an error-level message
func Error(message string) {
	Get().output("ERROR", "%s", message)
}

// ErrorF logs a formatted error-level message
func ErrorF(format string, args ...any) {
	Get().output("ERROR", format, args...)
}

// Fatal logs a fatal-level message and exits the program
func Fatal(message string) {
	Get().output("FATAL", "%s", message)
	// Give goroutine a moment to write to database before exiting
	time.Sleep(100 * time.Millisecond)
	os.Exit(1)
}

// FatalF logs a formatted fatal-level message and exits the program
func FatalF(format string, args ...any) {
	Get().output("FATAL", format, args...)
	// Give goroutine a moment to write to database before exiting
	time.Sleep(100 * time.Millisecond)
	os.Exit(1)
}
