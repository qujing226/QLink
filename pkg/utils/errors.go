package utils

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorType 错误类型
type ErrorType string

const (
	ErrorTypeValidation    ErrorType = "VALIDATION"
	ErrorTypeNotFound      ErrorType = "NOT_FOUND"
	ErrorTypeConflict      ErrorType = "CONFLICT"
	ErrorTypeUnauthorized  ErrorType = "UNAUTHORIZED"
	ErrorTypeInternal      ErrorType = "INTERNAL"
	ErrorTypeNetwork       ErrorType = "NETWORK"
	ErrorTypeTimeout       ErrorType = "TIMEOUT"
	ErrorTypeInvalidFormat ErrorType = "INVALID_FORMAT"
	ErrorTypeBlockchain    ErrorType = "BLOCKCHAIN"
)

// AppError 应用程序错误
type AppError struct {
	Type    ErrorType `json:"type"`
	Code    string    `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Cause   error     `json:"-"`
	Stack   string    `json:"stack,omitempty"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s:%s] %s - %s", e.Type, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Type, e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewError 创建新的应用程序错误
func NewError(errorType ErrorType, code, message string) *AppError {
	return &AppError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Stack:   getStackTrace(),
	}
}

// NewErrorWithCause 创建带原因的应用程序错误
func NewErrorWithCause(errorType ErrorType, code, message string, cause error) *AppError {
	return &AppError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Cause:   cause,
		Stack:   getStackTrace(),
	}
}

// NewErrorWithDetails 创建带详细信息的应用程序错误
func NewErrorWithDetails(errorType ErrorType, code, message, details string) *AppError {
	return &AppError{
		Type:    errorType,
		Code:    code,
		Message: message,
		Details: details,
		Stack:   getStackTrace(),
	}
}

// 预定义的常见错误
var (
	// 验证错误
	ErrInvalidDID     = NewError(ErrorTypeValidation, "INVALID_DID", "DID格式无效")
	ErrInvalidNodeID  = NewError(ErrorTypeValidation, "INVALID_NODE_ID", "节点ID无效")
	ErrInvalidAddress = NewError(ErrorTypeValidation, "INVALID_ADDRESS", "地址格式无效")
	ErrRequiredField  = NewError(ErrorTypeValidation, "REQUIRED_FIELD", "必填字段缺失")

	// 未找到错误
	ErrDIDNotFound      = NewError(ErrorTypeNotFound, "DID_NOT_FOUND", "DID不存在")
	ErrNodeNotFound     = NewError(ErrorTypeNotFound, "NODE_NOT_FOUND", "节点不存在")
	ErrResourceNotFound = NewError(ErrorTypeNotFound, "RESOURCE_NOT_FOUND", "资源不存在")

	// 冲突错误
	ErrDIDExists    = NewError(ErrorTypeConflict, "DID_EXISTS", "DID已存在")
	ErrNodeExists   = NewError(ErrorTypeConflict, "NODE_EXISTS", "节点已存在")
	ErrConflictData = NewError(ErrorTypeConflict, "CONFLICT_DATA", "数据冲突")

	// 内部错误
	ErrInternalServer     = NewError(ErrorTypeInternal, "INTERNAL_SERVER", "内部服务器错误")
	ErrDatabaseError      = NewError(ErrorTypeInternal, "DATABASE_ERROR", "数据库错误")
	ErrSerializationError = NewError(ErrorTypeInternal, "SERIALIZATION_ERROR", "序列化错误")

	// 网络错误
	ErrNetworkTimeout   = NewError(ErrorTypeNetwork, "NETWORK_TIMEOUT", "网络超时")
	ErrConnectionFailed = NewError(ErrorTypeNetwork, "CONNECTION_FAILED", "连接失败")
	ErrPeerNotReachable = NewError(ErrorTypeNetwork, "PEER_NOT_REACHABLE", "节点不可达")
)

// WrapError 包装错误
func WrapError(err error, errorType ErrorType, code, message string) *AppError {
	if err == nil {
		return nil
	}

	// 如果已经是AppError，直接返回
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return NewErrorWithCause(errorType, code, message, err)
}

// WrapValidationError 包装验证错误
func WrapValidationError(err error, field string) *AppError {
	return WrapError(err, ErrorTypeValidation, "VALIDATION_FAILED",
		fmt.Sprintf("字段 %s 验证失败", field))
}

// WrapNotFoundError 包装未找到错误
func WrapNotFoundError(err error, resource string) *AppError {
	return WrapError(err, ErrorTypeNotFound, "RESOURCE_NOT_FOUND",
		fmt.Sprintf("资源 %s 未找到", resource))
}

// WrapInternalError 包装内部错误
func WrapInternalError(err error, operation string) *AppError {
	return WrapError(err, ErrorTypeInternal, "OPERATION_FAILED",
		fmt.Sprintf("操作 %s 失败", operation))
}

// IsErrorType 检查错误类型
func IsErrorType(err error, errorType ErrorType) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type == errorType
	}
	return false
}

// IsErrorCode 检查错误代码
func IsErrorCode(err error, code string) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// GetErrorType 获取错误类型
func GetErrorType(err error) ErrorType {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Type
	}
	return ErrorTypeInternal
}

// GetErrorCode 获取错误代码
func GetErrorCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return "UNKNOWN"
}

// getStackTrace 获取堆栈跟踪
func getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])

	var builder strings.Builder
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		builder.WriteString(fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function))

		if !more {
			break
		}
	}

	return builder.String()
}

// HandlePanic 处理panic并转换为错误
func HandlePanic() error {
	if r := recover(); r != nil {
		stack := getStackTrace()
		return &AppError{
			Type:    ErrorTypeInternal,
			Code:    "PANIC",
			Message: fmt.Sprintf("发生panic: %v", r),
			Stack:   stack,
		}
	}
	return nil
}

// SafeExecute 安全执行函数，捕获panic
func SafeExecute(fn func() error) (err error) {
	defer func() {
		if panicErr := HandlePanic(); panicErr != nil {
			err = panicErr
		}
	}()

	return fn()
}
