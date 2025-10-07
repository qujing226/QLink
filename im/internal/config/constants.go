package config

import "time"

// 服务器配置常量
const (
	// 默认服务器配置
	DefaultServerPort = "8082"
	DefaultServerHost = "0.0.0.0"
	
	// 数据库配置
	DefaultDatabasePath = "./qlink.db"
	DefaultDatabaseType = "sqlite"
	
	// JWT配置
	DefaultJWTExpiration = 24 * time.Hour
	DefaultJWTIssuer     = "qlink-im"
	
	// WebSocket配置
	DefaultWebSocketReadBufferSize  = 1024
	DefaultWebSocketWriteBufferSize = 1024
	DefaultWebSocketPingPeriod      = 54 * time.Second
	DefaultWebSocketPongWait        = 60 * time.Second
	DefaultWebSocketWriteWait       = 10 * time.Second
	
	// 消息配置
	DefaultMessageChannelSize = 256
	DefaultMaxMessageSize     = 512
	
	// 加密配置
	DefaultAESKeySize = 32 // AES-256
	DefaultNonceSize  = 12 // GCM nonce size
	
	// 速率限制配置
	DefaultRateLimit       = 100  // 每分钟请求数
	DefaultRateLimitBurst  = 10   // 突发请求数
	DefaultRateLimitWindow = time.Minute
	
	// DID配置
	DefaultDIDPrefix = "did:qlink:"
	DefaultDIDMethod = "qlink"
	
	// 日志配置
	DefaultLogLevel = "info"
	DefaultLogFile  = "./logs/qlink-im.log"
	
	// 健康检查配置
	DefaultHealthCheckPath = "/health"
	
	// API配置
	DefaultAPIVersion = "v1"
	DefaultAPIPrefix  = "/api"
	
	// CORS配置
	DefaultCORSMaxAge = 12 * time.Hour
)

// 错误消息常量
const (
	ErrMsgInvalidDID        = "Invalid DID format"
	ErrMsgUserNotFound      = "User not found"
	ErrMsgUnauthorized      = "Unauthorized access"
	ErrMsgInternalServer    = "Internal server error"
	ErrMsgBadRequest        = "Bad request"
	ErrMsgForbidden         = "Forbidden"
	ErrMsgNotFound          = "Resource not found"
	ErrMsgConflict          = "Resource conflict"
	ErrMsgTooManyRequests   = "Too many requests"
	ErrMsgServiceUnavailable = "Service unavailable"
)

// 状态常量
const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusAway    = "away"
	StatusBusy    = "busy"
)

// 消息类型常量
const (
	MessageTypeText     = "text"
	MessageTypeImage    = "image"
	MessageTypeFile     = "file"
	MessageTypeSystem   = "system"
	MessageTypePing     = "ping"
	MessageTypePong     = "pong"
	MessageTypeTyping   = "typing"
	MessageTypeRead     = "read"
	MessageTypeDelivered = "delivered"
)

// 好友状态常量
const (
	FriendStatusPending  = "pending"
	FriendStatusAccepted = "accepted"
	FriendStatusRejected = "rejected"
	FriendStatusBlocked  = "blocked"
)

// 环境变量键名
const (
	EnvServerPort     = "SERVER_PORT"
	EnvServerHost     = "SERVER_HOST"
	EnvDatabaseURL    = "DATABASE_URL"
	EnvJWTSecret      = "JWT_SECRET"
	EnvDIDNodeURL     = "DID_NODE_URL"
	EnvLogLevel       = "LOG_LEVEL"
	EnvLogFile        = "LOG_FILE"
	EnvCORSOrigins    = "CORS_ORIGINS"
	EnvRateLimit      = "RATE_LIMIT"
	EnvEnvironment    = "ENVIRONMENT"
)

// 环境类型
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
	EnvTesting     = "testing"
)