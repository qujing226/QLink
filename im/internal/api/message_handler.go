package api

import (
	"net/http"
	"qlink-im/internal/models"
	"qlink-im/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService    service.MessageService
	encryptionService service.EncryptionService
}

func NewMessageHandler(messageService service.MessageService, encryptionService service.EncryptionService) *MessageHandler {
	return &MessageHandler{
		messageService:    messageService,
		encryptionService: encryptionService,
	}
}

// SendMessage 发送消息
func (h *MessageHandler) SendMessage(c *gin.Context) {
    userDID := c.GetString("userDID")
	if userDID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取或创建会话
	session, err := h.encryptionService.GetOrCreateSession(userDID, req.To)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session: " + err.Error()})
		return
	}

	// 解密消息内容（客户端发送的是加密的）
	plaintext, err := h.encryptionService.DecryptMessage(session.ID, req.Ciphertext, req.Nonce)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decrypt message: " + err.Error()})
		return
	}

	// 重新加密消息用于存储
	encryptedMessage, err := h.encryptionService.EncryptMessage(session.ID, plaintext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt message: " + err.Error()})
		return
	}

	// 发送消息
	message, err := h.messageService.SendMessage(userDID, req.To, encryptedMessage.Ciphertext, encryptedMessage.Nonce)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Message sent successfully",
		"data":    message,
	})
}

// GetMessages 获取消息列表
func (h *MessageHandler) GetMessages(c *gin.Context) {
    userDID := c.GetString("userDID")
	if userDID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	friendDID := c.Query("friend_did")
	if friendDID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "friend_did is required"})
		return
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, err := h.messageService.GetMessages(userDID, friendDID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 解密消息内容（可选，根据需要）
	decryptedMessages := make([]gin.H, len(messages))
	for i, msg := range messages {
		// 尝试解密消息
		plaintext, err := h.encryptionService.DecryptMessage(msg.SessionID, msg.Ciphertext, msg.Nonce)
		if err != nil {
			// 如果解密失败，返回加密的消息
			decryptedMessages[i] = gin.H{
				"id":         msg.ID,
				"from":       msg.From,
				"to":         msg.To,
				"ciphertext": msg.Ciphertext,
				"nonce":      msg.Nonce,
				"timestamp":  msg.Timestamp,
				"status":     msg.Status,
				"encrypted":  true,
			}
		} else {
			decryptedMessages[i] = gin.H{
				"id":        msg.ID,
				"from":      msg.From,
				"to":        msg.To,
				"content":   plaintext,
				"timestamp": msg.Timestamp,
				"status":    msg.Status,
				"encrypted": false,
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": decryptedMessages})
}

// MarkAsRead 标记消息为已读
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
    userDID := c.GetString("userDID")
	if userDID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	if err := h.messageService.MarkMessageAsRead(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message marked as read"})
}