package consensus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/qujing226/QLink/did/network"
	"github.com/qujing226/QLink/did/types"
)

// PoANode PoA共识节点
type PoANode struct {
	id         string
	authorities []string // 权威节点列表
	isAuthority bool     // 是否为权威节点
	
	// 网络通信
	p2pNetwork *network.P2PNetwork
	
	// 状态管理
	currentBlock *PoABlock
	blockHeight  int64
	
	// 提案管理
	proposals map[string]*PoAProposal
	votes     map[string]map[string]bool // proposalID -> nodeID -> vote
	
	// 控制
	mu     sync.RWMutex
	stopCh chan struct{}
	
	// 配置
	blockTime     time.Duration // 出块时间间隔
	voteThreshold float64       // 投票阈值
}

// PoABlock PoA区块结构
type PoABlock struct {
	Height    int64       `json:"height"`
	Hash      string      `json:"hash"`
	PrevHash  string      `json:"prev_hash"`
	Timestamp time.Time   `json:"timestamp"`
	Proposer  string      `json:"proposer"`
	Data      interface{} `json:"data"`
	Signature string      `json:"signature"`
}

// PoAProposal PoA提案结构
type PoAProposal struct {
	ID        string                `json:"id"`
	Height    int64                 `json:"height"`
	Block     *PoABlock             `json:"block"`
	Proposer  string                `json:"proposer"`
	Timestamp time.Time             `json:"timestamp"`
	Status    types.OperationStatus `json:"status"`
}

// NewPoANode 创建PoA节点
func NewPoANode(id string, authorities []string, p2pNetwork *network.P2PNetwork) *PoANode {
	// 检查是否为权威节点
	isAuthority := false
	for _, auth := range authorities {
		if auth == id {
			isAuthority = true
			break
		}
	}
	
	return &PoANode{
		id:            id,
		authorities:   authorities,
		isAuthority:   isAuthority,
		p2pNetwork:    p2pNetwork,
		proposals:     make(map[string]*PoAProposal),
		votes:         make(map[string]map[string]bool),
		stopCh:        make(chan struct{}),
		blockTime:     5 * time.Second, // 默认5秒出块
		voteThreshold: 0.67,            // 默认67%阈值
	}
}

// Start 启动PoA节点
func (poa *PoANode) Start(ctx context.Context) error {
	log.Printf("启动PoA节点: %s (权威节点: %v)", poa.id, poa.isAuthority)
	
	// 注册网络消息处理器
	if poa.p2pNetwork != nil {
		poa.p2pNetwork.RegisterMessageHandler(network.MessageTypeConsensus, poa.handleNetworkMessage)
	}
	
	// 如果是权威节点，启动出块循环
	if poa.isAuthority {
		go poa.blockProducerLoop(ctx)
	}
	
	// 启动提案处理循环
	go poa.proposalProcessorLoop(ctx)
	
	return nil
}

// Stop 停止PoA节点
func (poa *PoANode) Stop() error {
	close(poa.stopCh)
	log.Printf("PoA节点已停止: %s", poa.id)
	return nil
}

// Submit 提交操作
func (poa *PoANode) Submit(command interface{}) error {
	if !poa.isAuthority {
		return fmt.Errorf("只有权威节点可以提交操作")
	}
	
	// 创建新区块
	block := poa.createBlock(command)
	
	// 创建提案
	proposal := &PoAProposal{
		ID:        poa.generateProposalID(block),
		Height:    block.Height,
		Block:     block,
		Proposer:  poa.id,
		Timestamp: time.Now(),
		Status:    types.OperationStatusPending,
	}
	
	poa.mu.Lock()
	poa.proposals[proposal.ID] = proposal
	poa.votes[proposal.ID] = make(map[string]bool)
	poa.mu.Unlock()
	
	// 广播提案
	poa.broadcastProposal(proposal)
	
	log.Printf("提交提案: %s (高度: %d)", proposal.ID, proposal.Height)
	return nil
}

// blockProducerLoop 出块循环
func (poa *PoANode) blockProducerLoop(ctx context.Context) {
	ticker := time.NewTicker(poa.blockTime)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-poa.stopCh:
			return
		case <-ticker.C:
			// 检查是否轮到自己出块
			if poa.isMyTurnToPropose() {
				poa.proposeEmptyBlock()
			}
		}
	}
}

// proposalProcessorLoop 提案处理循环
func (poa *PoANode) proposalProcessorLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-poa.stopCh:
			return
		case <-ticker.C:
			poa.processProposals()
		}
	}
}

// isMyTurnToPropose 检查是否轮到自己出块
func (poa *PoANode) isMyTurnToPropose() bool {
	if !poa.isAuthority {
		return false
	}
	
	// 简单的轮询算法：根据时间戳和权威节点列表确定出块顺序
	now := time.Now().Unix()
	slotDuration := int64(poa.blockTime.Seconds())
	currentSlot := now / slotDuration
	
	// 排序权威节点列表以确保一致性
	sortedAuthorities := make([]string, len(poa.authorities))
	copy(sortedAuthorities, poa.authorities)
	sort.Strings(sortedAuthorities)
	
	proposerIndex := currentSlot % int64(len(sortedAuthorities))
	expectedProposer := sortedAuthorities[proposerIndex]
	
	return expectedProposer == poa.id
}

// proposeEmptyBlock 提议空块
func (poa *PoANode) proposeEmptyBlock() {
	poa.Submit(nil) // 提交空操作
}

// createBlock 创建区块
func (poa *PoANode) createBlock(data interface{}) *PoABlock {
	poa.mu.RLock()
	prevHash := ""
	height := int64(1)
	
	if poa.currentBlock != nil {
		prevHash = poa.currentBlock.Hash
		height = poa.currentBlock.Height + 1
	}
	poa.mu.RUnlock()
	
	block := &PoABlock{
		Height:    height,
		PrevHash:  prevHash,
		Timestamp: time.Now(),
		Proposer:  poa.id,
		Data:      data,
	}
	
	// 计算区块哈希
	block.Hash = poa.calculateBlockHash(block)
	
	// 签名区块（简化实现）
	block.Signature = poa.signBlock(block)
	
	return block
}

// calculateBlockHash 计算区块哈希
func (poa *PoANode) calculateBlockHash(block *PoABlock) string {
	data := fmt.Sprintf("%d%s%s%s%v", 
		block.Height, 
		block.PrevHash, 
		block.Timestamp.Format(time.RFC3339), 
		block.Proposer, 
		block.Data)
	
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// signBlock 签名区块
func (poa *PoANode) signBlock(block *PoABlock) string {
	// 简化实现，实际应该使用私钥签名
	data := fmt.Sprintf("%s%s", block.Hash, poa.id)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateProposalID 生成提案ID
func (poa *PoANode) generateProposalID(block *PoABlock) string {
	data := fmt.Sprintf("%s%s%d", block.Hash, block.Proposer, block.Timestamp.UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16] // 取前16位作为ID
}

// broadcastProposal 广播提案
func (poa *PoANode) broadcastProposal(proposal *PoAProposal) {
	if poa.p2pNetwork == nil {
		return
	}
	
	poa.p2pNetwork.BroadcastMessage(network.MessageTypeConsensus, map[string]interface{}{
		"type":     "poa_proposal",
		"proposal": proposal,
	})
}

// processProposals 处理提案
func (poa *PoANode) processProposals() {
	poa.mu.Lock()
	defer poa.mu.Unlock()
	
	for proposalID, proposal := range poa.proposals {
		if proposal.Status != types.OperationStatusPending {
			continue
		}
		
		// 检查投票结果
		votes := poa.votes[proposalID]
		approvedCount := 0
		totalVotes := len(votes)
		
		for _, vote := range votes {
			if vote {
				approvedCount++
			}
		}
		
		// 计算投票率
		voteRate := float64(approvedCount) / float64(len(poa.authorities))
		
		// 检查是否达到阈值
		if voteRate >= poa.voteThreshold {
			proposal.Status = types.OperationStatusCommitted
			poa.applyBlock(proposal.Block)
			log.Printf("提案 %s 已批准并应用", proposalID)
		} else if totalVotes >= len(poa.authorities) {
			// 所有权威节点都已投票但未达到阈值
			proposal.Status = types.OperationStatusRejected
			log.Printf("提案 %s 已拒绝", proposalID)
		}
	}
}

// applyBlock 应用区块
func (poa *PoANode) applyBlock(block *PoABlock) {
	poa.currentBlock = block
	poa.blockHeight = block.Height
	
	log.Printf("应用区块: 高度=%d, 哈希=%s, 提议者=%s", 
		block.Height, block.Hash[:8], block.Proposer)
}

// handleNetworkMessage 处理网络消息
func (poa *PoANode) handleNetworkMessage(peer *network.Peer, msg *network.Message) error {
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("无效的消息数据格式")
	}
	
	msgType, ok := data["type"].(string)
	if !ok {
		return fmt.Errorf("消息类型缺失")
	}
	
	switch msgType {
	case "poa_proposal":
		return poa.handleProposal(data["proposal"])
	case "poa_vote":
		return poa.handleVote(data["vote"])
	default:
		return fmt.Errorf("未知的PoA消息类型: %s", msgType)
	}
}

// handleProposal 处理提案
func (poa *PoANode) handleProposal(data interface{}) error {
	// 解析提案（简化实现）
	log.Printf("收到PoA提案")
	
	// 如果是权威节点，进行投票
	if poa.isAuthority {
		// 简化投票逻辑：总是投赞成票
		// 实际实现应该验证提案的有效性
		poa.voteOnProposal("dummy_proposal_id", true)
	}
	
	return nil
}

// handleVote 处理投票
func (poa *PoANode) handleVote(data interface{}) error {
	log.Printf("收到PoA投票")
	return nil
}

// voteOnProposal 对提案投票
func (poa *PoANode) voteOnProposal(proposalID string, approve bool) {
	if !poa.isAuthority {
		return
	}
	
	poa.mu.Lock()
	if _, exists := poa.votes[proposalID]; !exists {
		poa.votes[proposalID] = make(map[string]bool)
	}
	poa.votes[proposalID][poa.id] = approve
	poa.mu.Unlock()
	
	// 广播投票
	if poa.p2pNetwork != nil {
		poa.p2pNetwork.BroadcastMessage(network.MessageTypeConsensus, map[string]interface{}{
			"type": "poa_vote",
			"vote": map[string]interface{}{
				"proposal_id": proposalID,
				"voter":       poa.id,
				"approve":     approve,
			},
		})
	}
	
	log.Printf("对提案 %s 投票: %v", proposalID, approve)
}

// GetStatus 获取节点状态
func (poa *PoANode) GetStatus() map[string]interface{} {
	poa.mu.RLock()
	defer poa.mu.RUnlock()
	
	return map[string]interface{}{
		"node_id":       poa.id,
		"is_authority":  poa.isAuthority,
		"authorities":   poa.authorities,
		"block_height":  poa.blockHeight,
		"current_hash":  func() string {
			if poa.currentBlock != nil {
				return poa.currentBlock.Hash
			}
			return ""
		}(),
		"proposals":     len(poa.proposals),
		"block_time":    poa.blockTime.String(),
		"vote_threshold": poa.voteThreshold,
	}
}

// GetAuthorities 获取权威节点列表
func (poa *PoANode) GetAuthorities() []string {
	return poa.authorities
}

// IsAuthority 检查是否为权威节点
func (poa *PoANode) IsAuthority() bool {
	return poa.isAuthority
}

// GetCurrentBlock 获取当前区块
func (poa *PoANode) GetCurrentBlock() *PoABlock {
	poa.mu.RLock()
	defer poa.mu.RUnlock()
	return poa.currentBlock
}