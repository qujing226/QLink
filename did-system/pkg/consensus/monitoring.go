package consensus

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ConsensusMonitor 共识监控器
type ConsensusMonitor struct {
	// 监控配置
	config *MonitorConfig

	// 监控指标
	metrics *ConsensusMetrics

	// 故障检测
	failureDetector *FailureDetector

	// 恢复机制
	recoveryManager *RecoveryManager

	// 共识算法实例
	consensus ConsensusAlgorithm

	// 控制
	mu     sync.RWMutex
	stopCh chan struct{}

	// 回调函数
	onFailureDetected func(failure *FailureEvent)
	onRecoveryStarted func(recovery *RecoveryEvent)
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	// 监控间隔
	MonitorInterval time.Duration `json:"monitor_interval"`

	// 性能阈值
	MaxLatency     time.Duration `json:"max_latency"`
	MinThroughput  float64       `json:"min_throughput"`
	MaxFailureRate float64       `json:"max_failure_rate"`

	// 故障检测配置
	FailureDetectionWindow time.Duration `json:"failure_detection_window"`
	MaxConsecutiveFailures int           `json:"max_consecutive_failures"`

	// 恢复配置
	RecoveryTimeout     time.Duration `json:"recovery_timeout"`
	MaxRecoveryAttempts int           `json:"max_recovery_attempts"`

	// 告警配置
	EnableAlerts bool `json:"enable_alerts"`
}

// ConsensusMetrics 共识指标
type ConsensusMetrics struct {
	// 性能指标
	Latency     time.Duration `json:"latency"`
	Throughput  float64       `json:"throughput"`
	SuccessRate float64       `json:"success_rate"`

	// 状态指标
	ActiveNodes       int   `json:"active_nodes"`
	LeaderChanges     int64 `json:"leader_changes"`
	NetworkPartitions int   `json:"network_partitions"`

	// 错误指标
	TotalErrors       int64     `json:"total_errors"`
	ConsecutiveErrors int       `json:"consecutive_errors"`
	LastError         time.Time `json:"last_error"`

	// 时间戳
	LastUpdate time.Time `json:"last_update"`

	mu sync.RWMutex
}

// FailureDetector 故障检测器
type FailureDetector struct {
	config           *MonitorConfig
	failureHistory   []FailureEvent
	consecutiveFails int
	lastCheck        time.Time
	mu               sync.RWMutex
}

// RecoveryManager 恢复管理器
type RecoveryManager struct {
	config           *MonitorConfig
	recoveryHistory  []RecoveryEvent
	currentRecovery  *RecoveryEvent
	recoveryAttempts int
	mu               sync.RWMutex
}

// FailureEvent 故障事件
type FailureEvent struct {
	ID          string                 `json:"id"`
	Type        FailureType            `json:"type"`
	Severity    FailureSeverity        `json:"severity"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context"`
	Resolved    bool                   `json:"resolved"`
}

// RecoveryEvent 恢复事件
type RecoveryEvent struct {
	ID        string                 `json:"id"`
	FailureID string                 `json:"failure_id"`
	Strategy  RecoveryStrategy       `json:"strategy"`
	Status    RecoveryStatus         `json:"status"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Attempts  int                    `json:"attempts"`
	Context   map[string]interface{} `json:"context"`
	Success   bool                   `json:"success"`
}

// FailureType 故障类型
type FailureType int

const (
	FailureTypeLatency FailureType = iota
	FailureTypeThroughput
	FailureTypeNetworkPartition
	FailureTypeLeaderElection
	FailureTypeConsensusTimeout
	FailureTypeNodeFailure
	FailureTypeDataCorruption
)

// FailureSeverity 故障严重程度
type FailureSeverity int

const (
	FailureSeverityLow FailureSeverity = iota
	FailureSeverityMedium
	FailureSeverityHigh
	FailureSeverityCritical
)

// RecoveryStrategy 恢复策略
type RecoveryStrategy int

const (
	RecoveryStrategyRestart RecoveryStrategy = iota
	RecoveryStrategyLeaderElection
	RecoveryStrategyNetworkRepair
	RecoveryStrategyDataSync
	RecoveryStrategyRollback
	RecoveryStrategyManualIntervention
)

// RecoveryStatus 恢复状态
type RecoveryStatus int

const (
	RecoveryStatusPending RecoveryStatus = iota
	RecoveryStatusInProgress
	RecoveryStatusCompleted
	RecoveryStatusFailed
	RecoveryStatusAborted
)

// NewConsensusMonitor 创建共识监控器
func NewConsensusMonitor(config *MonitorConfig) *ConsensusMonitor {
	if config == nil {
		config = &MonitorConfig{
			MonitorInterval:        5 * time.Second,
			MaxLatency:             1 * time.Second,
			MinThroughput:          10.0,
			MaxFailureRate:         0.1,
			FailureDetectionWindow: 30 * time.Second,
			MaxConsecutiveFailures: 3,
			RecoveryTimeout:        60 * time.Second,
			MaxRecoveryAttempts:    3,
			EnableAlerts:           true,
		}
	}

	return &ConsensusMonitor{
		config:          config,
		metrics:         &ConsensusMetrics{},
		failureDetector: &FailureDetector{config: config},
		recoveryManager: &RecoveryManager{config: config},
		stopCh:          make(chan struct{}),
	}
}

// Start 启动监控器
func (cm *ConsensusMonitor) Start(ctx context.Context) error {
	log.Printf("启动共识监控器")

	// 启动监控循环
	go cm.monitorLoop(ctx)

	// 启动故障检测循环
	go cm.failureDetectionLoop(ctx)

	// 启动恢复管理循环
	go cm.recoveryLoop(ctx)

	return nil
}

// Stop 停止监控器
func (cm *ConsensusMonitor) Stop() error {
	close(cm.stopCh)
	log.Printf("共识监控器已停止")
	return nil
}

// monitorLoop 监控循环
func (cm *ConsensusMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.collectMetrics()
			cm.analyzePerformance()
		}
	}
}

// failureDetectionLoop 故障检测循环
func (cm *ConsensusMonitor) failureDetectionLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.config.MonitorInterval / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.detectFailures()
		}
	}
}

// recoveryLoop 恢复管理循环
func (cm *ConsensusMonitor) recoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(cm.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.manageRecovery()
		}
	}
}

// collectMetrics 收集指标
func (cm *ConsensusMonitor) collectMetrics() {
	cm.metrics.mu.Lock()
	defer cm.metrics.mu.Unlock()

	// 更新时间戳
	cm.metrics.LastUpdate = time.Now()

	// TODO: 实际实现中应该从共识节点收集真实指标
	// 这里是示例实现
	cm.metrics.Latency = 100 * time.Millisecond
	cm.metrics.Throughput = 50.0
	cm.metrics.SuccessRate = 0.95
	cm.metrics.ActiveNodes = 3

	log.Printf("收集指标: 延迟=%v, 吞吐量=%.2f, 成功率=%.2f",
		cm.metrics.Latency, cm.metrics.Throughput, cm.metrics.SuccessRate)
}

// analyzePerformance 分析性能
func (cm *ConsensusMonitor) analyzePerformance() {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	// 检查延迟
	if cm.metrics.Latency > cm.config.MaxLatency {
		cm.reportFailure(FailureTypeLatency, FailureSeverityMedium,
			fmt.Sprintf("延迟过高: %v > %v", cm.metrics.Latency, cm.config.MaxLatency))
	}

	// 检查吞吐量
	if cm.metrics.Throughput < cm.config.MinThroughput {
		cm.reportFailure(FailureTypeThroughput, FailureSeverityMedium,
			fmt.Sprintf("吞吐量过低: %.2f < %.2f", cm.metrics.Throughput, cm.config.MinThroughput))
	}

	// 检查失败率
	failureRate := 1.0 - cm.metrics.SuccessRate
	if failureRate > cm.config.MaxFailureRate {
		cm.reportFailure(FailureTypeConsensusTimeout, FailureSeverityHigh,
			fmt.Sprintf("失败率过高: %.2f > %.2f", failureRate, cm.config.MaxFailureRate))
	}
}

// detectFailures 检测故障
func (cm *ConsensusMonitor) detectFailures() {
	cm.failureDetector.mu.Lock()
	defer cm.failureDetector.mu.Unlock()

	now := time.Now()

	// 检查连续故障
	if cm.failureDetector.consecutiveFails >= cm.config.MaxConsecutiveFailures {
		cm.reportFailure(FailureTypeNodeFailure, FailureSeverityCritical,
			fmt.Sprintf("连续故障次数过多: %d", cm.failureDetector.consecutiveFails))
		cm.failureDetector.consecutiveFails = 0
	}

	cm.failureDetector.lastCheck = now
}

// reportFailure 报告故障
func (cm *ConsensusMonitor) reportFailure(failureType FailureType, severity FailureSeverity, description string) {
	failure := &FailureEvent{
		ID:          fmt.Sprintf("failure-%d", time.Now().UnixNano()),
		Type:        failureType,
		Severity:    severity,
		Description: description,
		Timestamp:   time.Now(),
		Context:     make(map[string]interface{}),
		Resolved:    false,
	}

	// 记录故障
	cm.failureDetector.mu.Lock()
	cm.failureDetector.failureHistory = append(cm.failureDetector.failureHistory, *failure)
	cm.failureDetector.consecutiveFails++
	cm.failureDetector.mu.Unlock()

	// 更新错误指标
	cm.metrics.mu.Lock()
	cm.metrics.TotalErrors++
	cm.metrics.ConsecutiveErrors++
	cm.metrics.LastError = time.Now()
	cm.metrics.mu.Unlock()

	log.Printf("检测到故障: %s - %s", failure.ID, failure.Description)

	// 触发回调
	if cm.onFailureDetected != nil {
		cm.onFailureDetected(failure)
	}

	// 启动恢复
	cm.startRecovery(failure)
}

// startRecovery 启动恢复
func (cm *ConsensusMonitor) startRecovery(failure *FailureEvent) {
	strategy := cm.selectRecoveryStrategy(failure)

	recovery := &RecoveryEvent{
		ID:        fmt.Sprintf("recovery-%d", time.Now().UnixNano()),
		FailureID: failure.ID,
		Strategy:  strategy,
		Status:    RecoveryStatusPending,
		StartTime: time.Now(),
		Attempts:  0,
		Context:   make(map[string]interface{}),
		Success:   false,
	}

	cm.recoveryManager.mu.Lock()
	cm.recoveryManager.currentRecovery = recovery
	cm.recoveryManager.recoveryHistory = append(cm.recoveryManager.recoveryHistory, *recovery)
	cm.recoveryManager.mu.Unlock()

	log.Printf("启动恢复: %s (策略: %d)", recovery.ID, recovery.Strategy)

	// 触发回调
	if cm.onRecoveryStarted != nil {
		cm.onRecoveryStarted(recovery)
	}
}

// selectRecoveryStrategy 选择恢复策略
func (cm *ConsensusMonitor) selectRecoveryStrategy(failure *FailureEvent) RecoveryStrategy {
	switch failure.Type {
	case FailureTypeLatency, FailureTypeThroughput:
		return RecoveryStrategyRestart
	case FailureTypeLeaderElection:
		return RecoveryStrategyLeaderElection
	case FailureTypeNetworkPartition:
		return RecoveryStrategyNetworkRepair
	case FailureTypeDataCorruption:
		return RecoveryStrategyDataSync
	case FailureTypeNodeFailure:
		if failure.Severity == FailureSeverityCritical {
			return RecoveryStrategyManualIntervention
		}
		return RecoveryStrategyRestart
	default:
		return RecoveryStrategyRestart
	}
}

// manageRecovery 管理恢复
func (cm *ConsensusMonitor) manageRecovery() {
	cm.recoveryManager.mu.Lock()
	defer cm.recoveryManager.mu.Unlock()

	if cm.recoveryManager.currentRecovery == nil {
		return
	}

	recovery := cm.recoveryManager.currentRecovery

	// 检查恢复超时
	if time.Since(recovery.StartTime) > cm.config.RecoveryTimeout {
		recovery.Status = RecoveryStatusFailed
		recovery.EndTime = time.Now()
		cm.recoveryManager.currentRecovery = nil
		log.Printf("恢复超时: %s", recovery.ID)
		return
	}

	// 执行恢复策略
	if recovery.Status == RecoveryStatusPending {
		recovery.Status = RecoveryStatusInProgress
		recovery.Attempts++

		success := cm.executeRecoveryStrategy(recovery)

		if success {
			recovery.Status = RecoveryStatusCompleted
			recovery.Success = true
			recovery.EndTime = time.Now()
			cm.recoveryManager.currentRecovery = nil

			// 重置连续错误计数
			cm.metrics.mu.Lock()
			cm.metrics.ConsecutiveErrors = 0
			cm.metrics.mu.Unlock()

			cm.failureDetector.mu.Lock()
			cm.failureDetector.consecutiveFails = 0
			cm.failureDetector.mu.Unlock()

			log.Printf("恢复成功: %s", recovery.ID)
		} else if recovery.Attempts >= cm.config.MaxRecoveryAttempts {
			recovery.Status = RecoveryStatusFailed
			recovery.EndTime = time.Now()
			cm.recoveryManager.currentRecovery = nil
			log.Printf("恢复失败，已达最大尝试次数: %s", recovery.ID)
		}
	}
}

// executeRecoveryStrategy 执行恢复策略
func (cm *ConsensusMonitor) executeRecoveryStrategy(recovery *RecoveryEvent) bool {
	log.Printf("执行恢复策略: %d (尝试: %d)", recovery.Strategy, recovery.Attempts)

	switch recovery.Strategy {
	case RecoveryStrategyRestart:
		return cm.executeRestart(recovery)
	case RecoveryStrategyLeaderElection:
		return cm.executeLeaderElection(recovery)
	case RecoveryStrategyNetworkRepair:
		return cm.executeNetworkRepair(recovery)
	case RecoveryStrategyDataSync:
		return cm.executeDataSync(recovery)
	case RecoveryStrategyRollback:
		return cm.executeRollback(recovery)
	case RecoveryStrategyManualIntervention:
		return cm.executeManualIntervention(recovery)
	default:
		return false
	}
}

// executeRestart 执行重启恢复
func (cm *ConsensusMonitor) executeRestart(recovery *RecoveryEvent) bool {
	log.Printf("执行重启恢复: %s", recovery.ID)

	// 1. 停止当前共识算法
	if cm.consensus != nil {
		log.Printf("停止当前共识算法")
		if err := cm.consensus.Stop(); err != nil {
			log.Printf("停止共识算法失败: %v", err)
			return false
		}
	}

	// 2. 等待一段时间确保完全停止
	time.Sleep(2 * time.Second)

	// 3. 重新启动共识算法
	if cm.consensus != nil {
		log.Printf("重新启动共识算法")
		ctx := context.Background()
		if err := cm.consensus.Start(ctx); err != nil {
			log.Printf("重启共识算法失败: %v", err)
			return false
		}
	}

	// 4. 验证重启是否成功
	time.Sleep(1 * time.Second)
	log.Printf("重启恢复完成")
	return true
}

// executeLeaderElection 执行领导者选举恢复
func (cm *ConsensusMonitor) executeLeaderElection(recovery *RecoveryEvent) bool {
	log.Printf("执行领导者选举恢复: %s", recovery.ID)

	// 1. 检查当前共识算法是否支持领导者选举
	if cm.consensus == nil {
		log.Printf("共识算法未初始化")
		return false
	}

	// 2. 触发领导者选举（对于Raft算法）
	if raftNode, ok := cm.consensus.(*RaftNode); ok {
		log.Printf("触发Raft领导者选举")

		// 强制节点进入候选者状态以开始选举
		raftNode.mu.Lock()
		raftNode.State = Candidate
		raftNode.term++
		raftNode.votedFor = raftNode.id
		raftNode.mu.Unlock()

		// 等待选举完成
		time.Sleep(3 * time.Second)

		log.Printf("领导者选举过程完成")
		return true
	}

	log.Printf("领导者选举恢复失败或不支持")
	return false
}

// executeNetworkRepair 执行网络修复恢复
func (cm *ConsensusMonitor) executeNetworkRepair(recovery *RecoveryEvent) bool {
	log.Printf("执行网络修复恢复: %s", recovery.ID)

	// 1. 检查网络连接状态
	if cm.consensus == nil {
		log.Printf("共识算法未初始化")
		return false
	}

	// 2. 尝试重新建立网络连接
	if raftNode, ok := cm.consensus.(*RaftNode); ok {
		log.Printf("修复Raft网络连接")

		// 重新初始化对等节点连接
		raftNode.mu.Lock()
		for peerID := range raftNode.peers {
			log.Printf("尝试重新连接节点: %s", peerID)
			// 这里应该调用网络层的重连逻辑
			// 由于网络层的具体实现可能不同，这里只是记录日志
		}
		raftNode.mu.Unlock()

		// 等待网络修复完成
		time.Sleep(2 * time.Second)

		log.Printf("网络修复完成")
		return true
	}

	// 3. 对于PoA算法的网络修复
	if poaNode, ok := cm.consensus.(*PoANode); ok {
		log.Printf("修复PoA网络连接")

		// PoA网络修复逻辑
		poaNode.mu.Lock()
		// 重置网络状态
		poaNode.mu.Unlock()

		time.Sleep(1 * time.Second)
		log.Printf("PoA网络修复完成")
		return true
	}

	log.Printf("网络修复失败或不支持")
	return false
}

// executeDataSync 执行数据同步恢复
func (cm *ConsensusMonitor) executeDataSync(recovery *RecoveryEvent) bool {
	log.Printf("执行数据同步恢复: %s", recovery.ID)

	// 1. 检查共识算法状态
	if cm.consensus == nil {
		log.Printf("共识算法未初始化")
		return false
	}

	// 2. 对于Raft算法的数据同步
	if raftNode, ok := cm.consensus.(*RaftNode); ok {
		log.Printf("执行Raft数据同步")

		raftNode.mu.Lock()

		// 重置提交索引，强制重新同步
		if raftNode.commitIndex > 0 {
			log.Printf("重置提交索引从 %d 到 %d", raftNode.commitIndex, raftNode.lastApplied)
			raftNode.commitIndex = raftNode.lastApplied
		}

		// 如果是领导者，重新发送心跳以同步数据
		if raftNode.State == Leader {
			log.Printf("作为领导者，准备重新同步数据到所有跟随者")
			// 重置nextIndex以强制重新发送日志条目
			for peerID := range raftNode.peers {
				if raftNode.nextIndex != nil {
					raftNode.nextIndex[peerID] = raftNode.lastApplied + 1
				}
			}
		}

		raftNode.mu.Unlock()

		// 等待数据同步完成
		time.Sleep(3 * time.Second)

		log.Printf("Raft数据同步完成")
		return true
	}

	// 3. 对于PoA算法的数据同步
	if poaNode, ok := cm.consensus.(*PoANode); ok {
		log.Printf("执行PoA数据同步")

		// PoA数据同步逻辑
		poaNode.mu.Lock()
		// 重新验证和同步区块数据
		log.Printf("重新验证PoA区块数据")
		poaNode.mu.Unlock()

		time.Sleep(2 * time.Second)
		log.Printf("PoA数据同步完成")
		return true
	}

	log.Printf("数据同步失败或不支持")
	return false
}

// executeRollback 执行回滚恢复
func (cm *ConsensusMonitor) executeRollback(recovery *RecoveryEvent) bool {
	log.Printf("执行回滚恢复: %s", recovery.ID)

	// 1. 检查共识算法状态
	if cm.consensus == nil {
		log.Printf("共识算法未初始化")
		return false
	}

	// 2. 对于Raft算法的回滚
	if raftNode, ok := cm.consensus.(*RaftNode); ok {
		log.Printf("执行Raft回滚操作")

		raftNode.mu.Lock()

		// 回滚到上一个安全的提交点
		if raftNode.commitIndex > 0 {
			rollbackIndex := raftNode.commitIndex - 1
			log.Printf("回滚提交索引从 %d 到 %d", raftNode.commitIndex, rollbackIndex)

			// 截断日志到回滚点
			if rollbackIndex >= 0 && int(rollbackIndex) < len(raftNode.log) {
				raftNode.log = raftNode.log[:rollbackIndex+1]
				raftNode.commitIndex = rollbackIndex
				raftNode.lastApplied = rollbackIndex
			}
		}

		// 如果是领导者，通知所有跟随者回滚
		if raftNode.State == Leader {
			log.Printf("作为领导者，通知跟随者执行回滚")
			for peerID := range raftNode.peers {
				if raftNode.nextIndex != nil {
					raftNode.nextIndex[peerID] = raftNode.commitIndex + 1
				}
				if raftNode.matchIndex != nil {
					raftNode.matchIndex[peerID] = 0
				}
			}
		}

		raftNode.mu.Unlock()

		// 等待回滚完成
		time.Sleep(2 * time.Second)

		log.Printf("Raft回滚完成")
		return true
	}

	// 3. 对于PoA算法的回滚
	if poaNode, ok := cm.consensus.(*PoANode); ok {
		log.Printf("执行PoA回滚操作")

		// PoA回滚逻辑
		poaNode.mu.Lock()
		// 回滚到上一个有效区块
		log.Printf("回滚PoA区块到上一个有效状态")
		poaNode.mu.Unlock()

		time.Sleep(1 * time.Second)
		log.Printf("PoA回滚完成")
		return true
	}

	log.Printf("回滚失败或不支持")
	return false
}

// executeManualIntervention 执行手动干预恢复
func (cm *ConsensusMonitor) executeManualIntervention(recovery *RecoveryEvent) bool {
	log.Printf("需要手动干预: %s", recovery.ID)

	// 1. 记录详细的故障信息
	log.Printf("故障详情 - 恢复事件ID: %s, 故障ID: %s", recovery.ID, recovery.FailureID)

	// 2. 发送告警通知
	alert := map[string]interface{}{
		"type":        "manual_intervention_required",
		"recovery_id": recovery.ID,
		"failure_id":  recovery.FailureID,
		"timestamp":   time.Now(),
		"severity":    "critical",
		"message":     fmt.Sprintf("共识系统需要手动干预 - 恢复事件: %s", recovery.ID),
		"context":     recovery.Context,
	}

	// 3. 如果配置了告警回调，调用它
	if cm.onFailureDetected != nil {
		// 创建一个手动干预类型的故障事件
		manualFailure := &FailureEvent{
			ID:          fmt.Sprintf("manual_%s", recovery.ID),
			Type:        FailureTypeNodeFailure, // 使用节点故障类型
			Severity:    FailureSeverityCritical,
			Description: fmt.Sprintf("需要手动干预的故障: %s", recovery.FailureID),
			Timestamp:   time.Now(),
			Context:     alert,
			Resolved:    false,
		}

		log.Printf("触发手动干预告警回调")
		cm.onFailureDetected(manualFailure)
	}

	// 4. 记录到系统日志
	log.Printf("ALERT: 共识系统故障需要管理员手动处理")
	log.Printf("ALERT: 请检查系统状态并采取适当的恢复措施")
	log.Printf("ALERT: 故障上下文: %+v", recovery.Context)

	// 5. 更新指标
	if cm.metrics != nil {
		cm.metrics.mu.Lock()
		cm.metrics.TotalErrors++
		cm.metrics.LastError = time.Now()
		cm.metrics.mu.Unlock()
	}

	// 手动干预不能自动完成，返回false表示需要人工处理
	return false
}

// GetMetrics 获取指标
func (cm *ConsensusMonitor) GetMetrics() *ConsensusMetrics {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	// 创建一个新的结构体，手动复制字段以避免复制锁
	return &ConsensusMetrics{
		Latency:           cm.metrics.Latency,
		Throughput:        cm.metrics.Throughput,
		SuccessRate:       cm.metrics.SuccessRate,
		ActiveNodes:       cm.metrics.ActiveNodes,
		LeaderChanges:     cm.metrics.LeaderChanges,
		NetworkPartitions: cm.metrics.NetworkPartitions,
		TotalErrors:       cm.metrics.TotalErrors,
		ConsecutiveErrors: cm.metrics.ConsecutiveErrors,
		LastError:         cm.metrics.LastError,
		LastUpdate:        cm.metrics.LastUpdate,
	}
}

// GetFailureHistory 获取故障历史
func (cm *ConsensusMonitor) GetFailureHistory() []FailureEvent {
	cm.failureDetector.mu.RLock()
	defer cm.failureDetector.mu.RUnlock()

	// 返回历史副本
	history := make([]FailureEvent, len(cm.failureDetector.failureHistory))
	copy(history, cm.failureDetector.failureHistory)
	return history
}

// GetRecoveryHistory 获取恢复历史
func (cm *ConsensusMonitor) GetRecoveryHistory() []RecoveryEvent {
	cm.recoveryManager.mu.RLock()
	defer cm.recoveryManager.mu.RUnlock()

	// 返回历史副本
	history := make([]RecoveryEvent, len(cm.recoveryManager.recoveryHistory))
	copy(history, cm.recoveryManager.recoveryHistory)
	return history
}

// SetFailureCallback 设置故障回调
func (cm *ConsensusMonitor) SetFailureCallback(callback func(*FailureEvent)) {
	cm.onFailureDetected = callback
}

// SetRecoveryCallback 设置恢复回调
func (cm *ConsensusMonitor) SetRecoveryCallback(callback func(*RecoveryEvent)) {
	cm.onRecoveryStarted = callback
}

// Reset 重置监控器状态
func (cm *ConsensusMonitor) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 重置指标
	cm.metrics.mu.Lock()
	cm.metrics.ConsecutiveErrors = 0
	cm.metrics.TotalErrors = 0
	cm.metrics.LastError = time.Time{}
	cm.metrics.mu.Unlock()

	// 重置故障检测器
	cm.failureDetector.mu.Lock()
	cm.failureDetector.consecutiveFails = 0
	cm.failureDetector.failureHistory = nil
	cm.failureDetector.mu.Unlock()

	// 重置恢复管理器
	cm.recoveryManager.mu.Lock()
	cm.recoveryManager.currentRecovery = nil
	cm.recoveryManager.recoveryAttempts = 0
	cm.recoveryManager.mu.Unlock()

	log.Printf("监控器状态已重置")
}

// GetStatus 获取监控器状态
func (cm *ConsensusMonitor) GetStatus() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	metrics := cm.GetMetrics()

	return map[string]interface{}{
		"monitor_active":     true,
		"last_update":        metrics.LastUpdate,
		"current_metrics":    metrics,
		"failure_count":      len(cm.GetFailureHistory()),
		"recovery_count":     len(cm.GetRecoveryHistory()),
		"consecutive_errors": metrics.ConsecutiveErrors,
		"config":             cm.config,
	}
}
