package gateway

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger provides structured logging
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger(level string) *Logger {
	logLevel := parseLogLevel(level)

	return &Logger{
		level:  logLevel,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// Debug logs debug level messages
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", format, args...)
	}
}

// Info logs info level messages
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", format, args...)
	}
}

// Warn logs warning level messages
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		l.log("WARN", format, args...)
	}
}

// Error logs error level messages
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= ERROR {
		l.log("ERROR", format, args...)
	}
}

// log formats and outputs log messages
func (l *Logger) log(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	l.logger.Printf("[%s] [%s] %s", timestamp, level, message)
}

// ConnectionLogger provides connection-specific logging
type ConnectionLogger struct {
	*Logger
	connectionID string
}

// NewConnectionLogger creates a logger for a specific connection
func NewConnectionLogger(baseLogger *Logger, connectionID string) *ConnectionLogger {
	return &ConnectionLogger{
		Logger:       baseLogger,
		connectionID: connectionID,
	}
}

// LogConnection logs connection-specific messages
func (cl *ConnectionLogger) LogConnection(level, message string, args ...interface{}) {
	formattedMessage := fmt.Sprintf("[Connection: %s] %s", cl.connectionID, message)
	switch level {
	case "debug":
		cl.Debug(formattedMessage, args...)
	case "info":
		cl.Info(formattedMessage, args...)
	case "warn":
		cl.Warn(formattedMessage, args...)
	case "error":
		cl.Error(formattedMessage, args...)
	}
}
