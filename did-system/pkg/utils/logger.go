package utils

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志记录器
type Logger struct {
	level  LogLevel
	prefix string
	logger *log.Logger
}

// NewLogger 创建新的日志记录器
func NewLogger(prefix string, level LogLevel) *Logger {
	return &Logger{
		level:  level,
		prefix: prefix,
		logger: log.New(os.Stdout, "", 0),
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// formatMessage 格式化日志消息
func (l *Logger) formatMessage(level LogLevel, message string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 获取调用者信息
	_, file, line, ok := runtime.Caller(3)
	caller := "unknown"
	if ok {
		// 只保留文件名，不包含完整路径
		parts := strings.Split(file, "/")
		if len(parts) > 0 {
			caller = fmt.Sprintf("%s:%d", parts[len(parts)-1], line)
		}
	}

	prefix := l.prefix
	if prefix != "" {
		prefix = fmt.Sprintf("[%s] ", prefix)
	}

	return fmt.Sprintf("%s %s%s [%s] %s",
		timestamp, prefix, level.String(), caller, message)
}

// shouldLog 检查是否应该记录此级别的日志
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// Debug 记录调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.shouldLog(LogLevelDebug) {
		message := fmt.Sprintf(format, args...)
		l.logger.Println(l.formatMessage(LogLevelDebug, message))
	}
}

// Info 记录信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	if l.shouldLog(LogLevelInfo) {
		message := fmt.Sprintf(format, args...)
		l.logger.Println(l.formatMessage(LogLevelInfo, message))
	}
}

// Warn 记录警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.shouldLog(LogLevelWarn) {
		message := fmt.Sprintf(format, args...)
		l.logger.Println(l.formatMessage(LogLevelWarn, message))
	}
}

// Error 记录错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	if l.shouldLog(LogLevelError) {
		message := fmt.Sprintf(format, args...)
		l.logger.Println(l.formatMessage(LogLevelError, message))
	}
}

// Fatal 记录致命错误日志并退出程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logger.Println(l.formatMessage(LogLevelFatal, message))
	os.Exit(1)
}

// ErrorWithStack 记录带堆栈信息的错误日志
func (l *Logger) ErrorWithStack(err error, format string, args ...interface{}) {
	if !l.shouldLog(LogLevelError) {
		return
	}

	message := fmt.Sprintf(format, args...)
	if err != nil {
		message = fmt.Sprintf("%s: %v", message, err)

		// 如果是AppError，包含堆栈信息
		if appErr, ok := err.(*AppError); ok && appErr.Stack != "" {
			message = fmt.Sprintf("%s\nStack trace:\n%s", message, appErr.Stack)
		}
	}

	l.logger.Println(l.formatMessage(LogLevelError, message))
}

// LogOperation 记录操作日志
func (l *Logger) LogOperation(operation, nodeID string, duration time.Duration, err error) {
	if err != nil {
		l.Error("操作失败 - 操作: %s, 节点: %s, 耗时: %v, 错误: %v",
			operation, nodeID, duration, err)
	} else {
		l.Info("操作成功 - 操作: %s, 节点: %s, 耗时: %v",
			operation, nodeID, duration)
	}
}

// LogRequest 记录请求日志
func (l *Logger) LogRequest(method, path, clientIP string, statusCode int, duration time.Duration) {
	l.Info("请求处理 - %s %s [%s] %d %v",
		method, path, clientIP, statusCode, duration)
}

// LogSync 记录同步日志
func (l *Logger) LogSync(nodeID, peerID string, syncType string, success bool, duration time.Duration) {
	if success {
		l.Info("同步成功 - 节点: %s, 对等节点: %s, 类型: %s, 耗时: %v",
			nodeID, peerID, syncType, duration)
	} else {
		l.Error("同步失败 - 节点: %s, 对等节点: %s, 类型: %s, 耗时: %v",
			nodeID, peerID, syncType, duration)
	}
}

// LogConsensus 记录共识日志
func (l *Logger) LogConsensus(nodeID string, term int64, action string, success bool) {
	if success {
		l.Info("共识操作成功 - 节点: %s, 任期: %d, 操作: %s",
			nodeID, term, action)
	} else {
		l.Error("共识操作失败 - 节点: %s, 任期: %d, 操作: %s",
			nodeID, term, action)
	}
}

// 全局日志记录器实例
var (
	defaultLogger = NewLogger("QLink", LogLevelInfo)

	// 各模块的专用日志记录器
	ConsensusLogger = NewLogger("CONSENSUS", LogLevelInfo)
	SyncLogger      = NewLogger("SYNC", LogLevelInfo)
	NetworkLogger   = NewLogger("NETWORK", LogLevelInfo)
	DIDLogger       = NewLogger("DID", LogLevelInfo)
	ClusterLogger   = NewLogger("CLUSTER", LogLevelInfo)
)

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

func ErrorWithStack(err error, format string, args ...interface{}) {
	defaultLogger.ErrorWithStack(err, format, args...)
}

// SetGlobalLogLevel 设置全局日志级别
func SetGlobalLogLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
	ConsensusLogger.SetLevel(level)
	SyncLogger.SetLevel(level)
	NetworkLogger.SetLevel(level)
	DIDLogger.SetLevel(level)
	ClusterLogger.SetLevel(level)
}

// ParseLogLevel 解析日志级别字符串
func ParseLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}
