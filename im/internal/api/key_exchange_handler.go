package api

import (
	"net/http"
	"qlink-im/internal/models"
	"qlink-im/internal/service"

	"github.com/gin-gonic/gin"
)

type KeyExchangeHandler struct {
	keyExchangeService service.KeyExchangeService
}

func NewKeyExchangeHandler(keyExchangeService service.KeyExchangeService) *KeyExchangeHandler {
	return &KeyExchangeHandler{
		keyExchangeService: keyExchangeService,
	}
}

// InitiateKeyExchange 发起密钥交换
func (h *KeyExchangeHandler) InitiateKeyExchange(c *gin.Context) {
    userDID := c.GetString("userDID")
    if userDID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
        return
    }

	var req models.KeyExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keyExchange, err := h.keyExchangeService.InitiateKeyExchange(userDID, req.TargetDID, req.Ciphertext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keyExchange)
}

// CompleteKeyExchange 完成密钥交换
func (h *KeyExchangeHandler) CompleteKeyExchange(c *gin.Context) {
    userDID := c.GetString("userDID")
    if userDID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
        return
    }

	keyExchangeID := c.Param("id")
	if keyExchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少密钥交换ID"})
		return
	}

	err := h.keyExchangeService.CompleteKeyExchange(userDID, keyExchangeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密钥交换完成"})
}

// GetPendingKeyExchanges 获取待处理的密钥交换
func (h *KeyExchangeHandler) GetPendingKeyExchanges(c *gin.Context) {
    userDID := c.GetString("userDID")
    if userDID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
        return
    }

	keyExchanges, err := h.keyExchangeService.GetPendingKeyExchanges(userDID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, keyExchanges)
}