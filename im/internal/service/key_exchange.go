package service

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"qlink-im/internal/models"
	"qlink-im/internal/storage"
	"strconv"
	"time"
)

type KeyExchangeService interface {
    InitiateKeyExchange(fromDID, toDID, ciphertext string) (*models.KeyExchange, error)
    CompleteKeyExchange(userDID, keyExchangeID string) error
    GetPendingKeyExchanges(userDID string) ([]*models.KeyExchange, error)
    // 生成共享密钥（当前版本不再使用 Kyber，改为确定性派生）
    GenerateSharedKey(fromDID, toDID string) ([]byte, error)
}

type keyExchangeService struct {
	storage storage.Storage
}

func NewKeyExchangeService(storage storage.Storage) KeyExchangeService {
	return &keyExchangeService{
		storage: storage,
	}
}

// InitiateKeyExchange 发起密钥交换
func (s *keyExchangeService) InitiateKeyExchange(fromDID, toDID, ciphertext string) (*models.KeyExchange, error) {
	// 检查目标用户是否存在
	_, err := s.storage.GetUserByDID(toDID)
	if err != nil {
		return nil, fmt.Errorf("目标用户不存在: %w", err)
	}

	// 创建密钥交换记录
	keyExchange := &models.KeyExchange{
		From:       fromDID,
		To:         toDID,
		Ciphertext: ciphertext,
		Status:     "pending",
		ExpiresAt:  time.Now().Add(24 * time.Hour), // 24小时过期
		CreatedAt:  time.Now(),
	}

	err = s.storage.CreateKeyExchange(keyExchange)
	if err != nil {
		return nil, fmt.Errorf("创建密钥交换记录失败: %w", err)
	}

	return keyExchange, nil
}

// CompleteKeyExchange 完成密钥交换
func (s *keyExchangeService) CompleteKeyExchange(userDID, keyExchangeIDStr string) error {
	keyExchangeID, err := strconv.ParseUint(keyExchangeIDStr, 10, 32)
	if err != nil {
		return fmt.Errorf("无效的密钥交换ID: %w", err)
	}

	keyExchange, err := s.storage.GetKeyExchange(uint(keyExchangeID))
	if err != nil {
		return fmt.Errorf("获取密钥交换记录失败: %w", err)
	}

	// 验证用户权限
	if keyExchange.To != userDID {
		return fmt.Errorf("无权限完成此密钥交换")
	}

	// 检查是否已过期
	if time.Now().After(keyExchange.ExpiresAt) {
		return fmt.Errorf("密钥交换已过期")
	}

	// 更新状态为已完成
	keyExchange.Status = "completed"
	err = s.storage.UpdateKeyExchange(keyExchange)
	if err != nil {
		return fmt.Errorf("更新密钥交换状态失败: %w", err)
	}

	// 生成会话密钥并创建会话
	sharedKey, err := s.GenerateSharedKey(keyExchange.From, keyExchange.To)
	if err != nil {
		return fmt.Errorf("生成共享密钥失败: %w", err)
	}

	// 创建会话
	session := &models.Session{
		UserDID:      keyExchange.From,
		FriendDID:    keyExchange.To,
		SessionKey:   base64.StdEncoding.EncodeToString(sharedKey),
		KeyVersion:   1,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7天过期
		LastActivity: time.Now(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = s.storage.CreateSession(session)
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}

	return nil
}

// GetPendingKeyExchanges 获取待处理的密钥交换
func (s *keyExchangeService) GetPendingKeyExchanges(userDID string) ([]*models.KeyExchange, error) {
	return s.storage.GetPendingKeyExchanges(userDID)
}

// GenerateSharedKey 生成共享密钥（确定性派生，不依赖 Kyber）
func (s *keyExchangeService) GenerateSharedKey(fromDID, toDID string) ([]byte, error) {
    // 使用两个 DID 的组合作为种子生成共享密钥，确保通信加密的一致性
    // 注意：这是演示用的确定性派生方式，生产环境请替换为安全的密钥协商协议（例如 X25519 或 PQC 方案）
    seed := fmt.Sprintf("qlink-shared:%s:%s", fromDID, toDID)

    // 使用 SHA-256 确保密钥长度为 32 字节（适配 AES-256）
    hash := sha256.Sum256([]byte(seed))
    return hash[:], nil
}