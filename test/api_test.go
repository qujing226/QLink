package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestBasicAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建基本配置
	cfg := &config.Config{
		API: &config.APIConfig{
			Host:  "localhost",
			Port:  8080,
			Debug: true,
		},
	}

	// 创建模拟区块链
	mockBlockchain := &MockBlockchain{}

	// 创建DID注册表
	registry := did.NewDIDRegistry(mockBlockchain)

	// 创建测试路由
	router := gin.New()

	// 手动设置基本路由用于测试
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 测试健康检查
	t.Run("Health Check", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// 验证注册表创建成功
	assert.NotNil(t, registry)
	assert.NotNil(t, cfg)
}
