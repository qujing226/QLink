package did

import (
	"fmt"
	"testing"
	"time"

	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/crypto"
)

func TestMemoryCache(t *testing.T) {
	// 创建内存缓存
	cache := NewMemoryCache(100, 5*time.Minute, 1*time.Minute)
	defer cache.Close()

	// 测试设置和获取
	testKey := "test-key"
	testValue := "test-value"
	
	cache.Set(testKey, testValue, 0)
	
	value, found := cache.Get(testKey)
	if !found {
		t.Error("缓存值应该存在")
	}
	
	if value != testValue {
		t.Errorf("期望值: %s, 实际值: %s", testValue, value)
	}

	// 测试缓存大小
	if cache.Size() != 1 {
		t.Errorf("期望缓存大小: 1, 实际大小: %d", cache.Size())
	}

	// 测试删除
	cache.Delete(testKey)
	_, found = cache.Get(testKey)
	if found {
		t.Error("删除后缓存值不应该存在")
	}
}

func TestCacheExpiration(t *testing.T) {
	// 创建短TTL的缓存
	cache := NewMemoryCache(100, 100*time.Millisecond, 50*time.Millisecond)
	defer cache.Close()

	testKey := "expire-key"
	testValue := "expire-value"
	
	// 设置短TTL的缓存
	cache.Set(testKey, testValue, 100*time.Millisecond)
	
	// 立即获取应该成功
	value, found := cache.Get(testKey)
	if !found || value != testValue {
		t.Error("缓存值应该立即可用")
	}

	// 等待过期
	time.Sleep(150 * time.Millisecond)
	
	// 过期后获取应该失败
	_, found = cache.Get(testKey)
	if found {
		t.Error("过期后缓存值不应该存在")
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewMemoryCache(100, 5*time.Minute, 1*time.Minute)
	defer cache.Close()

	// 初始统计
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("初始统计应该为0")
	}

	// 设置值
	cache.Set("key1", "value1", 0)
	
	// 命中
	cache.Get("key1")
	
	// 未命中
	cache.Get("nonexistent")
	
	// 检查统计
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("期望命中次数: 1, 实际: %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("期望未命中次数: 1, 实际: %d", stats.Misses)
	}
	if stats.HitRate != 0.5 {
		t.Errorf("期望命中率: 0.5, 实际: %f", stats.HitRate)
	}
}

func TestCachedDIDResolver(t *testing.T) {
	// 创建测试环境
	cfg := &config.Config{}
	registry := NewDIDRegistry(cfg, nil)
	resolver := NewDIDResolver(cfg, registry, nil)
	cache := NewMemoryCache(100, 5*time.Minute, 1*time.Minute)
	defer cache.Close()

	cachedResolver := NewCachedDIDResolver(resolver, cache, 5*time.Minute)

	// 生成测试DID
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	didStr, err := crypto.GenerateDIDFromKeyPair(keyPair)
	if err != nil {
		t.Fatalf("生成DID失败: %v", err)
	}

	// 创建验证方法
	jwk, err := keyPair.ToJWK()
	if err != nil {
		t.Fatalf("转换JWK失败: %v", err)
	}

	// 注册DID
	req := &RegisterRequest{
		DID: didStr,
		VerificationMethod: []VerificationMethod{
			{
				ID:           didStr + "#key-1",
				Type:         "JsonWebKey2020",
				Controller:   didStr,
				PublicKeyJwk: jwk,
			},
		},
	}

	_, err = registry.Register(req)
	if err != nil {
		t.Fatalf("注册DID失败: %v", err)
	}

	// 第一次解析（应该从原始解析器获取）
	result1, err := cachedResolver.Resolve(didStr)
	if err != nil {
		t.Fatalf("解析DID失败: %v", err)
	}

	if result1.DIDDocument == nil {
		t.Error("DID文档不应该为空")
	}

	// 第二次解析（应该从缓存获取）
	result2, err := cachedResolver.Resolve(didStr)
	if err != nil {
		t.Fatalf("第二次解析DID失败: %v", err)
	}

	if result2.DIDDocument == nil {
		t.Error("缓存的DID文档不应该为空")
	}

	// 验证缓存统计
	stats := cachedResolver.GetCacheStats()
	if stats.Hits == 0 {
		t.Error("应该有缓存命中")
	}
}

func TestCachedDIDRegistry(t *testing.T) {
	// 创建测试环境
	cfg := &config.Config{}
	registry := NewDIDRegistry(cfg, nil)
	cache := NewMemoryCache(100, 5*time.Minute, 1*time.Minute)
	defer cache.Close()

	cachedRegistry := NewCachedDIDRegistry(registry, cache, 5*time.Minute)

	// 生成测试DID
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	didStr, err := crypto.GenerateDIDFromKeyPair(keyPair)
	if err != nil {
		t.Fatalf("生成DID失败: %v", err)
	}

	// 创建验证方法
	jwk, err := keyPair.ToJWK()
	if err != nil {
		t.Fatalf("转换JWK失败: %v", err)
	}

	// 注册DID（应该更新缓存）
	req := &RegisterRequest{
		DID: didStr,
		VerificationMethod: []VerificationMethod{
			{
				ID:           didStr + "#key-1",
				Type:         "JsonWebKey2020",
				Controller:   didStr,
				PublicKeyJwk: jwk,
			},
		},
	}

	doc, err := cachedRegistry.Register(req)
	if err != nil {
		t.Fatalf("注册DID失败: %v", err)
	}

	if doc == nil {
		t.Error("注册的DID文档不应该为空")
	}

	// 解析DID（应该从缓存获取）
	resolvedDoc, err := cachedRegistry.Resolve(didStr)
	if err != nil {
		t.Fatalf("解析DID失败: %v", err)
	}

	if resolvedDoc.ID != didStr {
		t.Errorf("期望DID: %s, 实际: %s", didStr, resolvedDoc.ID)
	}

	// 验证缓存统计
	stats := cachedRegistry.GetCacheStats()
	if stats.Hits == 0 {
		t.Error("应该有缓存命中")
	}

	// 测试撤销DID（应该删除缓存）
	err = cachedRegistry.Revoke(didStr, nil)
	if err != nil {
		t.Fatalf("撤销DID失败: %v", err)
	}

	// 再次解析应该失败（因为已撤销）
	_, err = cachedRegistry.Resolve(didStr)
	if err == nil {
		t.Error("解析已撤销的DID应该失败")
	}
}

func TestCacheEviction(t *testing.T) {
	// 创建小容量缓存
	cache := NewMemoryCache(2, 5*time.Minute, 1*time.Minute)
	defer cache.Close()

	// 添加超过容量的条目
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0) // 应该触发驱逐

	// 验证缓存大小
	if cache.Size() != 2 {
		t.Errorf("期望缓存大小: 2, 实际: %d", cache.Size())
	}

	// 最旧的条目应该被驱逐
	_, found := cache.Get("key1")
	if found {
		t.Error("最旧的条目应该被驱逐")
	}

	// 较新的条目应该存在
	_, found = cache.Get("key2")
	if !found {
		t.Error("较新的条目应该存在")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("最新的条目应该存在")
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewMemoryCache(1000, 5*time.Minute, 1*time.Minute)
	defer cache.Close()

	// 预填充缓存
	for i := 0; i < 100; i++ {
		cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(fmt.Sprintf("key%d", i%100))
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewMemoryCache(1000, 5*time.Minute, 1*time.Minute)
	defer cache.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 0)
	}
}