package integration

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/tests/testutils"
)

// TestDIDRegistryIntegration 测试DID注册表集成功能
func TestDIDRegistryIntegration(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 创建测试配置
	cfg := &config.Config{
		DID: &config.DIDConfig{
			Method:  "QLink",
			ChainID: "test123",
		},
	}

	// 创建DID注册表
	registry := did.NewDIDRegistry(cfg, nil)

	// 生成测试DID
	didStr := testutils.GenerateTestDID("test123")

	// 测试DID格式验证
	if !strings.HasPrefix(didStr, "did:QLink:test123:") {
		t.Errorf("DID格式不正确: %s", didStr)
	}

	// 测试解析不存在的DID
	_, err := registry.Resolve(didStr)
	testutils.AssertError(t, err, "解析不存在的DID应该返回错误")

	// 测试列出DID（应该为空）
	docs, err := registry.List()
	testutils.AssertNoError(t, err, "列出DID")
	testutils.AssertEqual(t, 0, len(docs), "初始DID数量应该为0")
}

// TestDIDErrorHandling 测试DID错误处理
func TestDIDErrorHandling(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 创建测试配置
	cfg := &config.Config{
		DID: &config.DIDConfig{
			Method:  "QLink",
			ChainID: "test123",
		},
	}

	// 创建DID注册表
	registry := did.NewDIDRegistry(cfg, nil)

	// 测试无效的DID格式
	invalidDIDs := []string{
		"invalid-did",
		"did:invalid:test123:1234567890abcdef",
		"did:QLink:1234567890abcdef",  // 缺少chain-id
		"did:QLink::1234567890abcdef", // 空chain-id
		"",                            // 空字符串
	}

	for _, invalidDID := range invalidDIDs {
		// 测试解析无效DID
		_, err := registry.Resolve(invalidDID)
		testutils.AssertError(t, err, "解析无效DID应该失败: "+invalidDID)
	}

	// 测试解析不存在的DID
	nonExistentDID := testutils.GenerateTestDID("nonexistent")
	_, err := registry.Resolve(nonExistentDID)
	testutils.AssertError(t, err, "解析不存在的DID应该返回错误")
}

// TestDIDRegistryBasicOperations 测试DID注册表基本操作
func TestDIDRegistryBasicOperations(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 创建测试配置
	cfg := &config.Config{
		DID: &config.DIDConfig{
			Method:  "QLink",
			ChainID: "test123",
		},
	}

	// 创建DID注册表
	registry := did.NewDIDRegistry(cfg, nil)

	// 验证注册表初始状态
	docs, err := registry.List()
	testutils.AssertNoError(t, err, "列出DID")
	testutils.AssertEqual(t, 0, len(docs), "初始DID数量")

	// 生成多个测试DID
	didStr1 := testutils.GenerateTestDID("test1")
	didStr2 := testutils.GenerateTestDID("test2")
	didStr3 := testutils.GenerateTestDID("test3")

	// 验证DID格式
	for _, did := range []string{didStr1, didStr2, didStr3} {
		if !strings.HasPrefix(did, "did:QLink:test") {
			t.Errorf("DID格式不正确: %s", did)
		}
		if !strings.Contains(did, ":") {
			t.Errorf("DID应该包含冒号分隔符: %s", did)
		}
	}

	// 测试DID唯一性
	if didStr1 == didStr2 || didStr1 == didStr3 || didStr2 == didStr3 {
		t.Error("生成的DID应该是唯一的")
	}
}

// TestDIDConcurrentAccess 测试DID并发访问
func TestDIDConcurrentAccess(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 创建测试配置
	cfg := &config.Config{
		DID: &config.DIDConfig{
			Method:  "QLink",
			ChainID: "test123",
		},
	}

	// 创建DID注册表
	registry := did.NewDIDRegistry(cfg, nil)

	// 并发访问测试
	concurrency := 10
	done := make(chan bool, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			// 生成唯一的DID
			didStr := testutils.GenerateTestDID("concurrent")

			// 尝试解析（应该失败）
			_, err := registry.Resolve(didStr)
			if err == nil {
				errors <- fmt.Errorf("解析不存在的DID应该失败")
				return
			}

			// 列出DID
			_, err = registry.List()
			if err != nil {
				errors <- err
				return
			}

			done <- true
		}(i)
	}

	// 等待所有操作完成
	for i := 0; i < concurrency; i++ {
		select {
		case <-done:
			// 成功
		case err := <-errors:
			t.Fatalf("并发访问失败: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("并发访问超时")
		}
	}
}

// TestDIDValidation 测试DID验证逻辑
func TestDIDValidation(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 测试有效的DID格式
	validDID := testutils.GenerateTestDID("valid")
	if !strings.HasPrefix(validDID, "did:QLink:") {
		t.Errorf("有效DID格式不正确: %s", validDID)
	}

	// 测试DID组件
	parts := strings.Split(validDID, ":")
	if len(parts) < 4 {
		t.Errorf("DID应该至少有4个组件: %s", validDID)
	}

	if parts[0] != "did" {
		t.Errorf("DID应该以'did'开头: %s", validDID)
	}

	if parts[1] != "QLink" {
		t.Errorf("DID方法应该是'QLink': %s", validDID)
	}

	// 验证链ID存在
	if parts[2] == "" {
		t.Errorf("DID应该包含链ID: %s", validDID)
	}

	// 验证标识符存在
	if len(parts) < 4 || parts[3] == "" {
		t.Errorf("DID应该包含标识符: %s", validDID)
	}
}
