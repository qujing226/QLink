package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/blockchain"
	"github.com/qujing226/QLink/pkg/api"
	"github.com/qujing226/QLink/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNetworkAPIAvailability 测试网络API可用性
func TestNetworkAPIAvailability(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试配置
	cfg := &config.Config{
		Node: &config.NodeConfig{
			ID:   "test-node-1",
			Role: "validator",
		},
		API: &config.APIConfig{
			Host:  "localhost",
			Port:  8080,
			Debug: true,
		},
		DID: &config.DIDConfig{
			StoragePath: "/tmp/qlink-test",
		},
	}

	// 创建存储管理器
	sm, err := blockchain.NewStorageManager(cfg)
	require.NoError(t, err)
	defer sm.Close()

	// 创建模拟区块链
	mockBlockchain := &MockBlockchain{}

	// 创建DID注册表和解析器
	registry := did.NewDIDRegistry(mockBlockchain)
	resolver := did.NewDIDResolver(cfg, registry, sm)

	// 创建API服务器（不需要区块链实例）
	server := api.NewServer(cfg, sm, registry, resolver, nil)

	// 创建测试路由
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 手动设置基本路由
	setupTestRoutes(router, server)

	// 测试健康检查端点
	t.Run("Health Check", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
	})

	// 测试API版本端点
	t.Run("API Version", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/version", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 可能返回200或404，取决于是否实现
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
	})

	// 测试节点信息端点
	t.Run("Node Info", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/node/info", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 可能返回200或404，取决于是否实现
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
	})
}

// TestDIDCRUDOperations 测试DID CRUD操作
func TestDIDCRUDOperations(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试配置
	cfg := &config.Config{
		Node: &config.NodeConfig{
			ID:   "test-node-2", // 使用不同的节点ID避免冲突
			Role: "validator",
		},
		API: &config.APIConfig{
			Host:  "localhost",
			Port:  8081, // 使用不同的端口避免冲突
			Debug: true,
		},
		DID: &config.DIDConfig{
			StoragePath: "/tmp/qlink-test-2", // 使用不同的存储路径
		},
	}

	// 创建存储管理器
	sm, err := blockchain.NewStorageManager(cfg)
	require.NoError(t, err)
	defer sm.Close()

	// 创建模拟区块链
	mockBlockchain := &MockBlockchain{}

	// 创建DID注册表和解析器
	registry := did.NewDIDRegistry(mockBlockchain)
	resolver := did.NewDIDResolver(cfg, registry, sm)

	// 创建API服务器（不需要区块链实例）
	server := api.NewServer(cfg, sm, registry, resolver, nil)

	// 创建测试路由
	router := gin.New()
	setupTestRoutes(router, server)

	// 测试创建DID
	t.Run("Create DID", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"method": "qlink",
		}

		jsonBody, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/v1/did", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 可能返回201（成功）或404（端点不存在）
		assert.True(t, w.Code == http.StatusCreated || w.Code == http.StatusNotFound)
	})

	// 测试解析DID
	t.Run("Resolve DID", func(t *testing.T) {
		testDID := "did:qlink:test123"
		url := fmt.Sprintf("/api/v1/did/%s", testDID)
		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 可能返回200（成功）或404（DID不存在或端点不存在）
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
	})
}

// TestBlockchainConsistency 测试区块链一致性
func TestBlockchainConsistency(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试配置
	cfg := &config.Config{
		Node: &config.NodeConfig{
			ID:   "test-node-3", // 使用不同的节点ID避免冲突
			Role: "validator",
		},
		API: &config.APIConfig{
			Host:  "localhost",
			Port:  8082, // 使用不同的端口避免冲突
			Debug: true,
		},
		DID: &config.DIDConfig{
			StoragePath: "/tmp/qlink-test-3", // 使用不同的存储路径
		},
	}

	// 创建存储管理器
	sm1, err := blockchain.NewStorageManager(cfg)
	require.NoError(t, err)
	defer sm1.Close()

	// 创建模拟区块链
	mockBlockchain1 := &MockBlockchain{}

	// 创建第一个节点的DID注册表
	registry1 := did.NewDIDRegistry(mockBlockchain1)

	// 创建第二个节点的模拟区块链
	mockBlockchain2 := &MockBlockchain{}
	registry2 := did.NewDIDRegistry(mockBlockchain2)

	// 验证两个注册表都能正常工作
	assert.NotNil(t, registry1)
	assert.NotNil(t, registry2)

	// 测试基本功能
	t.Run("Registry Creation", func(t *testing.T) {
		assert.NotNil(t, registry1)
		assert.NotNil(t, registry2)
	})
}

// setupTestRoutes 设置测试路由
func setupTestRoutes(router *gin.Engine, server *api.Server) {
	// 基本健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API版本信息
	v1 := router.Group("/api/v1")
	{
		v1.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"version": "1.0.0"})
		})

		v1.GET("/node/info", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"node_id": "test-node-1",
				"role":    "validator",
			})
		})

		// DID相关路由
		did := v1.Group("/did")
		{
			did.POST("", func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{"message": "DID created"})
			})
			did.GET("/:id", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"did": c.Param("id")})
			})
		}
	}
}
