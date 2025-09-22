package did

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics DID系统监控指标
type Metrics struct {
	// 操作计数器
	RegisterCount   int64 `json:"register_count"`
	ResolveCount    int64 `json:"resolve_count"`
	UpdateCount     int64 `json:"update_count"`
	RevokeCount     int64 `json:"revoke_count"`
	
	// 批量操作计数器
	BatchRegisterCount int64 `json:"batch_register_count"`
	BatchResolveCount  int64 `json:"batch_resolve_count"`
	BatchUpdateCount   int64 `json:"batch_update_count"`
	BatchRevokeCount   int64 `json:"batch_revoke_count"`
	
	// 错误计数器
	RegisterErrors  int64 `json:"register_errors"`
	ResolveErrors   int64 `json:"resolve_errors"`
	UpdateErrors    int64 `json:"update_errors"`
	RevokeErrors    int64 `json:"revoke_errors"`
	
	// 批量操作错误计数器
	BatchRegisterErrors int64 `json:"batch_register_errors"`
	BatchResolveErrors  int64 `json:"batch_resolve_errors"`
	BatchUpdateErrors   int64 `json:"batch_update_errors"`
	BatchRevokeErrors   int64 `json:"batch_revoke_errors"`
	
	// 性能指标
	AvgRegisterTime time.Duration `json:"avg_register_time"`
	AvgResolveTime  time.Duration `json:"avg_resolve_time"`
	AvgUpdateTime   time.Duration `json:"avg_update_time"`
	AvgRevokeTime   time.Duration `json:"avg_revoke_time"`
	
	// 批量操作性能指标
	AvgBatchRegisterTime time.Duration `json:"avg_batch_register_time"`
	AvgBatchResolveTime  time.Duration `json:"avg_batch_resolve_time"`
	AvgBatchUpdateTime   time.Duration `json:"avg_batch_update_time"`
	AvgBatchRevokeTime   time.Duration `json:"avg_batch_revoke_time"`
	
	// 缓存指标
	CacheHits       int64 `json:"cache_hits"`
	CacheMisses     int64 `json:"cache_misses"`
	CacheSize       int64 `json:"cache_size"`
	
	// 系统指标
	ActiveDIDs      int64     `json:"active_dids"`
	RevokedDIDs     int64     `json:"revoked_dids"`
	LastUpdated     time.Time `json:"last_updated"`
	
	// 并发指标
	ConcurrentOperations int64 `json:"concurrent_operations"`
	MaxConcurrency       int64 `json:"max_concurrency"`
	
	// 内部统计
	mu                   sync.RWMutex
	registerTimes        []time.Duration
	resolveTimes         []time.Duration
	updateTimes          []time.Duration
	revokeTimes          []time.Duration
	batchRegisterTimes   []time.Duration
	batchResolveTimes    []time.Duration
	batchUpdateTimes     []time.Duration
	batchRevokeTimes     []time.Duration
}

// NewMetrics 创建新的监控指标实例
func NewMetrics() *Metrics {
	return &Metrics{
		LastUpdated:   time.Now(),
		registerTimes: make([]time.Duration, 0, 1000),
		resolveTimes:  make([]time.Duration, 0, 1000),
		updateTimes:   make([]time.Duration, 0, 1000),
		revokeTimes:   make([]time.Duration, 0, 1000),
	}
}

// RecordRegister 记录注册操作
func (m *Metrics) RecordRegister(duration time.Duration, success bool) {
	atomic.AddInt64(&m.RegisterCount, 1)
	if !success {
		atomic.AddInt64(&m.RegisterErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 过滤负数持续时间
	if duration < 0 {
		duration = 0
	}
	
	m.registerTimes = append(m.registerTimes, duration)
	if len(m.registerTimes) > 1000 {
		m.registerTimes = m.registerTimes[1:]
	}
	m.updateAverage(&m.AvgRegisterTime, m.registerTimes)
	m.LastUpdated = time.Now()
}

// RecordResolve 记录解析操作
func (m *Metrics) RecordResolve(duration time.Duration, success bool) {
	atomic.AddInt64(&m.ResolveCount, 1)
	if !success {
		atomic.AddInt64(&m.ResolveErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.resolveTimes = append(m.resolveTimes, duration)
	if len(m.resolveTimes) > 1000 {
		m.resolveTimes = m.resolveTimes[1:]
	}
	m.updateAverage(&m.AvgResolveTime, m.resolveTimes)
	m.LastUpdated = time.Now()
}

// RecordUpdate 记录更新操作
func (m *Metrics) RecordUpdate(duration time.Duration, success bool) {
	atomic.AddInt64(&m.UpdateCount, 1)
	if !success {
		atomic.AddInt64(&m.UpdateErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.updateTimes = append(m.updateTimes, duration)
	if len(m.updateTimes) > 1000 {
		m.updateTimes = m.updateTimes[1:]
	}
	m.updateAverage(&m.AvgUpdateTime, m.updateTimes)
	m.LastUpdated = time.Now()
}

// RecordRevoke 记录撤销操作
func (m *Metrics) RecordRevoke(duration time.Duration, success bool) {
	atomic.AddInt64(&m.RevokeCount, 1)
	if !success {
		atomic.AddInt64(&m.RevokeErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.revokeTimes = append(m.revokeTimes, duration)
	if len(m.revokeTimes) > 1000 {
		m.revokeTimes = m.revokeTimes[1:]
	}
	m.updateAverage(&m.AvgRevokeTime, m.revokeTimes)
	m.LastUpdated = time.Now()
}

// RecordBatchRegister 记录批量注册操作
func (m *Metrics) RecordBatchRegister(duration time.Duration, success bool, count int) {
	atomic.AddInt64(&m.BatchRegisterCount, 1)
	if !success {
		atomic.AddInt64(&m.BatchRegisterErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.batchRegisterTimes = append(m.batchRegisterTimes, duration)
	if len(m.batchRegisterTimes) > 1000 {
		m.batchRegisterTimes = m.batchRegisterTimes[1:]
	}
	m.updateAverage(&m.AvgBatchRegisterTime, m.batchRegisterTimes)
	m.LastUpdated = time.Now()
}

// RecordBatchResolve 记录批量解析操作
func (m *Metrics) RecordBatchResolve(duration time.Duration, success bool, count int) {
	atomic.AddInt64(&m.BatchResolveCount, 1)
	if !success {
		atomic.AddInt64(&m.BatchResolveErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.batchResolveTimes = append(m.batchResolveTimes, duration)
	if len(m.batchResolveTimes) > 1000 {
		m.batchResolveTimes = m.batchResolveTimes[1:]
	}
	m.updateAverage(&m.AvgBatchResolveTime, m.batchResolveTimes)
	m.LastUpdated = time.Now()
}

// RecordBatchUpdate 记录批量更新操作
func (m *Metrics) RecordBatchUpdate(duration time.Duration, success bool, count int) {
	atomic.AddInt64(&m.BatchUpdateCount, 1)
	if !success {
		atomic.AddInt64(&m.BatchUpdateErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.batchUpdateTimes = append(m.batchUpdateTimes, duration)
	if len(m.batchUpdateTimes) > 1000 {
		m.batchUpdateTimes = m.batchUpdateTimes[1:]
	}
	m.updateAverage(&m.AvgBatchUpdateTime, m.batchUpdateTimes)
	m.LastUpdated = time.Now()
}

// RecordBatchRevoke 记录批量撤销操作
func (m *Metrics) RecordBatchRevoke(duration time.Duration, success bool, count int) {
	atomic.AddInt64(&m.BatchRevokeCount, 1)
	if !success {
		atomic.AddInt64(&m.BatchRevokeErrors, 1)
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.batchRevokeTimes = append(m.batchRevokeTimes, duration)
	if len(m.batchRevokeTimes) > 1000 {
		m.batchRevokeTimes = m.batchRevokeTimes[1:]
	}
	m.updateAverage(&m.AvgBatchRevokeTime, m.batchRevokeTimes)
	m.LastUpdated = time.Now()
}

// IncrementConcurrency 增加并发操作计数
func (m *Metrics) IncrementConcurrency() {
	current := atomic.AddInt64(&m.ConcurrentOperations, 1)
	// 更新最大并发数
	for {
		max := atomic.LoadInt64(&m.MaxConcurrency)
		if current <= max || atomic.CompareAndSwapInt64(&m.MaxConcurrency, max, current) {
			break
		}
	}
}

// DecrementConcurrency 减少并发操作计数
func (m *Metrics) DecrementConcurrency() {
	atomic.AddInt64(&m.ConcurrentOperations, -1)
}

// RecordCacheHit 记录缓存命中
func (m *Metrics) RecordCacheHit() {
	atomic.AddInt64(&m.CacheHits, 1)
}

// RecordCacheMiss 记录缓存未命中
func (m *Metrics) RecordCacheMiss() {
	atomic.AddInt64(&m.CacheMisses, 1)
}

// UpdateCacheSize 更新缓存大小
func (m *Metrics) UpdateCacheSize(size int64) {
	atomic.StoreInt64(&m.CacheSize, size)
}

// UpdateDIDCounts 更新DID计数
func (m *Metrics) UpdateDIDCounts(active, revoked int64) {
	atomic.StoreInt64(&m.ActiveDIDs, active)
	atomic.StoreInt64(&m.RevokedDIDs, revoked)
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return &Metrics{
		RegisterCount:   atomic.LoadInt64(&m.RegisterCount),
		ResolveCount:    atomic.LoadInt64(&m.ResolveCount),
		UpdateCount:     atomic.LoadInt64(&m.UpdateCount),
		RevokeCount:     atomic.LoadInt64(&m.RevokeCount),
		RegisterErrors:  atomic.LoadInt64(&m.RegisterErrors),
		ResolveErrors:   atomic.LoadInt64(&m.ResolveErrors),
		UpdateErrors:    atomic.LoadInt64(&m.UpdateErrors),
		RevokeErrors:    atomic.LoadInt64(&m.RevokeErrors),
		AvgRegisterTime: m.AvgRegisterTime,
		AvgResolveTime:  m.AvgResolveTime,
		AvgUpdateTime:   m.AvgUpdateTime,
		AvgRevokeTime:   m.AvgRevokeTime,
		CacheHits:       atomic.LoadInt64(&m.CacheHits),
		CacheMisses:     atomic.LoadInt64(&m.CacheMisses),
		CacheSize:       atomic.LoadInt64(&m.CacheSize),
		ActiveDIDs:      atomic.LoadInt64(&m.ActiveDIDs),
		RevokedDIDs:     atomic.LoadInt64(&m.RevokedDIDs),
		LastUpdated:     m.LastUpdated,
	}
}

// GetSuccessRate 获取成功率
func (m *Metrics) GetSuccessRate() map[string]float64 {
	registerCount := atomic.LoadInt64(&m.RegisterCount)
	resolveCount := atomic.LoadInt64(&m.ResolveCount)
	updateCount := atomic.LoadInt64(&m.UpdateCount)
	revokeCount := atomic.LoadInt64(&m.RevokeCount)
	
	registerErrors := atomic.LoadInt64(&m.RegisterErrors)
	resolveErrors := atomic.LoadInt64(&m.ResolveErrors)
	updateErrors := atomic.LoadInt64(&m.UpdateErrors)
	revokeErrors := atomic.LoadInt64(&m.RevokeErrors)
	
	rates := make(map[string]float64)
	
	if registerCount > 0 {
		rates["register"] = float64(registerCount-registerErrors) / float64(registerCount)
	}
	if resolveCount > 0 {
		rates["resolve"] = float64(resolveCount-resolveErrors) / float64(resolveCount)
	}
	if updateCount > 0 {
		rates["update"] = float64(updateCount-updateErrors) / float64(updateCount)
	}
	if revokeCount > 0 {
		rates["revoke"] = float64(revokeCount-revokeErrors) / float64(revokeCount)
	}
	
	return rates
}

// GetCacheHitRate 获取缓存命中率
func (m *Metrics) GetCacheHitRate() float64 {
	hits := atomic.LoadInt64(&m.CacheHits)
	misses := atomic.LoadInt64(&m.CacheMisses)
	total := hits + misses
	
	if total == 0 {
		return 0.0
	}
	
	return float64(hits) / float64(total)
}

// Reset 重置所有指标
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	atomic.StoreInt64(&m.RegisterCount, 0)
	atomic.StoreInt64(&m.ResolveCount, 0)
	atomic.StoreInt64(&m.UpdateCount, 0)
	atomic.StoreInt64(&m.RevokeCount, 0)
	atomic.StoreInt64(&m.RegisterErrors, 0)
	atomic.StoreInt64(&m.ResolveErrors, 0)
	atomic.StoreInt64(&m.UpdateErrors, 0)
	atomic.StoreInt64(&m.RevokeErrors, 0)
	atomic.StoreInt64(&m.CacheHits, 0)
	atomic.StoreInt64(&m.CacheMisses, 0)
	atomic.StoreInt64(&m.CacheSize, 0)
	atomic.StoreInt64(&m.ActiveDIDs, 0)
	atomic.StoreInt64(&m.RevokedDIDs, 0)
	
	m.AvgRegisterTime = 0
	m.AvgResolveTime = 0
	m.AvgUpdateTime = 0
	m.AvgRevokeTime = 0
	
	m.registerTimes = m.registerTimes[:0]
	m.resolveTimes = m.resolveTimes[:0]
	m.updateTimes = m.updateTimes[:0]
	m.revokeTimes = m.revokeTimes[:0]
	
	m.LastUpdated = time.Now()
}

// updateAverage 更新平均值
func (m *Metrics) updateAverage(avg *time.Duration, times []time.Duration) {
	if len(times) == 0 {
		*avg = 0
		return
	}
	
	var total time.Duration
	for _, t := range times {
		total += t
	}
	*avg = total / time.Duration(len(times))
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	metrics *Metrics
	ticker  *time.Ticker
	done    chan struct{}
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(metrics *Metrics, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		metrics: metrics,
		ticker:  time.NewTicker(interval),
		done:    make(chan struct{}),
	}
}

// Start 启动指标收集
func (mc *MetricsCollector) Start() {
	go func() {
		for {
			select {
			case <-mc.ticker.C:
				// 这里可以添加定期收集系统指标的逻辑
				// 例如：更新活跃DID数量、内存使用情况等
			case <-mc.done:
				return
			}
		}
	}()
}

// Stop 停止指标收集
func (mc *MetricsCollector) Stop() {
	mc.ticker.Stop()
	close(mc.done)
}