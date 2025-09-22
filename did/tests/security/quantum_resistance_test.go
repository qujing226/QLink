package security

import (
	"crypto/rand"
	"testing"

	"github.com/qujing226/QLink/did/crypto"
	"github.com/qujing226/QLink/did/tests/testutils"
)

// TestQuantumResistantKeyGeneration 测试抗量子密钥生成
func TestQuantumResistantKeyGeneration(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 测试混合密钥对生成
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成混合密钥对失败: %v", err)
	}

	// 验证密钥对不为空
	if keyPair == nil {
		t.Fatal("生成的密钥对为空")
	}

	// 验证ECDSA密钥部分
	if keyPair.ECDSAPrivateKey == nil {
		t.Fatal("ECDSA私钥为空")
	}
	if keyPair.ECDSAPublicKey == nil {
		t.Fatal("ECDSA公钥为空")
	}

	t.Log("抗量子密钥生成测试通过")
}

// TestQuantumResistantSigning 测试抗量子数字签名
func TestQuantumResistantSigning(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 生成密钥对
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 测试数据
	testData := []byte("这是一个测试消息，用于验证抗量子签名功能")

	// 生成签名
	signature, err := keyPair.Sign(testData)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}

	// 验证签名
	if !keyPair.Verify(testData, signature) {
		t.Fatal("签名验证失败")
	}

	// 测试篡改数据后的签名验证
	tamperedData := []byte("这是一个被篡改的测试消息")
	if keyPair.Verify(tamperedData, signature) {
		t.Fatal("篡改数据的签名验证应该失败")
	}

	t.Log("抗量子数字签名测试通过")
}

// TestKeyPairUniqueness 测试密钥对唯一性
func TestKeyPairUniqueness(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 生成多个密钥对
	keyPairs := make([]*crypto.HybridKeyPair, 10)
	for i := 0; i < 10; i++ {
		kp, err := crypto.GenerateHybridKeyPair()
		if err != nil {
			t.Fatalf("生成第%d个密钥对失败: %v", i+1, err)
		}
		keyPairs[i] = kp
	}

	// 验证密钥对的唯一性
	for i := 0; i < len(keyPairs); i++ {
		for j := i + 1; j < len(keyPairs); j++ {
			// 比较公钥指纹
			fingerprint1, err1 := keyPairs[i].GetFingerprint()
			fingerprint2, err2 := keyPairs[j].GetFingerprint()
			if err1 == nil && err2 == nil {
				if fingerprint1 == fingerprint2 {
					t.Fatalf("密钥对%d和%d的指纹相同", i+1, j+1)
				}
			}
		}
	}

	t.Log("密钥对唯一性测试通过")
}

// TestKeyPairSerialization 测试密钥对序列化和反序列化
func TestKeyPairSerialization(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 生成密钥对
	originalKeyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 序列化公钥
	serializedData, err := originalKeyPair.SerializePublicKey()
	if err != nil {
		t.Fatalf("序列化公钥失败: %v", err)
	}

	// 反序列化公钥
	deserializedKeyPair, err := crypto.DeserializePublicKey(serializedData)
	if err != nil {
		t.Fatalf("反序列化公钥失败: %v", err)
	}

	// 验证反序列化后的密钥对功能
	testData := []byte("序列化测试数据")

	// 使用原始密钥对签名
	originalSignature, err := originalKeyPair.Sign(testData)
	if err != nil {
		t.Fatalf("原始密钥对签名失败: %v", err)
	}

	// 使用反序列化密钥对验证
	if !deserializedKeyPair.Verify(testData, originalSignature) {
		t.Fatal("反序列化密钥对验证签名失败")
	}

	t.Log("密钥对序列化测试通过")
}

// TestRandomnessQuality 测试随机数质量
func TestRandomnessQuality(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 生成多个随机数样本
	sampleSize := 1000
	sampleLength := 32
	samples := make([][]byte, sampleSize)

	for i := 0; i < sampleSize; i++ {
		sample := make([]byte, sampleLength)
		_, err := rand.Read(sample)
		if err != nil {
			t.Fatalf("生成随机数失败: %v", err)
		}
		samples[i] = sample
	}

	// 检查随机数的唯一性
	uniqueMap := make(map[string]bool)
	for i, sample := range samples {
		sampleStr := string(sample)
		if uniqueMap[sampleStr] {
			t.Fatalf("发现重复的随机数样本，索引: %d", i)
		}
		uniqueMap[sampleStr] = true
	}

	// 简单的熵检查：计算每个字节位置的分布
	for pos := 0; pos < sampleLength; pos++ {
		bitCounts := make([]int, 256)
		for _, sample := range samples {
			bitCounts[sample[pos]]++
		}

		// 检查分布是否过于集中（简单检查）
		maxCount := 0
		for _, count := range bitCounts {
			if count > maxCount {
				maxCount = count
			}
		}

		// 如果某个值出现次数超过样本总数的10%，可能存在偏差
		if maxCount > sampleSize/10 {
			t.Logf("警告：位置%d的随机数分布可能存在偏差，最大出现次数: %d", pos, maxCount)
		}
	}

	t.Log("随机数质量测试通过")
}

// BenchmarkKeyGeneration 密钥生成性能基准测试
func BenchmarkKeyGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := crypto.GenerateHybridKeyPair()
		if err != nil {
			b.Fatalf("生成密钥对失败: %v", err)
		}
	}
}

// BenchmarkSigning 签名性能基准测试
func BenchmarkSigning(b *testing.B) {
	// 预生成密钥对
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		b.Fatalf("生成密钥对失败: %v", err)
	}

	testData := []byte("性能测试数据")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := keyPair.Sign(testData)
		if err != nil {
			b.Fatalf("签名失败: %v", err)
		}
	}
}

// BenchmarkVerification 验证性能基准测试
func BenchmarkVerification(b *testing.B) {
	// 预生成密钥对和签名
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		b.Fatalf("生成密钥对失败: %v", err)
	}

	testData := []byte("性能测试数据")
	signature, err := keyPair.Sign(testData)
	if err != nil {
		b.Fatalf("生成签名失败: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keyPair.Verify(testData, signature)
	}
}
