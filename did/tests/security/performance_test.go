package security

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/crypto"
)

// BenchmarkDIDCreation 基准测试DID创建性能
func BenchmarkDIDCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 生成密钥对
		keyPair, err := crypto.GenerateHybridKeyPair()
		if err != nil {
			b.Fatalf("生成密钥对失败: %v", err)
		}

		// 创建DID
		_, err = crypto.GenerateDIDFromKeyPair(keyPair)
		if err != nil {
			b.Fatalf("生成DID失败: %v", err)
		}
	}
}

// BenchmarkDIDResolution 基准测试DID解析性能
func BenchmarkDIDResolution(b *testing.B) {
	// 创建测试配置
	cfg := &config.Config{}
	registry := did.NewDIDRegistry(cfg, nil)

	// 预先创建一些DID用于测试
	testDIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		keyPair, err := crypto.GenerateHybridKeyPair()
		if err != nil {
			b.Fatalf("生成密钥对失败: %v", err)
		}

		didStr, err := crypto.GenerateDIDFromKeyPair(keyPair)
		if err != nil {
			b.Fatalf("生成DID失败: %v", err)
		}
		testDIDs[i] = didStr

		// 创建验证方法
		jwk, err := keyPair.ToJWK()
		if err != nil {
			b.Fatalf("转换JWK失败: %v", err)
		}

		// 创建注册请求
		req := &did.RegisterRequest{
			DID: didStr,
			VerificationMethod: []did.VerificationMethod{
				{
					ID:           didStr + "#key-1",
					Type:         "JsonWebKey2020",
					Controller:   didStr,
					PublicKeyJwk: jwk,
				},
			},
		}

		// 注册DID到同一个registry实例
		_, err = registry.Register(req)
		if err != nil {
			b.Fatalf("注册DID失败: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 随机选择一个DID进行解析
		didToResolve := testDIDs[i%len(testDIDs)]
		_, err := registry.Resolve(didToResolve)
		if err != nil {
			b.Fatalf("解析DID失败: %v", err)
		}
	}
}

// TestConcurrentDIDOperations 测试并发DID操作
func TestConcurrentDIDOperations(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{}
	registry := did.NewDIDRegistry(cfg, nil)

	// 并发参数
	concurrentUsers := 50
	operationsPerUser := 10

	var wg sync.WaitGroup
	errorChan := make(chan error, concurrentUsers*operationsPerUser)

	// 启动多个goroutine进行并发操作
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			for j := 0; j < operationsPerUser; j++ {
				// 生成密钥对
				keyPair, err := crypto.GenerateHybridKeyPair()
				if err != nil {
					errorChan <- fmt.Errorf("用户%d操作%d: 生成密钥对失败: %v", userID, j, err)
					continue
				}

				// 生成DID
				didStr, err := crypto.GenerateDIDFromKeyPair(keyPair)
				if err != nil {
					errorChan <- fmt.Errorf("用户%d操作%d: 生成DID失败: %v", userID, j, err)
					continue
				}

				// 创建验证方法
				jwk, err := keyPair.ToJWK()
				if err != nil {
					errorChan <- fmt.Errorf("用户%d操作%d: 转换JWK失败: %v", userID, j, err)
					continue
				}

				// 创建注册请求
				req := &did.RegisterRequest{
					DID: didStr,
					VerificationMethod: []did.VerificationMethod{
						{
							ID:           didStr + "#key-1",
							Type:         "JsonWebKey2020",
							Controller:   didStr,
							PublicKeyJwk: jwk,
						},
					},
				}

				// 注册DID
				_, err = registry.Register(req)
				if err != nil {
					errorChan <- fmt.Errorf("用户%d操作%d: 注册DID失败: %v", userID, j, err)
					continue
				}

				// 解析DID
				_, err = registry.Resolve(didStr)
				if err != nil {
					errorChan <- fmt.Errorf("用户%d操作%d: 解析DID失败: %v", userID, j, err)
					continue
				}
			}
		}(i)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errorChan)

	// 检查是否有错误
	errorCount := 0
	for err := range errorChan {
		t.Errorf("并发操作错误: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("并发测试中发生了%d个错误", errorCount)
	}

	t.Logf("成功完成%d个用户的并发操作，每个用户执行%d次操作", concurrentUsers, operationsPerUser)
}

// TestMemoryUsage 测试内存使用情况
func TestMemoryUsage(t *testing.T) {
	// 创建测试配置
	cfg := &config.Config{}
	registry := did.NewDIDRegistry(cfg, nil)

	// 创建大量DID来测试内存使用
	didCount := 1000
	createdDIDs := make([]string, 0, didCount)

	start := time.Now()

	for i := 0; i < didCount; i++ {
		// 生成密钥对
		keyPair, err := crypto.GenerateHybridKeyPair()
		if err != nil {
			t.Fatalf("生成密钥对失败: %v", err)
		}

		// 生成DID
		didStr, err := crypto.GenerateDIDFromKeyPair(keyPair)
		if err != nil {
			t.Fatalf("生成DID失败: %v", err)
		}

		// 创建验证方法
		jwk, err := keyPair.ToJWK()
		if err != nil {
			t.Fatalf("转换JWK失败: %v", err)
		}

		// 创建注册请求
		req := &did.RegisterRequest{
			DID: didStr,
			VerificationMethod: []did.VerificationMethod{
				{
					ID:           didStr + "#key-1",
					Type:         "JsonWebKey2020",
					Controller:   didStr,
					PublicKeyJwk: jwk,
				},
			},
		}

		// 注册DID
		_, err = registry.Register(req)
		if err != nil {
			t.Fatalf("注册DID失败: %v", err)
		}

		createdDIDs = append(createdDIDs, didStr)

		// 每100个DID输出一次进度
		if (i+1)%100 == 0 {
			t.Logf("已创建%d个DID", i+1)
		}
	}

	elapsed := time.Since(start)
	t.Logf("创建%d个DID耗时: %v", didCount, elapsed)
	t.Logf("平均每个DID创建时间: %v", elapsed/time.Duration(didCount))

	// 测试解析性能
	resolveStart := time.Now()
	for _, didStr := range createdDIDs {
		_, err := registry.Resolve(didStr)
		if err != nil {
			t.Fatalf("解析DID失败: %v", err)
		}
	}
	resolveElapsed := time.Since(resolveStart)

	t.Logf("解析%d个DID耗时: %v", didCount, resolveElapsed)
	t.Logf("平均每个DID解析时间: %v", resolveElapsed/time.Duration(didCount))
}
