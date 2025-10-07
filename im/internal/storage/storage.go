package storage

import (
	"fmt"
	"qlink-im/internal/config"
	"qlink-im/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Storage interface {
	// User operations
	CreateUser(user *models.User) error
	GetUserByDID(did string) (*models.User, error)
	UpdateUser(user *models.User) error
	SearchUsers(query string) ([]*models.User, error)

	// Friend operations
	CreateFriend(friend *models.Friend) error
	GetFriends(userDID string) ([]*models.Friend, error)
	GetFriendRequest(userDID, friendDID string) (*models.Friend, error)
	GetFriendRequests(userDID string) ([]*models.Friend, error)
	UpdateFriendStatus(id uint, status string) error

	// Session operations
	CreateSession(session *models.Session) error
	GetSession(userDID, friendDID string) (*models.Session, error)
	GetSessionByID(id uint) (*models.Session, error)
	UpdateSession(session *models.Session) error
	DeleteExpiredSessions() error

	// Message operations
	CreateMessage(message *models.Message) error
	GetMessages(userDID, friendDID string, limit, offset int) ([]*models.Message, error)
	UpdateMessageStatus(id uint, status string) error

	// Challenge operations
	CreateChallenge(challenge *models.Challenge) error
	GetChallenge(id uint) (*models.Challenge, error)
	GetChallengeByNonce(nonce string) (*models.Challenge, error)
	UpdateChallenge(challenge *models.Challenge) error
	DeleteExpiredChallenges() error

	// Key exchange operations
	CreateKeyExchange(keyExchange *models.KeyExchange) error
	GetKeyExchange(id uint) (*models.KeyExchange, error)
	GetCompletedKeyExchange(userDID, friendDID string) (*models.KeyExchange, error)
	UpdateKeyExchange(keyExchange *models.KeyExchange) error
	DeleteExpiredKeyExchanges() error
	GetPendingKeyExchanges(userDID string) ([]*models.KeyExchange, error)

	Close() error
}

type storage struct {
	db *gorm.DB
}

func NewStorage(cfg config.DatabaseConfig) (Storage, error) {
	var db *gorm.DB
	var err error

	switch cfg.Type {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.URL), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 自动迁移数据库表
	if err := db.AutoMigrate(
		&models.User{},
		&models.Friend{},
		&models.Session{},
		&models.Message{},
		&models.Challenge{},
		&models.KeyExchange{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &storage{db: db}, nil
}

func (s *storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// User operations
func (s *storage) CreateUser(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *storage) GetUserByDID(did string) (*models.User, error) {
	var user models.User
	err := s.db.Where("d_id = ?", did).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *storage) UpdateUser(user *models.User) error {
	return s.db.Save(user).Error
}

// SearchUsers 搜索用户（通过DID或昵称）
func (s *storage) SearchUsers(query string) ([]*models.User, error) {
	var users []*models.User
	err := s.db.Where("d_id LIKE ? OR nickname LIKE ?", "%"+query+"%", "%"+query+"%").
		Limit(20).Find(&users).Error
	return users, err
}

// Friend operations
func (s *storage) CreateFriend(friend *models.Friend) error {
	return s.db.Create(friend).Error
}

func (s *storage) GetFriends(userDID string) ([]*models.Friend, error) {
	var friends []*models.Friend
	err := s.db.Where("user_d_id = ? AND status = ?", userDID, "accepted").Find(&friends).Error
	return friends, err
}

func (s *storage) GetFriendRequest(userDID, friendDID string) (*models.Friend, error) {
	var friend models.Friend
	err := s.db.Where("user_d_id = ? AND friend_d_id = ?", userDID, friendDID).First(&friend).Error
	if err != nil {
		return nil, err
	}
	return &friend, nil
}

func (s *storage) UpdateFriendStatus(id uint, status string) error {
	return s.db.Model(&models.Friend{}).Where("id = ?", id).Update("status", status).Error
}

func (s *storage) GetFriendRequests(userDID string) ([]*models.Friend, error) {
	var requests []*models.Friend
	err := s.db.Where("friend_d_id = ? AND status = ?", userDID, "pending").Find(&requests).Error
	return requests, err
}

// Session operations
func (s *storage) CreateSession(session *models.Session) error {
	return s.db.Create(session).Error
}

func (s *storage) GetSession(userDID, friendDID string) (*models.Session, error) {
	var session models.Session
	err := s.db.Where("(user_d_id = ? AND friend_d_id = ?) OR (user_d_id = ? AND friend_d_id = ?)",
		userDID, friendDID, friendDID, userDID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSessionByID 根据ID获取会话
func (s *storage) GetSessionByID(id uint) (*models.Session, error) {
	var session models.Session
	err := s.db.First(&session, id).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *storage) UpdateSession(session *models.Session) error {
	return s.db.Save(session).Error
}

func (s *storage) DeleteExpiredSessions() error {
	return s.db.Where("expires_at < NOW()").Delete(&models.Session{}).Error
}

// Message operations
func (s *storage) CreateMessage(message *models.Message) error {
	return s.db.Create(message).Error
}

func (s *storage) GetMessages(userDID, friendDID string, limit, offset int) ([]*models.Message, error) {
	var messages []*models.Message
	err := s.db.Where("(`from` = ? AND `to` = ?) OR (`from` = ? AND `to` = ?)",
		userDID, friendDID, friendDID, userDID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error
	return messages, err
}

func (s *storage) UpdateMessageStatus(id uint, status string) error {
	return s.db.Model(&models.Message{}).Where("id = ?", id).Update("status", status).Error
}

// Challenge operations
func (s *storage) CreateChallenge(challenge *models.Challenge) error {
	return s.db.Create(challenge).Error
}

func (s *storage) GetChallenge(id uint) (*models.Challenge, error) {
	var challenge models.Challenge
	err := s.db.First(&challenge, id).Error
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *storage) GetChallengeByNonce(nonce string) (*models.Challenge, error) {
	var challenge models.Challenge
	if err := s.db.Where("nonce = ?", nonce).First(&challenge).Error; err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (s *storage) UpdateChallenge(challenge *models.Challenge) error {
	return s.db.Save(challenge).Error
}

func (s *storage) DeleteExpiredChallenges() error {
	return s.db.Where("expires_at < NOW()").Delete(&models.Challenge{}).Error
}

// Key exchange operations
func (s *storage) CreateKeyExchange(keyExchange *models.KeyExchange) error {
	return s.db.Create(keyExchange).Error
}

func (s *storage) GetKeyExchange(id uint) (*models.KeyExchange, error) {
	var keyExchange models.KeyExchange
	err := s.db.First(&keyExchange, id).Error
	if err != nil {
		return nil, err
	}
	return &keyExchange, nil
}

// GetCompletedKeyExchange 获取已完成的密钥交换
func (s *storage) GetCompletedKeyExchange(userDID, friendDID string) (*models.KeyExchange, error) {
	var keyExchange models.KeyExchange
	err := s.db.Where("((from_did = ? AND to_did = ?) OR (from_did = ? AND to_did = ?)) AND status = ?", 
		userDID, friendDID, friendDID, userDID, "completed").First(&keyExchange).Error
	if err != nil {
		return nil, err
	}
	return &keyExchange, nil
}

func (s *storage) UpdateKeyExchange(keyExchange *models.KeyExchange) error {
	return s.db.Save(keyExchange).Error
}

func (s *storage) DeleteExpiredKeyExchanges() error {
	return s.db.Where("expires_at < NOW()").Delete(&models.KeyExchange{}).Error
}

// GetPendingKeyExchanges 获取用户的待处理密钥交换
func (s *storage) GetPendingKeyExchanges(userDID string) ([]*models.KeyExchange, error) {
	var keyExchanges []*models.KeyExchange
	err := s.db.Where("to_did = ? AND status = ?", userDID, "pending").Find(&keyExchanges).Error
	return keyExchanges, err
}