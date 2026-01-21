package consensus

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
	"github.com/qujing226/QLink/pkg/network"
)

// ConsensusSwitcherAdapter 共识算法切换器适配器，使用统一接口
type ConsensusSwitcherAdapter struct {
	// 当前共识算法
	currentConsensus interfaces.ConsensusAlgorithm
	currentType      interfaces.ConsensusType

	// 可用的共识算法适配器
	raftAdapter *RaftAdapter
	poaAdapter  *PoAAdapter

	// 切换配置
	config *SwitcherAdapterConfig

	// 状态管理
	mu           sync.RWMutex
	switching    bool
	switchCtx    context.Context
	switchCancel context.CancelFunc

	// 监控器
	monitor *ConsensusMonitor

	// 回调函数
	onSwitchStarted   func(from, to interfaces.ConsensusType)
	onSwitchCompleted func(from, to interfaces.ConsensusType, success bool)
}

// SwitcherAdapterConfig 切换器适配器配置
type SwitcherAdapterConfig struct {
	// 切换策略
	SwitchStrategy SwitchStrategy `json:"switch_strategy"`

	// 切换超时
	SwitchTimeout time.Duration `json:"switch_timeout"`

	// 数据同步超时
	DataSyncTimeout time.Duration `json:"data_sync_timeout"`

	// 自动切换配置
	EnableAutoSwitch    bool          `json:"enable_auto_switch"`
	AutoSwitchThreshold float64       `json:"auto_switch_threshold"`
	AutoSwitchCooldown  time.Duration `json:"auto_switch_cooldown"`

	// 安全配置
	RequireConfirmation bool `json:"require_confirmation"`
	BackupBeforeSwitch  bool `json:"backup_before_switch"`

	// 回滚配置
	EnableRollback   bool          `json:"enable_rollback"`
	RollbackTimeout  time.Duration `json:"rollback_timeout"`
	MaxRollbackDepth int           `json:"max_rollback_depth"`
}

// SwitchAdapterEvent 切换事件
type SwitchAdapterEvent struct {
	ID        string                   `json:"id"`
	FromType  interfaces.ConsensusType `json:"from_type"`
	ToType    interfaces.ConsensusType `json:"to_type"`
	Strategy  SwitchStrategy           `json:"strategy"`
	StartTime time.Time                `json:"start_time"`
	EndTime   time.Time                `json:"end_time"`
	Success   bool                     `json:"success"`
	Error     string                   `json:"error,omitempty"`
	Context   map[string]interface{}   `json:"context"`
	Rollback  bool                     `json:"rollback"`
}

// SwitchAdapterState 切换状态
type SwitchAdapterState struct {
	InProgress    bool                     `json:"in_progress"`
	CurrentType   interfaces.ConsensusType `json:"current_type"`
	TargetType    interfaces.ConsensusType `json:"target_type"`
	Progress      float64                  `json:"progress"`
	Stage         string                   `json:"stage"`
	StartTime     time.Time                `json:"start_time"`
	EstimatedTime time.Duration            `json:"estimated_time"`
}

// NewConsensusSwitcherAdapter 创建新的共识切换器适配器
func NewConsensusSwitcherAdapter(config *SwitcherAdapterConfig) *ConsensusSwitcherAdapter {
	if config == nil {
		config = &SwitcherAdapterConfig{
			SwitchStrategy:      SwitchStrategyGraceful,
			SwitchTimeout:       30 * time.Second,
			DataSyncTimeout:     10 * time.Second,
			EnableAutoSwitch:    false,
			AutoSwitchThreshold: 0.8,
			AutoSwitchCooldown:  5 * time.Minute,
			RequireConfirmation: true,
			BackupBeforeSwitch:  true,
			EnableRollback:      true,
			RollbackTimeout:     15 * time.Second,
			MaxRollbackDepth:    3,
		}
	}

	return &ConsensusSwitcherAdapter{
		config:      config,
		currentType: interfaces.ConsensusTypeRaft, // 默认使用Raft
	}
}

// Initialize 初始化切换器
func (csa *ConsensusSwitcherAdapter) Initialize(nodeID string, peers []string, authorities []string, p2pNetwork *network.P2PNetwork, monitor *ConsensusMonitor) error {
	csa.mu.Lock()
	defer csa.mu.Unlock()

	// 创建Raft适配器
	csa.raftAdapter = NewRaftAdapter(nodeID, peers, p2pNetwork)

	// 创建PoA适配器
	csa.poaAdapter = NewPoAAdapter(nodeID, authorities, p2pNetwork)

	// 设置监控器
	csa.monitor = monitor

	// 设置默认共识算法
	csa.currentConsensus = csa.raftAdapter
	csa.currentType = interfaces.ConsensusTypeRaft

	log.Printf("ConsensusSwitcherAdapter initialized with Raft as default")
	return nil
}

// SwitchTo 切换到指定的共识算法
func (csa *ConsensusSwitcherAdapter) SwitchTo(targetType interfaces.ConsensusType) error {
	csa.mu.Lock()
	defer csa.mu.Unlock()

	if csa.switching {
		return fmt.Errorf("共识切换正在进行中")
	}

	if csa.currentType == targetType {
		return fmt.Errorf("已经在使用 %s 共识算法", csa.getConsensusTypeName(targetType))
	}

	if !csa.IsSupported(targetType) {
		return fmt.Errorf("不支持的共识算法类型: %s", csa.getConsensusTypeName(targetType))
	}

	// 开始切换
	csa.switching = true
	go csa.performSwitch(csa.currentType, targetType)

	return nil
}

// performSwitch 执行切换
func (csa *ConsensusSwitcherAdapter) performSwitch(fromType, toType interfaces.ConsensusType) {
	defer func() {
		csa.mu.Lock()
		csa.switching = false
		csa.mu.Unlock()
	}()

	// 创建切换事件
	event := &SwitchAdapterEvent{
		ID:        fmt.Sprintf("switch_%d", time.Now().Unix()),
		FromType:  fromType,
		ToType:    toType,
		Strategy:  csa.config.SwitchStrategy,
		StartTime: time.Now(),
		Context:   make(map[string]interface{}),
	}

	// 触发切换开始回调
	if csa.onSwitchStarted != nil {
		csa.onSwitchStarted(fromType, toType)
	}

	var err error
	switch csa.config.SwitchStrategy {
	case SwitchStrategyGraceful:
		err = csa.performGracefulSwitch(fromType, toType, event)
	case SwitchStrategyImmediate:
		err = csa.performImmediateSwitch(fromType, toType, event)
	default:
		err = fmt.Errorf("不支持的切换策略: %v", csa.config.SwitchStrategy)
	}

	// 更新事件结果
	event.EndTime = time.Now()
	event.Success = (err == nil)
	if err != nil {
		event.Error = err.Error()
		log.Printf("共识切换失败: %v", err)
	} else {
		log.Printf("共识切换成功: %s -> %s", csa.getConsensusTypeName(fromType), csa.getConsensusTypeName(toType))
	}

	// 触发切换完成回调
	if csa.onSwitchCompleted != nil {
		csa.onSwitchCompleted(fromType, toType, event.Success)
	}
}

// performGracefulSwitch 执行优雅切换
func (csa *ConsensusSwitcherAdapter) performGracefulSwitch(fromType, toType interfaces.ConsensusType, event *SwitchAdapterEvent) error {
	// 1. 获取源和目标共识算法
	source, err := csa.getConsensusAlgorithm(fromType)
	if err != nil {
		return fmt.Errorf("获取源共识算法失败: %v", err)
	}

	target, err := csa.getConsensusAlgorithm(toType)
	if err != nil {
		return fmt.Errorf("获取目标共识算法失败: %v", err)
	}

	// 2. 备份当前状态（如果启用）
	if csa.config.BackupBeforeSwitch {
		if err := csa.backupCurrentState(event); err != nil {
			return fmt.Errorf("备份状态失败: %v", err)
		}
	}

	// 3. 启动目标共识算法
	if err := target.Start(context.Background()); err != nil {
		return fmt.Errorf("启动目标共识算法失败: %v", err)
	}

	// 4. 数据同步
	if err := csa.syncDataToTarget(source, target, event); err != nil {
		target.Stop()
		return fmt.Errorf("数据同步失败: %v", err)
	}

	// 5. 验证切换
	if err := csa.validateSwitch(target, event); err != nil {
		target.Stop()
		return fmt.Errorf("切换验证失败: %v", err)
	}

	// 6. 停止源共识算法
	if err := source.Stop(); err != nil {
		log.Printf("停止源共识算法时出现警告: %v", err)
	}

	// 7. 更新当前共识算法
	csa.mu.Lock()
	csa.currentConsensus = target
	csa.currentType = toType
	csa.mu.Unlock()

	return nil
}

// performImmediateSwitch 执行立即切换
func (csa *ConsensusSwitcherAdapter) performImmediateSwitch(fromType, toType interfaces.ConsensusType, event *SwitchAdapterEvent) error {
	// 1. 获取源和目标共识算法
	source, err := csa.getConsensusAlgorithm(fromType)
	if err != nil {
		return fmt.Errorf("获取源共识算法失败: %v", err)
	}

	target, err := csa.getConsensusAlgorithm(toType)
	if err != nil {
		return fmt.Errorf("获取目标共识算法失败: %v", err)
	}

	// 2. 立即停止源共识算法
	if err := source.Stop(); err != nil {
		log.Printf("停止源共识算法时出现警告: %v", err)
	}

	// 3. 启动目标共识算法
	if err := target.Start(context.Background()); err != nil {
		return fmt.Errorf("启动目标共识算法失败: %v", err)
	}

	// 4. 更新当前共识算法
	csa.mu.Lock()
	csa.currentConsensus = target
	csa.currentType = toType
	csa.mu.Unlock()

	return nil
}

// backupCurrentState 备份当前状态
func (csa *ConsensusSwitcherAdapter) backupCurrentState(event *SwitchAdapterEvent) error {
	// 实现状态备份逻辑
	log.Printf("备份当前状态...")
	event.Context["backup_completed"] = true
	return nil
}

// syncDataToTarget 同步数据到目标共识算法
func (csa *ConsensusSwitcherAdapter) syncDataToTarget(source, target interfaces.ConsensusAlgorithm, event *SwitchAdapterEvent) error {
	// 实现数据同步逻辑
	log.Printf("同步数据到目标共识算法...")

	// 获取源状态
	sourceStatus := source.GetStatus()

	// 这里可以实现具体的数据同步逻辑
	// 例如同步区块数据、状态数据等

	event.Context["sync_completed"] = true
	event.Context["source_status"] = sourceStatus

	return nil
}

// validateSwitch 验证切换
func (csa *ConsensusSwitcherAdapter) validateSwitch(target interfaces.ConsensusAlgorithm, event *SwitchAdapterEvent) error {
	// 实现切换验证逻辑
	log.Printf("验证切换...")

	// 检查目标共识算法状态
	status := target.GetStatus()
	if running, ok := status["running"].(bool); !ok || !running {
		return fmt.Errorf("目标共识算法未正常运行")
	}

	event.Context["validation_completed"] = true
	return nil
}

// getConsensusAlgorithm 获取共识算法实例
func (csa *ConsensusSwitcherAdapter) getConsensusAlgorithm(consensusType interfaces.ConsensusType) (interfaces.ConsensusAlgorithm, error) {
	switch consensusType {
	case interfaces.ConsensusTypeRaft:
		if csa.raftAdapter == nil {
			return nil, fmt.Errorf("Raft适配器未初始化")
		}
		return csa.raftAdapter, nil
	case interfaces.ConsensusTypePoA:
		if csa.poaAdapter == nil {
			return nil, fmt.Errorf("PoA适配器未初始化")
		}
		return csa.poaAdapter, nil
	default:
		return nil, fmt.Errorf("不支持的共识算法类型: %v", consensusType)
	}
}

// getConsensusTypeName 获取共识算法类型名称
func (csa *ConsensusSwitcherAdapter) getConsensusTypeName(consensusType interfaces.ConsensusType) string {
	switch consensusType {
	case interfaces.ConsensusTypeRaft:
		return "Raft"
	case interfaces.ConsensusTypePoA:
		return "PoA"
	case interfaces.ConsensusTypePBFT:
		return "PBFT"
	case interfaces.ConsensusTypePoS:
		return "PoS"
	default:
		return "Unknown"
	}
}

// GetCurrentType 获取当前共识算法类型
func (csa *ConsensusSwitcherAdapter) GetCurrentType() interfaces.ConsensusType {
	csa.mu.RLock()
	defer csa.mu.RUnlock()

	return csa.currentType
}

// GetCurrentConsensus 获取当前共识算法
func (csa *ConsensusSwitcherAdapter) GetCurrentConsensus() interfaces.ConsensusAlgorithm {
	csa.mu.RLock()
	defer csa.mu.RUnlock()

	return csa.currentConsensus
}

// GetSwitchState 获取切换状态
func (csa *ConsensusSwitcherAdapter) GetSwitchState() *SwitchAdapterState {
	csa.mu.RLock()
	defer csa.mu.RUnlock()

	state := &SwitchAdapterState{
		InProgress:  csa.switching,
		CurrentType: csa.currentType,
		Progress:    1.0, // 简化处理
		Stage:       "idle",
	}

	if csa.switching {
		state.Progress = 0.5 // 简化处理
		state.Stage = "switching"
	}

	return state
}

// IsSupported 检查是否支持指定的共识算法
func (csa *ConsensusSwitcherAdapter) IsSupported(consensusType interfaces.ConsensusType) bool {
	switch consensusType {
	case interfaces.ConsensusTypeRaft, interfaces.ConsensusTypePoA:
		return true
	default:
		return false
	}
}

// GetSupportedTypes 获取支持的共识算法类型
func (csa *ConsensusSwitcherAdapter) GetSupportedTypes() []interfaces.ConsensusType {
	return []interfaces.ConsensusType{
		interfaces.ConsensusTypeRaft,
		interfaces.ConsensusTypePoA,
	}
}

// SetSwitchStartedCallback 设置切换开始回调
func (csa *ConsensusSwitcherAdapter) SetSwitchStartedCallback(callback func(from, to interfaces.ConsensusType)) {
	csa.mu.Lock()
	defer csa.mu.Unlock()

	csa.onSwitchStarted = callback
}

// SetSwitchCompletedCallback 设置切换完成回调
func (csa *ConsensusSwitcherAdapter) SetSwitchCompletedCallback(callback func(from, to interfaces.ConsensusType, success bool)) {
	csa.mu.Lock()
	defer csa.mu.Unlock()

	csa.onSwitchCompleted = callback
}

// GetStatus 获取切换器状态
func (csa *ConsensusSwitcherAdapter) GetStatus() map[string]interface{} {
	csa.mu.RLock()
	defer csa.mu.RUnlock()

	status := map[string]interface{}{
		"current_type":    csa.getConsensusTypeName(csa.currentType),
		"switching":       csa.switching,
		"supported_types": csa.getSupportedTypeNames(),
		"switch_strategy": csa.config.SwitchStrategy,
		"auto_switch":     csa.config.EnableAutoSwitch,
	}

	if csa.currentConsensus != nil {
		status["current_consensus_status"] = csa.currentConsensus.GetStatus()
	}

	return status
}

// getSupportedTypeNames 获取支持的共识算法类型名称
func (csa *ConsensusSwitcherAdapter) getSupportedTypeNames() []string {
	types := csa.GetSupportedTypes()
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = csa.getConsensusTypeName(t)
	}
	return names
}
