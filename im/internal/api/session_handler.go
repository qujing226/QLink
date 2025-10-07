package api

import (
    "net/http"
    "qlink-im/internal/service"

    "github.com/gin-gonic/gin"
)

type SessionHandler struct {
	messageService service.MessageService
}

func NewSessionHandler(messageService service.MessageService) *SessionHandler {
	return &SessionHandler{
		messageService: messageService,
	}
}

func (h *SessionHandler) CreateSession(c *gin.Context) {
	userDID := c.GetString("userDID")
	
	var req struct {
		FriendDID  string `json:"friend_did" binding:"required"`
		SessionKey string `json:"session_key" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	session, err := h.messageService.CreateSession(userDID, req.FriendDID, req.SessionKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

func (h *SessionHandler) GetSession(c *gin.Context) {
    userDID := c.GetString("userDID")
    friendDID := c.Param("friend_did")
	
	if friendDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Friend DID is required"})
		return
	}

	session, err := h.messageService.GetSession(userDID, friendDID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}