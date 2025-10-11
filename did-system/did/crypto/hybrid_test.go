package crypto

import (
    "testing"
)

func TestECDSAKeyGeneration(t *testing.T) {
    // 测试密钥对生成（仅 ECDSA）
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
}

func TestECDSASignVerify(t *testing.T) {
    keyPair, err := GenerateHybridKeyPair()
    if err != nil {
        t.Fatalf("生成密钥对失败: %v", err)
    }

    data := []byte("hello world")
    sig, err := keyPair.Sign(data)
    if err != nil {
        t.Fatalf("签名失败: %v", err)
    }

    if !keyPair.Verify(data, sig) {
        t.Fatal("签名验证失败")
    }
}

func TestJWKRoundTrip(t *testing.T) {
    keyPair, err := GenerateHybridKeyPair()
    if err != nil {
        t.Fatalf("生成密钥对失败: %v", err)
    }

    jwk, err := keyPair.ToJWK()
    if err != nil {
        t.Fatalf("转换为JWK失败: %v", err)
    }

    restored, err := FromJWK(jwk)
    if err != nil {
        t.Fatalf("从JWK恢复失败: %v", err)
    }

    if restored.ECDSAPublicKey == nil {
        t.Fatalf("恢复的ECDSA公钥为空")
    }
}

func TestDIDGeneration(t *testing.T) {
    keyPair, err := GenerateHybridKeyPair()
    if err != nil {
        t.Fatalf("生成密钥对失败: %v", err)
    }

    did, err := GenerateDIDFromKeyPair(keyPair)
    if err != nil {
        t.Fatalf("生成DID失败: %v", err)
    }

    if did == "" {
        t.Fatal("DID为空")
    }

    expectedPrefix := "did:qlink:"
    if len(did) < len(expectedPrefix) || did[:len(expectedPrefix)] != expectedPrefix {
        t.Fatalf("DID前缀不正确，期望: %s, 实际: %s", expectedPrefix, did)
    }
}
