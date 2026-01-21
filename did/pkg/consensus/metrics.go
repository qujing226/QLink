package consensus

import (
	"sync"
	"time"
)

// ConsensusMetricsData 共识指标数据
type ConsensusMetricsData struct {
	mu sync.RWMutex

	// 基础指标
	StartTime         time.Time
	LastUpdateTime    time.Time
	TotalProposals    uint64
	AcceptedProposals uint64
	RejectedProposals uint64

	// 性能指标
	AverageLatency   time.Duration
	ThroughputPerSec float64

	// 网络指标
	PeerCount         int
	ActiveConnections int
	NetworkLatency    time.Duration

	// 状态指标
	CurrentTerm   uint64
	CurrentHeight uint64
	IsLeader      bool
	IsRunning     bool

	// 错误指标
	ErrorCount    uint64
	LastError     string
	LastErrorTime time.Time

	// 自定义指标
	CustomMetrics map[string]interface{}
}

// NewConsensusMetrics 创建新的共识指标
func NewConsensusMetrics() *ConsensusMetricsData {
	return &ConsensusMetricsData{
		StartTime:     time.Now(),
		CustomMetrics: make(map[string]interface{}),
	}
}

// UpdateProposal 更新提案指标
func (m *ConsensusMetricsData) UpdateProposal(accepted bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalProposals++
	if accepted {
		m.AcceptedProposals++
	} else {
		m.RejectedProposals++
	}

	// 更新平均延迟
	if m.TotalProposals == 1 {
		m.AverageLatency = latency
	} else {
		// 使用指数移动平均
		alpha := 0.1
		m.AverageLatency = time.Duration(float64(m.AverageLatency)*(1-alpha) + float64(latency)*alpha)
	}

	m.LastUpdateTime = time.Now()
	m.updateThroughput()
}

// UpdateNetworkMetrics 更新网络指标
func (m *ConsensusMetricsData) UpdateNetworkMetrics(peerCount, activeConnections int, networkLatency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PeerCount = peerCount
	m.ActiveConnections = activeConnections
	m.NetworkLatency = networkLatency
	m.LastUpdateTime = time.Now()
}

// UpdateStatus 更新状态指标
func (m *ConsensusMetricsData) UpdateStatus(term, height uint64, isLeader, isRunning bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CurrentTerm = term
	m.CurrentHeight = height
	m.IsLeader = isLeader
	m.IsRunning = isRunning
	m.LastUpdateTime = time.Now()
}

// RecordError 记录错误
func (m *ConsensusMetricsData) RecordError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ErrorCount++
	m.LastError = err.Error()
	m.LastErrorTime = time.Now()
}

// SetCustomMetric 设置自定义指标
func (m *ConsensusMetricsData) SetCustomMetric(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CustomMetrics[key] = value
	m.LastUpdateTime = time.Now()
}

// GetCustomMetric 获取自定义指标
func (m *ConsensusMetricsData) GetCustomMetric(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, exists := m.CustomMetrics[key]
	return value, exists
}

// updateThroughput 更新吞吐量
func (m *ConsensusMetricsData) updateThroughput() {
	duration := time.Since(m.StartTime).Seconds()
	if duration > 0 {
		m.ThroughputPerSec = float64(m.AcceptedProposals) / duration
	}
}

// GetSnapshot 获取指标快照
func (m *ConsensusMetricsData) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := map[string]interface{}{
		"start_time":         m.StartTime,
		"last_update_time":   m.LastUpdateTime,
		"uptime":             time.Since(m.StartTime),
		"total_proposals":    m.TotalProposals,
		"accepted_proposals": m.AcceptedProposals,
		"rejected_proposals": m.RejectedProposals,
		"acceptance_rate":    m.getAcceptanceRate(),
		"average_latency":    m.AverageLatency,
		"throughput_per_sec": m.ThroughputPerSec,
		"peer_count":         m.PeerCount,
		"active_connections": m.ActiveConnections,
		"network_latency":    m.NetworkLatency,
		"current_term":       m.CurrentTerm,
		"current_height":     m.CurrentHeight,
		"is_leader":          m.IsLeader,
		"is_running":         m.IsRunning,
		"error_count":        m.ErrorCount,
		"last_error":         m.LastError,
		"last_error_time":    m.LastErrorTime,
	}

	// 添加自定义指标
	for key, value := range m.CustomMetrics {
		snapshot["custom_"+key] = value
	}

	return snapshot
}

// getAcceptanceRate 计算接受率
func (m *ConsensusMetricsData) getAcceptanceRate() float64 {
	if m.TotalProposals == 0 {
		return 0.0
	}
	return float64(m.AcceptedProposals) / float64(m.TotalProposals)
}

// Reset 重置指标
func (m *ConsensusMetricsData) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartTime = time.Now()
	m.LastUpdateTime = time.Time{}
	m.TotalProposals = 0
	m.AcceptedProposals = 0
	m.RejectedProposals = 0
	m.AverageLatency = 0
	m.ThroughputPerSec = 0
	m.ErrorCount = 0
	m.LastError = ""
	m.LastErrorTime = time.Time{}
	m.CustomMetrics = make(map[string]interface{})
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*ConsensusMetricsData
}

// NewMetricsCollector 创建新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*ConsensusMetricsData),
	}
}

// RegisterConsensus 注册共识算法指标
func (c *MetricsCollector) RegisterConsensus(name string, metrics *ConsensusMetricsData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics[name] = metrics
}

// UnregisterConsensus 注销共识算法指标
func (c *MetricsCollector) UnregisterConsensus(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.metrics, name)
}

// GetMetrics 获取指定共识算法的指标
func (c *MetricsCollector) GetMetrics(name string) (*ConsensusMetricsData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics, exists := c.metrics[name]
	return metrics, exists
}

// GetAllMetrics 获取所有指标
func (c *MetricsCollector) GetAllMetrics() map[string]*ConsensusMetricsData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*ConsensusMetricsData)
	for name, metrics := range c.metrics {
		result[name] = metrics
	}
	return result
}

// GetSummary 获取指标摘要
func (c *MetricsCollector) GetSummary() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	summary := map[string]interface{}{
		"total_consensus_algorithms": len(c.metrics),
		"algorithms":                 make(map[string]interface{}),
	}

	var totalProposals uint64
	var totalAccepted uint64
	var totalErrors uint64
	var runningCount int

	algorithms := make(map[string]interface{})
	for name, metrics := range c.metrics {
		snapshot := metrics.GetSnapshot()
		algorithms[name] = snapshot

		// 累计统计
		totalProposals += metrics.TotalProposals
		totalAccepted += metrics.AcceptedProposals
		totalErrors += metrics.ErrorCount
		if metrics.IsRunning {
			runningCount++
		}
	}

	summary["algorithms"] = algorithms
	summary["total_proposals"] = totalProposals
	summary["total_accepted"] = totalAccepted
	summary["total_errors"] = totalErrors
	summary["running_algorithms"] = runningCount

	if totalProposals > 0 {
		summary["overall_acceptance_rate"] = float64(totalAccepted) / float64(totalProposals)
	} else {
		summary["overall_acceptance_rate"] = 0.0
	}

	return summary
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	collector *MetricsCollector
	interval  time.Duration
	stopCh    chan struct{}
	callbacks []func(map[string]interface{})
	mu        sync.RWMutex
}

// NewPerformanceMonitor 创建新的性能监控器
func NewPerformanceMonitor(collector *MetricsCollector, interval time.Duration) *PerformanceMonitor {
	return &PerformanceMonitor{
		collector: collector,
		interval:  interval,
		stopCh:    make(chan struct{}),
		callbacks: make([]func(map[string]interface{}), 0),
	}
}

// AddCallback 添加监控回调
func (p *PerformanceMonitor) AddCallback(callback func(map[string]interface{})) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.callbacks = append(p.callbacks, callback)
}

// Start 启动性能监控
func (p *PerformanceMonitor) Start() {
	go p.monitorLoop()
}

// Stop 停止性能监控
func (p *PerformanceMonitor) Stop() {
	close(p.stopCh)
}

// monitorLoop 监控循环
func (p *PerformanceMonitor) monitorLoop() {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			summary := p.collector.GetSummary()
			p.notifyCallbacks(summary)
		case <-p.stopCh:
			return
		}
	}
}

// notifyCallbacks 通知回调函数
func (p *PerformanceMonitor) notifyCallbacks(summary map[string]interface{}) {
	p.mu.RLock()
	callbacks := make([]func(map[string]interface{}), len(p.callbacks))
	copy(callbacks, p.callbacks)
	p.mu.RUnlock()

	for _, callback := range callbacks {
		go func(cb func(map[string]interface{})) {
			defer func() {
				if r := recover(); r != nil {
					// 忽略回调函数中的panic
				}
			}()
			cb(summary)
		}(callback)
	}
}

// 全局指标收集器
var globalMetricsCollector = NewMetricsCollector()

// GetGlobalMetricsCollector 获取全局指标收集器
func GetGlobalMetricsCollector() *MetricsCollector {
	return globalMetricsCollector
}

// RegisterConsensusMetrics 注册共识算法指标到全局收集器
func RegisterConsensusMetrics(name string, metrics *ConsensusMetricsData) {
	globalMetricsCollector.RegisterConsensus(name, metrics)
}

// UnregisterConsensusMetrics 从全局收集器注销共识算法指标
func UnregisterConsensusMetrics(name string) {
	globalMetricsCollector.UnregisterConsensus(name)
}

// GetConsensusMetrics 从全局收集器获取共识算法指标
func GetConsensusMetrics(name string) (*ConsensusMetricsData, bool) {
	return globalMetricsCollector.GetMetrics(name)
}

// GetAllConsensusMetrics 从全局收集器获取所有指标
func GetAllConsensusMetrics() map[string]*ConsensusMetricsData {
	return globalMetricsCollector.GetAllMetrics()
}

// GetConsensusMetricsSummary 从全局收集器获取指标摘要
func GetConsensusMetricsSummary() map[string]interface{} {
	return globalMetricsCollector.GetSummary()
}
