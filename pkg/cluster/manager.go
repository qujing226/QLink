package cluster

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/config"
	"github.com/qujing226/QLink/pkg/consensus"
	"github.com/qujing226/QLink/pkg/network"
	syncpkg "github.com/qujing226/QLink/pkg/sync"
)

// ClusterManager 集群管理器
type ClusterManager struct {
	nodeID       string
	clusterID    string
	p2pNetwork   *network.P2PNetwork
	raftNode     *consensus.RaftNode
	synchronizer *syncpkg.Synchronizer

	// 集群状态
	clusterState *ClusterState
	stateMutex   sync.RWMutex

	// 节点管理
	nodes      map[string]*NodeInfo
	nodesMutex sync.RWMutex

	// 配置
	config *config.ClusterConfig

	// 控制通道
	stopCh chan struct{}
}

// ClusterState 集群状态
type ClusterState struct {
	ID         string                 `json:"id"`
	Leader     string                 `json:"leader"`
	Term       int64                  `json:"term"`
	Status     ClusterStatus          `json:"status"`
	NodeCount  int                    `json:"node_count"`
	LastUpdate time.Time              `json:"last_update"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ClusterStatus 集群状态枚举
type ClusterStatus int

const (
	ClusterStatusInitializing ClusterStatus = iota
	ClusterStatusActive
	ClusterStatusElecting
	ClusterStatusSplitBrain
	ClusterStatusShutdown
)

// NodeInfo 节点信息
type NodeInfo struct {
	ID           string            `json:"id"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	Role         NodeRole          `json:"role"`
	Status       NodeStatus        `json:"status"`
	LastSeen     time.Time         `json:"last_seen"`
	Version      string            `json:"version"`
	Metadata     map[string]string `json:"metadata"`
	Capabilities []string          `json:"capabilities"`
}

// NodeRole 节点角色
type NodeRole int

const (
	NodeRoleFollower NodeRole = iota
	NodeRoleCandidate
	NodeRoleLeader
)

// NodeStatus 节点状态
type NodeStatus int

const (
	NodeStatusJoining NodeStatus = iota
	NodeStatusActive
	NodeStatusInactive
	NodeStatusLeaving
	NodeStatusFailed
)

// JoinRequest 加入集群请求
type JoinRequest struct {
	NodeID       string            `json:"node_id"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	Version      string            `json:"version"`
	Capabilities []string          `json:"capabilities"`
	Metadata     map[string]string `json:"metadata"`
}

// JoinResponse 加入集群响应
type JoinResponse struct {
	Accepted  bool                   `json:"accepted"`
	ClusterID string                 `json:"cluster_id"`
	Leader    string                 `json:"leader"`
	Nodes     []*NodeInfo            `json:"nodes"`
	Config    *config.ClusterConfig  `json:"config"`
	Reason    string                 `json:"reason,omitempty"`
}

// LeaveRequest 离开集群请求
type LeaveRequest struct {
	NodeID string `json:"node_id"`
	Reason string `json:"reason"`
}

// NewClusterManager 创建集群管理器
func NewClusterManager(nodeID, clusterID string, p2pNetwork *network.P2PNetwork,
	raftNode *consensus.RaftNode, synchronizer *syncpkg.Synchronizer, cfg *config.ClusterConfig) *ClusterManager {

	if cfg == nil {
		cfg = &config.ClusterConfig{
			ID:                  clusterID,
			MaxNodes:            10,
			MinNodes:            1,
			JoinTimeout:         30 * time.Second,
			SyncInterval:        10 * time.Second,
			HealthCheckInterval: 5 * time.Second,
			HeartbeatInterval:   3 * time.Second,
			ElectionTimeout:     5 * time.Second,
			AutoJoin:            false,
			BootstrapNodes:      []string{},
			Enabled:             true,
			Peers:               []string{},
		}
	}

	return &ClusterManager{
		nodeID:       nodeID,
		clusterID:    clusterID,
		p2pNetwork:   p2pNetwork,
		raftNode:     raftNode,
		synchronizer: synchronizer,
		config:       cfg,
		nodes:        make(map[string]*NodeInfo),
		clusterState: &ClusterState{
			ID:         clusterID,
			Status:     ClusterStatusInitializing,
			LastUpdate: time.Now(),
			Metadata:   make(map[string]interface{}),
		},
		stopCh: make(chan struct{}),
	}
}

// Start 启动集群管理器
func (cm *ClusterManager) Start(ctx context.Context) error {
	log.Printf("启动集群管理器，节点ID: %s, 集群ID: %s", cm.nodeID, cm.clusterID)

	// 注册网络消息处理器
	cm.p2pNetwork.RegisterMessageHandler(network.MessageTypeConsensus, cm.handleConsensusMessage)

	// 启动心跳检查
	go cm.heartbeatLoop(ctx)

	// 启动状态监控
	go cm.statusMonitor(ctx)

	// 启动集群同步
	go cm.clusterSync(ctx)

	// 更新集群状态
	cm.stateMutex.Lock()
	cm.clusterState.Status = ClusterStatusActive
	cm.clusterState.LastUpdate = time.Now()
	cm.stateMutex.Unlock()

	return nil
}

// Stop 停止集群管理器
func (cm *ClusterManager) Stop() error {
	close(cm.stopCh)

	// 通知其他节点离开
	cm.broadcastLeaveMessage()

	log.Printf("集群管理器已停止")
	return nil
}

// JoinCluster 加入集群
func (cm *ClusterManager) JoinCluster(leaderAddress string, leaderPort int) error {
	log.Printf("尝试加入集群，Leader: %s:%d", leaderAddress, leaderPort)

	// 构造加入请求
	joinReq := &JoinRequest{
		NodeID:       cm.nodeID,
		Address:      cm.p2pNetwork.GetNetworkStatus()["listening_address"].(string),
		Port:         leaderPort, // 简化处理
		Version:      "1.0.0",
		Capabilities: []string{"did", "consensus", "sync"},
		Metadata:     map[string]string{"role": "follower"},
	}

	// 发送加入请求
	response, err := cm.sendJoinRequest(leaderAddress, leaderPort, joinReq)
	if err != nil {
		return fmt.Errorf("发送加入请求失败: %w", err)
	}

	if !response.Accepted {
		return fmt.Errorf("加入集群被拒绝: %s", response.Reason)
	}

	// 更新集群状态
	cm.stateMutex.Lock()
	cm.clusterState.ID = response.ClusterID
	cm.clusterState.Leader = response.Leader
	cm.clusterState.Status = ClusterStatusActive
	cm.clusterState.LastUpdate = time.Now()
	cm.stateMutex.Unlock()

	// 添加集群节点
	cm.nodesMutex.Lock()
	for _, node := range response.Nodes {
		cm.nodes[node.ID] = node
		// 添加到P2P网络
		cm.p2pNetwork.AddPeer(node.ID, node.Address, node.Port)
	}
	cm.nodesMutex.Unlock()

	log.Printf("成功加入集群: %s", response.ClusterID)
	return nil
}

// LeaveCluster 离开集群
func (cm *ClusterManager) LeaveCluster(reason string) error {
	log.Printf("离开集群，原因: %s", reason)

	// 发送离开消息
	cm.broadcastLeaveMessage()

	// 更新状态
	cm.stateMutex.Lock()
	cm.clusterState.Status = ClusterStatusShutdown
	cm.clusterState.LastUpdate = time.Now()
	cm.stateMutex.Unlock()

	return nil
}

// AddNode 添加节点到集群
func (cm *ClusterManager) AddNode(nodeInfo *NodeInfo) error {
	cm.nodesMutex.Lock()
	defer cm.nodesMutex.Unlock()

	if _, exists := cm.nodes[nodeInfo.ID]; exists {
		return fmt.Errorf("节点已存在: %s", nodeInfo.ID)
	}

	// 检查集群容量
	if len(cm.nodes) >= cm.config.MaxNodes {
		return fmt.Errorf("集群已达到最大节点数: %d", cm.config.MaxNodes)
	}

	cm.nodes[nodeInfo.ID] = nodeInfo

	// 添加到P2P网络
	cm.p2pNetwork.AddPeer(nodeInfo.ID, nodeInfo.Address, nodeInfo.Port)

	// 更新集群状态
	cm.stateMutex.Lock()
	cm.clusterState.NodeCount = len(cm.nodes)
	cm.clusterState.LastUpdate = time.Now()
	cm.stateMutex.Unlock()

	log.Printf("节点 %s 已添加到集群", nodeInfo.ID)
	return nil
}

// RemoveNode 从集群移除节点
func (cm *ClusterManager) RemoveNode(nodeID string) error {
	cm.nodesMutex.Lock()
	defer cm.nodesMutex.Unlock()

	if _, exists := cm.nodes[nodeID]; !exists {
		return fmt.Errorf("节点不存在: %s", nodeID)
	}

	delete(cm.nodes, nodeID)

	// 从P2P网络移除
	cm.p2pNetwork.RemovePeer(nodeID)

	// 更新集群状态
	cm.stateMutex.Lock()
	cm.clusterState.NodeCount = len(cm.nodes)
	cm.clusterState.LastUpdate = time.Now()
	cm.stateMutex.Unlock()

	log.Printf("节点 %s 已从集群移除", nodeID)
	return nil
}

// GetClusterStatus 获取集群状态
func (cm *ClusterManager) GetClusterStatus() *ClusterState {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	// 深拷贝状态
	status := &ClusterState{
		ID:         cm.clusterState.ID,
		Leader:     cm.clusterState.Leader,
		Term:       cm.clusterState.Term,
		Status:     cm.clusterState.Status,
		NodeCount:  cm.clusterState.NodeCount,
		LastUpdate: cm.clusterState.LastUpdate,
		Metadata:   make(map[string]interface{}),
	}

	for k, v := range cm.clusterState.Metadata {
		status.Metadata[k] = v
	}

	return status
}

// GetNodes 获取集群节点列表
func (cm *ClusterManager) GetNodes() []*NodeInfo {
	cm.nodesMutex.RLock()
	defer cm.nodesMutex.RUnlock()

	nodes := make([]*NodeInfo, 0, len(cm.nodes))
	for _, node := range cm.nodes {
		// 深拷贝节点信息
		nodeCopy := &NodeInfo{
			ID:           node.ID,
			Address:      node.Address,
			Port:         node.Port,
			Role:         node.Role,
			Status:       node.Status,
			LastSeen:     node.LastSeen,
			Version:      node.Version,
			Capabilities: make([]string, len(node.Capabilities)),
			Metadata:     make(map[string]string),
		}

		copy(nodeCopy.Capabilities, node.Capabilities)
		for k, v := range node.Metadata {
			nodeCopy.Metadata[k] = v
		}

		nodes = append(nodes, nodeCopy)
	}

	return nodes
}

// IsLeader 检查当前节点是否为Leader
func (cm *ClusterManager) IsLeader() bool {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	return cm.clusterState.Leader == cm.nodeID
}

// GetLeader 获取当前Leader节点ID
func (cm *ClusterManager) GetLeader() string {
	cm.stateMutex.RLock()
	defer cm.stateMutex.RUnlock()

	return cm.clusterState.Leader
}

// heartbeatLoop 心跳循环
func (cm *ClusterManager) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.sendHeartbeats()
			cm.checkNodeHealth()
		}
	}
}

// statusMonitor 状态监控
func (cm *ClusterManager) statusMonitor(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.updateClusterMetrics()
		}
	}
}

// clusterSync 集群同步
func (cm *ClusterManager) clusterSync(ctx context.Context) {
	ticker := time.NewTicker(cm.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			if cm.IsLeader() {
				cm.triggerClusterSync()
			}
		}
	}
}

// handleConsensusMessage 处理共识消息
func (cm *ClusterManager) handleConsensusMessage(peer *network.Peer, msg *network.Message) error {
	log.Printf("收到来自 %s 的共识消息", msg.From)

	// 这里应该将消息转发给Raft节点处理
	// 简化实现，直接记录日志

	return nil
}

// sendJoinRequest 发送加入请求
func (cm *ClusterManager) sendJoinRequest(address string, port int, req *JoinRequest) (*JoinResponse, error) {
	// 简化实现，实际应该通过HTTP或gRPC发送请求
	log.Printf("发送加入请求到 %s:%d", address, port)

	// 模拟响应
	return &JoinResponse{
		Accepted:  true,
		ClusterID: cm.clusterID,
		Leader:    "leader-node",
		Nodes:     []*NodeInfo{},
		Config:    cm.config,
	}, nil
}

// broadcastLeaveMessage 广播离开消息
func (cm *ClusterManager) broadcastLeaveMessage() {
	leaveReq := &LeaveRequest{
		NodeID: cm.nodeID,
		Reason: "shutdown",
	}

	cm.p2pNetwork.BroadcastMessage(network.MessageTypeConsensus, leaveReq)
}

// sendHeartbeats 发送心跳
func (cm *ClusterManager) sendHeartbeats() {
	heartbeat := map[string]interface{}{
		"node_id":   cm.nodeID,
		"timestamp": time.Now(),
		"status":    "active",
	}

	cm.p2pNetwork.BroadcastMessage(network.MessageTypeHeartbeat, heartbeat)
}

// checkNodeHealth 检查节点健康状态
func (cm *ClusterManager) checkNodeHealth() {
	cm.nodesMutex.Lock()
	defer cm.nodesMutex.Unlock()

	now := time.Now()
	timeout := cm.config.HeartbeatInterval * 3

	for nodeID, node := range cm.nodes {
		if now.Sub(node.LastSeen) > timeout {
			log.Printf("节点 %s 心跳超时，标记为失败", nodeID)
			node.Status = NodeStatusFailed
		}
	}
}

// updateClusterMetrics 更新集群指标
func (cm *ClusterManager) updateClusterMetrics() {
	cm.stateMutex.Lock()
	defer cm.stateMutex.Unlock()

	activeNodes := 0
	cm.nodesMutex.RLock()
	for _, node := range cm.nodes {
		if node.Status == NodeStatusActive {
			activeNodes++
		}
	}
	cm.nodesMutex.RUnlock()

	cm.clusterState.Metadata["active_nodes"] = activeNodes
	cm.clusterState.Metadata["total_nodes"] = len(cm.nodes)
	cm.clusterState.LastUpdate = time.Now()
}

// triggerClusterSync 触发集群同步
func (cm *ClusterManager) triggerClusterSync() {
	log.Printf("触发集群数据同步")

	// 触发数据同步器
	if err := cm.synchronizer.TriggerSync(); err != nil {
		log.Printf("触发同步失败: %v", err)
	}
}

// ElectLeader 选举Leader
func (cm *ClusterManager) ElectLeader() error {
	log.Printf("开始Leader选举")

	cm.stateMutex.Lock()
	cm.clusterState.Status = ClusterStatusElecting
	cm.clusterState.LastUpdate = time.Now()
	cm.stateMutex.Unlock()

	// 这里应该调用Raft选举逻辑
	// 简化实现，直接设置当前节点为Leader
	if cm.raftNode != nil {
		// 实际应该调用raft选举
		log.Printf("Raft选举逻辑待实现")
	}

	return nil
}

// HandleNodeJoin 处理节点加入
func (cm *ClusterManager) HandleNodeJoin(req *JoinRequest) (*JoinResponse, error) {
	log.Printf("处理节点 %s 的加入请求", req.NodeID)

	// 检查是否为Leader
	if !cm.IsLeader() {
		return &JoinResponse{
			Accepted: false,
			Reason:   "not leader",
		}, nil
	}

	// 检查集群容量
	if len(cm.nodes) >= cm.config.MaxNodes {
		return &JoinResponse{
			Accepted: false,
			Reason:   "cluster full",
		}, nil
	}

	// 创建节点信息
	nodeInfo := &NodeInfo{
		ID:           req.NodeID,
		Address:      req.Address,
		Port:         req.Port,
		Role:         NodeRoleFollower,
		Status:       NodeStatusJoining,
		LastSeen:     time.Now(),
		Version:      req.Version,
		Capabilities: req.Capabilities,
		Metadata:     req.Metadata,
	}

	// 添加节点
	if err := cm.AddNode(nodeInfo); err != nil {
		return &JoinResponse{
			Accepted: false,
			Reason:   err.Error(),
		}, nil
	}

	return &JoinResponse{
		Accepted:  true,
		ClusterID: cm.clusterID,
		Leader:    cm.nodeID,
		Nodes:     cm.GetNodes(),
		Config:    cm.config,
	}, nil
}
