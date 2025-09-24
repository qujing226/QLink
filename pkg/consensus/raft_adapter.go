package consensus

import (
	"context"
	"fmt"
	"sync"

	"github.com/qujing226/QLink/pkg/interfaces"
	"github.com/qujing226/QLink/pkg/network"
)

// RaftAdapter Raft共识算法适配器
type RaftAdapter struct {
	raftNode   *RaftNode
	p2pNetwork *network.P2PNetwork
	mu         sync.RWMutex
	metrics    *ConsensusMetricsData
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// NewRaftAdapter 创建新的Raft适配器
func NewRaftAdapter(nodeID string, peers []string, p2pNetwork *network.P2PNetwork) *RaftAdapter {
	raftNode := NewRaftNode(nodeID, p2pNetwork)
	
	// 添加对等节点
	for _, peer := range peers {
		if peer != nodeID {
			raftNode.AddPeer(peer, peer) // 简化处理，实际应该是地址
		}
	}
	
	return &RaftAdapter{
		raftNode: raftNode,
	}
}

// GetType 获取共识算法类型
func (ra *RaftAdapter) GetType() interfaces.ConsensusType {
	return interfaces.ConsensusTypeRaft
}

// GetName 获取共识算法名称
func (ra *RaftAdapter) GetName() string {
	return "Raft"
}

// StartConsensus 启动共识算法
func (ra *RaftAdapter) StartConsensus() error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	
	if ra.running {
		return fmt.Errorf("Raft共识已经在运行")
	}
	
	ra.ctx, ra.cancel = context.WithCancel(context.Background())
	err := ra.raftNode.Start(ra.ctx)
	if err != nil {
		return fmt.Errorf("启动Raft节点失败: %v", err)
	}
	
	ra.running = true
	return nil
}

// StopConsensus 停止共识算法
func (ra *RaftAdapter) StopConsensus() error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	
	if !ra.running {
		return fmt.Errorf("Raft共识未在运行")
	}
	
	if ra.cancel != nil {
		ra.cancel()
	}
	
	err := ra.raftNode.Stop()
	if err != nil {
		return fmt.Errorf("停止Raft节点失败: %v", err)
	}
	
	ra.running = false
	return nil
}

// Submit 提交提案
func (ra *RaftAdapter) Submit(proposal interface{}) error {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	if !ra.running {
		return fmt.Errorf("Raft共识未在运行")
	}
	
	return ra.raftNode.Submit(proposal)
}

// GetStatus 获取共识状态
func (ra *RaftAdapter) GetStatus() map[string]interface{} {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	state, term, isLeader := ra.raftNode.GetState()
	
	status := map[string]interface{}{
		"type":      "Raft",
		"running":   ra.running,
		"state":     state.String(),
		"term":      term,
		"isLeader":  isLeader,
		"nodeID":    ra.raftNode.id,
	}
	
	// 添加对等节点信息
	peers := make([]string, 0)
	for peerID := range ra.raftNode.peers {
		peers = append(peers, peerID)
	}
	status["peers"] = peers
	
	return status
}

// GetLeader 获取当前Leader
func (ra *RaftAdapter) GetLeader() string {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	state, _, isLeader := ra.raftNode.GetState()
	if isLeader {
		return ra.raftNode.id
	}
	
	// 如果不是Leader，返回空字符串或已知的Leader ID
	// 在实际实现中，可以维护Leader信息
	if state == Follower {
		return ra.raftNode.votedFor // 简化处理
	}
	
	return ""
}

// GetNodes 获取所有节点列表
func (ra *RaftAdapter) GetNodes() []string {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	nodes := make([]string, 0, len(ra.raftNode.peers)+1)
	nodes = append(nodes, ra.raftNode.id) // 添加自己
	
	for peerID := range ra.raftNode.peers {
		nodes = append(nodes, peerID)
	}
	
	return nodes
}

// ValidateBlock 验证区块（Raft适配器实现）
func (ra *RaftAdapter) ValidateBlock(block interface{}) error {
	// Raft本身不直接验证区块，这里可以添加基本验证逻辑
	if block == nil {
		return fmt.Errorf("区块不能为空")
	}
	
	// 可以添加更多验证逻辑，如区块格式、签名等
	return nil
}

// ValidateProposer 验证提案者
func (ra *RaftAdapter) ValidateProposer(proposer string, blockNumber uint64) error {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	// 在Raft中，只有Leader可以提出提案
	_, _, isLeader := ra.raftNode.GetState()
	if !isLeader && proposer == ra.raftNode.id {
		return fmt.Errorf("节点 %s 不是Leader，无法提出提案", proposer)
	}
	
	// 检查提案者是否是已知节点
	if proposer != ra.raftNode.id {
		if _, exists := ra.raftNode.peers[proposer]; !exists {
			return fmt.Errorf("未知的提案者: %s", proposer)
		}
	}
	
	return nil
}

// GetNextProposer 获取下一个提案者
func (ra *RaftAdapter) GetNextProposer(blockNumber uint64) string {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	// 在Raft中，只有Leader可以提出提案
	_, _, isLeader := ra.raftNode.GetState()
	if isLeader {
		return ra.raftNode.id
	}
	
	// 如果当前节点不是Leader，返回已知的Leader
	return ra.GetLeader()
}

// IsAuthority 检查是否为权威节点
func (ra *RaftAdapter) IsAuthority(address string) bool {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	// 在Raft中，所有节点都是权威节点
	if address == ra.raftNode.id {
		return true
	}
	
	_, exists := ra.raftNode.peers[address]
	return exists
}

// GetAuthorities 获取所有权威节点
func (ra *RaftAdapter) GetAuthorities() []string {
	// 在Raft中，所有节点都是权威节点
	return ra.GetNodes()
}

// Start 启动共识算法（实现ConsensusAlgorithm接口）
func (ra *RaftAdapter) Start(ctx context.Context) error {
	return ra.StartConsensus()
}

// Stop 停止共识算法（实现ConsensusAlgorithm接口）
func (ra *RaftAdapter) Stop() error {
	return ra.StopConsensus()
}

// StartEngine 启动共识引擎（实现ConsensusEngine接口）
func (ra *RaftAdapter) StartEngine() error {
	return ra.StartConsensus()
}

// StopEngine 停止共识引擎（实现ConsensusEngine接口）
func (ra *RaftAdapter) StopEngine() error {
	return ra.StopConsensus()
}

// AddPeer 添加对等节点
func (ra *RaftAdapter) AddPeer(nodeID, address string) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	
	ra.raftNode.AddPeer(nodeID, address)
	return nil
}

// RemovePeer 移除对等节点
func (ra *RaftAdapter) RemovePeer(nodeID string) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	
	ra.raftNode.RemovePeer(nodeID)
	return nil
}

// IsRunning 检查是否正在运行
func (ra *RaftAdapter) IsRunning() bool {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	return ra.running
}

// GetMetrics 获取性能指标
func (ra *RaftAdapter) GetMetrics() map[string]interface{} {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	
	state, term, _ := ra.raftNode.GetState()
	
	metrics := map[string]interface{}{
		"current_term":    term,
		"log_entries":     len(ra.raftNode.log),
		"commit_index":    ra.raftNode.commitIndex,
		"last_applied":    ra.raftNode.lastApplied,
		"peer_count":      len(ra.raftNode.peers),
		"state":           state.String(),
		"election_timeout": ra.raftNode.electionTimeout.String(),
		"heartbeat_interval": ra.raftNode.heartbeatInterval.String(),
	}
	
	return metrics
}