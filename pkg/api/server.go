package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/blockchain"
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
}

// PeerInfo 节点信息
type PeerInfo struct {
	ID       string    `json:"id"`
	Address  string    `json:"address"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
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

					// 处理格基公钥
					if publicKeyLattice, exists := vmMap["publicKeyLattice"]; exists {
						verificationMethod.PublicKeyLattice = publicKeyLattice.(map[string]interface{})
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
	// 这里应该实现实际的DID生成逻辑
	// 暂时返回一个示例DID
	return "did:qlink:example123", nil
}
