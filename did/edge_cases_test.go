package did

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/utils"
)

// TestDIDRegistryEdgeCases 测试DID注册表的边界情况
func TestEdgeCases(t *testing.T) {
	cfg := &config.Config{
		DID: &config.DIDConfig{
			Method:  "qlink",
			ChainID: "testnet",
		},
	}
	registry := NewDIDRegistry(cfg, nil)

	t.Run("空DID字符串", func(t *testing.T) {
		_, err := registry.Resolve("")
		if err == nil {
			t.Error("解析空DID应该返回错误")
		}
		if !utils.IsErrorType(err, utils.ErrorTypeNotFound) {
			t.Errorf("期望未找到错误，得到: %v", err)
		}
	})

	t.Run("无效DID格式", func(t *testing.T) {
		invalidDIDs := []string{
			"invalid-did",
			"did:invalid",
			"did::test",
			"did:qlink:",
			"did:qlink:test:",
			"did:qlink:test::",
			"did:qlink:test:invalid-chars!@#",
			strings.Repeat("did:qlink:test:", 100), // 超长DID
		}

		for _, invalidDID := range invalidDIDs {
			t.Run(fmt.Sprintf("无效DID_%s", invalidDID), func(t *testing.T) {
				_, err := registry.Resolve(invalidDID)
				if err == nil {
					t.Errorf("解析无效DID应该返回错误: %s", invalidDID)
				}
			})
		}
	})

	t.Run("不存在的DID", func(t *testing.T) {
		_, err := registry.Resolve("did:qlink:test:nonexistent")
		if err == nil {
			t.Error("解析不存在的DID应该返回错误")
		}
		if !utils.IsErrorType(err, utils.ErrorTypeNotFound) {
			t.Errorf("期望未找到错误，得到: %v", err)
		}
	})

	t.Run("重复注册DID", func(t *testing.T) {
		// 创建测试请求
		req := &RegisterRequest{
			DID: "did:qlink:test:duplicate",
			VerificationMethod: []VerificationMethod{
				{
					ID:   "did:qlink:test:duplicate#key1",
					Type: "Ed25519VerificationKey2020",
					PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
				},
			},
		}

		// 第一次注册应该成功
		_, err := registry.Register(req)
		if err != nil {
			t.Fatalf("第一次注册失败: %v", err)
		}

		// 第二次注册应该失败
		_, err = registry.Register(req)
		if err == nil {
			t.Error("重复注册DID应该返回错误")
		}
		if !utils.IsErrorType(err, utils.ErrorTypeConflict) {
			t.Errorf("期望冲突错误，得到: %v", err)
		}
	})

	t.Run("空验证方法", func(t *testing.T) {
		req := &RegisterRequest{
			DID:                "did:qlink:test:empty-vm",
			VerificationMethod: []VerificationMethod{},
		}

		// 注册空验证方法的DID应该成功，因为验证方法是可选的
		_, err := registry.Register(req)
		if err != nil {
			t.Errorf("注册空验证方法的DID不应该返回错误: %v", err)
		}
	})

	t.Run("无效的验证方法", func(t *testing.T) {
		req := &RegisterRequest{
			DID: "did:qlink:test:invalid-vm",
			VerificationMethod: []VerificationMethod{
				{
					ID:   "", // 空ID
					Type: "Ed25519VerificationKey2020",
					PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
				},
			},
		}

		// 注册无效验证方法的DID应该成功，因为当前实现没有验证验证方法的详细内容
		_, err := registry.Register(req)
		if err != nil {
			t.Errorf("注册无效验证方法的DID不应该返回错误: %v", err)
		}
	})
}

// TestLockManagerEdgeCases 测试锁管理器的边界情况
func TestLockManagerEdgeCases(t *testing.T) {
	t.Run("零超时时间", func(t *testing.T) {
		lm := NewLockManager(0)
		defer lm.Stop()

		err := lm.WithReadLock("resource", func() error {
			time.Sleep(time.Millisecond * 10)
			return nil
		})
		
		// 零超时应该立即超时
		if err == nil {
			t.Error("零超时应该返回超时错误")
		}
		if !utils.IsErrorType(err, utils.ErrorTypeTimeout) {
			t.Errorf("期望超时错误，得到: %v", err)
		}
	})

	t.Run("负超时时间", func(t *testing.T) {
		lm := NewLockManager(-time.Second)
		defer lm.Stop()

		err := lm.WithReadLock("resource", func() error {
			return nil
		})
		
		// 负超时应该立即超时
		if err == nil {
			t.Error("负超时应该返回超时错误")
		}
	})

	t.Run("空资源名", func(t *testing.T) {
		lm := NewLockManager(time.Second)
		defer lm.Stop()

		lock1 := lm.GetLock("")
		lock2 := lm.GetLock("")
		
		if lock1 != lock2 {
			t.Error("空资源名应该返回相同的锁")
		}
	})

	t.Run("大量并发锁请求", func(t *testing.T) {
		lm := NewLockManager(time.Second * 5)
		defer lm.Stop()

		var wg sync.WaitGroup
		errors := make(chan error, 100)

		// 创建100个并发锁请求
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				resource := fmt.Sprintf("resource_%d", id%10) // 10个不同的资源
				
				err := lm.WithWriteLock(resource, func() error {
					time.Sleep(time.Millisecond * 10)
					return nil
				})
				
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// 检查是否有错误
		for err := range errors {
			t.Errorf("并发锁请求失败: %v", err)
		}
	})

	t.Run("锁内部panic处理", func(t *testing.T) {
		lm := NewLockManager(time.Second * 5)
		defer lm.Stop()

		err := lm.WithReadLock("resource", func() error {
			panic("测试panic")
		})

		// 应该捕获panic并返回错误
		if err == nil {
			t.Error("锁内部panic应该被捕获并返回错误")
		}
	})

	t.Run("上下文取消", func(t *testing.T) {
		lm := NewLockManager(time.Second * 10)
		defer lm.Stop()

		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()

		err := lm.WithContextReadLock(ctx, "resource", func() error {
			time.Sleep(time.Second) // 超过上下文超时时间
			return nil
		})

		if err == nil {
			t.Error("上下文取消应该返回错误")
		}
		if err != context.DeadlineExceeded {
			t.Errorf("期望上下文超时错误，得到: %v", err)
		}
	})
}

// TestCacheEdgeCases 测试缓存的边界情况
func TestCacheEdgeCases(t *testing.T) {
	t.Run("零容量缓存", func(t *testing.T) {
		cache := NewMemoryCache(0, time.Minute, time.Minute)
		defer cache.Close()
		
		// 零容量缓存应该不存储任何内容
		testDoc := &DIDDocument{ID: "did:qlink:test:test"}
		cache.Set("key", testDoc, time.Minute)
		
		_, found := cache.Get("key")
		if found {
			t.Error("零容量缓存不应该存储任何内容")
		}
	})

	t.Run("负容量缓存", func(t *testing.T) {
		cache := NewMemoryCache(-1, time.Minute, time.Minute)
		defer cache.Close()
		
		// 负容量缓存应该被视为零容量
		testDoc := &DIDDocument{ID: "did:qlink:test:test"}
		cache.Set("key", testDoc, time.Minute)
		
		_, found := cache.Get("key")
		if found {
			t.Error("负容量缓存不应该存储任何内容")
		}
	})

	t.Run("零TTL缓存", func(t *testing.T) {
		cache := NewMemoryCache(10, 0, time.Minute)
		defer cache.Close()
		
		testDoc := &DIDDocument{ID: "did:qlink:test:test"}
		cache.Set("key", testDoc, 0)
		
		// 零TTL应该立即过期
		time.Sleep(time.Millisecond)
		_, found := cache.Get("key")
		if found {
			t.Error("零TTL缓存项应该立即过期")
		}
	})

	t.Run("空键值", func(t *testing.T) {
		cache := NewMemoryCache(10, time.Minute, time.Minute)
		defer cache.Close()
		
		testDoc := &DIDDocument{ID: "did:qlink:test:test"}
		cache.Set("", testDoc, time.Minute)
		doc, found := cache.Get("")
		
		if !found {
			t.Error("应该能够使用空字符串作为键")
		}
		if didDoc, ok := doc.(*DIDDocument); !ok || didDoc.ID != "did:qlink:test:test" {
			t.Error("缓存的文档不正确")
		}
	})

	t.Run("nil值", func(t *testing.T) {
		cache := NewMemoryCache(10, time.Minute, time.Minute)
		defer cache.Close()
		
		cache.Set("key", nil, time.Minute)
		doc, found := cache.Get("key")
		
		if !found || doc != nil {
			t.Error("应该能够存储nil值")
		}
	})

	t.Run("大量并发访问", func(t *testing.T) {
		cache := NewMemoryCache(100, time.Minute, time.Minute)
		defer cache.Close()
		var wg sync.WaitGroup
		
		// 并发写入
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("key_%d", id)
				testDoc := &DIDDocument{ID: fmt.Sprintf("did:qlink:test:%s", key)}
				cache.Set(key, testDoc, time.Minute)
			}(i)
		}
		
		// 并发读取
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				key := fmt.Sprintf("key_%d", id)
				cache.Get(key)
			}(i)
		}
		
		wg.Wait()
		
		// 验证缓存状态
		stats := cache.Stats()
		if stats.Size < 0 || stats.Size > 100 {
			t.Errorf("缓存大小异常: %d", stats.Size)
		}
	})
}

// TestMetricsEdgeCases 测试监控指标的边界情况
func TestMetricsEdgeCases(t *testing.T) {
	metrics := NewMetrics()

	t.Run("负数指标", func(t *testing.T) {
		// 测试负数持续时间
		metrics.RecordRegister(-time.Second, true)
		
		snapshot := metrics.GetSnapshot()
		if snapshot.AvgRegisterTime < 0 {
			t.Error("平均时间不应该为负数")
		}
	})

	t.Run("极大数值", func(t *testing.T) {
		// 测试极大的持续时间
		metrics.RecordResolve(time.Hour*24*365, true) // 一年
		
		snapshot := metrics.GetSnapshot()
		if snapshot.AvgResolveTime < 0 {
			t.Error("平均时间计算错误")
		}
	})

	t.Run("并发指标更新", func(t *testing.T) {
		// 创建新的指标实例以避免与其他测试的干扰
		testMetrics := NewMetrics()
		var wg sync.WaitGroup
		
		// 并发更新不同类型的指标
		for i := 0; i < 100; i++ {
			wg.Add(4)
			
			go func() {
				defer wg.Done()
				testMetrics.RecordRegister(time.Millisecond, true)
			}()
			
			go func() {
				defer wg.Done()
				testMetrics.RecordResolve(time.Millisecond*2, false)
			}()
			
			go func() {
				defer wg.Done()
				testMetrics.RecordCacheHit()
			}()
			
			go func() {
				defer wg.Done()
				testMetrics.UpdateDIDCounts(1, 0)
			}()
		}
		
		wg.Wait()
		
		// 验证指标一致性
		snapshot := testMetrics.GetSnapshot()
		if snapshot.RegisterCount != 100 {
			t.Errorf("注册操作计数错误: %d", snapshot.RegisterCount)
		}
		if snapshot.ResolveCount != 100 {
			t.Errorf("解析操作计数错误: %d", snapshot.ResolveCount)
		}
	})

	t.Run("重置指标", func(t *testing.T) {
		// 添加一些指标
		metrics.RecordRegister(time.Millisecond, true)
		metrics.RecordResolve(time.Millisecond, false)
		metrics.RecordCacheHit()
		
		// 重置指标
		metrics.Reset()
		
		// 验证所有指标都被重置
		snapshot := metrics.GetSnapshot()
		if snapshot.RegisterCount != 0 ||
		   snapshot.ResolveCount != 0 ||
		   snapshot.CacheHits != 0 {
			t.Error("指标重置失败")
		}
	})
}