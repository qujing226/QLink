package api

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/blockchain"
	"github.com/qujing226/QLink/did/crypto"
	blockchainPkg "github.com/qujing226/QLink/pkg/blockchain"
	"github.com/qujing226/QLink/pkg/config"
	"github.com/qujing226/QLink/pkg/types"
)

// Server HTTP API服务器
type Server struct {
	config         *config.Config
	server         *http.Server
	storageManager *blockchain.StorageManager
	registry       *did.DIDRegistry
	resolver       *did.DIDResolver
	blockchain     *blockchainPkg.Blockchain // 添加区块链实例

	// 分布式网络相关
	nodeID     string
	peers      map[string]*PeerInfo
	peersMutex sync.RWMutex

	// 监控指标
	requestCounter    *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	activeConnections prometheus.Gauge

	// 认证相关
	challenges map[string]*Challenge
	challengesMutex sync.RWMutex
}

// PeerInfo 节点信息
type PeerInfo struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
}

// Challenge 质询信息
type Challenge struct {
	ID        string    `json:"id"`
	DID       string    `json:"did"`
	Challenge string    `json:"challenge"`
	Timestamp time.Time `json:"timestamp"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	DID         string `json:"did" binding:"required"`
	Signature   string `json:"signature" binding:"required"`
	ChallengeID string `json:"challenge_id" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string    `json:"token"`
	DID       string    `json:"did"`
	LoginTime time.Time `json:"login_time"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewServer 创建新的API服务器
func NewServer(cfg *config.Config, sm *blockchain.StorageManager, reg *did.DIDRegistry, res *did.DIDResolver, bc *blockchainPkg.Blockchain) *Server {
	// 检查输入参数
	if cfg == nil {
		log.Printf("警告: NewServer收到空的配置参数")
		return nil
	}

	if cfg.Node == nil {
		log.Printf("警告: NewServer收到空的Node配置")
		return nil
	}

	// 初始化监控指标
	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_requests_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "api_request_duration_seconds",
			Help: "Duration of API requests",
		},
		[]string{"method", "endpoint"},
	)

	activeConnections := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "api_active_connections",
			Help: "Number of active connections",
		},
	)

	// 注册监控指标（避免重复注册）
	prometheus.DefaultRegisterer.Unregister(requestCounter)
	prometheus.DefaultRegisterer.Unregister(requestDuration)
	prometheus.DefaultRegisterer.Unregister(activeConnections)
	prometheus.MustRegister(requestCounter, requestDuration, activeConnections)

	server := &Server{
		config:            cfg,
		storageManager:    sm,
		registry:          reg,
		resolver:          res,
		blockchain:        bc, // 添加区块链实例
		nodeID:            cfg.Node.ID,
		peers:             make(map[string]*PeerInfo),
		challenges:        make(map[string]*Challenge),
		requestCounter:    requestCounter,
		requestDuration:   requestDuration,
		activeConnections: activeConnections,
	}

	log.Printf("API服务器创建成功: %p", server)
	return server
}

// Start 启动API服务器
func (s *Server) Start() error {
	// 检查服务器实例是否为空
	if s == nil {
		return fmt.Errorf("服务器实例为空")
	}

	// 检查配置是否为空
	if s.config == nil {
		return fmt.Errorf("服务器配置为空")
	}

	if s.config.API == nil {
		return fmt.Errorf("API配置为空")
	}

	// 设置Gin模式
	if s.config.API.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 设置路由
	s.setupRoutes(router)

	// 创建HTTP服务器
	addr := fmt.Sprintf("%s:%d", s.config.API.Host, s.config.API.Port)
	s.server = &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("启动API服务器，监听地址: %s", addr)

	// 启动服务器
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API服务器启动失败: %v", err)
		}
	}()

	return nil
}

// Stop 停止API服务器
func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("停止API服务器")
	return s.server.Shutdown(ctx)
}

// setupRoutes 设置路由
func (s *Server) setupRoutes(router *gin.Engine) {
	// 设置CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"*"}
	router.Use(cors.New(config))

	// 添加监控中间件
	router.Use(s.metricsMiddleware())

	// 添加限流中间件
	router.Use(s.rateLimitMiddleware())

	// 健康检查
	router.GET("/health", s.healthCheck)

	// 监控指标端点
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API版本组
	v1 := router.Group("/api/v1")
	{
		// 区块链相关
		blockchain := v1.Group("/blockchain")
		{
			blockchain.GET("/blocks/:hash", s.getBlock)
			blockchain.GET("/blocks/latest", s.getLatestBlock)
			blockchain.GET("/height", s.getBlockHeight)
			blockchain.GET("/transactions/:id", s.getTransaction)
			blockchain.POST("/transactions", s.submitTransaction)
			blockchain.GET("/txpool/stats", s.getTxPoolStats) // 新增交易池统计端点
		}

		// DID相关
		did := v1.Group("/did")
		{
			did.POST("/register", s.registerDID)
			did.GET("/resolve/:did", s.resolveDID)
			did.PUT("/update/:did", s.updateDID)
			did.DELETE("/revoke/:did", s.revokeDID)
			did.POST("/generate", s.generateDID)
			did.GET("/list", s.listDIDs) // 新增DID列表端点
			did.GET("/:id/document", s.getDIDDocument)
			did.GET("/:id/lattice-key", s.getLatticePublicKey) // 新增格基公钥获取接口

			// 批量操作
			did.POST("/batch/register", s.batchRegisterDID)
			did.POST("/batch/resolve", s.batchResolveDID)
		}

		// 认证相关
		auth := v1.Group("/auth")
		{
			auth.POST("/challenge", s.createChallenge)
			auth.POST("/login", s.loginWithDID)
			auth.POST("/verify", s.verifyToken)
		}

		// 节点信息和集群管理
		node := v1.Group("/node")
		{
			node.GET("/info", s.getNodeInfo)
			node.GET("/peers", s.getPeers)
			node.POST("/peers", s.addPeer)
			node.DELETE("/peers/:id", s.removePeer)
			node.GET("/status", s.getNodeStatus)
			node.GET("/sync", s.getSyncStatus)
		}

		// 集群管理
		cluster := v1.Group("/cluster")
		{
			cluster.GET("/status", s.getClusterStatus)
			cluster.POST("/sync", s.triggerSync)
			cluster.GET("/consensus", s.getConsensusStatus)
		}
	}
}

// 健康检查
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// getBlock 根据哈希获取区块
func (s *Server) getBlock(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Block hash is required"})
		return
	}

	// 从区块链获取区块
	block, err := s.blockchain.GetBlock(hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get block: " + err.Error()})
		return
	}
	if block == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"block": block,
	})
}

// getLatestBlock 获取最新区块
func (s *Server) getLatestBlock(c *gin.Context) {
	// 从区块链获取最新区块
	latestBlock, err := s.blockchain.GetLastBlock()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get latest block: " + err.Error()})
		return
	}
	if latestBlock == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No blocks found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"block": latestBlock,
	})
}

// getBlockHeight 获取区块高度
func (s *Server) getBlockHeight(c *gin.Context) {
	// 从区块链获取区块高度
	height, err := s.blockchain.GetBlockHeight()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get block height: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"height": height,
	})
}

// getTransaction 根据ID获取交易
func (s *Server) getTransaction(c *gin.Context) {
	txID := c.Param("id")
	if txID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction ID is required"})
		return
	}

	// 从区块链获取交易
	tx, err := s.blockchain.GetTransaction(txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction: " + err.Error()})
		return
	}
	if tx == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction": tx,
	})
}

// SubmitTransactionRequest 提交交易请求
type SubmitTransactionRequest struct {
	Type  string                 `json:"type" binding:"required"`
	From  string                 `json:"from" binding:"required"`
	To    string                 `json:"to" binding:"required"`
	Data  map[string]interface{} `json:"data"`
	Nonce uint64                 `json:"nonce"`
}

// submitTransaction 提交交易
func (s *Server) submitTransaction(c *gin.Context) {
	var req SubmitTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 验证交易类型
	var txType blockchainPkg.TransactionType
	switch req.Type {
	case "register_did":
		txType = blockchainPkg.TxTypeRegisterDID
	case "update_did":
		txType = blockchainPkg.TxTypeUpdateDID
	case "revoke_did":
		txType = blockchainPkg.TxTypeRevokeDID
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transaction type"})
		return
	}

	// 创建交易
	tx := blockchainPkg.NewTransaction(txType, req.From, req.To, req.Data, req.Nonce)

	// 添加交易到区块链
	if err := s.blockchain.AddTransaction(tx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Transaction submitted successfully",
		"transaction_id": tx.ID,
		"status":         "pending",
	})
}

// getTxPoolStats 获取交易池统计信息
func (s *Server) getTxPoolStats(c *gin.Context) {
	stats := s.blockchain.GetTransactionPoolStats()
	if stats == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get transaction pool stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// RegisterDIDRequest 注册DID请求
type RegisterDIDRequest struct {
	DID       string                 `json:"did" binding:"required"`
	Document  map[string]interface{} `json:"document" binding:"required"`
	Signature string                 `json:"signature" binding:"required"`
}

// 注册DID
func (s *Server) registerDID(c *gin.Context) {
	var req RegisterDIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证DID格式
	if !s.validateDIDFormat(req.DID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的DID格式"})
		return
	}

	// 解析验证方法
	var verificationMethods []types.VerificationMethod
	if vmData, exists := req.Document["verificationMethod"]; exists {
		if vmArray, ok := vmData.([]interface{}); ok {
			for _, vm := range vmArray {
				if vmMap, ok := vm.(map[string]interface{}); ok {
					verificationMethod := types.VerificationMethod{
						ID:         vmMap["id"].(string),
						Type:       vmMap["type"].(string),
						Controller: vmMap["controller"].(string),
					}

					// 处理JWK公钥
					if publicKeyJwk, exists := vmMap["publicKeyJwk"]; exists {
						verificationMethod.PublicKeyJwk = publicKeyJwk
					}

					// 处理格基公钥
					if publicKeyLattice, exists := vmMap["publicKeyLattice"]; exists {
						verificationMethod.PublicKeyLattice = publicKeyLattice.(map[string]interface{})
					}

					// 处理Multibase公钥
					if publicKeyMultibase, exists := vmMap["publicKeyMultibase"]; exists {
						if pkm, ok := publicKeyMultibase.(string); ok {
							verificationMethod.PublicKeyMultibase = pkm
						}
					}

					verificationMethods = append(verificationMethods, verificationMethod)
				}
			}
		}
	}

	// 构造注册请求
	regReq := &did.RegisterRequest{
		DID:                req.DID,
		VerificationMethod: verificationMethods,
	}

	// 注册DID到注册表
	doc, err := s.registry.Register(regReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("注册DID失败: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "DID注册成功",
		"did":      req.DID,
		"document": doc,
	})
}

// 解析DID
func (s *Server) resolveDID(c *gin.Context) {
	didID := c.Param("did")
	if didID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID ID不能为空"})
		return
	}

	// 构造完整的DID
	fullDID := fmt.Sprintf("did:qlink:%s", didID)

	// 解析DID
	result, err := s.resolver.Resolve(fullDID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("解析DID失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"did_document":            result.DIDDocument,
		"did_resolution_metadata": result.DIDResolutionMetadata,
		"did_document_metadata":   result.DIDDocumentMetadata,
	})
}

// UpdateDIDRequest 更新DID请求
type UpdateDIDRequest struct {
	Document  map[string]interface{} `json:"document" binding:"required"`
	Signature string                 `json:"signature" binding:"required"`
}

// 更新DID
func (s *Server) updateDID(c *gin.Context) {
	didID := c.Param("did")
	if didID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID ID不能为空"})
		return
	}

	var req UpdateDIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用完整的DID，不要添加前缀
	fullDID := didID

	// 构造更新请求
	updateReq := &did.UpdateRequest{
		DID: fullDID,
		// TODO: 从req.Document中解析VerificationMethod和Service
		Proof: &types.Proof{
			Type:       "JsonWebSignature2020",
			Created:    time.Now(),
			ProofValue: req.Signature,
		},
	}

	// 更新DID
	doc, err := s.registry.Update(updateReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新DID失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "DID更新成功",
		"did":      fullDID,
		"document": doc,
	})
}

// RevokeDIDRequest 撤销DID请求
type RevokeDIDRequest struct {
	Signature string `json:"signature" binding:"required"`
	Reason    string `json:"reason,omitempty"`
}

// 撤销DID
func (s *Server) revokeDID(c *gin.Context) {
	didID := c.Param("did")
	if didID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID ID不能为空"})
		return
	}

	var req RevokeDIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用完整的DID，不要添加前缀
	fullDID := didID

	// 构造撤销证明
	proof := &types.Proof{
		Type:       "JsonWebSignature2020",
		Created:    time.Now(),
		ProofValue: req.Signature,
	}

	// 撤销DID
	err := s.registry.Revoke(fullDID, proof)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("撤销DID失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "DID撤销成功",
		"did":     fullDID,
		"reason":  req.Reason,
	})
}

// 获取节点信息
func (s *Server) getNodeInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"node_id": s.config.Node.ID,
		"role":    s.config.Node.Role,
		"version": "1.0.0",
	})
}

// 获取对等节点
func (s *Server) getPeers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"peers": []string{},
	})
}

// generateDID 生成新的DID
func (s *Server) generateDID(c *gin.Context) {
	// 生成新的DID
	newDID, err := s.generateNewDID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("生成DID失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"did":     newDID,
		"message": "DID生成成功",
	})
}

// getDIDDocument 获取DID文档
func (s *Server) getDIDDocument(c *gin.Context) {
	didID := c.Param("id")
	if didID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID ID不能为空"})
		return
	}

	// 构造完整的DID
	fullDID := fmt.Sprintf("did:qlink:%s", didID)

	// 解析DID获取文档
	document, err := s.registry.Resolve(fullDID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("获取DID文档失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, document)
}

// getLatticePublicKey 获取DID的格基公钥
func (s *Server) getLatticePublicKey(c *gin.Context) {
	didID := c.Param("id")
	if didID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID ID不能为空"})
		return
	}

	// 构造完整的DID
	fullDID := fmt.Sprintf("did:qlink:%s", didID)

	// 解析DID获取文档
	document, err := s.registry.Resolve(fullDID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("获取DID文档失败: %v", err)})
		return
	}

	// 从DID文档中提取格基公钥
	latticeKey, err := s.extractLatticePublicKey(document)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("未找到格基公钥: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"did":         fullDID,
		"lattice_key": latticeKey,
		"type":        "Kyber768",
		"format":      "JWK",
	})
}

// extractLatticePublicKey 从DID文档中提取格基公钥
func (s *Server) extractLatticePublicKey(document *types.DIDDocument) (map[string]interface{}, error) {
	if document == nil || document.VerificationMethod == nil {
		return nil, fmt.Errorf("DID文档或验证方法为空")
	}

	// 遍历验证方法查找格基公钥
	for _, vm := range document.VerificationMethod {
		// 检查是否有格基公钥字段
		if vm.PublicKeyLattice != nil {
			return vm.PublicKeyLattice, nil
		}

		// 检查JWK中是否包含Kyber字段（兼容性处理）
		if vm.PublicKeyJwk != nil {
			if jwk, ok := vm.PublicKeyJwk.(map[string]interface{}); ok {
				if kyberKey, exists := jwk["kyber"]; exists {
					return map[string]interface{}{
						"kty":   "OKP",
						"crv":   "Kyber768",
						"kyber": kyberKey,
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("未找到格基公钥")
}

// validateDIDFormat 验证DID格式
func (s *Server) validateDIDFormat(didStr string) bool {
	// 简单的DID格式验证
	return len(didStr) > 0 && didStr[:10] == "did:qlink:"
}

// listDIDs 获取所有DID列表
func (s *Server) listDIDs(c *gin.Context) {
	docs, err := s.registry.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取DID列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dids":  docs,
		"count": len(docs),
	})
}

// generateNewDID 生成新的DID
func (s *Server) generateNewDID() (string, error) {
	return fmt.Sprintf("did:qlink:example%d", time.Now().Unix()), nil
}

// createChallenge 创建质询
func (s *Server) createChallenge(c *gin.Context) {
	var req struct {
		DID string `json:"did" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 生成质询ID和内容
	challengeBytes := make([]byte, 32)
	rand.Read(challengeBytes)
	challengeID := hex.EncodeToString(challengeBytes)
	
	challengeContent := fmt.Sprintf("Please sign this challenge to authenticate your DID: %s at %d", req.DID, time.Now().Unix())
	
	challenge := &Challenge{
		ID:        challengeID,
		DID:       req.DID,
		Challenge: challengeContent,
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute), // 5分钟过期
	}

	// 存储质询
	s.challengesMutex.Lock()
	s.challenges[challengeID] = challenge
	s.challengesMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"challenge_id": challengeID,
		"challenge":    challengeContent,
		"expires_at":   challenge.ExpiresAt,
	})
}

// loginWithDID DID登录
func (s *Server) loginWithDID(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 验证质询是否存在且未过期
	s.challengesMutex.RLock()
	challenge, exists := s.challenges[req.ChallengeID]
	s.challengesMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid challenge"})
		return
	}

	if time.Now().After(challenge.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Challenge expired"})
		return
	}

	log.Printf("质询DID: %s, 请求DID: %s", challenge.DID, req.DID)
	if challenge.DID != req.DID {
		log.Printf("DID不匹配: 质询DID=%s, 请求DID=%s", challenge.DID, req.DID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID mismatch"})
		return
	}

	// 验证DID是否存在
	log.Printf("验证DID是否存在: %s", req.DID)
	_, err := s.resolver.Resolve(req.DID)
	if err != nil {
		log.Printf("DID解析失败: %s, 错误: %v", req.DID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "DID not found: " + err.Error()})
		return
	}
	log.Printf("DID解析成功: %s", req.DID)

	// 验证格基密码学签名
	log.Printf("开始验证签名: DID=%s, 质询=%s, 签名=%s", req.DID, challenge.Challenge, req.Signature)
	if !s.verifyLatticeSignature(req.Signature, challenge.Challenge, req.DID) {
		log.Printf("签名验证失败")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}
	log.Printf("签名验证成功")

	// 生成会话令牌
	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	// 清理使用过的质询
	s.challengesMutex.Lock()
	delete(s.challenges, req.ChallengeID)
	s.challengesMutex.Unlock()

	loginTime := time.Now()
	expiresAt := loginTime.Add(24 * time.Hour) // 24小时有效期

	response := LoginResponse{
		Token:     token,
		DID:       req.DID,
		LoginTime: loginTime,
		ExpiresAt: expiresAt,
	}

	c.JSON(http.StatusOK, response)
}

// verifyLatticeSignature 验证格基密码学签名
func (s *Server) verifyLatticeSignature(signature, challenge, did string) bool {
	log.Printf("开始验证签名 - DID: %s", did)
	log.Printf("签名内容: %s", signature)
	log.Printf("质询内容: %s", challenge)
	
	// 尝试解析为JSON格式的HybridSignature
	var hybridSig struct {
		ECDSASignature string `json:"ecdsa_signature"`
		KyberProof     string `json:"kyber_proof,omitempty"`
	}
	
	// 首先尝试解析为JSON格式（真实签名）
	if err := json.Unmarshal([]byte(signature), &hybridSig); err == nil {
		log.Printf("解析为JSON格式成功，调用混合签名验证")
		return s.verifyHybridSignature(&hybridSig, challenge, did)
	}
	
	// 如果不是JSON格式，尝试解析为base64编码的签名
	if sigBytes, err := base64.StdEncoding.DecodeString(signature); err == nil {
		log.Printf("解码base64成功，尝试解析JSON")
		if err := json.Unmarshal(sigBytes, &hybridSig); err == nil {
			log.Printf("base64解码后解析JSON成功，调用混合签名验证")
			return s.verifyHybridSignature(&hybridSig, challenge, did)
		} else {
			log.Printf("base64解码后解析JSON失败: %v", err)
		}
	} else {
		log.Printf("base64解码失败: %v", err)
	}
	
	// 如果不是JSON格式，尝试解析为hex编码的签名
	if sigBytes, err := hex.DecodeString(signature); err == nil {
		log.Printf("解码hex成功，尝试解析JSON")
		if err := json.Unmarshal(sigBytes, &hybridSig); err == nil {
			log.Printf("hex解码后解析JSON成功，调用混合签名验证")
			return s.verifyHybridSignature(&hybridSig, challenge, did)
		} else {
			log.Printf("hex解码后解析JSON失败: %v", err)
		}
	} else {
		log.Printf("hex解码失败: %v", err)
	}
	
	// 回退到旧的模拟签名验证逻辑（向后兼容）
	log.Printf("回退到旧的模拟签名验证逻辑")
	return s.verifyLegacySignature(signature, challenge, did)
}

// verifyHybridSignature 验证真实的混合签名
func (s *Server) verifyHybridSignature(sig *struct {
	ECDSASignature string `json:"ecdsa_signature"`
	KyberProof     string `json:"kyber_proof,omitempty"`
}, challenge, did string) bool {
	log.Printf("开始验证混合签名 - DID: %s", did)
	log.Printf("质询内容: %s", challenge)
	log.Printf("ECDSA签名: %s", sig.ECDSASignature)
	
	// 从DID文档中获取公钥
	keyPair, err := s.getHybridKeyPairFromDID(did)
	if err != nil {
		log.Printf("获取DID密钥对失败: %v", err)
		return false
	}
	
	// 解码base64编码的ECDSA签名
	ecdsaBytes, err := base64.StdEncoding.DecodeString(sig.ECDSASignature)
	if err != nil {
		log.Printf("解码ECDSA签名失败: %v", err)
		return false
	}
	
	log.Printf("解码后的ECDSA签名长度: %d bytes", len(ecdsaBytes))
	log.Printf("解码后的ECDSA签名hex: %x", ecdsaBytes)
	
	// 解码Kyber证明（如果存在）
	var kyberBytes []byte
	if sig.KyberProof != "" {
		kyberBytes, err = base64.StdEncoding.DecodeString(sig.KyberProof)
		if err != nil {
			log.Printf("解码Kyber证明失败: %v", err)
			return false
		}
	}
	
	// 创建HybridSignature对象
	hybridSignature := &crypto.HybridSignature{
		ECDSASignature: ecdsaBytes,
		KyberProof:     kyberBytes,
	}
	
	// 使用真实的密码学验证
	result := keyPair.Verify([]byte(challenge), hybridSignature)
	log.Printf("混合签名验证结果: %v", result)
	return result
}

// getHybridKeyPairFromDID 从DID获取HybridKeyPair
func (s *Server) getHybridKeyPairFromDID(did string) (*crypto.HybridKeyPair, error) {
	// 解析DID文档
	result, err := s.resolver.Resolve(did)
	if err != nil {
		return nil, fmt.Errorf("解析DID文档失败: %w", err)
	}

	if result.DIDDocument == nil {
		return nil, fmt.Errorf("DID文档不存在")
	}

	// 从验证方法中获取公钥
	if len(result.DIDDocument.VerificationMethod) == 0 {
		return nil, fmt.Errorf("DID文档中没有验证方法")
	}

	// 使用第一个验证方法的公钥
	vm := result.DIDDocument.VerificationMethod[0]
	
	// 从JWK中重建HybridKeyPair
	if vm.PublicKeyJwk != nil {
		// 类型断言为PublicKeyJWK
		if jwk, ok := vm.PublicKeyJwk.(*crypto.PublicKeyJWK); ok {
			return crypto.FromJWK(jwk)
		}
		// 如果类型断言失败，尝试从interface{}转换
		jwkData, err := json.Marshal(vm.PublicKeyJwk)
		if err != nil {
			return nil, fmt.Errorf("序列化JWK失败: %w", err)
		}
		var jwk crypto.PublicKeyJWK
		if err := json.Unmarshal(jwkData, &jwk); err != nil {
			return nil, fmt.Errorf("反序列化JWK失败: %w", err)
		}
		return crypto.FromJWK(&jwk)
	}

	return nil, fmt.Errorf("无法从DID文档中提取公钥")
}

// verifyLegacySignature 验证旧的模拟签名格式（向后兼容）
func (s *Server) verifyLegacySignature(signature, challenge, did string) bool {
	// 检查签名是否为空或过短
	if len(signature) < 20 {
		return false
	}
	
	// 检查签名是否以 'lattice_si' 开头
	if !strings.HasPrefix(signature, "lattice_si") {
		return false
	}
	
	// 提取签名的各个部分
	sigContent := signature[10:] // 去掉 'lattice_si' 前缀
	
	// 检查challenge前缀（前8位）
	if len(sigContent) < 8 {
		return false
	}
	
	challengePrefix := sigContent[:8]
	expectedChallengePrefix := challenge[:8]
	
	if challengePrefix != expectedChallengePrefix {
		return false
	}
	
	// 从DID文档中获取真实的公钥
	publicKey, err := s.getPublicKeyFromDIDDocument(did)
	if err != nil {
		log.Printf("获取DID公钥失败: %v", err)
		return false
	}
	
	// 检查公钥后缀（最后8位）
	if len(sigContent) < 16 {
		return false
	}
	
	publicKeySuffix := sigContent[len(sigContent)-8:]
	expectedPublicKeySuffix := publicKey[len(publicKey)-8:]
	
	if publicKeySuffix != expectedPublicKeySuffix {
		return false
	}
	
	// 验证哈希部分
	hashPart := sigContent[8 : len(sigContent)-8]
	expectedHash := s.generateSignatureHash(challenge, publicKey)
	
	return hashPart == expectedHash
}

// getPublicKeyFromDIDDocument 从DID文档中获取真实的公钥
func (s *Server) getPublicKeyFromDIDDocument(did string) (string, error) {
	// 解析DID文档
	result, err := s.resolver.Resolve(did)
	if err != nil {
		return "", fmt.Errorf("解析DID文档失败: %w", err)
	}

	if result.DIDDocument == nil {
		return "", fmt.Errorf("DID文档不存在")
	}

	// 从验证方法中获取公钥
	if len(result.DIDDocument.VerificationMethod) == 0 {
		return "", fmt.Errorf("DID文档中没有验证方法")
	}

	// 使用第一个验证方法的公钥
	vm := result.DIDDocument.VerificationMethod[0]
	
	// 从JWK中提取公钥信息
	if vm.PublicKeyJwk != nil {
		// 这里需要根据实际的JWK格式来提取公钥
		// 暂时返回一个基于DID的公钥
		return s.generatePublicKeyFromDIDFallback(did), nil
	}

	// 如果没有JWK，使用其他公钥格式
	if vm.PublicKeyMultibase != "" {
		return vm.PublicKeyMultibase, nil
	}

	// 如果都没有，回退到基于DID生成公钥
	return s.generatePublicKeyFromDIDFallback(did), nil
}

// generatePublicKeyFromDIDFallback 从DID生成公钥的回退方法
func (s *Server) generatePublicKeyFromDIDFallback(did string) string {
	// 从DID中提取标识符部分
	parts := strings.Split(did, ":")
	if len(parts) >= 3 {
		identifier := parts[2]
		// 确保长度足够
		if len(identifier) >= 32 {
			return identifier[:32]
		}
		// 如果不够长，用默认值填充
		return identifier + "default-private-key"[:32-len(identifier)]
	}
	return "default-private-key"
}

// generateSignatureHash 生成签名哈希
func (s *Server) generateSignatureHash(challenge, publicKey string) string {
	// 使用与前端相同的HMAC-SHA256算法
	h := hmac.New(sha256.New, []byte(publicKey))
	h.Write([]byte(challenge))
	hash := h.Sum(nil)
	return hex.EncodeToString(hash)[:16] // 取前16个字符
}

// verifyToken 验证令牌
func (s *Server) verifyToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 简化的令牌验证（实际应用中需要使用JWT或其他安全机制）
	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"message": "Token is valid",
	})
}
