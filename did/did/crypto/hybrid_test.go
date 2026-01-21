package crypto

import (
	"testing"
)

func TestKyber768KeyGeneration(t *testing.T) {
	// 测试密钥对生成
	keyPair, err := GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 验证ECDSA密钥
	if keyPair.ECDSAPrivateKey == nil {
		t.Error("ECDSA私钥为空")
	}
	if keyPair.ECDSAPublicKey == nil {
		t.Error("ECDSA公钥为空")
	}

	// 验证Kyber768密钥
	if keyPair.KyberDecapsulationKey == nil {
		t.Error("Kyber768私钥为空")
	}
	if keyPair.KyberEncapsulationKey == nil {
		t.Error("Kyber768公钥为空")
	}
}

func TestKyber768KeyEncapsulation(t *testing.T) {
	// 生成密钥对
	keyPair, err := GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 测试密钥封装
	ciphertext, sharedKey1, err := keyPair.EncapsulateSharedKey()
	if err != nil {
		t.Fatalf("密钥封装失败: %v", err)
	}

	if len(ciphertext) == 0 {
		t.Error("密文为空")
	}
	if len(sharedKey1) == 0 {
		t.Error("共享密钥为空")
	}

	// 测试密钥解封装
	sharedKey2, err := keyPair.DecapsulateSharedKey(ciphertext)
	if err != nil {
		t.Fatalf("密钥解封装失败: %v", err)
	}

	// 验证共享密钥一致性
	if len(sharedKey1) != len(sharedKey2) {
		t.Error("共享密钥长度不一致")
	}

	for i := range sharedKey1 {
		if sharedKey1[i] != sharedKey2[i] {
			t.Error("共享密钥内容不一致")
			break
		}
	}
}

func TestHybridEncryptDecrypt(t *testing.T) {
	// 生成发送方和接收方密钥对
	senderKeyPair, err := GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成发送方密钥对失败: %v", err)
	}

	recipientKeyPair, err := GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成接收方密钥对失败: %v", err)
	}

	// 测试数据
	originalData := []byte("这是一个测试消息，用于验证Kyber768混合加密功能")

	// 加密
	encryptedData, signature, err := senderKeyPair.HybridEncrypt(originalData, recipientKeyPair)
	if err != nil {
		t.Fatalf("混合加密失败: %v", err)
	}

	if len(encryptedData) == 0 {
		t.Error("加密数据为空")
	}
	if signature == nil {
		t.Error("签名为空")
	}

	// 解密
	decryptedData, err := recipientKeyPair.HybridDecrypt(encryptedData, signature, senderKeyPair)
	if err != nil {
		t.Fatalf("混合解密失败: %v", err)
	}

	// 验证数据一致性
	if len(originalData) != len(decryptedData) {
		t.Error("解密数据长度不一致")
	}

	for i := range originalData {
		if originalData[i] != decryptedData[i] {
			t.Error("解密数据内容不一致")
			break
		}
	}
}

func TestJWKWithKyber768(t *testing.T) {
	// 生成密钥对
	keyPair, err := GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 转换为JWK
	jwk, err := keyPair.ToJWK()
	if err != nil {
		t.Fatalf("转换为JWK失败: %v", err)
	}

	// 验证JWK包含Kyber768公钥
	if jwk.Kyber == "" {
		t.Error("JWK中缺少Kyber768公钥")
	}

	// 从JWK恢复公钥
	restoredKeyPair, err := FromJWK(jwk)
	if err != nil {
		t.Fatalf("从JWK恢复公钥失败: %v", err)
	}

	// 验证恢复的公钥
	if restoredKeyPair.ECDSAPublicKey == nil {
		t.Error("恢复的ECDSA公钥为空")
	}
	if restoredKeyPair.KyberEncapsulationKey == nil {
		t.Error("恢复的Kyber768公钥为空")
	}

	// 测试使用恢复的公钥进行密钥封装
	_, _, err = restoredKeyPair.EncapsulateSharedKey()
	if err != nil {
		t.Fatalf("使用恢复的公钥进行密钥封装失败: %v", err)
	}
}

func TestDIDGenerationWithKyber768(t *testing.T) {
	// 生成密钥对
	keyPair, err := GenerateHybridKeyPair()
	if err != nil {
		t.Fatalf("生成密钥对失败: %v", err)
	}

	// 生成DID
	did, err := GenerateDIDFromKeyPair(keyPair)
	if err != nil {
		t.Fatalf("生成DID失败: %v", err)
	}

	// 验证DID格式
	if len(did) == 0 {
		t.Error("DID为空")
	}

	expectedPrefix := "did:qlink:"
	if len(did) < len(expectedPrefix) || did[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("DID格式不正确，期望前缀: %s, 实际DID: %s", expectedPrefix, did)
	}

	t.Logf("生成的DID: %s", did)
}
