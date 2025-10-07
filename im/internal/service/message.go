package service

import (
	"fmt"
	"qlink-im/internal/models"
	"qlink-im/internal/storage"
	"time"
)

type MessageService interface {
	SendMessage(from, to, ciphertext, nonce string) (*models.Message, error)
	GetMessages(userDID, friendDID string, page, limit int) ([]*models.Message, error)
	MarkMessageAsRead(messageID uint) error
	CreateSession(userDID, friendDID, sessionKey string) (*models.Session, error)
	GetSession(userDID, friendDID string) (*models.Session, error)
	UpdateSessionKey(sessionID uint, newKey string) error
}

type messageService struct {
	storage storage.Storage
}

func NewMessageService(storage storage.Storage) MessageService {
	return &messageService{
		storage: storage,
	}
}

// SendMessage 发送消息
func (m *messageService) SendMessage(from, to, ciphertext, nonce string) (*models.Message, error) {
	// 验证发送者和接收者是否为好友
	friendRelation, err := m.storage.GetFriendRequest(from, to)
	if err != nil || friendRelation.Status != "accepted" {
		return nil, fmt.Errorf("users are not friends")
	}

	// 检查是否存在有效会话
	session, err := m.storage.GetSession(from, to)
	if err != nil {
		return nil, fmt.Errorf("no active session found")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	// 创建消息
	message := &models.Message{
		From:       from,
		To:         to,
		SessionID:  session.ID,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Timestamp:  time.Now(),
		Status:     "sent",
	}

	if err := m.storage.CreateMessage(message); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// 更新会话最后活动时间
	session.LastActivity = time.Now()
	m.storage.UpdateSession(session)

	return message, nil
}

// GetMessages 获取消息历史
func (m *messageService) GetMessages(userDID, friendDID string, page, limit int) ([]*models.Message, error) {
	// 验证用户关系
	friendRelation, err := m.storage.GetFriendRequest(userDID, friendDID)
	if err != nil || friendRelation.Status != "accepted" {
		return nil, fmt.Errorf("users are not friends")
	}

	// 计算offset
	offset := (page - 1) * limit
	return m.storage.GetMessages(userDID, friendDID, limit, offset)
}

// MarkMessageAsRead 标记消息为已读
func (m *messageService) MarkMessageAsRead(messageID uint) error {
	return m.storage.UpdateMessageStatus(messageID, "read")
}

// CreateSession 创建会话
func (m *messageService) CreateSession(userDID, friendDID, sessionKey string) (*models.Session, error) {
	// 检查是否已存在会话
	existingSession, _ := m.storage.GetSession(userDID, friendDID)
	if existingSession != nil && time.Now().Before(existingSession.ExpiresAt) {
		return existingSession, nil
	}

	// 创建新会话
	session := &models.Session{
		UserDID:      userDID,
		FriendDID:    friendDID,
		SessionKey:   sessionKey, // 应该加密存储
		KeyVersion:   1,
		ExpiresAt:    time.Now().Add(30 * time.Minute), // 30分钟过期
		LastActivity: time.Now(),
	}

	if err := m.storage.CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// GetSession 获取会话
func (m *messageService) GetSession(userDID, friendDID string) (*models.Session, error) {
	session, err := m.storage.GetSession(userDID, friendDID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// UpdateSessionKey 更新会话密钥
func (m *messageService) UpdateSessionKey(sessionID uint, newKey string) error {
	// 这里需要实现获取会话并更新密钥的逻辑
	// 暂时返回nil
	return nil
}