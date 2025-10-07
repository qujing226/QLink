package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"qlink-im/internal/models"
	"qlink-im/internal/storage"
)

// EncryptionService 加密服务接口
type EncryptionService interface {
	// EncryptMessage 加密消息
	EncryptMessage(sessionID uint, plaintext string) (*models.MessageRequest, error)
	// DecryptMessage 解密消息
	DecryptMessage(sessionID uint, ciphertext, nonce string) (string, error)
	// DeriveSessionKey 从共享密钥派生会话密钥
	DeriveSessionKey(sharedKey []byte, userDID, friendDID string) []byte
	// GetOrCreateSession 获取或创建会话
	GetOrCreateSession(userDID, friendDID string) (*models.Session, error)
}

// encryptionService 加密服务实现
type encryptionService struct {
	storage storage.Storage
}

// NewEncryptionService 创建加密服务
func NewEncryptionService(storage storage.Storage) EncryptionService {
	return &encryptionService{
		storage: storage,
	}
}

// EncryptMessage 加密消息
func (s *encryptionService) EncryptMessage(sessionID uint, plaintext string) (*models.MessageRequest, error) {
	// 获取会话信息
	session, err := s.storage.GetSessionByID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// 解码会话密钥
	sessionKey, err := base64.StdEncoding.DecodeString(session.SessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session key: %w", err)
	}

	// 创建AES-GCM加密器
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密消息
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	return &models.MessageRequest{
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
	}, nil
}

// DecryptMessage 解密消息
func (s *encryptionService) DecryptMessage(sessionID uint, ciphertext, nonce string) (string, error) {
	// 获取会话信息
	session, err := s.storage.GetSessionByID(sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	// 解码会话密钥
	sessionKey, err := base64.StdEncoding.DecodeString(session.SessionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode session key: %w", err)
	}

	// 解码密文和nonce
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	// 创建AES-GCM解密器
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 解密消息
	plaintext, err := gcm.Open(nil, nonceBytes, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt message: %w", err)
	}

	return string(plaintext), nil
}

// DeriveSessionKey 从共享密钥派生会话密钥
func (s *encryptionService) DeriveSessionKey(sharedKey []byte, userDID, friendDID string) []byte {
	// 使用SHA-256派生会话密钥
	h := sha256.New()
	h.Write(sharedKey)
	h.Write([]byte(userDID))
	h.Write([]byte(friendDID))
	return h.Sum(nil)
}

// GetOrCreateSession 获取或创建会话
func (s *encryptionService) GetOrCreateSession(userDID, friendDID string) (*models.Session, error) {
	// 首先尝试获取现有会话
	session, err := s.storage.GetSession(userDID, friendDID)
	if err == nil {
		return session, nil
	}

	// 如果没有现有会话，检查是否有已完成的密钥交换
	keyExchange, err := s.storage.GetCompletedKeyExchange(userDID, friendDID)
	if err != nil {
		return nil, fmt.Errorf("no completed key exchange found: %w", err)
	}

	// 从密钥交换中获取共享密钥（这里需要实现从ciphertext中解封装共享密钥的逻辑）
	// 这是一个简化的实现，实际应该使用Kyber768解封装
	sharedKey := []byte(keyExchange.Ciphertext) // 简化处理
	
	// 派生会话密钥
	sessionKey := s.DeriveSessionKey(sharedKey, userDID, friendDID)

	// 创建新会话
	newSession := &models.Session{
		UserDID:    userDID,
		FriendDID:  friendDID,
		SessionKey: base64.StdEncoding.EncodeToString(sessionKey),
		KeyVersion: 1,
	}

	if err := s.storage.CreateSession(newSession); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return newSession, nil
}