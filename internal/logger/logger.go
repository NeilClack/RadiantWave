// Package logger provides a simple, singleton utility for logging to both a file and standard output.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger is configured to write log messages to multiple destinations.
type Logger struct {
	// stdLogger holds the underlying standard library logger.
	stdLogger *log.Logger
	// file is a reference to the log file, kept so it can be closed.
	file *os.File
}

var (
	instance *Logger
	once     sync.Once
)

// InitLogger initializes the singleton Logger instance.
// It must be called once at application startup before any calls to Get.
// It opens the specified log file and sets up a multi-writer to output
// to both the file and standard error.
func InitLogger(logDir string) error {
	var err error
	once.Do(func() {
		logFilePath := filepath.Join(logDir, ".radiantwave.log")
		file, e := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if e != nil {
			err = fmt.Errorf("failed to open log file %s: %w", logFilePath, e)
			return
		}

		// Create a multi-writer to log to both the file and the console (os.Stderr).
		multiWriter := io.MultiWriter(file, os.Stderr)
		stdLogger := log.New(multiWriter, "", 0)

		instance = &Logger{stdLogger: stdLogger, file: file}
		InfoF("Logger initialized. Logging to %s", logFilePath)
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

// Close closes the log file. It should be called when the logger is no longer needed,
// typically with a defer statement in the main function.
func (l *Logger) Close() error {
	if l.file != nil {
		InfoF("Closing logger.")
		return l.file.Close()
	}
	return nil
}

// output is the internal method that formats and writes the log message.
func (l *Logger) output(level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("%s [%s] %s", timestamp, level, message)
	l.stdLogger.Println(logEntry)
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
	os.Exit(1)
}

// FatalF logs a formatted fatal-level message and exits the program
func FatalF(format string, args ...any) {
	Get().output("FATAL", format, args...)
	os.Exit(1)
}
