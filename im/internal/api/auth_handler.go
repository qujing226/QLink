package api

import (
    "net/http"
    "qlink-im/internal/models"
    "qlink-im/internal/service"
    "strconv"

    "github.com/gin-gonic/gin"
)

type AuthHandler struct {
    authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// 验证DID格式
	if req.DID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID is required"})
		return
	}

	// 验证签名
	if req.Signature == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Signature is required"})
		return
	}

	// 执行登录
	response, err := h.authService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateChallenge 获取区块链质询
func (h *AuthHandler) CreateChallenge(c *gin.Context) {
    var req struct {
        DID string `json:"did"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
        return
    }

    if req.DID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "DID is required"})
    }
    // 使用本地服务生成质询（去除对区块链的依赖）
    challenge, err := h.authService.CreateChallenge(req.DID, "im-service")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
            "challenge_id": challenge.ID,
            "challenge":    challenge.Nonce,
            "expires_at":   challenge.ExpiresAt,
    })
}

// VerifyChallenge 验证区块链签名
func (h *AuthHandler) VerifyChallenge(c *gin.Context) {
    var req struct {
        DID         string `json:"did"`
        Signature   string `json:"signature"`
        ChallengeID string `json:"challenge_id"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // 本地验证签名（通过挑战ID）
    id64, err := strconv.ParseUint(req.ChallengeID, 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid challenge_id"})
        return
    }
    if err := h.authService.VerifyChallenge(uint(id64), req.Signature); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    // 生成并返回登录令牌
    loginResp, err := h.authService.Login(&models.LoginRequest{DID: req.DID, Signature: req.Signature})
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":  "Challenge verified successfully",
        "token":    loginResp.Token,
        "user":     loginResp.User,
        "expires_at": loginResp.ExpiresAt,
    })
}