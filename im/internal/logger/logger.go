package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// Logger 日志记录器
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

var defaultLogger *Logger

func init() {
	defaultLogger = New(INFO)
}

// New 创建新的日志记录器
func New(level LogLevel) *Logger {
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

// SetLevel 设置日志级别
func SetLevel(level LogLevel) {
	defaultLogger.level = level
}

// log 记录日志
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	// 获取调用者信息
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	filename := filepath.Base(file)
	levelName := levelNames[level]

	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] %s %s:%d %s", levelName, timestamp, filename, line, message)

	l.logger.Println(logLine)

	if level == FATAL {
		os.Exit(1)
	}
}

// Debug 记录调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info 记录信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn 记录警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error 记录错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

// Fatal 记录致命错误日志并退出程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FATAL, format, args...)
}

// 全局日志函数
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// WithFields 创建带字段的日志记录器（简化版本）
func WithFields(fields map[string]interface{}) *FieldLogger {
	return &FieldLogger{
		logger: defaultLogger,
		fields: fields,
	}
}

// FieldLogger 带字段的日志记录器
type FieldLogger struct {
	logger *Logger
	fields map[string]interface{}
}

func (fl *FieldLogger) formatMessage(format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)
	
	if len(fl.fields) > 0 {
		fieldsStr := ""
		for k, v := range fl.fields {
			if fieldsStr != "" {
				fieldsStr += " "
			}
			fieldsStr += fmt.Sprintf("%s=%v", k, v)
		}
		message = fmt.Sprintf("%s [%s]", message, fieldsStr)
	}
	
	return message
}

func (fl *FieldLogger) Debug(format string, args ...interface{}) {
	message := fl.formatMessage(format, args...)
	fl.logger.Debug("%s", message)
}

func (fl *FieldLogger) Info(format string, args ...interface{}) {
	message := fl.formatMessage(format, args...)
	fl.logger.Info("%s", message)
}

func (fl *FieldLogger) Warn(format string, args ...interface{}) {
	message := fl.formatMessage(format, args...)
	fl.logger.Warn("%s", message)
}

func (fl *FieldLogger) Error(format string, args ...interface{}) {
	message := fl.formatMessage(format, args...)
	fl.logger.Error("%s", message)
}

func (fl *FieldLogger) Fatal(format string, args ...interface{}) {
	message := fl.formatMessage(format, args...)
	fl.logger.Fatal("%s", message)
}