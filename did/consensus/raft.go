package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/qujing226/QLink/did/network"
	"github.com/qujing226/QLink/did/types"
)

// RaftNode Raft节点
type RaftNode struct {
	id       string
	peers    map[string]*PeerConnection
	State    NodeState // 改为公开字段以便测试
	term     int64
	votedFor string
	log      []LogEntry

	// 状态管理
	commitIndex int64
	lastApplied int64

	// Leader状态
	nextIndex  map[string]int64
	matchIndex map[string]int64

	// 选举超时
	electionTimeout   time.Duration
	heartbeatInterval time.Duration

	// 网络通信
	p2pNetwork *network.P2PNetwork

	// 同步
	mu sync.RWMutex

	// 通道
	appendEntriesCh chan *AppendEntriesRequest
	requestVoteCh   chan *RequestVoteRequest
	stopCh          chan struct{}
}

// NodeState 节点状态
type NodeState = types.NodeState

const (
	Follower  = types.NodeStateFollower
	Candidate = types.NodeStateCandidate
	Leader    = types.NodeStateLeader
)

// LogEntry 日志条目
type LogEntry = types.LogEntry

// PeerConnection 对等节点连接
type PeerConnection = types.PeerInfo

// AppendEntriesRequest 追加条目请求
type AppendEntriesRequest struct {
	Term         int64      `json:"term"`
	LeaderID     string     `json:"leader_id"`
	PrevLogIndex int64      `json:"prev_log_index"`
	PrevLogTerm  int64      `json:"prev_log_term"`
	Entries      []LogEntry `json:"entries"`
	LeaderCommit int64      `json:"leader_commit"`
}

// AppendEntriesResponse 追加条目响应
type AppendEntriesResponse struct {
	Term    int64 `json:"term"`
	Success bool  `json:"success"`
}

// RequestVoteRequest 请求投票请求
type RequestVoteRequest struct {
	Term         int64  `json:"term"`
	CandidateID  string `json:"candidate_id"`
	LastLogIndex int64  `json:"last_log_index"`
	LastLogTerm  int64  `json:"last_log_term"`
}

// RequestVoteResponse 请求投票响应
type RequestVoteResponse struct {
	Term        int64 `json:"term"`
	VoteGranted bool  `json:"vote_granted"`
}

// NewRaftNode 创建新的Raft节点
func NewRaftNode(id string, p2pNetwork *network.P2PNetwork) *RaftNode {
	return &RaftNode{
		id:                id,
		peers:             make(map[string]*PeerConnection),
		State:             Follower,
		term:              0,
		log:               make([]LogEntry, 0),
		nextIndex:         make(map[string]int64),
		matchIndex:        make(map[string]int64),
		electionTimeout:   time.Duration(150+rand.Intn(150)) * time.Millisecond,
		heartbeatInterval: 50 * time.Millisecond,
		p2pNetwork:        p2pNetwork,
		appendEntriesCh:   make(chan *AppendEntriesRequest, 100),
		requestVoteCh:     make(chan *RequestVoteRequest, 100),
		stopCh:            make(chan struct{}),
	}
}

// Start 启动Raft节点
func (rn *RaftNode) Start(ctx context.Context) error {
	log.Printf("启动Raft节点: %s", rn.id)
	
	// 注册Raft消息处理器
	if rn.p2pNetwork != nil {
		rn.p2pNetwork.RegisterMessageHandler(network.MessageTypeConsensus, rn.handleNetworkMessage)
	}
	
	go rn.run(ctx)
	return nil
}

// Stop 停止Raft节点
func (rn *RaftNode) Stop() error {
	close(rn.stopCh)
	return nil
}

// AddPeer 添加对等节点
func (rn *RaftNode) AddPeer(id, address string) {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	rn.peers[id] = &PeerConnection{
		NodeID:  id,
		Address: address,
		Active:  true,
	}

	if rn.State == Leader {
		rn.nextIndex[id] = rn.getLastLogIndex() + 1
		rn.matchIndex[id] = 0
	}
}

// RemovePeer 移除对等节点
func (rn *RaftNode) RemovePeer(id string) {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	delete(rn.peers, id)
	delete(rn.nextIndex, id)
	delete(rn.matchIndex, id)
}

// GetState 获取节点状态
func (rn *RaftNode) GetState() (NodeState, int64, bool) {
	rn.mu.RLock()
	defer rn.mu.RUnlock()

	return rn.State, rn.term, rn.State == Leader
}

// Submit 提交命令
func (rn *RaftNode) Submit(command interface{}) error {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	if rn.State != Leader {
		return fmt.Errorf("只有Leader可以接受命令")
	}

	// 创建新的日志条目
	entry := LogEntry{
		Term:    rn.term,
		Index:   rn.getLastLogIndex() + 1,
		Command: command,
	}

	rn.log = append(rn.log, entry)
	log.Printf("Leader %s 添加日志条目: %+v", rn.id, entry)

	// 立即发送给所有followers
	go rn.sendAppendEntries()

	return nil
}

// run 主运行循环
func (rn *RaftNode) run(ctx context.Context) {
	ticker := time.NewTicker(rn.electionTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-rn.stopCh:
			return
		case <-ticker.C:
			rn.handleElectionTimeout()
		case req := <-rn.appendEntriesCh:
			rn.handleAppendEntries(req)
		case req := <-rn.requestVoteCh:
			rn.handleRequestVote(req)
		}
	}
}

// handleElectionTimeout 处理选举超时
func (rn *RaftNode) handleElectionTimeout() {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	if rn.State == Leader {
		// Leader发送心跳
		go rn.sendHeartbeat()
		return
	}

	// 开始选举
	rn.startElection()
}

// startElection 开始选举
func (rn *RaftNode) startElection() {
	rn.State = Candidate
	rn.term++
	rn.votedFor = rn.id

	log.Printf("节点 %s 开始选举，任期: %d", rn.id, rn.term)

	votes := 1                      // 投票给自己
	totalPeers := len(rn.peers) + 1 // 包括自己

	// 向所有peers请求投票
	for peerID := range rn.peers {
		go func(id string) {
			if rn.requestVote(id) {
				rn.mu.Lock()
				votes++
				if votes > totalPeers/2 && rn.State == Candidate {
					rn.becomeLeader()
				}
				rn.mu.Unlock()
			}
		}(peerID)
	}
}

// becomeLeader 成为Leader
func (rn *RaftNode) becomeLeader() {
	rn.State = Leader
	log.Printf("节点 %s 成为Leader，任期: %d", rn.id, rn.term)

	// 初始化Leader状态
	for peerID := range rn.peers {
		rn.nextIndex[peerID] = rn.getLastLogIndex() + 1
		rn.matchIndex[peerID] = 0
	}

	// 立即发送心跳
	go rn.sendHeartbeat()
}

// sendHeartbeat 发送心跳
func (rn *RaftNode) sendHeartbeat() {
	rn.mu.RLock()
	defer rn.mu.RUnlock()

	if rn.State != Leader {
		return
	}

	for peerID := range rn.peers {
		go rn.sendAppendEntriesToPeer(peerID, []LogEntry{})
	}
}

// sendAppendEntries 发送追加条目
func (rn *RaftNode) sendAppendEntries() {
	rn.mu.RLock()
	defer rn.mu.RUnlock()

	if rn.State != Leader {
		return
	}

	for peerID := range rn.peers {
		nextIdx := rn.nextIndex[peerID]
		if nextIdx <= rn.getLastLogIndex() {
			entries := rn.log[nextIdx:]
			go rn.sendAppendEntriesToPeer(peerID, entries)
		}
	}
}

// sendAppendEntriesToPeer 向指定节点发送追加条目
func (rn *RaftNode) sendAppendEntriesToPeer(peerID string, entries []LogEntry) {
	if rn.p2pNetwork == nil {
		log.Printf("P2P网络未初始化，无法发送追加条目到节点 %s", peerID)
		return
	}

	// 构造追加条目请求
	req := &AppendEntriesRequest{
		Term:         rn.term,
		LeaderID:     rn.id,
		PrevLogIndex: rn.getPrevLogIndex(peerID),
		PrevLogTerm:  rn.getPrevLogTerm(peerID),
		Entries:      entries,
		LeaderCommit: rn.commitIndex,
	}

	// 发送消息
	if err := rn.p2pNetwork.SendMessage(peerID, network.MessageTypeConsensus, map[string]interface{}{
		"type": "append_entries",
		"data": req,
	}); err != nil {
		log.Printf("发送追加条目到节点 %s 失败: %v", peerID, err)
		// 处理发送失败，可能需要重试或调整nextIndex
		if nextIndex, exists := rn.nextIndex[peerID]; exists && nextIndex > 1 {
			rn.nextIndex[peerID] = nextIndex - 1
		}
		return
	}

	log.Printf("向节点 %s 发送追加条目，条目数量: %d", peerID, len(entries))
}

// requestVote 向指定节点请求投票
func (rn *RaftNode) requestVote(peerID string) bool {
	if rn.p2pNetwork == nil {
		log.Printf("P2P网络未初始化，无法向节点 %s 请求投票", peerID)
		return false
	}

	// 构造投票请求
	req := &RequestVoteRequest{
		Term:         rn.term,
		CandidateID:  rn.id,
		LastLogIndex: rn.getLastLogIndex(),
		LastLogTerm:  rn.getLastLogTerm(),
	}

	// 发送消息
	if err := rn.p2pNetwork.SendMessage(peerID, network.MessageTypeConsensus, map[string]interface{}{
		"type": "request_vote",
		"data": req,
	}); err != nil {
		log.Printf("向节点 %s 请求投票失败: %v", peerID, err)
		return false
	}

	log.Printf("向节点 %s 请求投票", peerID)
	return true // 这里应该等待响应，简化实现先返回true
}

// handleAppendEntries 处理追加条目请求
func (rn *RaftNode) handleAppendEntries(req *AppendEntriesRequest) {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	// 如果请求的任期小于当前任期，拒绝
	if req.Term < rn.term {
		return
	}

	// 如果请求的任期大于当前任期，更新任期并成为follower
	if req.Term > rn.term {
		rn.term = req.Term
		rn.votedFor = ""
		rn.State = Follower
	}

	// 重置选举超时
	rn.resetElectionTimeout()

	// 检查日志一致性
	if req.PrevLogIndex > 0 {
		// 检查前一个日志条目是否存在且任期匹配
		if req.PrevLogIndex > rn.getLastLogIndex() ||
			(req.PrevLogIndex > 0 && rn.log[req.PrevLogIndex-1].Term != req.PrevLogTerm) {
			log.Printf("节点 %s 日志不一致，拒绝追加条目", rn.id)
			return
		}
	}

	// 删除冲突的日志条目
	if len(req.Entries) > 0 {
		// 找到第一个冲突的条目
		conflictIndex := -1
		for i, entry := range req.Entries {
			logIndex := req.PrevLogIndex + int64(i) + 1
			if logIndex <= rn.getLastLogIndex() {
				if rn.log[logIndex-1].Term != entry.Term {
					conflictIndex = i
					break
				}
			} else {
				conflictIndex = i
				break
			}
		}

		// 如果有冲突，删除冲突及之后的所有条目
		if conflictIndex >= 0 {
			newLogLength := req.PrevLogIndex + int64(conflictIndex)
			if newLogLength < int64(len(rn.log)) {
				rn.log = rn.log[:newLogLength]
			}

			// 追加新的条目
			for i := conflictIndex; i < len(req.Entries); i++ {
				rn.log = append(rn.log, req.Entries[i])
			}
		}
	}

	// 更新提交索引
	if req.LeaderCommit > rn.commitIndex {
		rn.commitIndex = min(req.LeaderCommit, rn.getLastLogIndex())
		// 应用已提交的日志条目
		rn.applyCommittedEntries()
	}

	log.Printf("节点 %s 成功处理来自Leader %s 的追加条目请求", rn.id, req.LeaderID)
}

// resetElectionTimeout 重置选举超时
func (rn *RaftNode) resetElectionTimeout() {
	// 这里应该重置选举超时定时器
	// 简化实现，实际应该有定时器管理
}

// applyCommittedEntries 应用已提交的日志条目
func (rn *RaftNode) applyCommittedEntries() {
	for rn.lastApplied < rn.commitIndex {
		rn.lastApplied++
		entry := rn.log[rn.lastApplied-1]
		log.Printf("应用日志条目 %d: %v", rn.lastApplied, entry.Command)
		// 这里应该将命令应用到状态机
	}
}

// min 返回两个int64中的较小值
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// handleRequestVote 处理投票请求
func (rn *RaftNode) handleRequestVote(req *RequestVoteRequest) {
	rn.mu.Lock()
	defer rn.mu.Unlock()

	// 如果请求的任期小于当前任期，拒绝投票
	if req.Term < rn.term {
		return
	}

	// 如果请求的任期大于当前任期，更新任期
	if req.Term > rn.term {
		rn.term = req.Term
		rn.votedFor = ""
		rn.State = Follower
	}

	// 投票逻辑
	if (rn.votedFor == "" || rn.votedFor == req.CandidateID) &&
		rn.isLogUpToDate(req.LastLogIndex, req.LastLogTerm) {
		rn.votedFor = req.CandidateID
		log.Printf("节点 %s 投票给候选人 %s", rn.id, req.CandidateID)
	}
}

// isLogUpToDate 检查日志是否是最新的
func (rn *RaftNode) isLogUpToDate(lastLogIndex, lastLogTerm int64) bool {
	if len(rn.log) == 0 {
		return true
	}

	lastEntry := rn.log[len(rn.log)-1]
	return lastLogTerm > lastEntry.Term ||
		(lastLogTerm == lastEntry.Term && lastLogIndex >= lastEntry.Index)
}

// getLastLogIndex 获取最后一个日志条目的索引
func (rn *RaftNode) getLastLogIndex() int64 {
	if len(rn.log) == 0 {
		return 0
	}
	return rn.log[len(rn.log)-1].Index
}

// GetPeers 获取对等节点列表
func (rn *RaftNode) GetPeers() map[string]*PeerConnection {
	rn.mu.RLock()
	defer rn.mu.RUnlock()

	peers := make(map[string]*PeerConnection)
	for id, peer := range rn.peers {
		peers[id] = &PeerConnection{
			NodeID:  peer.NodeID,
			Address: peer.Address,
			Active:  peer.Active,
		}
	}
	return peers
}

// GetStatus 获取节点状态信息
func (rn *RaftNode) GetStatus() map[string]interface{} {
	rn.mu.RLock()
	defer rn.mu.RUnlock()

	return map[string]interface{}{
		"id":           rn.id,
		"state":        rn.getStateString(),
		"term":         rn.term,
		"voted_for":    rn.votedFor,
		"log_length":   len(rn.log),
		"commit_index": rn.commitIndex,
		"last_applied": rn.lastApplied,
		"peer_count":   len(rn.peers),
	}
}

// getLastLogTerm 获取最后一个日志条目的任期
func (rn *RaftNode) getLastLogTerm() int64 {
	if len(rn.log) == 0 {
		return 0
	}
	return rn.log[len(rn.log)-1].Term
}

// getPrevLogIndex 获取指定peer的前一个日志索引
func (rn *RaftNode) getPrevLogIndex(peerID string) int64 {
	if nextIndex, exists := rn.nextIndex[peerID]; exists {
		return nextIndex - 1
	}
	return rn.getLastLogIndex()
}

// getPrevLogTerm 获取指定peer的前一个日志任期
func (rn *RaftNode) getPrevLogTerm(peerID string) int64 {
	prevIndex := rn.getPrevLogIndex(peerID)
	if prevIndex < 0 || prevIndex >= int64(len(rn.log)) {
		return 0
	}
	return rn.log[prevIndex].Term
}

// handleNetworkMessage 处理网络消息
func (rn *RaftNode) handleNetworkMessage(peer *network.Peer, msg *network.Message) error {
	// 解析消息数据
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("无效的消息数据格式")
	}

	msgType, ok := data["type"].(string)
	if !ok {
		return fmt.Errorf("消息类型缺失")
	}

	switch msgType {
	case "append_entries":
		return rn.handleAppendEntriesMessage(data["data"])
	case "request_vote":
		return rn.handleRequestVoteMessage(data["data"])
	case "append_entries_response":
		return rn.handleAppendEntriesResponse(data["data"])
	case "request_vote_response":
		return rn.handleRequestVoteResponse(data["data"])
	default:
		return fmt.Errorf("未知的消息类型: %s", msgType)
	}
}

// handleAppendEntriesMessage 处理追加条目消息
func (rn *RaftNode) handleAppendEntriesMessage(data interface{}) error {
	// 将interface{}转换为AppendEntriesRequest
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化追加条目请求失败: %w", err)
	}

	var req AppendEntriesRequest
	if err := json.Unmarshal(dataBytes, &req); err != nil {
		return fmt.Errorf("反序列化追加条目请求失败: %w", err)
	}

	rn.handleAppendEntries(&req)
	return nil
}

// handleRequestVoteMessage 处理投票请求消息
func (rn *RaftNode) handleRequestVoteMessage(data interface{}) error {
	// 将interface{}转换为RequestVoteRequest
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化投票请求失败: %w", err)
	}

	var req RequestVoteRequest
	if err := json.Unmarshal(dataBytes, &req); err != nil {
		return fmt.Errorf("反序列化投票请求失败: %w", err)
	}

	rn.handleRequestVote(&req)
	return nil
}

// handleAppendEntriesResponse 处理追加条目响应
func (rn *RaftNode) handleAppendEntriesResponse(data interface{}) error {
	// TODO: 实现追加条目响应处理
	log.Printf("收到追加条目响应")
	return nil
}

// handleRequestVoteResponse 处理投票响应
func (rn *RaftNode) handleRequestVoteResponse(data interface{}) error {
	// TODO: 实现投票响应处理
	log.Printf("收到投票响应")
	return nil
}

// getStateString 获取状态字符串
func (rn *RaftNode) getStateString() string {
	switch rn.State {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}