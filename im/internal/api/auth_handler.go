package api

import (
	"net/http"
	"qlink-im/internal/models"
	"qlink-im/internal/service"

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

	// 从区块链系统获取质询
	challengeResp, err := h.authService.GetChallenge(req.DID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"challenge_id": challengeResp.ChallengeID,
		"challenge":    challengeResp.Challenge,
		"expires_at":   challengeResp.ExpiresAt,
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

	// 向区块链系统验证签名
	loginResp, err := h.authService.VerifyWithBlockchain(req.DID, req.Signature, req.ChallengeID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Challenge verified successfully",
		"token":      loginResp.Token,
		"did":        loginResp.DID,
		"login_time": loginResp.LoginTime,
		"expires_at": loginResp.ExpiresAt,
	})
}

// GetLatticePublicKey 获取用户的格加密公钥（从区块链DID文档解析）
func (h *AuthHandler) GetLatticePublicKey(c *gin.Context) {
    did := c.Param("did")
    if did == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "DID is required"})
        return
    }

    pk, err := h.authService.GetLatticePublicKey(did)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "did": did,
        "lattice_public_key": pk,
    })
}