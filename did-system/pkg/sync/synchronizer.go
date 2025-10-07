package sync

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

// Synchronizer 数据同步器
type Synchronizer struct {
	nodeID     string
	registry   *did.DIDRegistry
	p2pNetwork *network.P2PNetwork

	// 同步状态
	syncState      *SyncState
	syncStateMutex sync.RWMutex

	// 同步配置
	config *config.SyncConfig

	// 控制通道
	stopCh chan struct{}
}

// SyncState 同步状态
type SyncState struct {
	LastSyncTime   time.Time           `json:"last_sync_time"`
	SyncInProgress bool                `json:"sync_in_progress"`
	PeerSyncStatus map[string]PeerSync `json:"peer_sync_status"`
	ConflictCount  int                 `json:"conflict_count"`
	ResolvedCount  int                 `json:"resolved_count"`
}

// PeerSync 节点同步状态
type PeerSync struct {
	NodeID       string    `json:"node_id"`
	LastSyncTime time.Time `json:"last_sync_time"`
	Status       string    `json:"status"` // "synced", "syncing", "failed", "conflict"
	Version      int64     `json:"version"`
}

// SyncMessage 同步消息
type SyncMessage struct {
	Type      SyncMessageType `json:"type"`
	NodeID    string          `json:"node_id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      interface{}     `json:"data"`
}

// SyncMessageType 同步消息类型
type SyncMessageType int

const (
	SyncMessageTypeRequest SyncMessageType = iota
	SyncMessageTypeResponse
	SyncMessageTypeDelta
	SyncMessageTypeConflict
	SyncMessageTypeResolution
)

// DIDSyncData DID同步数据
type DIDSyncData struct {
	DIDs     []*types.DIDDocument `json:"dids"`
	Version  int64                `json:"version"`
	Checksum string               `json:"checksum"`
}

// ConflictData 冲突数据
type ConflictData = types.ConflictData

// ConflictEntry 冲突条目
type ConflictEntry = types.ConflictEntry

// NewSynchronizer 创建同步器
func NewSynchronizer(nodeID string, registry *did.DIDRegistry, p2pNetwork *network.P2PNetwork, cfg *config.SyncConfig) *Synchronizer {
	if cfg == nil {
		cfg = &config.SyncConfig{
			SyncInterval:       30 * time.Second,
			BatchSize:          100,
			MaxRetries:         3,
			ConflictResolution: "timestamp",
		}
	}

	return &Synchronizer{
		nodeID:     nodeID,
		registry:   registry,
		p2pNetwork: p2pNetwork,
		config:     cfg,
		syncState: &SyncState{
			PeerSyncStatus: make(map[string]PeerSync),
		},
		stopCh: make(chan struct{}),
	}
}

// Start 启动同步器
func (s *Synchronizer) Start(ctx context.Context) error {
	log.Printf("启动数据同步器，节点ID: %s", s.nodeID)

	// 注册网络消息处理器
	s.p2pNetwork.RegisterMessageHandler(network.MessageTypeSync, s.handleSyncMessage)

	// 启动定期同步
	go s.periodicSync(ctx)

	// 启动冲突检测
	go s.conflictDetection(ctx)

	return nil
}

// Stop 停止同步器
func (s *Synchronizer) Stop() error {
	close(s.stopCh)
	log.Printf("数据同步器已停止")
	return nil
}

// TriggerSync 触发同步
func (s *Synchronizer) TriggerSync() error {
	s.syncStateMutex.Lock()
	if s.syncState.SyncInProgress {
		s.syncStateMutex.Unlock()
		return fmt.Errorf("同步正在进行中")
	}
	s.syncState.SyncInProgress = true
	s.syncStateMutex.Unlock()

	defer func() {
		s.syncStateMutex.Lock()
		s.syncState.SyncInProgress = false
		s.syncState.LastSyncTime = time.Now()
		s.syncStateMutex.Unlock()
	}()

	log.Printf("开始数据同步")

	// 获取本地数据版本
	localVersion, err := s.getLocalVersion()
	if err != nil {
		return fmt.Errorf("获取本地版本失败: %w", err)
	}

	// 向所有节点请求同步
	peers := s.p2pNetwork.GetPeers()
	for peerID := range peers {
		go s.requestSyncFromPeer(peerID, localVersion)
	}

	return nil
}

// GetSyncStatus 获取同步状态
func (s *Synchronizer) GetSyncStatus() *SyncState {
	s.syncStateMutex.RLock()
	defer s.syncStateMutex.RUnlock()

	// 深拷贝状态
	status := &SyncState{
		LastSyncTime:   s.syncState.LastSyncTime,
		SyncInProgress: s.syncState.SyncInProgress,
		ConflictCount:  s.syncState.ConflictCount,
		ResolvedCount:  s.syncState.ResolvedCount,
		PeerSyncStatus: make(map[string]PeerSync),
	}

	for k, v := range s.syncState.PeerSyncStatus {
		status.PeerSyncStatus[k] = v
	}

	return status
}

// periodicSync 定期同步
func (s *Synchronizer) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(s.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			if err := s.TriggerSync(); err != nil {
				log.Printf("定期同步失败: %v", err)
			}
		}
	}
}

// conflictDetection 冲突检测
func (s *Synchronizer) conflictDetection(ctx context.Context) {
	ticker := time.NewTicker(time.Minute) // 每分钟检查一次冲突
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.detectAndResolveConflicts()
		}
	}
}

// handleSyncMessage 处理同步消息
func (s *Synchronizer) handleSyncMessage(peer *network.Peer, msg *network.Message) error {
	var syncMsg SyncMessage
	data, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("序列化同步消息失败: %w", err)
	}

	if err := json.Unmarshal(data, &syncMsg); err != nil {
		return fmt.Errorf("反序列化同步消息失败: %w", err)
	}

	switch syncMsg.Type {
	case SyncMessageTypeRequest:
		return s.handleSyncRequest(peer, &syncMsg)
	case SyncMessageTypeResponse:
		return s.handleSyncResponse(peer, &syncMsg)
	case SyncMessageTypeDelta:
		return s.handleSyncDelta(peer, &syncMsg)
	case SyncMessageTypeConflict:
		return s.handleSyncConflict(peer, &syncMsg)
	case SyncMessageTypeResolution:
		return s.handleSyncResolution(peer, &syncMsg)
	default:
		return fmt.Errorf("未知同步消息类型: %d", syncMsg.Type)
	}
}

// requestSyncFromPeer 从节点请求同步
func (s *Synchronizer) requestSyncFromPeer(peerID string, localVersion int64) {
	syncMsg := SyncMessage{
		Type:      SyncMessageTypeRequest,
		NodeID:    s.nodeID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"version": localVersion,
		},
	}

	if err := s.p2pNetwork.SendMessage(peerID, network.MessageTypeSync, syncMsg); err != nil {
		log.Printf("向节点 %s 发送同步请求失败: %v", peerID, err)
	}
}

// handleSyncRequest 处理同步请求
func (s *Synchronizer) handleSyncRequest(peer *network.Peer, msg *SyncMessage) error {
	log.Printf("收到来自 %s 的同步请求", msg.NodeID)

	// 解析请求版本
	requestData, ok := msg.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("无效的同步请求数据")
	}

	peerVersion, ok := requestData["version"].(float64)
	if !ok {
		return fmt.Errorf("无效的版本信息")
	}

	// 获取本地版本和数据
	localVersion, err := s.getLocalVersion()
	if err != nil {
		return fmt.Errorf("获取本地版本失败: %w", err)
	}

	// 如果对方版本较旧，发送增量数据
	if int64(peerVersion) < localVersion {
		deltaData, err := s.getDeltaData(int64(peerVersion), localVersion)
		if err != nil {
			return fmt.Errorf("获取增量数据失败: %w", err)
		}

		responseMsg := SyncMessage{
			Type:      SyncMessageTypeDelta,
			NodeID:    s.nodeID,
			Timestamp: time.Now(),
			Data:      deltaData,
		}

		return s.p2pNetwork.SendMessage(msg.NodeID, network.MessageTypeSync, responseMsg)
	}

	// 发送响应
	responseMsg := SyncMessage{
		Type:      SyncMessageTypeResponse,
		NodeID:    s.nodeID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"version": localVersion,
			"status":  "up_to_date",
		},
	}

	return s.p2pNetwork.SendMessage(msg.NodeID, network.MessageTypeSync, responseMsg)
}

// handleSyncResponse 处理同步响应
func (s *Synchronizer) handleSyncResponse(peer *network.Peer, msg *SyncMessage) error {
	log.Printf("收到来自 %s 的同步响应", msg.NodeID)

	// 更新节点同步状态
	s.syncStateMutex.Lock()
	s.syncState.PeerSyncStatus[msg.NodeID] = PeerSync{
		NodeID:       msg.NodeID,
		LastSyncTime: time.Now(),
		Status:       "synced",
	}
	s.syncStateMutex.Unlock()

	return nil
}

// handleSyncDelta 处理增量同步
func (s *Synchronizer) handleSyncDelta(peer *network.Peer, msg *SyncMessage) error {
	log.Printf("收到来自 %s 的增量数据", msg.NodeID)

	// 解析增量数据
	var deltaData DIDSyncData
	data, err := json.Marshal(msg.Data)
	if err != nil {
		return fmt.Errorf("序列化增量数据失败: %w", err)
	}

	if err := json.Unmarshal(data, &deltaData); err != nil {
		return fmt.Errorf("反序列化增量数据失败: %w", err)
	}

	// 应用增量数据
	conflicts, err := s.applyDeltaData(&deltaData)
	if err != nil {
		return fmt.Errorf("应用增量数据失败: %w", err)
	}

	// 如果有冲突，发送冲突消息
	if len(conflicts) > 0 {
		conflictMsg := SyncMessage{
			Type:      SyncMessageTypeConflict,
			NodeID:    s.nodeID,
			Timestamp: time.Now(),
			Data:      conflicts,
		}

		return s.p2pNetwork.SendMessage(msg.NodeID, network.MessageTypeSync, conflictMsg)
	}

	// 更新同步状态
	s.syncStateMutex.Lock()
	s.syncState.PeerSyncStatus[msg.NodeID] = PeerSync{
		NodeID:       msg.NodeID,
		LastSyncTime: time.Now(),
		Status:       "synced",
		Version:      deltaData.Version,
	}
	s.syncStateMutex.Unlock()

	return nil
}

// handleSyncConflict 处理同步冲突
func (s *Synchronizer) handleSyncConflict(peer *network.Peer, msg *SyncMessage) error {
	log.Printf("收到来自 %s 的冲突报告", msg.NodeID)

	s.syncStateMutex.Lock()
	s.syncState.ConflictCount++
	s.syncStateMutex.Unlock()

	// 这里应该实现冲突解决逻辑
	// 简化实现，记录冲突
	log.Printf("检测到数据冲突，需要手动解决")

	return nil
}

// handleSyncResolution 处理冲突解决
func (s *Synchronizer) handleSyncResolution(peer *network.Peer, msg *SyncMessage) error {
	log.Printf("收到来自 %s 的冲突解决方案", msg.NodeID)

	s.syncStateMutex.Lock()
	s.syncState.ResolvedCount++
	s.syncStateMutex.Unlock()

	return nil
}

// getLocalVersion 获取本地数据版本
func (s *Synchronizer) getLocalVersion() (int64, error) {
	// 简化实现，使用时间戳作为版本
	return time.Now().Unix(), nil
}

// getDeltaData 获取增量数据
func (s *Synchronizer) getDeltaData(fromVersion, toVersion int64) (*DIDSyncData, error) {
	// 简化实现，返回所有DID文档
	// 实际应该根据版本范围返回增量数据

	// 这里需要从registry获取数据，但registry接口需要扩展
	// 暂时返回空数据
	return &DIDSyncData{
		DIDs:     []*types.DIDDocument{},
		Version:  toVersion,
		Checksum: fmt.Sprintf("checksum_%d", toVersion),
	}, nil
}

// applyDeltaData 应用增量数据
func (s *Synchronizer) applyDeltaData(deltaData *DIDSyncData) ([]*ConflictData, error) {
	var conflicts []*ConflictData

	// 简化实现，直接应用所有数据
	// 实际应该检查冲突并合并数据
	for _, didDoc := range deltaData.DIDs {
		// 检查是否存在冲突
		existing, err := s.registry.Resolve(didDoc.ID)
		if err == nil && existing != nil {
			// 存在冲突，需要解决
			conflict := &ConflictData{
				DID: didDoc.ID,
				Conflicts: []*ConflictEntry{
					{
						NodeID:    s.nodeID,
						Document:  existing,
						Timestamp: time.Now(),
						Version:   1, // 简化版本
					},
					{
						NodeID:    "remote",
						Document:  didDoc,
						Timestamp: time.Now(),
						Version:   2,
					},
				},
			}
			conflicts = append(conflicts, conflict)
		}
	}

	return conflicts, nil
}

// detectAndResolveConflicts 检测和解决冲突
func (s *Synchronizer) detectAndResolveConflicts() {
	// 简化实现，实际应该实现复杂的冲突检测和解决逻辑
	log.Printf("执行冲突检测和解决")
}

// ForceSync 强制同步
func (s *Synchronizer) ForceSync(peerID string) error {
	log.Printf("强制与节点 %s 同步", peerID)

	localVersion, err := s.getLocalVersion()
	if err != nil {
		return fmt.Errorf("获取本地版本失败: %w", err)
	}

	s.requestSyncFromPeer(peerID, localVersion)
	return nil
}

// GetConflicts 获取冲突列表
func (s *Synchronizer) GetConflicts() ([]*ConflictData, error) {
	// 简化实现，返回空列表
	return []*ConflictData{}, nil
}

// ResolveConflict 解决冲突
func (s *Synchronizer) ResolveConflict(conflictID string, resolution string) error {
	log.Printf("解决冲突 %s，解决方案: %s", conflictID, resolution)

	s.syncStateMutex.Lock()
	s.syncState.ResolvedCount++
	s.syncStateMutex.Unlock()

	return nil
}
