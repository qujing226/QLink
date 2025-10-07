package service

import (
	"fmt"
	"qlink-im/internal/models"
	"qlink-im/internal/storage"
)

type FriendService interface {
	AddFriend(userDID, targetDID, requestMsg string) error
	AcceptFriend(userDID, friendDID string) error
	RejectFriend(userDID, friendDID string) error
	GetFriends(userDID string) ([]*models.Friend, error)
	GetFriendRequests(userDID string) ([]*models.Friend, error)
	BlockFriend(userDID, friendDID string) error
	SearchUsers(query string) ([]*models.User, error)
}

type friendService struct {
	storage storage.Storage
}

func NewFriendService(storage storage.Storage) FriendService {
	return &friendService{
		storage: storage,
	}
}

// AddFriend 添加好友请求
func (f *friendService) AddFriend(userDID, targetDID, requestMsg string) error {
	if userDID == targetDID {
		return fmt.Errorf("cannot add yourself as friend")
	}

	// 检查目标用户是否存在
	_, err := f.storage.GetUserByDID(targetDID)
	if err != nil {
		return fmt.Errorf("target user not found")
	}

	// 检查是否已经是好友或已发送请求
	existing, _ := f.storage.GetFriendRequest(userDID, targetDID)
	if existing != nil {
		switch existing.Status {
		case "accepted":
			return fmt.Errorf("already friends")
		case "pending":
			return fmt.Errorf("friend request already sent")
		case "blocked":
			return fmt.Errorf("user has blocked you")
		}
	}

	// 创建好友请求
	friend := &models.Friend{
		UserDID:    userDID,
		FriendDID:  targetDID,
		Status:     "pending",
		RequestMsg: requestMsg,
	}

	return f.storage.CreateFriend(friend)
}

// AcceptFriend 接受好友请求
func (f *friendService) AcceptFriend(userDID, friendDID string) error {
	// 查找好友请求
	friendRequest, err := f.storage.GetFriendRequest(friendDID, userDID)
	if err != nil {
		return fmt.Errorf("friend request not found")
	}

	if friendRequest.Status != "pending" {
		return fmt.Errorf("friend request is not pending")
	}

	// 更新请求状态为已接受
	if err := f.storage.UpdateFriendStatus(friendRequest.ID, "accepted"); err != nil {
		return fmt.Errorf("failed to accept friend request: %w", err)
	}

	// 创建反向好友关系
	reverseFriend := &models.Friend{
		UserDID:   userDID,
		FriendDID: friendDID,
		Status:    "accepted",
	}

	return f.storage.CreateFriend(reverseFriend)
}

// RejectFriend 拒绝好友请求
func (f *friendService) RejectFriend(userDID, friendDID string) error {
	friendRequest, err := f.storage.GetFriendRequest(friendDID, userDID)
	if err != nil {
		return fmt.Errorf("friend request not found")
	}

	if friendRequest.Status != "pending" {
		return fmt.Errorf("friend request is not pending")
	}

	return f.storage.UpdateFriendStatus(friendRequest.ID, "rejected")
}

// GetFriends 获取好友列表
func (f *friendService) GetFriends(userDID string) ([]*models.Friend, error) {
	return f.storage.GetFriends(userDID)
}

// GetFriendRequests 获取好友请求列表
func (f *friendService) GetFriendRequests(userDID string) ([]*models.Friend, error) {
	return f.storage.GetFriendRequests(userDID)
}

// BlockFriend 屏蔽好友
func (f *friendService) BlockFriend(userDID, friendDID string) error {
	// 查找好友关系
	friendRequest, err := f.storage.GetFriendRequest(userDID, friendDID)
	if err != nil {
		return fmt.Errorf("friend relationship not found")
	}

	// 更新状态为已屏蔽
	if err := f.storage.UpdateFriendStatus(friendRequest.ID, "blocked"); err != nil {
		return fmt.Errorf("failed to block friend: %w", err)
	}

	return nil
}

// SearchUsers 搜索用户（通过DID或昵称）
func (f *friendService) SearchUsers(query string) ([]*models.User, error) {
	return f.storage.SearchUsers(query)
}