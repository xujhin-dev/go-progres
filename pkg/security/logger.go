package security

import (
	"log"
)

// SecurityLogger 安全日志接口
type SecurityLogger interface {
	Log(event SecurityEvent) error
	LogError(message string, fields ...interface{})
	LogWarning(message string, fields ...interface{})
	LogInfo(message string, fields ...interface{})
	Critical(message string, fields ...interface{})
	Error(message string, fields ...interface{})
	Warn(message string, fields ...interface{})
	Info(message string, fields ...interface{})
}

// DefaultSecurityLogger 默认安全日志实现
type DefaultSecurityLogger struct {
	logger *log.Logger
}

// NewDefaultSecurityLogger 创建默认安全日志器
func NewDefaultSecurityLogger() *DefaultSecurityLogger {
	return &DefaultSecurityLogger{
		logger: log.Default(),
	}
}

// Log 记录安全事件
func (l *DefaultSecurityLogger) Log(event SecurityEvent) error {
	l.logger.Printf("[%s] %s - %s from %s at %s",
		event.Level, event.Type, event.Message, event.IP, event.Timestamp)
	return nil
}

// LogError 记录错误
func (l *DefaultSecurityLogger) LogError(message string, fields ...interface{}) {
	l.logger.Printf("[ERROR] %s %v", message, fields)
}

// LogWarning 记录警告
func (l *DefaultSecurityLogger) LogWarning(message string, fields ...interface{}) {
	l.logger.Printf("[WARNING] %s %v", message, fields)
}

// LogInfo 记录信息
func (l *DefaultSecurityLogger) LogInfo(message string, fields ...interface{}) {
	l.logger.Printf("[INFO] %s %v", message, fields)
}

// Critical 记录严重错误
func (l *DefaultSecurityLogger) Critical(message string, fields ...interface{}) {
	l.logger.Printf("[CRITICAL] %s %v", message, fields)
}

// Error 记录错误
func (l *DefaultSecurityLogger) Error(message string, fields ...interface{}) {
	l.logger.Printf("[ERROR] %s %v", message, fields)
}

// Warn 记录警告
func (l *DefaultSecurityLogger) Warn(message string, fields ...interface{}) {
	l.logger.Printf("[WARNING] %s %v", message, fields)
}

// Info 记录信息
func (l *DefaultSecurityLogger) Info(message string, fields ...interface{}) {
	l.logger.Printf("[INFO] %s %v", message, fields)
}
