package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qujing226/QLink/did"
)

// 批量注册DID
func (s *Server) batchRegisterDID(c *gin.Context) {
	type BatchRegisterRequest struct {
		DIDs []RegisterDIDRequest `json:"dids" binding:"required"`
	}

	var req BatchRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results := make([]gin.H, 0, len(req.DIDs))
	for _, didReq := range req.DIDs {
		// 验证DID格式
		if !s.validateDIDFormat(didReq.DID) {
			results = append(results, gin.H{
				"did":     didReq.DID,
				"success": false,
				"error":   "无效的DID格式",
			})
			continue
		}

		// 创建注册请求
		regReq := &did.RegisterRequest{
			DID: didReq.DID,
			// TODO: 从didReq.Document中解析VerificationMethod和Service
		}

		// 注册DID
		_, err := s.registry.Register(regReq)
		if err != nil {
			results = append(results, gin.H{
				"did":     didReq.DID,
				"success": false,
				"error":   err.Error(),
			})
		} else {
			results = append(results, gin.H{
				"did":     didReq.DID,
				"success": true,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(req.DIDs),
	})
}

// 批量解析DID
func (s *Server) batchResolveDID(c *gin.Context) {
	type BatchResolveRequest struct {
		DIDs []string `json:"dids" binding:"required"`
	}

	var req BatchResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results := make([]gin.H, 0, len(req.DIDs))
	for _, did := range req.DIDs {
		document, err := s.resolver.Resolve(did)
		if err != nil {
			results = append(results, gin.H{
				"did":     did,
				"success": false,
				"error":   err.Error(),
			})
		} else {
			results = append(results, gin.H{
				"did":      did,
				"success":  true,
				"document": document,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(req.DIDs),
	})
}

// 添加对等节点
func (s *Server) addPeer(c *gin.Context) {
	type AddPeerRequest struct {
		ID      string `json:"id" binding:"required"`
		Address string `json:"address" binding:"required"`
	}

	var req AddPeerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.peersMutex.Lock()
	s.peers[req.ID] = &PeerInfo{
		ID:       req.ID,
		Address:  req.Address,
		Status:   "connected",
		LastSeen: time.Now(),
	}
	s.peersMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "节点添加成功",
		"peer": gin.H{
			"id":      req.ID,
			"address": req.Address,
		},
	})
}

// 移除对等节点
func (s *Server) removePeer(c *gin.Context) {
	peerID := c.Param("id")
	if peerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "节点ID不能为空"})
		return
	}

	s.peersMutex.Lock()
	delete(s.peers, peerID)
	s.peersMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"message": "节点移除成功",
		"peer_id": peerID,
	})
}

// 获取节点状态
func (s *Server) getNodeStatus(c *gin.Context) {
	s.peersMutex.RLock()
	peerCount := len(s.peers)
	s.peersMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"node_id":    s.nodeID,
		"status":     "running",
		"peer_count": peerCount,
		"uptime":     time.Since(time.Now()).String(), // 这里应该记录实际启动时间
		"version":    "1.0.0",
	})
}

// 获取同步状态
func (s *Server) getSyncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"is_syncing":    false,
		"sync_progress": 100.0,
		"last_sync":     time.Now(),
		"block_height":  0, // 这里应该获取实际区块高度
	})
}

// 获取集群状态
func (s *Server) getClusterStatus(c *gin.Context) {
	s.peersMutex.RLock()
	peers := make([]gin.H, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, gin.H{
			"id":        peer.ID,
			"address":   peer.Address,
			"status":    peer.Status,
			"last_seen": peer.LastSeen,
		})
	}
	s.peersMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"cluster_size": len(peers) + 1, // +1 for current node
		"leader":       s.nodeID,       // 简化实现，实际应该通过共识算法确定
		"peers":        peers,
		"consensus":    "raft",
		"status":       "healthy",
	})
}

// 触发同步
func (s *Server) triggerSync(c *gin.Context) {
	// 这里应该实现实际的同步逻辑
	c.JSON(http.StatusOK, gin.H{
		"message":   "同步已触发",
		"timestamp": time.Now(),
	})
}

// 获取共识状态
func (s *Server) getConsensusStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"algorithm":    "raft",
		"leader":       s.nodeID,
		"term":         1,
		"status":       "stable",
		"last_applied": 0,
		"commit_index": 0,
	})
}
