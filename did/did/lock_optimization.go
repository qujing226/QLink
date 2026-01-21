package did

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/config"
	"github.com/qujing226/QLink/pkg/types"
	"github.com/qujing226/QLink/pkg/utils"
)

// LockManager 锁管理器
type LockManager struct {
	locks         map[string]*sync.RWMutex
	locksMu       sync.RWMutex
	timeout       time.Duration
	lockUsage     map[string]time.Time // 跟踪锁的最后使用时间
	cleanupTicker *time.Ticker         // 定期清理未使用的锁
	stats         *LockStats           // 锁统计信息
	statsMu       sync.RWMutex         // 保护统计信息的锁
}

// NewLockManager 创建锁管理器
func NewLockManager(timeout time.Duration) *LockManager {
	lm := &LockManager{
		locks:     make(map[string]*sync.RWMutex),
		timeout:   timeout,
		lockUsage: make(map[string]time.Time),
		stats: &LockStats{
			TotalLocks:      0,
			ActiveLocks:     0,
			WaitingThreads:  0,
			AverageWaitTime: 0,
			DeadlockCount:   0,
		},
	}

	// 启动定期清理
	lm.cleanupTicker = time.NewTicker(5 * time.Minute)
	go lm.periodicCleanup()

	return lm
}

// GetLock 获取指定资源的锁
func (lm *LockManager) GetLock(resource string) *sync.RWMutex {
	lm.locksMu.RLock()
	lock, exists := lm.locks[resource]
	lm.locksMu.RUnlock()

	if exists {
		// 更新使用时间
		lm.locksMu.Lock()
		lm.lockUsage[resource] = time.Now()
		lm.locksMu.Unlock()
		return lock
	}

	lm.locksMu.Lock()
	defer lm.locksMu.Unlock()

	// 双重检查
	if lock, exists := lm.locks[resource]; exists {
		lm.lockUsage[resource] = time.Now()
		return lock
	}

	lock = &sync.RWMutex{}
	lm.locks[resource] = lock
	lm.lockUsage[resource] = time.Now()

	// 更新统计信息
	lm.statsMu.Lock()
	lm.stats.TotalLocks++
	lm.statsMu.Unlock()

	return lock
}

// periodicCleanup 定期清理未使用的锁
func (lm *LockManager) periodicCleanup() {
	for range lm.cleanupTicker.C {
		lm.CleanupUnusedLocks()
	}
}

// WithReadLock 使用读锁执行函数
func (lm *LockManager) WithReadLock(resource string, fn func() error) error {
	lock := lm.GetLock(resource)

	// 更新统计信息
	lm.statsMu.Lock()
	lm.stats.ActiveLocks++
	lm.statsMu.Unlock()

	start := time.Now()
	done := make(chan struct{})
	var err error

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				err = utils.NewErrorWithDetails(utils.ErrorTypeInternal, "LOCK_PANIC", "锁内部发生panic", fmt.Sprintf("panic: %v", r))
			}
		}()
		lock.RLock()
		defer lock.RUnlock()
		err = fn()
	}()

	select {
	case <-done:
		// 更新统计信息
		lm.statsMu.Lock()
		lm.stats.ActiveLocks--
		duration := time.Since(start)
		lm.stats.AverageWaitTime = (lm.stats.AverageWaitTime + duration) / 2
		lm.statsMu.Unlock()
		return err
	case <-time.After(lm.timeout):
		lm.statsMu.Lock()
		lm.stats.ActiveLocks--
		lm.statsMu.Unlock()
		return utils.NewErrorWithDetails(utils.ErrorTypeTimeout, "LOCK_TIMEOUT", "读锁超时",
			fmt.Sprintf("resource: %s, timeout: %v", resource, lm.timeout))
	}
}

// WithWriteLock 使用写锁执行函数
func (lm *LockManager) WithWriteLock(resource string, fn func() error) error {
	lock := lm.GetLock(resource)

	// 更新统计信息
	lm.statsMu.Lock()
	lm.stats.ActiveLocks++
	lm.statsMu.Unlock()

	start := time.Now()
	done := make(chan struct{})
	var err error

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				err = utils.NewErrorWithDetails(utils.ErrorTypeInternal, "LOCK_PANIC", "锁内部发生panic", fmt.Sprintf("panic: %v", r))
			}
		}()
		lock.Lock()
		defer lock.Unlock()
		err = fn()
	}()

	select {
	case <-done:
		// 更新统计信息
		lm.statsMu.Lock()
		lm.stats.ActiveLocks--
		duration := time.Since(start)
		lm.stats.AverageWaitTime = (lm.stats.AverageWaitTime + duration) / 2
		lm.statsMu.Unlock()
		return err
	case <-time.After(lm.timeout):
		lm.statsMu.Lock()
		lm.stats.ActiveLocks--
		lm.statsMu.Unlock()
		return utils.NewErrorWithDetails(utils.ErrorTypeTimeout, "LOCK_TIMEOUT", "写锁超时",
			fmt.Sprintf("resource: %s, timeout: %v", resource, lm.timeout))
	}
}

// WithContextReadLock 使用上下文和读锁执行函数
func (lm *LockManager) WithContextReadLock(ctx context.Context, resource string, fn func() error) error {
	lock := lm.GetLock(resource)

	done := make(chan struct{})
	var err error

	go func() {
		defer close(done)
		lock.RLock()
		defer lock.RUnlock()
		err = fn()
	}()

	select {
	case <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(lm.timeout):
		return fmt.Errorf("读锁超时: %s", resource)
	}
}

// WithContextWriteLock 使用上下文和写锁执行函数
func (lm *LockManager) WithContextWriteLock(ctx context.Context, resource string, fn func() error) error {
	lock := lm.GetLock(resource)

	done := make(chan struct{})
	var err error

	go func() {
		defer close(done)
		lock.Lock()
		defer lock.Unlock()
		err = fn()
	}()

	select {
	case <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(lm.timeout):
		return fmt.Errorf("写锁超时: %s", resource)
	}
}

// CleanupUnusedLocks 清理未使用的锁
func (lm *LockManager) CleanupUnusedLocks() {
	lm.locksMu.Lock()
	defer lm.locksMu.Unlock()

	now := time.Now()
	cleanupThreshold := 10 * time.Minute // 10分钟未使用的锁将被清理

	for resource, lastUsed := range lm.lockUsage {
		if now.Sub(lastUsed) > cleanupThreshold {
			delete(lm.locks, resource)
			delete(lm.lockUsage, resource)

			// 更新统计信息
			lm.statsMu.Lock()
			lm.stats.TotalLocks--
			lm.statsMu.Unlock()
		}
	}
}

// Stop 停止锁管理器
func (lm *LockManager) Stop() {
	if lm.cleanupTicker != nil {
		lm.cleanupTicker.Stop()
	}
}

// OptimizedDIDRegistry 优化锁使用的DID注册表
type OptimizedDIDRegistry struct {
	*DIDRegistry
	lockManager *LockManager
	metrics     *Metrics
}

// NewOptimizedDIDRegistry 创建优化的DID注册表
func NewOptimizedDIDRegistry(lockTimeout time.Duration, cfg *config.Config, blockchain BlockchainInterface) *OptimizedDIDRegistry {
    return &OptimizedDIDRegistry{
        DIDRegistry: NewDIDRegistry(blockchain),
        lockManager: NewLockManager(lockTimeout),
        metrics:     NewMetrics(),
    }
}

// Register 注册DID（优化版本）
func (r *OptimizedDIDRegistry) Register(req *RegisterRequest) (*types.DIDDocument, error) {
	start := time.Now()
	var doc *types.DIDDocument
	var err error

	// 使用DID特定的锁，减少全局锁竞争
	err = r.lockManager.WithWriteLock(req.DID, func() error {
		doc, err = r.DIDRegistry.Register(req)
		return err
	})

	// 记录指标
	duration := time.Since(start)
	r.metrics.RecordRegister(duration, err == nil)

	return doc, err
}

// Resolve 解析DID（优化版本）
func (r *OptimizedDIDRegistry) Resolve(didStr string) (*types.DIDDocument, error) {
	start := time.Now()
	var doc *types.DIDDocument
	var err error

	// 使用读锁，允许并发读取
	err = r.lockManager.WithReadLock(didStr, func() error {
		doc, err = r.DIDRegistry.Resolve(didStr)
		return err
	})

	// 记录指标
	duration := time.Since(start)
	r.metrics.RecordResolve(duration, err == nil)

	return doc, err
}

// Update 更新DID（优化版本）
func (r *OptimizedDIDRegistry) Update(req *UpdateRequest) (*types.DIDDocument, error) {
	start := time.Now()
	var doc *types.DIDDocument
	var err error

	// 使用写锁
	err = r.lockManager.WithWriteLock(req.DID, func() error {
		doc, err = r.DIDRegistry.Update(req)
		return err
	})

	// 记录指标
	duration := time.Since(start)
	r.metrics.RecordUpdate(duration, err == nil)

	return doc, err
}

// Revoke 撤销DID（优化版本）
func (r *OptimizedDIDRegistry) Revoke(didStr string, proof *types.Proof) error {
	start := time.Now()
	var err error

	// 使用写锁
	err = r.lockManager.WithWriteLock(didStr, func() error {
		return r.DIDRegistry.Revoke(didStr, proof)
	})

	// 记录指标
	duration := time.Since(start)
	r.metrics.RecordRevoke(duration, err == nil)

	return err
}

// BatchResolve 批量解析DID（优化版本）
func (r *OptimizedDIDRegistry) BatchResolve(didStrs []string) ([]*types.DIDDocument, []error) {
	results := make([]*types.DIDDocument, len(didStrs))
	errors := make([]error, len(didStrs))

	// 使用goroutine并发解析
	var wg sync.WaitGroup
	for i, didStr := range didStrs {
		wg.Add(1)
		go func(index int, did string) {
			defer wg.Done()
			doc, err := r.Resolve(did)
			results[index] = doc
			errors[index] = err
		}(i, didStr)
	}

	wg.Wait()
	return results, errors
}

// GetMetrics 获取指标
func (r *OptimizedDIDRegistry) GetMetrics() *Metrics {
	return r.metrics
}

// DeadlockDetector 死锁检测器
type DeadlockDetector struct {
	lockGraph   map[string][]string // 锁依赖图
	graphMu     sync.RWMutex
	timeout     time.Duration
	checkTicker *time.Ticker
	stopChan    chan struct{}
}

// NewDeadlockDetector 创建死锁检测器
func NewDeadlockDetector(checkInterval time.Duration) *DeadlockDetector {
	return &DeadlockDetector{
		lockGraph:   make(map[string][]string),
		timeout:     30 * time.Second,
		checkTicker: time.NewTicker(checkInterval),
		stopChan:    make(chan struct{}),
	}
}

// Start 启动死锁检测
func (dd *DeadlockDetector) Start() {
	go func() {
		for {
			select {
			case <-dd.checkTicker.C:
				dd.detectDeadlock()
			case <-dd.stopChan:
				return
			}
		}
	}()
}

// Stop 停止死锁检测
func (dd *DeadlockDetector) Stop() {
	dd.checkTicker.Stop()
	close(dd.stopChan)
}

// AddLockDependency 添加锁依赖关系
func (dd *DeadlockDetector) AddLockDependency(from, to string) {
	dd.graphMu.Lock()
	defer dd.graphMu.Unlock()

	if dd.lockGraph[from] == nil {
		dd.lockGraph[from] = make([]string, 0)
	}

	// 避免重复添加
	for _, dep := range dd.lockGraph[from] {
		if dep == to {
			return
		}
	}

	dd.lockGraph[from] = append(dd.lockGraph[from], to)
}

// RemoveLockDependency 移除锁依赖关系
func (dd *DeadlockDetector) RemoveLockDependency(from, to string) {
	dd.graphMu.Lock()
	defer dd.graphMu.Unlock()

	deps := dd.lockGraph[from]
	for i, dep := range deps {
		if dep == to {
			dd.lockGraph[from] = append(deps[:i], deps[i+1:]...)
			break
		}
	}
}

// detectDeadlock 检测死锁
func (dd *DeadlockDetector) detectDeadlock() {
	dd.graphMu.RLock()
	defer dd.graphMu.RUnlock()

	// 使用DFS检测环
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range dd.lockGraph {
		if !visited[node] {
			if dd.hasCycleDFS(node, visited, recStack) {
				// 检测到死锁，记录日志或采取其他措施
				fmt.Printf("检测到潜在死锁，涉及节点: %s\n", node)
			}
		}
	}
}

// hasCycleDFS 使用DFS检测环
func (dd *DeadlockDetector) hasCycleDFS(node string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range dd.lockGraph[node] {
		if !visited[neighbor] {
			if dd.hasCycleDFS(neighbor, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// LockStats 锁统计信息
type LockStats struct {
	TotalLocks      int           `json:"total_locks"`
	ActiveLocks     int           `json:"active_locks"`
	WaitingThreads  int           `json:"waiting_threads"`
	AverageWaitTime time.Duration `json:"average_wait_time"`
	DeadlockCount   int           `json:"deadlock_count"`
}

// GetLockStats 获取锁统计信息
func (lm *LockManager) GetLockStats() *LockStats {
	lm.locksMu.RLock()
	defer lm.locksMu.RUnlock()

	return &LockStats{
		TotalLocks:      len(lm.locks),
		ActiveLocks:     len(lm.locks), // 简化实现
		WaitingThreads:  0,             // 需要更复杂的跟踪
		AverageWaitTime: 0,             // 需要更复杂的跟踪
		DeadlockCount:   0,             // 需要与死锁检测器集成
	}
}
