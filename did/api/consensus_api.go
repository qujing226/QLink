package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qujing226/QLink/did/consensus"
)

// ConsensusAPI 共识API接口
type ConsensusAPI struct {
	consensusIntegration *consensus.ConsensusIntegration
}

// NewConsensusAPI 创建共识API
func NewConsensusAPI(ci *consensus.ConsensusIntegration) *ConsensusAPI {
	return &ConsensusAPI{
		consensusIntegration: ci,
	}
}

// RegisterRoutes 注册路由
func (api *ConsensusAPI) RegisterRoutes(router *gin.Engine) {
	consensusGroup := router.Group("/consensus")
	{
		consensusGroup.POST("/propose", api.ProposeOperation)
		consensusGroup.GET("/status", api.GetConsensusStatus)
		consensusGroup.GET("/nodes", api.GetNodes)
		consensusGroup.GET("/leader", api.GetLeader)
		consensusGroup.GET("/metrics", api.GetMetrics)
		consensusGroup.GET("/health", api.HealthCheck)
	}
}

// ProposeOperationRequest 提议操作请求
type ProposeOperationRequest struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ProposeOperation 提议操作
func (api *ConsensusAPI) ProposeOperation(c *gin.Context) {
	var req ProposeOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("解析请求失败: %v", err)})
		return
	}

	// 验证必需字段
	if req.Data == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少数据字段"})
		return
	}

	// 将字符串类型转换为ProposalType
	var proposalType consensus.ProposalType
	switch req.Type {
	case "did_operation":
		proposalType = consensus.ProposalTypeDIDCreate
	case "config_change":
		proposalType = consensus.ProposalTypeConfigUpdate
	case "node_join":
		proposalType = consensus.ProposalTypeNodeJoin
	case "node_leave":
		proposalType = consensus.ProposalTypeNodeLeave
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的操作类型"})
		return
	}

	// 提议操作
	proposal, err := api.consensusIntegration.ProposeOperation(proposalType, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("提议操作失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "操作提议成功",
		"proposal_id": proposal.ID,
	})
}

// GetConsensusStatus 获取共识状态
func (api *ConsensusAPI) GetConsensusStatus(c *gin.Context) {
	status := api.consensusIntegration.GetStatus()
	c.JSON(http.StatusOK, status)
}

// GetNodes 获取节点列表
func (api *ConsensusAPI) GetNodes(c *gin.Context) {
	nodes := api.consensusIntegration.GetNodes()

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"count": len(nodes),
	})
}

// GetLeader 获取当前领导者
func (api *ConsensusAPI) GetLeader(c *gin.Context) {
	leader := api.consensusIntegration.GetLeader()

	c.JSON(http.StatusOK, gin.H{
		"leader": leader,
	})
}

// ConsensusMetrics 共识指标
type ConsensusMetrics struct {
	TotalProposals    int64   `json:"total_proposals"`
	SuccessfulCommits int64   `json:"successful_commits"`
	FailedCommits     int64   `json:"failed_commits"`
	AverageLatency    float64 `json:"average_latency_ms"`
	CurrentTerm       int64   `json:"current_term"`
	LastCommitIndex   int64   `json:"last_commit_index"`
}

// GetMetrics 获取共识指标
func (api *ConsensusAPI) GetMetrics(c *gin.Context) {
	// 这里应该从共识模块获取实际指标
	metrics := &ConsensusMetrics{
		TotalProposals:    100,
		SuccessfulCommits: 95,
		FailedCommits:     5,
		AverageLatency:    50.5,
		CurrentTerm:       1,
		LastCommitIndex:   95,
	}

	c.JSON(http.StatusOK, metrics)
}

// HealthCheck 健康检查
func (api *ConsensusAPI) HealthCheck(c *gin.Context) {
	isHealthy := api.consensusIntegration.IsHealthy()

	status := "healthy"
	statusCode := http.StatusOK

	if !isHealthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": fmt.Sprintf("%d", c.Request.Context().Value("timestamp")),
	})
}