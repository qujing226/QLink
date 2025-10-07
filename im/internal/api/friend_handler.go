package api

import (
	"net/http"
	"qlink-im/internal/models"
	"qlink-im/internal/service"

	"github.com/gin-gonic/gin"
)

type FriendHandler struct {
	friendService service.FriendService
}

func NewFriendHandler(friendService service.FriendService) *FriendHandler {
	return &FriendHandler{
		friendService: friendService,
	}
}

func (h *FriendHandler) AddFriend(c *gin.Context) {
	userDID := c.GetString("userDID")
	
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.FriendDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Friend DID is required"})
		return
	}

	if userDID == req.FriendDID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot add yourself as friend"})
		return
	}

	err := h.friendService.AddFriend(userDID, req.FriendDID, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Friend request sent"})
}

func (h *FriendHandler) AcceptFriend(c *gin.Context) {
	userDID := c.GetString("userDID")
	
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.FriendDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Friend DID is required"})
		return
	}

	err := h.friendService.AcceptFriend(userDID, req.FriendDID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Friend request accepted"})
}

func (h *FriendHandler) RejectFriend(c *gin.Context) {
	userDID := c.GetString("userDID")
	
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.FriendDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Friend DID is required"})
		return
	}

	err := h.friendService.RejectFriend(userDID, req.FriendDID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Friend request rejected"})
}

func (h *FriendHandler) GetFriends(c *gin.Context) {
	userDID := c.GetString("userDID")

	friends, err := h.friendService.GetFriends(userDID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"friends": friends})
}

func (h *FriendHandler) GetFriendRequests(c *gin.Context) {
	userDID := c.GetString("userDID")

	requests, err := h.friendService.GetFriendRequests(userDID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

func (h *FriendHandler) BlockFriend(c *gin.Context) {
	userDID := c.GetString("userDID")
	
	var req models.FriendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if req.FriendDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Friend DID is required"})
		return
	}

	err := h.friendService.BlockFriend(userDID, req.FriendDID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Friend blocked"})
}

// SearchUsers 搜索用户（通过DID）
func (h *FriendHandler) SearchUsers(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	users, err := h.friendService.SearchUsers(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}