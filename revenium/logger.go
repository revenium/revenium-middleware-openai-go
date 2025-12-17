package revenium

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger interface {
	Debug(message string, args ...interface{})
	Info(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Error(message string, args ...interface{})
}

type DefaultLogger struct {
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{}
}

func (l *DefaultLogger) Debug(message string, args ...interface{}) {
	if globalDebugEnabled || os.Getenv("REVENIUM_DEBUG") == "true" {
		l.log("Debug", message, args...)
	}
}

func (l *DefaultLogger) Info(message string, args ...interface{}) {
	l.log("", message, args...)
}

func (l *DefaultLogger) Warn(message string, args ...interface{}) {
	l.log("Warning", message, args...)
}

func (l *DefaultLogger) Error(message string, args ...interface{}) {
	l.log("Error", message, args...)
}

// log is the internal logging method
func (l *DefaultLogger) log(level, message string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	var prefix string
	if level == "" {
		prefix = fmt.Sprintf("[%s] [Revenium]", timestamp)
	} else {
		prefix = fmt.Sprintf("[%s] [Revenium %s]", timestamp, level)
	}

	if len(args) > 0 {
		message = fmt.Sprintf(message, args...)
	}

	log.Printf("%s %s", prefix, message)
}

var globalLogger Logger = NewDefaultLogger()
var globalDebugEnabled bool

func GetLogger() Logger {
	return globalLogger
}

func SetLogger(logger Logger) {
	globalLogger = logger
}

func Debug(message string, args ...interface{}) {
	globalLogger.Debug(message, args...)
}

func Info(message string, args ...interface{}) {
	globalLogger.Info(message, args...)
}

func Warn(message string, args ...interface{}) {
	globalLogger.Warn(message, args...)
}

func Error(message string, args ...interface{}) {
	globalLogger.Error(message, args...)
}

func SetGlobalDebug(enabled bool) {
	globalDebugEnabled = enabled
}
