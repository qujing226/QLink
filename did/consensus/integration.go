package consensus

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/qujing226/QLink/did/network"
)

// ConsensusManager 共识管理器
type ConsensusManager struct {
	// 共识算法实例
	raftNode *RaftNode
	poaNode  *PoANode
	
	// 管理组件
	monitor  *ConsensusMonitor
	switcher *ConsensusSwitcher
	
	// 网络通信
	p2pNetwork *network.P2PNetwork
	
	// 配置
	config *ManagerConfig
	
	// 状态管理
	mu      sync.RWMutex
	started bool
	stopCh  chan struct{}
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	// 节点配置
	NodeID      string   `json:"node_id"`
	Authorities []string `json:"authorities"`
	
	// 默认共识算法
	DefaultConsensus ConsensusType `json:"default_consensus"`
	
	// 监控配置
	MonitorConfig *MonitorConfig `json:"monitor_config"`
	
	// 切换器配置
	SwitcherConfig *SwitcherConfig `json:"switcher_config"`
	
	// Raft配置
	RaftConfig *RaftConfig `json:"raft_config"`
	
	// PoA配置
	PoAConfig *PoAConfig `json:"poa_config"`
}

// RaftConfig Raft配置
type RaftConfig struct {
	ElectionTimeout   time.Duration `json:"election_timeout"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}

// PoAConfig PoA配置
type PoAConfig struct {
	BlockTime     time.Duration `json:"block_time"`
	VoteThreshold float64       `json:"vote_threshold"`
}

// NewConsensusManager 创建共识管理器
func NewConsensusManager(config *ManagerConfig, p2pNetwork *network.P2PNetwork) *ConsensusManager {
	if config == nil {
		config = &ManagerConfig{
			DefaultConsensus: ConsensusTypeRaft,
			MonitorConfig:    nil, // 使用默认配置
			SwitcherConfig:   nil, // 使用默认配置
			RaftConfig: &RaftConfig{
				ElectionTimeout:   300 * time.Millisecond,
				HeartbeatInterval: 50 * time.Millisecond,
			},
			PoAConfig: &PoAConfig{
				BlockTime:     5 * time.Second,
				VoteThreshold: 0.67,
			},
		}
	}
	
	return &ConsensusManager{
		config:     config,
		p2pNetwork: p2pNetwork,
		stopCh:     make(chan struct{}),
	}
}

// Initialize 初始化共识管理器
func (cm *ConsensusManager) Initialize() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	log.Printf("初始化共识管理器，节点ID: %s", cm.config.NodeID)
	
	// 创建Raft节点
	cm.raftNode = NewRaftNode(cm.config.NodeID, cm.p2pNetwork)
	if cm.config.RaftConfig != nil {
		// TODO: 应用Raft配置
	}
	
	// 创建PoA节点
	cm.poaNode = NewPoANode(cm.config.NodeID, cm.config.Authorities, cm.p2pNetwork)
	if cm.config.PoAConfig != nil {
		// TODO: 应用PoA配置
	}
	
	// 创建监控器
	cm.monitor = NewConsensusMonitor(cm.config.MonitorConfig)
	
	// 创建切换器
	cm.switcher = NewConsensusSwitcher(cm.config.SwitcherConfig)
	
	// 初始化切换器
	if err := cm.switcher.Initialize(cm.raftNode, cm.poaNode, cm.monitor); err != nil {
		return fmt.Errorf("初始化切换器失败: %v", err)
	}
	
	// 设置回调函数
	cm.setupCallbacks()
	
	log.Printf("共识管理器初始化完成")
	return nil
}

// Start 启动共识管理器
func (cm *ConsensusManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	if cm.started {
		cm.mu.Unlock()
		return fmt.Errorf("共识管理器已启动")
	}
	cm.started = true
	cm.mu.Unlock()
	
	log.Printf("启动共识管理器")
	
	// 启动监控器
	if err := cm.monitor.Start(ctx); err != nil {
		return fmt.Errorf("启动监控器失败: %v", err)
	}
	
	// 根据默认配置启动相应的共识算法
	switch cm.config.DefaultConsensus {
	case ConsensusTypeRaft:
		if err := cm.raftNode.Start(ctx); err != nil {
			return fmt.Errorf("启动Raft节点失败: %v", err)
		}
	case ConsensusTypePoA:
		if err := cm.poaNode.Start(ctx); err != nil {
			return fmt.Errorf("启动PoA节点失败: %v", err)
		}
	default:
		return fmt.Errorf("不支持的默认共识算法: %d", cm.config.DefaultConsensus)
	}
	
	// 启动管理循环
	go cm.managementLoop(ctx)
	
	log.Printf("共识管理器启动完成，当前算法: %s", cm.getConsensusTypeName(cm.config.DefaultConsensus))
	return nil
}

// Stop 停止共识管理器
func (cm *ConsensusManager) Stop() error {
	cm.mu.Lock()
	if !cm.started {
		cm.mu.Unlock()
		return nil
	}
	cm.started = false
	cm.mu.Unlock()
	
	log.Printf("停止共识管理器")
	
	// 发送停止信号
	close(cm.stopCh)
	
	// 停止当前共识算法
	currentConsensus := cm.switcher.GetCurrentConsensus()
	if currentConsensus != nil {
		currentConsensus.Stop()
	}
	
	// 停止监控器
	if cm.monitor != nil {
		cm.monitor.Stop()
	}
	
	log.Printf("共识管理器已停止")
	return nil
}

// managementLoop 管理循环
func (cm *ConsensusManager) managementLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.performPeriodicTasks()
		}
	}
}

// performPeriodicTasks 执行周期性任务
func (cm *ConsensusManager) performPeriodicTasks() {
	// 检查是否需要自动切换
	if cm.switcher != nil {
		if err := cm.switcher.AutoSwitch(); err != nil {
			log.Printf("自动切换检查失败: %v", err)
		}
	}
	
	// 检查系统健康状态
	cm.checkSystemHealth()
}

// checkSystemHealth 检查系统健康状态
func (cm *ConsensusManager) checkSystemHealth() {
	if cm.monitor == nil {
		return
	}
	
	metrics := cm.monitor.GetMetrics()
	if metrics == nil {
		return
	}
	
	// 检查关键指标
	if metrics.ConsecutiveErrors > 5 {
		log.Printf("警告: 连续错误次数过多 (%d)", metrics.ConsecutiveErrors)
	}
	
	if metrics.SuccessRate < 0.9 {
		log.Printf("警告: 成功率过低 (%.2f)", metrics.SuccessRate)
	}
	
	if time.Since(metrics.LastUpdate) > 30*time.Second {
		log.Printf("警告: 指标更新超时")
	}
}

// setupCallbacks 设置回调函数
func (cm *ConsensusManager) setupCallbacks() {
	// 设置故障检测回调
	cm.monitor.SetFailureCallback(func(failure *FailureEvent) {
		log.Printf("检测到故障: %s - %s", failure.ID, failure.Description)
		
		// 根据故障类型决定是否需要切换共识算法
		if failure.Severity == FailureSeverityCritical {
			cm.handleCriticalFailure(failure)
		}
	})
	
	// 设置恢复回调
	cm.monitor.SetRecoveryCallback(func(recovery *RecoveryEvent) {
		log.Printf("开始恢复: %s (策略: %d)", recovery.ID, recovery.Strategy)
	})
	
	// 设置切换回调
	cm.switcher.SetSwitchStartedCallback(func(from, to ConsensusType) {
		log.Printf("开始切换共识算法: %s -> %s", 
			cm.getConsensusTypeName(from), cm.getConsensusTypeName(to))
	})
	
	cm.switcher.SetSwitchCompletedCallback(func(from, to ConsensusType, success bool) {
		if success {
			log.Printf("共识算法切换成功: %s -> %s", 
				cm.getConsensusTypeName(from), cm.getConsensusTypeName(to))
		} else {
			log.Printf("共识算法切换失败: %s -> %s", 
				cm.getConsensusTypeName(from), cm.getConsensusTypeName(to))
		}
	})
}

// handleCriticalFailure 处理关键故障
func (cm *ConsensusManager) handleCriticalFailure(failure *FailureEvent) {
	currentType := cm.switcher.GetCurrentType()
	
	// 选择备用共识算法
	var targetType ConsensusType
	switch currentType {
	case ConsensusTypeRaft:
		if cm.switcher.IsSupported(ConsensusTypePoA) {
			targetType = ConsensusTypePoA
		}
	case ConsensusTypePoA:
		if cm.switcher.IsSupported(ConsensusTypeRaft) {
			targetType = ConsensusTypeRaft
		}
	default:
		log.Printf("无可用的备用共识算法")
		return
	}
	
	log.Printf("关键故障触发自动切换: %s -> %s", 
		cm.getConsensusTypeName(currentType), cm.getConsensusTypeName(targetType))
	
	if err := cm.switcher.SwitchTo(targetType); err != nil {
		log.Printf("自动切换失败: %v", err)
	}
}

// getConsensusTypeName 获取共识算法类型名称
func (cm *ConsensusManager) getConsensusTypeName(consensusType ConsensusType) string {
	switch consensusType {
	case ConsensusTypeRaft:
		return "Raft"
	case ConsensusTypePoA:
		return "PoA"
	case ConsensusTypePBFT:
		return "PBFT"
	case ConsensusTypePoS:
		return "PoS"
	default:
		return "Unknown"
	}
}

// Submit 提交提案
func (cm *ConsensusManager) Submit(proposal interface{}) error {
	currentConsensus := cm.switcher.GetCurrentConsensus()
	if currentConsensus == nil {
		return fmt.Errorf("当前没有活跃的共识算法")
	}
	
	return currentConsensus.Submit(proposal)
}

// SwitchConsensus 切换共识算法
func (cm *ConsensusManager) SwitchConsensus(targetType ConsensusType) error {
	return cm.switcher.SwitchTo(targetType)
}

// GetCurrentConsensusType 获取当前共识算法类型
func (cm *ConsensusManager) GetCurrentConsensusType() ConsensusType {
	return cm.switcher.GetCurrentType()
}

// GetStatus 获取管理器状态
func (cm *ConsensusManager) GetStatus() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	status := map[string]interface{}{
		"started":           cm.started,
		"node_id":          cm.config.NodeID,
		"current_consensus": cm.getConsensusTypeName(cm.switcher.GetCurrentType()),
		"supported_types":   cm.getSupportedTypeNames(),
	}
	
	// 添加监控状态
	if cm.monitor != nil {
		status["monitor"] = cm.monitor.GetStatus()
	}
	
	// 添加切换器状态
	if cm.switcher != nil {
		status["switcher"] = cm.switcher.GetStatus()
	}
	
	// 添加当前共识算法状态
	currentConsensus := cm.switcher.GetCurrentConsensus()
	if currentConsensus != nil {
		status["consensus_status"] = currentConsensus.GetStatus()
	}
	
	return status
}

// getSupportedTypeNames 获取支持的算法类型名称
func (cm *ConsensusManager) getSupportedTypeNames() []string {
	if cm.switcher == nil {
		return []string{}
	}
	
	types := cm.switcher.GetSupportedTypes()
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = cm.getConsensusTypeName(t)
	}
	return names
}

// GetMetrics 获取监控指标
func (cm *ConsensusManager) GetMetrics() *ConsensusMetrics {
	if cm.monitor == nil {
		return nil
	}
	return cm.monitor.GetMetrics()
}

// GetFailureHistory 获取故障历史
func (cm *ConsensusManager) GetFailureHistory() []FailureEvent {
	if cm.monitor == nil {
		return nil
	}
	return cm.monitor.GetFailureHistory()
}

// GetRecoveryHistory 获取恢复历史
func (cm *ConsensusManager) GetRecoveryHistory() []RecoveryEvent {
	if cm.monitor == nil {
		return nil
	}
	return cm.monitor.GetRecoveryHistory()
}

// GetSwitchState 获取切换状态
func (cm *ConsensusManager) GetSwitchState() *SwitchState {
	if cm.switcher == nil {
		return nil
	}
	return cm.switcher.GetSwitchState()
}

// AddPeer 添加对等节点
func (cm *ConsensusManager) AddPeer(id, address string) error {
	// 添加到Raft节点
	if cm.raftNode != nil {
		cm.raftNode.AddPeer(id, address)
	}
	
	// TODO: 添加到PoA节点（如果需要）
	
	log.Printf("添加对等节点: %s (%s)", id, address)
	return nil
}

// RemovePeer 移除对等节点
func (cm *ConsensusManager) RemovePeer(id string) error {
	// 从Raft节点移除
	if cm.raftNode != nil {
		cm.raftNode.RemovePeer(id)
	}
	
	// TODO: 从PoA节点移除（如果需要）
	
	log.Printf("移除对等节点: %s", id)
	return nil
}

// IsLeader 检查是否为领导者
func (cm *ConsensusManager) IsLeader() bool {
	currentType := cm.switcher.GetCurrentType()
	
	switch currentType {
	case ConsensusTypeRaft:
		if cm.raftNode != nil {
			_, _, isLeader := cm.raftNode.GetState()
			return isLeader
		}
	case ConsensusTypePoA:
		if cm.poaNode != nil {
			return cm.poaNode.IsAuthority()
		}
	}
	
	return false
}

// GetLeaderID 获取领导者ID
func (cm *ConsensusManager) GetLeaderID() string {
	// TODO: 实现获取领导者ID的逻辑
	// 这需要根据不同的共识算法实现
	return ""
}

// GetPeers 获取对等节点列表
func (cm *ConsensusManager) GetPeers() map[string]interface{} {
	currentType := cm.switcher.GetCurrentType()
	
	switch currentType {
	case ConsensusTypeRaft:
		if cm.raftNode != nil {
			peers := cm.raftNode.GetPeers()
			result := make(map[string]interface{})
			for id, peer := range peers {
				result[id] = map[string]interface{}{
					"node_id": peer.NodeID,
					"address": peer.Address,
					"active":  peer.Active,
				}
			}
			return result
		}
	case ConsensusTypePoA:
		if cm.poaNode != nil {
			authorities := cm.poaNode.GetAuthorities()
			result := make(map[string]interface{})
			for i, auth := range authorities {
				result[fmt.Sprintf("authority_%d", i)] = map[string]interface{}{
					"node_id": auth,
					"active":  true, // PoA中权威节点默认活跃
				}
			}
			return result
		}
	}
	
	return make(map[string]interface{})
}