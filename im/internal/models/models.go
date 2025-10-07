package models

import (
	"time"
)

// User 用户模型
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	DID       string    `json:"did" gorm:"column:d_id;uniqueIndex;not null"`
	PublicKey string    `json:"public_key" gorm:"not null"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Status    string    `json:"status" gorm:"default:offline"` // online, offline, away
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Friend 好友关系模型
type Friend struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserDID    string    `json:"user_did" gorm:"not null"`
	FriendDID  string    `json:"friend_did" gorm:"not null"`
	Status     string    `json:"status" gorm:"default:pending"` // pending, accepted, blocked
	RequestMsg string    `json:"request_msg"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Session 会话模型
type Session struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserDID      string    `json:"user_did" gorm:"not null"`
	FriendDID    string    `json:"friend_did" gorm:"not null"`
	SessionKey   string    `json:"session_key"` // 加密存储的会话密钥
	KeyVersion   int       `json:"key_version" gorm:"default:1"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastActivity time.Time `json:"last_activity"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Message 消息模型
type Message struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	From       string    `json:"from" gorm:"not null"`
	To         string    `json:"to" gorm:"not null"`
	SessionID  uint      `json:"session_id"`
	Ciphertext string    `json:"ciphertext" gorm:"not null"`
	Nonce      string    `json:"nonce" gorm:"not null"`
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status" gorm:"default:sent"` // sent, delivered, read
	CreatedAt  time.Time `json:"created_at"`
}

// Challenge 认证质询模型
type Challenge struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	From      string    `json:"from" gorm:"not null"`
	To        string    `json:"to" gorm:"not null"`
	Nonce     string    `json:"nonce" gorm:"not null"`
	Signature string    `json:"signature"`
	Status    string    `json:"status" gorm:"default:pending"` // pending, completed, failed
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// KeyExchange Kyber768密钥交换模型
type KeyExchange struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	From       string    `json:"from" gorm:"not null"`
	To         string    `json:"to" gorm:"not null"`
	Ciphertext string    `json:"ciphertext"` // Kyber768 encapsulation结果
	Status     string    `json:"status" gorm:"default:pending"` // pending, completed, failed
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// LoginRequest 登录请求
// LoginRequest 登录请求
type LoginRequest struct {
	DID       string `json:"did" binding:"required"`
	Signature string `json:"signature" binding:"required"`
	PublicKey string `json:"public_key,omitempty"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	User      User   `json:"user"`
}

// FriendRequest 好友请求
type FriendRequest struct {
	FriendDID string `json:"friend_did" binding:"required"`
	Message   string `json:"message,omitempty"`
}

// MessageRequest 发送消息请求
type MessageRequest struct {
	To         string `json:"to" binding:"required"`
	Ciphertext string `json:"ciphertext" binding:"required"`
	Nonce      string `json:"nonce" binding:"required"`
}

// ChallengeRequest 认证质询请求
type ChallengeRequest struct {
	DID string `json:"did" binding:"required"`
}

// ChallengeResponse 认证质询响应
type ChallengeResponse struct {
	DID       string `json:"did" binding:"required"`
	Challenge string `json:"challenge" binding:"required"`
	Signature string `json:"signature" binding:"required"`
}

// KeyExchangeRequest 密钥交换请求
type KeyExchangeRequest struct {
	TargetDID  string `json:"target_did" binding:"required"`
	Ciphertext string `json:"ciphertext" binding:"required"`
}