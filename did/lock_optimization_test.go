package did

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/qujing226/QLink/did/config"
)

func TestLockManager(t *testing.T) {
	lm := NewLockManager(time.Second * 5)
	
	// 测试获取锁
	lock1 := lm.GetLock("resource1")
	lock2 := lm.GetLock("resource1")
	
	if lock1 != lock2 {
		t.Error("同一资源应该返回相同的锁")
	}
	
	// 测试不同资源的锁
	lock3 := lm.GetLock("resource2")
	if lock1 == lock3 {
		t.Error("不同资源应该返回不同的锁")
	}
}

func TestLockManagerWithReadLock(t *testing.T) {
	lm := NewLockManager(time.Second * 5)
	counter := 0
	
	var wg sync.WaitGroup
	
	// 多个读锁应该可以并发执行
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := lm.WithReadLock("resource", func() error {
				counter++
				time.Sleep(time.Millisecond * 100)
				return nil
			})
			if err != nil {
				t.Errorf("读锁执行失败: %v", err)
			}
		}()
	}
	
	wg.Wait()
	
	if counter != 5 {
		t.Errorf("期望计数器为5，实际为%d", counter)
	}
}

func TestLockManagerWithWriteLock(t *testing.T) {
	lm := NewLockManager(time.Second * 5)
	counter := 0
	
	var wg sync.WaitGroup
	
	// 写锁应该串行执行
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := lm.WithWriteLock("resource", func() error {
				temp := counter
				time.Sleep(time.Millisecond * 50)
				counter = temp + 1
				return nil
			})
			if err != nil {
				t.Errorf("写锁执行失败: %v", err)
			}
		}()
	}
	
	wg.Wait()
	
	if counter != 3 {
		t.Errorf("期望计数器为3，实际为%d", counter)
	}
}

func TestLockManagerWithContext(t *testing.T) {
	lm := NewLockManager(time.Second * 5)
	
	// 测试上下文取消
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	
	err := lm.WithContextReadLock(ctx, "resource", func() error {
		time.Sleep(time.Millisecond * 200) // 超过上下文超时时间
		return nil
	})
	
	if err == nil {
		t.Error("期望上下文超时错误")
	}
}

func TestOptimizedDIDRegistry(t *testing.T) {
	cfg := &config.Config{}
	registry := NewOptimizedDIDRegistry(time.Second*5, cfg, nil)
	
	// 测试注册
	req := &RegisterRequest{
		DID: "did:qlink:test123",
		VerificationMethod: []VerificationMethod{
			{
				ID:                 "did:qlink:test123#key1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         "did:qlink:test123",
				PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
			},
		},
	}
	
	doc, err := registry.Register(req)
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	
	if doc.ID != req.DID {
		t.Errorf("期望DID为%s，实际为%s", req.DID, doc.ID)
	}
	
	// 测试解析
	resolvedDoc, err := registry.Resolve(req.DID)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	
	if resolvedDoc.ID != req.DID {
		t.Errorf("期望解析的DID为%s，实际为%s", req.DID, resolvedDoc.ID)
	}
}

func TestOptimizedDIDRegistryBatchResolve(t *testing.T) {
	cfg := &config.Config{}
	registry := NewOptimizedDIDRegistry(time.Second*5, cfg, nil)
	
	// 注册多个DID
	dids := []string{"did:qlink:test1", "did:qlink:test2", "did:qlink:test3"}
	
	for _, did := range dids {
		req := &RegisterRequest{
			DID: did,
			VerificationMethod: []VerificationMethod{
				{
					ID:                 did + "#key1",
					Type:               "Ed25519VerificationKey2020",
					Controller:         did,
					PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
				},
			},
		}
		
		_, err := registry.Register(req)
		if err != nil {
			t.Fatalf("注册DID %s失败: %v", did, err)
		}
	}
	
	// 批量解析
	docs, errs := registry.BatchResolve(dids)
	
	if len(docs) != len(dids) {
		t.Errorf("期望返回%d个文档，实际返回%d个", len(dids), len(docs))
	}
	
	for i, doc := range docs {
		if errs[i] != nil {
			t.Errorf("解析DID %s失败: %v", dids[i], errs[i])
		}
		if doc.ID != dids[i] {
			t.Errorf("期望DID为%s，实际为%s", dids[i], doc.ID)
		}
	}
}

func TestDeadlockDetector(t *testing.T) {
	detector := NewDeadlockDetector(time.Millisecond * 100)
	
	// 添加锁依赖
	detector.AddLockDependency("A", "B")
	detector.AddLockDependency("B", "C")
	detector.AddLockDependency("C", "A") // 形成环
	
	// 启动检测器
	detector.Start()
	defer detector.Stop()
	
	// 等待检测
	time.Sleep(time.Millisecond * 200)
	
	// 移除依赖
	detector.RemoveLockDependency("C", "A")
	
	time.Sleep(time.Millisecond * 200)
}

func TestLockStats(t *testing.T) {
	lm := NewLockManager(time.Second * 5)
	
	// 获取一些锁
	lm.GetLock("resource1")
	lm.GetLock("resource2")
	lm.GetLock("resource3")
	
	stats := lm.GetLockStats()
	
	if stats.TotalLocks != 3 {
		t.Errorf("期望总锁数为3，实际为%d", stats.TotalLocks)
	}
}

// 基准测试
func BenchmarkLockManagerGetLock(b *testing.B) {
	lm := NewLockManager(time.Second * 5)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lm.GetLock("resource")
		}
	})
}

func BenchmarkLockManagerWithReadLock(b *testing.B) {
	lm := NewLockManager(time.Second * 5)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lm.WithReadLock("resource", func() error {
				return nil
			})
		}
	})
}

func BenchmarkOptimizedDIDRegistryResolve(b *testing.B) {
	cfg := &config.Config{}
	registry := NewOptimizedDIDRegistry(time.Second*5, cfg, nil)
	
	// 预先注册一个DID
	req := &RegisterRequest{
		DID: "did:qlink:benchmark",
		VerificationMethod: []VerificationMethod{
			{
				ID:                 "did:qlink:benchmark#key1",
				Type:               "Ed25519VerificationKey2020",
				Controller:         "did:qlink:benchmark",
				PublicKeyMultibase: "z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK",
			},
		},
	}
	registry.Register(req)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			registry.Resolve("did:qlink:benchmark")
		}
	})
}