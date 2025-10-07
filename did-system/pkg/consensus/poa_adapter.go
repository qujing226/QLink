package consensus

import (
	"context"
	"fmt"
	"sync"

	"github.com/qujing226/QLink/pkg/interfaces"
	"github.com/qujing226/QLink/pkg/network"
)

// PoAAdapter PoA共识算法适配器，实现统一的共识接口
type PoAAdapter struct {
	poaNode    *PoANode
	p2pNetwork *network.P2PNetwork
	mu         sync.RWMutex
	metrics    *ConsensusMetricsData
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// NewPoAAdapter 创建新的PoA适配器
func NewPoAAdapter(nodeID string, authorities []string, p2pNetwork *network.P2PNetwork) *PoAAdapter {
	poaNode := NewPoANode(nodeID, authorities, p2pNetwork)

	return &PoAAdapter{
		poaNode: poaNode,
	}
}

// GetType 获取共识算法类型
func (pa *PoAAdapter) GetType() interfaces.ConsensusType {
	return interfaces.ConsensusTypePoA
}

// GetName 获取共识算法名称
func (pa *PoAAdapter) GetName() string {
	return "Proof of Authority"
}

// StartConsensus 启动共识算法
func (pa *PoAAdapter) StartConsensus() error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	if pa.running {
		return fmt.Errorf("PoA共识已经在运行")
	}

	pa.ctx, pa.cancel = context.WithCancel(context.Background())
	if err := pa.poaNode.Start(pa.ctx); err != nil {
		return fmt.Errorf("启动PoA节点失败: %v", err)
	}

	pa.running = true
	return nil
}

// StopConsensus 停止共识算法
func (pa *PoAAdapter) StopConsensus() error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	if !pa.running {
		return fmt.Errorf("PoA共识未在运行")
	}

	if pa.cancel != nil {
		pa.cancel()
	}

	err := pa.poaNode.Stop()
	if err != nil {
		return fmt.Errorf("停止PoA节点失败: %v", err)
	}

	pa.running = false
	return nil
}

// Submit 提交提案
func (pa *PoAAdapter) Submit(proposal interface{}) error {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	if !pa.running {
		return fmt.Errorf("PoA共识未在运行")
	}

	return pa.poaNode.Submit(proposal)
}

// GetStatus 获取共识状态
func (pa *PoAAdapter) GetStatus() map[string]interface{} {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	status := pa.poaNode.GetStatus()

	// 添加适配器特定信息
	status["type"] = "PoA"
	status["running"] = pa.running

	return status
}

// GetLeader 获取当前Leader（在PoA中是当前轮次的提案者）
func (pa *PoAAdapter) GetLeader() string {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	// 在PoA中，Leader是当前轮次应该出块的权威节点
	// 这里简化处理，返回当前区块的提案者
	currentBlock := pa.poaNode.GetCurrentBlock()
	if currentBlock != nil {
		return currentBlock.Proposer
	}

	return ""
}

// GetNodes 获取所有节点列表
func (pa *PoAAdapter) GetNodes() []string {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	return pa.poaNode.GetAuthorities()
}

// ValidateBlock 验证区块
func (pa *PoAAdapter) ValidateBlock(block interface{}) error {
	if block == nil {
		return fmt.Errorf("区块不能为空")
	}

	// 可以添加PoA特定的区块验证逻辑
	// 例如验证提案者是否为权威节点、签名验证等

	return nil
}

// ValidateProposer 验证提案者
func (pa *PoAAdapter) ValidateProposer(proposer string, blockNumber uint64) error {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	// 检查提案者是否为权威节点
	authorities := pa.poaNode.GetAuthorities()
	for _, authority := range authorities {
		if authority == proposer {
			return nil // 提案者是权威节点
		}
	}

	return fmt.Errorf("提案者 %s 不是权威节点", proposer)
}

// GetNextProposer 获取下一个提案者
func (pa *PoAAdapter) GetNextProposer(blockNumber uint64) string {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	authorities := pa.poaNode.GetAuthorities()
	if len(authorities) == 0 {
		return ""
	}

	// 使用轮询方式确定下一个提案者
	nextIndex := int(blockNumber) % len(authorities)
	return authorities[nextIndex]
}

// IsAuthority 检查是否为权威节点
func (pa *PoAAdapter) IsAuthority(address string) bool {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	authorities := pa.poaNode.GetAuthorities()
	for _, authority := range authorities {
		if authority == address {
			return true
		}
	}

	return false
}

// GetAuthorities 获取所有权威节点
func (pa *PoAAdapter) GetAuthorities() []string {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	return pa.poaNode.GetAuthorities()
}

// Start 启动共识算法（实现ConsensusAlgorithm接口）
func (pa *PoAAdapter) Start(ctx context.Context) error {
	return pa.StartConsensus()
}

// Stop 停止共识算法（实现ConsensusAlgorithm接口）
func (pa *PoAAdapter) Stop() error {
	return pa.StopConsensus()
}

// StartEngine 启动共识引擎（实现ConsensusEngine接口）
func (pa *PoAAdapter) StartEngine() error {
	return pa.StartConsensus()
}

// StopEngine 停止共识引擎（实现ConsensusEngine接口）
func (pa *PoAAdapter) StopEngine() error {
	return pa.StopConsensus()
}

// IsRunning 检查是否正在运行
func (pa *PoAAdapter) IsRunning() bool {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	return pa.running
}

// GetMetrics 获取性能指标
func (pa *PoAAdapter) GetMetrics() map[string]interface{} {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	status := pa.poaNode.GetStatus()
	currentBlock := pa.poaNode.GetCurrentBlock()

	metrics := map[string]interface{}{
		"block_height":     status["block_height"],
		"authority_count":  len(pa.poaNode.GetAuthorities()),
		"is_authority":     pa.poaNode.IsAuthorityNode(),
		"proposal_count":   len(pa.poaNode.proposals),
		"current_proposer": "",
	}

	if currentBlock != nil {
		metrics["current_proposer"] = currentBlock.Proposer
		metrics["last_block_time"] = currentBlock.Timestamp
	}

	return metrics
}

// AddAuthority 添加权威节点
func (pa *PoAAdapter) AddAuthority(nodeID string) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// 直接检查权威节点列表，避免调用IsAuthority方法导致死锁
	authorities := pa.poaNode.GetAuthorities()
	for _, authority := range authorities {
		if authority == nodeID {
			return fmt.Errorf("节点 %s 已经是权威节点", nodeID)
		}
	}

	// 添加到权威节点列表
	pa.poaNode.authorities = append(pa.poaNode.authorities, nodeID)

	return nil
}

// RemoveAuthority 移除权威节点
func (pa *PoAAdapter) RemoveAuthority(nodeID string) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// 查找并移除权威节点
	for i, authority := range pa.poaNode.authorities {
		if authority == nodeID {
			pa.poaNode.authorities = append(pa.poaNode.authorities[:i], pa.poaNode.authorities[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("节点 %s 不是权威节点", nodeID)
}

// GetCurrentBlockHeight 获取当前区块高度
func (pa *PoAAdapter) GetCurrentBlockHeight() int64 {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	return pa.poaNode.blockHeight
}

// GetCurrentBlock 获取当前区块
func (pa *PoAAdapter) GetCurrentBlock() interface{} {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	return pa.poaNode.GetCurrentBlock()
}
