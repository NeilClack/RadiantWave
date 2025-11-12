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

	"radiantwavetech.com/radiantwave/internal/config"
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
// It must be called once at application startup before any calls to GetLogger.
// It opens the specified log file and sets up a multi-writer to output
// to both the file and standard error.
func InitLogger() error {
	config := config.Get()
	logDir := config.LogDir
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
		l := log.New(multiWriter, "", 0) // The built-in logger handles timestamps and prefixes.

		instance = &Logger{stdLogger: l, file: file}
		instance.Infof("Logger initialized. Logging to %s", logFilePath)
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
// typically with a defer statement in the main Run function.
func (l *Logger) Close() {
	if l.file != nil {
		l.Infof("Closing logger.")
		l.file.Close()
	}
}

// output is the internal method that formats and writes the log message.
func (l *Logger) output(level, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("%s [%s] %s", timestamp, level, message)
	l.stdLogger.Println(logEntry)
}

// Info logs a message with the INFO level.
func (l *Logger) Info(message string) {
	l.output("[DEPRECATED] INFO", message)
}

func LogInfo(message string) {
	instance.output("INFO", message)
}

// Infof logs a formatted message with the INFO level.
func (l *Logger) Infof(format string, args ...any) {
	l.output("[DEPRECATED] INFO", format, args...)
}

// LogInfoF is an attempt to create a function that does not require a logger pointer to utilize
func LogInfoF(format string, args ...any) {
	instance.output("INFO", format, args...)
}

// Warning logs a message with the WARNING level.
func (l *Logger) Warning(message string) {
	l.output("[DEPRECATED] WARNING", message)
}

func LogWarning(message string) {
	instance.output("WARNING", message)
}

// Warningf logs a formatted message with the WARNING level.
func (l *Logger) Warningf(format string, args ...any) {
	l.output("[DEPRECATED] WARNING", format, args...)
}

func LogWarningF(format string, args ...any) {
	instance.output("WARNING", format, args...)
}

// Error logs a message with the ERROR level.
func (l *Logger) Error(message string) {
	l.output("[DEPRECATED] ERROR", message)
}

func LogError(message string) {
	instance.output("ERROR", message)
}

// Errorf logs a formatted message with the ERROR level.
func (l *Logger) Errorf(format string, args ...any) {
	l.output("[DEPRECATED] ERROR", format, args...)
}

func LogErrorF(format string, args ...any) {
	instance.output("ERROR", format, args...)
}
