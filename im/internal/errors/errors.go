package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode 定义错误代码类型
type ErrorCode string

const (
	// 认证相关错误
	ErrInvalidDID        ErrorCode = "INVALID_DID"
	ErrInvalidToken      ErrorCode = "INVALID_TOKEN"
	ErrTokenExpired      ErrorCode = "TOKEN_EXPIRED"
	ErrUnauthorized      ErrorCode = "UNAUTHORIZED"
	
	// 用户相关错误
	ErrUserNotFound      ErrorCode = "USER_NOT_FOUND"
	ErrUserAlreadyExists ErrorCode = "USER_ALREADY_EXISTS"
	
	// 好友相关错误
	ErrFriendNotFound       ErrorCode = "FRIEND_NOT_FOUND"
	ErrFriendRequestExists  ErrorCode = "FRIEND_REQUEST_EXISTS"
	ErrAlreadyFriends      ErrorCode = "ALREADY_FRIENDS"
	ErrCannotAddSelf       ErrorCode = "CANNOT_ADD_SELF"
	ErrFriendBlocked       ErrorCode = "FRIEND_BLOCKED"
	
	// 消息相关错误
	ErrMessageNotFound     ErrorCode = "MESSAGE_NOT_FOUND"
	ErrInvalidMessage      ErrorCode = "INVALID_MESSAGE"
	ErrSessionExpired      ErrorCode = "SESSION_EXPIRED"
	ErrSessionNotFound     ErrorCode = "SESSION_NOT_FOUND"
	
	// 系统相关错误
	ErrInternalServer      ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrBadRequest          ErrorCode = "BAD_REQUEST"
	ErrValidation          ErrorCode = "VALIDATION_ERROR"
	ErrDatabase            ErrorCode = "DATABASE_ERROR"
)

// AppError 应用错误结构
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	HTTPStatus int       `json:"-"`
}

func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New 创建新的应用错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatus(code),
	}
}

// Newf 创建带格式化消息的应用错误
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: getHTTPStatus(code),
	}
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithDetailsf 添加格式化的错误详情
func (e *AppError) WithDetailsf(format string, args ...interface{}) *AppError {
	e.Details = fmt.Sprintf(format, args...)
	return e
}

// getHTTPStatus 根据错误代码获取HTTP状态码
func getHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrInvalidDID, ErrBadRequest, ErrValidation, ErrInvalidMessage:
		return http.StatusBadRequest
	case ErrInvalidToken, ErrTokenExpired, ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrUserNotFound, ErrFriendNotFound, ErrMessageNotFound, ErrSessionNotFound:
		return http.StatusNotFound
	case ErrUserAlreadyExists, ErrFriendRequestExists, ErrAlreadyFriends:
		return http.StatusConflict
	case ErrCannotAddSelf, ErrFriendBlocked:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// IsAppError 检查是否为应用错误
func IsAppError(err error) (*AppError, bool) {
	if appErr, ok := err.(*AppError); ok {
		return appErr, true
	}
	return nil, false
}

// WrapDatabaseError 包装数据库错误
func WrapDatabaseError(err error, operation string) *AppError {
	return &AppError{
		Code:       ErrDatabase,
		Message:    fmt.Sprintf("Database operation failed: %s", operation),
		Details:    err.Error(),
		HTTPStatus: http.StatusInternalServerError,
	}
}