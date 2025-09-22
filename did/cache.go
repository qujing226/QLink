package did

import (
	"fmt"
	"sync"
	"time"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      interface{} `json:"data"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// IsExpired 检查缓存是否过期
func (ce *CacheEntry) IsExpired() bool {
	return time.Now().After(ce.ExpiresAt)
}

// DIDCache DID缓存接口
type DIDCache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
	Clear()
	Size() int
	Stats() CacheStats
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	Size        int   `json:"size"`
	HitRate     float64 `json:"hit_rate"`
	LastCleanup time.Time `json:"last_cleanup"`
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	mu          sync.RWMutex
	data        map[string]*CacheEntry
	maxSize     int
	defaultTTL  time.Duration
	cleanupTick time.Duration
	stats       CacheStats
	stopCleanup chan struct{}
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int, defaultTTL, cleanupTick time.Duration) *MemoryCache {
	cache := &MemoryCache{
		data:        make(map[string]*CacheEntry),
		maxSize:     maxSize,
		defaultTTL:  defaultTTL,
		cleanupTick: cleanupTick,
		stopCleanup: make(chan struct{}),
	}

	// 启动清理协程
	go cache.startCleanup()

	return cache
}

// Get 获取缓存值
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.data[key]
	if !exists {
		mc.stats.Misses++
		return nil, false
	}

	if entry.IsExpired() {
		mc.stats.Misses++
		// 延迟删除过期条目
		go func() {
			mc.mu.Lock()
			delete(mc.data, key)
			mc.mu.Unlock()
		}()
		return nil, false
	}

	mc.stats.Hits++
	return entry.Data, true
}

// Set 设置缓存值
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 如果最大容量为0或负数，不存储任何内容
	if mc.maxSize <= 0 {
		return
	}

	if ttl == 0 {
		ttl = mc.defaultTTL
	}

	// 如果缓存已满，删除最旧的条目
	if len(mc.data) >= mc.maxSize {
		mc.evictOldest()
	}

	mc.data[key] = &CacheEntry{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
}

// Delete 删除缓存值
func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	delete(mc.data, key)
}

// Clear 清空缓存
func (mc *MemoryCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.data = make(map[string]*CacheEntry)
}

// Size 获取缓存大小
func (mc *MemoryCache) Size() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.data)
}

// Stats 获取缓存统计信息
func (mc *MemoryCache) Stats() CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	stats := mc.stats
	stats.Size = len(mc.data)
	
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}
	
	return stats
}

// Close 关闭缓存
func (mc *MemoryCache) Close() {
	close(mc.stopCleanup)
}

// startCleanup 启动清理协程
func (mc *MemoryCache) startCleanup() {
	ticker := time.NewTicker(mc.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.cleanup()
		case <-mc.stopCleanup:
			return
		}
	}
}

// cleanup 清理过期条目
func (mc *MemoryCache) cleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	for key, entry := range mc.data {
		if entry.IsExpired() {
			delete(mc.data, key)
		}
	}
	mc.stats.LastCleanup = now
}

// evictOldest 删除最旧的条目
func (mc *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range mc.data {
		if oldestKey == "" || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(mc.data, oldestKey)
	}
}

// CachedDIDResolver 带缓存的DID解析器
type CachedDIDResolver struct {
	resolver *DIDResolver
	cache    DIDCache
	cacheTTL time.Duration
}

// NewCachedDIDResolver 创建带缓存的DID解析器
func NewCachedDIDResolver(resolver *DIDResolver, cache DIDCache, cacheTTL time.Duration) *CachedDIDResolver {
	return &CachedDIDResolver{
		resolver: resolver,
		cache:    cache,
		cacheTTL: cacheTTL,
	}
}

// Resolve 解析DID（带缓存）
func (cr *CachedDIDResolver) Resolve(didStr string) (*ResolutionResult, error) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("did:resolve:%s", didStr)
	if cached, found := cr.cache.Get(cacheKey); found {
		if result, ok := cached.(*ResolutionResult); ok {
			return result, nil
		}
	}

	// 缓存未命中，从原始解析器获取
	result, err := cr.resolver.Resolve(didStr)
	if err != nil {
		return nil, err
	}

	// 只缓存成功的解析结果
	if result.DIDDocument != nil {
		cr.cache.Set(cacheKey, result, cr.cacheTTL)
	}

	return result, nil
}

// ResolveVerificationMethod 解析验证方法（带缓存）
func (cr *CachedDIDResolver) ResolveVerificationMethod(didURL string) (*VerificationMethod, error) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("did:vm:%s", didURL)
	if cached, found := cr.cache.Get(cacheKey); found {
		if vm, ok := cached.(*VerificationMethod); ok {
			return vm, nil
		}
	}

	// 缓存未命中，从原始解析器获取
	vm, err := cr.resolver.ResolveVerificationMethod(didURL)
	if err != nil {
		return nil, err
	}

	// 缓存验证方法
	cr.cache.Set(cacheKey, vm, cr.cacheTTL)

	return vm, nil
}

// InvalidateCache 使缓存失效
func (cr *CachedDIDResolver) InvalidateCache(didStr string) {
	// 删除DID解析缓存
	resolveKey := fmt.Sprintf("did:resolve:%s", didStr)
	cr.cache.Delete(resolveKey)

	// 删除相关的验证方法缓存
	// 这里简化处理，实际应该维护DID与验证方法的映射关系
	vmKey := fmt.Sprintf("did:vm:%s#", didStr)
	cr.cache.Delete(vmKey)
}

// GetCacheStats 获取缓存统计信息
func (cr *CachedDIDResolver) GetCacheStats() CacheStats {
	return cr.cache.Stats()
}

// CachedDIDRegistry 带缓存的DID注册表
type CachedDIDRegistry struct {
	registry *DIDRegistry
	cache    DIDCache
	cacheTTL time.Duration
}

// NewCachedDIDRegistry 创建带缓存的DID注册表
func NewCachedDIDRegistry(registry *DIDRegistry, cache DIDCache, cacheTTL time.Duration) *CachedDIDRegistry {
	return &CachedDIDRegistry{
		registry: registry,
		cache:    cache,
		cacheTTL: cacheTTL,
	}
}

// Register 注册DID（更新缓存）
func (cr *CachedDIDRegistry) Register(req *RegisterRequest) (*DIDDocument, error) {
	doc, err := cr.registry.Register(req)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	cacheKey := fmt.Sprintf("did:doc:%s", req.DID)
	cr.cache.Set(cacheKey, doc, cr.cacheTTL)

	return doc, nil
}

// Resolve 解析DID（带缓存）
func (cr *CachedDIDRegistry) Resolve(didStr string) (*DIDDocument, error) {
	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("did:doc:%s", didStr)
	if cached, found := cr.cache.Get(cacheKey); found {
		if doc, ok := cached.(*DIDDocument); ok {
			return doc, nil
		}
	}

	// 缓存未命中，从原始注册表获取
	doc, err := cr.registry.Resolve(didStr)
	if err != nil {
		return nil, err
	}

	// 缓存DID文档
	cr.cache.Set(cacheKey, doc, cr.cacheTTL)

	return doc, nil
}

// Update 更新DID（更新缓存）
func (cr *CachedDIDRegistry) Update(req *UpdateRequest) (*DIDDocument, error) {
	doc, err := cr.registry.Update(req)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	cacheKey := fmt.Sprintf("did:doc:%s", req.DID)
	cr.cache.Set(cacheKey, doc, cr.cacheTTL)

	return doc, nil
}

// Revoke 撤销DID（删除缓存）
func (cr *CachedDIDRegistry) Revoke(didStr string, proof *Proof) error {
	err := cr.registry.Revoke(didStr, proof)
	if err != nil {
		return err
	}

	// 删除缓存
	cacheKey := fmt.Sprintf("did:doc:%s", didStr)
	cr.cache.Delete(cacheKey)

	return nil
}

// List 列出所有DID
func (cr *CachedDIDRegistry) List() ([]*DIDDocument, error) {
	return cr.registry.List()
}

// GetCacheStats 获取缓存统计信息
func (cr *CachedDIDRegistry) GetCacheStats() CacheStats {
	return cr.cache.Stats()
}