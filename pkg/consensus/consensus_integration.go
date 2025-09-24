package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/pkg/config"
	"github.com/qujing226/QLink/pkg/network"
	"github.com/qujing226/QLink/pkg/types"
)

// ConsensusIntegration 共识集成器
type ConsensusIntegration struct {
	nodeID      string
	raftNode    *RaftNode
	didRegistry *did.DIDRegistry
	p2pNetwork  *network.P2PNetwork

	// 状态管理
	state      ConsensusState
	stateMutex sync.RWMutex

	// 提案管理
	proposals      map[string]*Proposal
	proposalsMutex sync.RWMutex

	// 配置
	config *config.ConsensusConfig

	// 控制通道
	stopCh chan struct{}
}

// ConsensusState 共识状态
type ConsensusState struct {
	Term             int64     `json:"term"`
	Leader           string    `json:"leader"`
	Status           string    `json:"status"` // "active", "electing", "inactive"
	LastCommitIndex  int64     `json:"last_commit_index"`
	PendingProposals int       `json:"pending_proposals"`
	LastUpdate       time.Time `json:"last_update"`
}



// Proposal 提案
type Proposal struct {
	ID          string          `json:"id"`
	Type        ProposalType    `json:"type"`
	Data        interface{}     `json:"data"`
	Proposer    string          `json:"proposer"`
	Term        int64           `json:"term"`
	Timestamp   time.Time       `json:"timestamp"`
	Status      ProposalStatus  `json:"status"`
	Votes       map[string]bool `json:"votes"`
	CommitIndex int64           `json:"commit_index"`
}

// ProposalType 提案类型
type ProposalType = types.OperationType

const (
	ProposalTypeDIDCreate     = types.OperationTypeDIDCreate
	ProposalTypeDIDUpdate     = types.OperationTypeDIDUpdate
	ProposalTypeDIDDeactivate = types.OperationTypeDIDDeactivate
	ProposalTypeNodeJoin      = types.OperationTypeNodeJoin
	ProposalTypeNodeLeave     = types.OperationTypeNodeLeave
	ProposalTypeConfigUpdate  = types.OperationTypeConfigUpdate
)

// ProposalStatus 提案状态
type ProposalStatus = types.OperationStatus

const (
	ProposalStatusPending   = types.OperationStatusPending
	ProposalStatusVoting    = types.OperationStatusProcessing
	ProposalStatusCommitted = types.OperationStatusCommitted
	ProposalStatusRejected  = types.OperationStatusRejected
	ProposalStatusTimeout   = types.OperationStatusTimeout
)

// DIDOperation DID操作
type DIDOperation = types.DIDOperation

// NewConsensusIntegration 创建共识集成实例
func NewConsensusIntegration(nodeID string, raftNode *RaftNode, didRegistry *did.DIDRegistry,
	p2pNetwork *network.P2PNetwork, cfg *config.ConsensusConfig) *ConsensusIntegration {

	return &ConsensusIntegration{
		nodeID:      nodeID,
		raftNode:    raftNode,
		didRegistry: didRegistry,
		p2pNetwork:  p2pNetwork,
		state: ConsensusState{
			Term:             0,
			Leader:           "",
			Status:           "inactive",
			LastCommitIndex:  0,
			PendingProposals: 0,
			LastUpdate:       time.Now(),
		},
		proposals: make(map[string]*Proposal),
		config:    cfg,
		stopCh:    make(chan struct{}),
	}
}

// Start 启动共识集成器
func (ci *ConsensusIntegration) Start(ctx context.Context) error {
	log.Printf("启动共识集成器，节点ID: %s", ci.nodeID)

	// 注册网络消息处理器
	ci.p2pNetwork.RegisterMessageHandler(network.MessageTypeConsensus, ci.handleConsensusMessage)

	// 启动状态监控
	go ci.stateMonitor(ctx)

	// 启动提案处理
	go ci.proposalProcessor(ctx)

	// 更新状态
	ci.stateMutex.Lock()
	ci.state.Status = "active"
	ci.state.LastUpdate = time.Now()
	ci.stateMutex.Unlock()

	return nil
}

// Stop 停止共识集成器
func (ci *ConsensusIntegration) Stop() error {
	close(ci.stopCh)
	log.Printf("共识集成器已停止")
	return nil
}

// ProposeOperation 提议操作
func (ci *ConsensusIntegration) ProposeOperation(opType ProposalType, data interface{}) (*Proposal, error) {
	// 检查是否为Leader
	if !ci.isLeader() {
		return nil, fmt.Errorf("只有Leader可以提议操作")
	}

	// 检查待处理提案数量
	ci.proposalsMutex.RLock()
	pendingCount := len(ci.proposals)
	ci.proposalsMutex.RUnlock()

	if pendingCount >= ci.config.MaxPendingProposals {
		return nil, fmt.Errorf("待处理提案过多: %d", pendingCount)
	}

	// 创建提案
	proposal := &Proposal{
		ID:        fmt.Sprintf("%s-%d-%d", ci.nodeID, time.Now().UnixNano(), opType),
		Type:      opType,
		Data:      data,
		Proposer:  ci.nodeID,
		Term:      ci.getCurrentTerm(),
		Timestamp: time.Now(),
		Status:    ProposalStatusPending,
		Votes:     make(map[string]bool),
	}

	// 保存提案
	ci.proposalsMutex.Lock()
	ci.proposals[proposal.ID] = proposal
	ci.proposalsMutex.Unlock()

	// 提交到Raft
	err := ci.raftNode.Submit(proposal)
	if err != nil {
		ci.proposalsMutex.Lock()
		delete(ci.proposals, proposal.ID)
		ci.proposalsMutex.Unlock()
		return nil, fmt.Errorf("提交提案到Raft失败: %w", err)
	}

	log.Printf("提案已提交: %s, 类型: %d", proposal.ID, proposal.Type)
	return proposal, nil
}

// ProposeDIDOperation 提议DID操作
func (ci *ConsensusIntegration) ProposeDIDOperation(operation string, didDoc *types.DIDDocument) (*Proposal, error) {
	didOp := &DIDOperation{
		Operation: operation,
		DID:       didDoc.ID,
		Document:  didDoc,
	}

	var proposalType ProposalType
	switch operation {
	case "create":
		proposalType = ProposalTypeDIDCreate
	case "update":
		proposalType = ProposalTypeDIDUpdate
	case "deactivate":
		proposalType = ProposalTypeDIDDeactivate
	default:
		return nil, fmt.Errorf("不支持的DID操作: %s", operation)
	}

	return ci.ProposeOperation(proposalType, didOp)
}

// GetProposal 获取提案
func (ci *ConsensusIntegration) GetProposal(proposalID string) (*Proposal, bool) {
	ci.proposalsMutex.RLock()
	defer ci.proposalsMutex.RUnlock()

	proposal, exists := ci.proposals[proposalID]
	return proposal, exists
}

// GetPendingProposals 获取待处理提案
func (ci *ConsensusIntegration) GetPendingProposals() []*Proposal {
	ci.proposalsMutex.RLock()
	defer ci.proposalsMutex.RUnlock()

	var pending []*Proposal
	for _, proposal := range ci.proposals {
		if proposal.Status == ProposalStatusPending || proposal.Status == ProposalStatusVoting {
			pending = append(pending, proposal)
		}
	}

	return pending
}

// GetConsensusState 获取共识状态
func (ci *ConsensusIntegration) GetConsensusState() ConsensusState {
	ci.stateMutex.RLock()
	defer ci.stateMutex.RUnlock()

	return ci.state
}

// isLeader 检查是否为Leader
func (ci *ConsensusIntegration) isLeader() bool {
	state, _, isLeader := ci.raftNode.GetState()
	return state == Leader && isLeader
}

// getCurrentTerm 获取当前任期
func (ci *ConsensusIntegration) getCurrentTerm() int64 {
	_, term, _ := ci.raftNode.GetState()
	return term
}

// stateMonitor 状态监控
func (ci *ConsensusIntegration) stateMonitor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ci.stopCh:
			return
		case <-ticker.C:
			ci.updateState()
		}
	}
}

// updateState 更新状态
func (ci *ConsensusIntegration) updateState() {
	state, term, isLeader := ci.raftNode.GetState()

	ci.stateMutex.Lock()
	ci.state.Term = term
	ci.state.LastUpdate = time.Now()

	if isLeader {
		ci.state.Leader = ci.nodeID
		ci.state.Status = "active"
	} else {
		switch state {
		case Follower:
			ci.state.Status = "active"
		case Candidate:
			ci.state.Status = "electing"
		default:
			ci.state.Status = "inactive"
		}
	}

	// 更新待处理提案数量
	ci.proposalsMutex.RLock()
	ci.state.PendingProposals = len(ci.proposals)
	ci.proposalsMutex.RUnlock()

	ci.stateMutex.Unlock()
}

// proposalProcessor 提案处理器
func (ci *ConsensusIntegration) proposalProcessor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ci.stopCh:
			return
		case <-ticker.C:
			ci.processTimeoutProposals()
		}
	}
}

// processTimeoutProposals 处理超时提案
func (ci *ConsensusIntegration) processTimeoutProposals() {
	ci.proposalsMutex.Lock()
	defer ci.proposalsMutex.Unlock()

	now := time.Now()
	for id, proposal := range ci.proposals {
		if proposal.Status == ProposalStatusPending || proposal.Status == ProposalStatusVoting {
			if now.Sub(proposal.Timestamp) > ci.config.ProposalTimeout {
				proposal.Status = ProposalStatusTimeout
				log.Printf("提案超时: %s", id)
			}
		}

		// 清理已完成的提案
		if proposal.Status == ProposalStatusCommitted ||
			proposal.Status == ProposalStatusRejected ||
			proposal.Status == ProposalStatusTimeout {
			if now.Sub(proposal.Timestamp) > time.Hour {
				delete(ci.proposals, id)
			}
		}
	}
}

// handleConsensusMessage 处理共识消息
func (ci *ConsensusIntegration) handleConsensusMessage(peer *network.Peer, msg *network.Message) error {
	// 将interface{}转换为[]byte
	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("序列化消息数据失败: %w", err)
	}

	var proposal Proposal
	if err := json.Unmarshal(dataBytes, &proposal); err != nil {
		return fmt.Errorf("解析共识消息失败: %w", err)
	}

	return ci.handleProposal(&proposal)
}

// handleProposal 处理提案
func (ci *ConsensusIntegration) handleProposal(proposal *Proposal) error {
	log.Printf("处理提案: %s, 类型: %d", proposal.ID, proposal.Type)

	// 验证提案
	if err := ci.validateProposal(proposal); err != nil {
		return fmt.Errorf("提案验证失败: %w", err)
	}

	// 执行提案
	switch proposal.Type {
	case ProposalTypeDIDCreate, ProposalTypeDIDUpdate, ProposalTypeDIDDeactivate:
		return ci.handleDIDProposal(proposal)
	case ProposalTypeNodeJoin, ProposalTypeNodeLeave:
		return ci.handleNodeProposal(proposal)
	case ProposalTypeConfigUpdate:
		return ci.handleConfigProposal(proposal)
	default:
		return fmt.Errorf("未知提案类型: %d", proposal.Type)
	}
}

// validateProposal 验证提案
func (ci *ConsensusIntegration) validateProposal(proposal *Proposal) error {
	// 基本验证
	if proposal.ID == "" || proposal.Proposer == "" {
		return fmt.Errorf("提案ID或提议者为空")
	}

	// 任期验证
	currentTerm := ci.getCurrentTerm()
	if proposal.Term > currentTerm {
		return fmt.Errorf("提案任期过新: %d > %d", proposal.Term, currentTerm)
	}

	return nil
}

// handleDIDProposal 处理DID提案
func (ci *ConsensusIntegration) handleDIDProposal(proposal *Proposal) error {
	didOp, ok := proposal.Data.(*DIDOperation)
	if !ok {
		return fmt.Errorf("无效的DID操作数据")
	}

	switch didOp.Operation {
	case "register":
		// 创建注册请求
		registerReq := &did.RegisterRequest{
			DID:                didOp.DID,
			VerificationMethod: didOp.Document.VerificationMethod,
			Service:            didOp.Document.Service,
		}
		_, err := ci.didRegistry.Register(registerReq)
		if err != nil {
			return fmt.Errorf("注册DID失败: %w", err)
		}
		log.Printf("DID创建成功: %s", didOp.DID)

	case "update":
		// 创建更新请求
		updateReq := &did.UpdateRequest{
			DID:                didOp.DID,
			VerificationMethod: didOp.Document.VerificationMethod,
			Service:            didOp.Document.Service,
			Proof:              didOp.Document.Proof,
		}
		_, err := ci.didRegistry.Update(updateReq)
		if err != nil {
			return fmt.Errorf("更新DID失败: %w", err)
		}
		log.Printf("DID更新成功: %s", didOp.DID)

	case "revoke":
		err := ci.didRegistry.Revoke(didOp.DID, didOp.Document.Proof)
		if err != nil {
			return fmt.Errorf("撤销DID失败: %w", err)
		}
		log.Printf("DID停用成功: %s", didOp.DID)

	default:
		return fmt.Errorf("不支持的DID操作: %s", didOp.Operation)
	}

	// 更新提案状态
	ci.proposalsMutex.Lock()
	if storedProposal, exists := ci.proposals[proposal.ID]; exists {
		storedProposal.Status = ProposalStatusCommitted
	}
	ci.proposalsMutex.Unlock()

	return nil
}

// handleNodeProposal 处理节点提案
func (ci *ConsensusIntegration) handleNodeProposal(proposal *Proposal) error {
	log.Printf("处理节点提案: %s", proposal.ID)

	// 更新提案状态
	ci.proposalsMutex.Lock()
	if storedProposal, exists := ci.proposals[proposal.ID]; exists {
		storedProposal.Status = ProposalStatusCommitted
	}
	ci.proposalsMutex.Unlock()

	return nil
}

// handleConfigProposal 处理配置提案
func (ci *ConsensusIntegration) handleConfigProposal(proposal *Proposal) error {
	log.Printf("处理配置提案: %s", proposal.ID)

	// 更新提案状态
	ci.proposalsMutex.Lock()
	if storedProposal, exists := ci.proposals[proposal.ID]; exists {
		storedProposal.Status = ProposalStatusCommitted
	}
	ci.proposalsMutex.Unlock()

	return nil
}

// GetStatus 获取共识状态
func (ci *ConsensusIntegration) GetStatus() map[string]interface{} {
	ci.stateMutex.RLock()
	defer ci.stateMutex.RUnlock()

	return map[string]interface{}{
		"is_leader":         ci.isLeader(),
		"current_term":      ci.getCurrentTerm(),
		"status":            ci.state.Status,
		"last_commit_index": ci.state.LastCommitIndex,
		"pending_proposals": ci.state.PendingProposals,
		"last_update":       ci.state.LastUpdate,
	}
}

// GetNodes 获取节点列表
func (ci *ConsensusIntegration) GetNodes() []string {
	// 从P2P网络获取连接的节点
	peers := ci.p2pNetwork.GetPeers()
	nodes := make([]string, 0, len(peers)+1)

	// 添加当前节点
	nodes = append(nodes, ci.nodeID)

	// 添加其他节点
	for peerID := range peers {
		nodes = append(nodes, peerID)
	}

	return nodes
}

// GetLeader 获取当前领导者
func (ci *ConsensusIntegration) GetLeader() string {
	ci.stateMutex.RLock()
	defer ci.stateMutex.RUnlock()
	return ci.state.Leader
}

// IsHealthy 检查共识模块是否健康
func (ci *ConsensusIntegration) IsHealthy() bool {
	// 检查Raft节点是否正常运行
	if ci.raftNode == nil {
		return false
	}

	// 检查网络连接是否正常
	if ci.p2pNetwork == nil {
		return false
	}

	// 检查状态是否正常
	ci.stateMutex.RLock()
	status := ci.state.Status
	ci.stateMutex.RUnlock()

	return status == "active" || status == "electing"
}
