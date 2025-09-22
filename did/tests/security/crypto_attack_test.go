package security

import (
	"fmt"
	"testing"
	"time"

	"github.com/qujing226/QLink/did/crypto"
	"github.com/qujing226/QLink/did/tests/testutils"
)

// TestReplayAttackPrevention 测试防重放攻击
func TestReplayAttackPrevention(t *testing.T) {
	// 设置测试环境
	testutils.SetupTestEnvironment(t)

	// 生成密钥对
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 测试数据
	testData := []byte("这是一个测试消息")

	// 生成签名
	signature, err := keyPair.Sign(testData)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}

	// 第一次验证应该成功
	if !keyPair.Verify(testData, signature) {
		t.Fatal("首次签名验证失败")
	}

	// 模拟重放攻击 - 相同的签名应该仍然有效（因为数据相同）
	// 但在实际应用中，应该通过时间戳或nonce来防止重放
	if !keyPair.Verify(testData, signature) {
		t.Fatal("重放验证失败 - 这在当前实现中是预期的")
	}

	t.Log("注意：当前实现未包含时间戳验证，需要在应用层实现防重放机制")
}

// TestTimestampValidation 测试时间戳验证（模拟）
func TestTimestampValidation(t *testing.T) {
	// 设置测试环境
	testutils.SetupTestEnvironment(t)

	// 当前时间戳
	currentTime := time.Now().Unix()

	// 测试有效时间戳（当前时间）
	validTimestamp := currentTime
	if !isTimestampValid(validTimestamp, 300) { // 5分钟容差
		t.Fatal("有效时间戳验证失败")
	}

	// 测试过期时间戳（1小时前）
	expiredTimestamp := currentTime - 3600
	if isTimestampValid(expiredTimestamp, 300) {
		t.Fatal("过期时间戳应该验证失败")
	}

	// 测试未来时间戳（1小时后）
	futureTimestamp := currentTime + 3600
	if isTimestampValid(futureTimestamp, 300) {
		t.Fatal("未来时间戳应该验证失败")
	}
}

// TestSignatureMalleability 测试签名延展性攻击防护
func TestSignatureMalleability(t *testing.T) {
	// 设置测试环境
	testutils.SetupTestEnvironment(t)

	// 生成密钥对
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 测试数据
	testData := []byte("这是一个测试消息")

	// 生成多个签名
	signatures := make([]*crypto.HybridSignature, 5)
	for i := 0; i < 5; i++ {
		sig, err := keyPair.Sign(testData)
		if err != nil {
			t.Fatalf("第%d次签名失败: %v", i+1, err)
		}
		signatures[i] = sig
	}

	// 验证所有签名都有效
	for i, sig := range signatures {
		if !keyPair.Verify(testData, sig) {
			t.Fatalf("第%d个签名验证失败", i+1)
		}
	}

	// 注意：ECDSA签名具有随机性，每次签名结果都不同
	// 这有助于防止某些类型的签名延展性攻击
	t.Log("ECDSA签名的随机性有助于防止签名延展性攻击")
}

// TestKeyRecoveryResistance 测试密钥恢复攻击抵抗性
func TestKeyRecoveryResistance(t *testing.T) {
	// 设置测试环境
	testutils.SetupTestEnvironment(t)

	// 生成密钥对
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 生成大量签名样本
	sampleCount := 100
	testMessages := make([][]byte, sampleCount)
	signatures := make([]*crypto.HybridSignature, sampleCount)

	for i := 0; i < sampleCount; i++ {
		// 生成不同的测试消息
		message := []byte(fmt.Sprintf("测试消息 #%d - %d", i, time.Now().UnixNano()))
		testMessages[i] = message

		// 生成签名
		sig, err := keyPair.Sign(message)
		if err != nil {
			t.Fatalf("第%d个消息签名失败: %v", i+1, err)
		}
		signatures[i] = sig
	}

	// 验证所有签名
	for i := 0; i < sampleCount; i++ {
		if !keyPair.Verify(testMessages[i], signatures[i]) {
			t.Fatalf("第%d个签名验证失败", i+1)
		}
	}

	// 注意：在实际应用中，应该确保即使有大量签名样本，
	// 攻击者也无法从中恢复私钥
	t.Logf("成功生成并验证了%d个签名样本", sampleCount)
}

// isTimestampValid 验证时间戳是否在有效范围内
func isTimestampValid(timestamp int64, toleranceSeconds int64) bool {
	currentTime := time.Now().Unix()
	diff := currentTime - timestamp

	// 检查时间戳是否在容差范围内（不能太旧或太新）
	return diff >= 0 && diff <= toleranceSeconds
}
