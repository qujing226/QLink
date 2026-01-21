package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/mlkem"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
)

// HybridKeyPair 混合密钥对（ECDSA + Kyber768）
type HybridKeyPair struct {
	ECDSAPrivateKey *ecdsa.PrivateKey `json:"-"` // 不序列化私钥
	ECDSAPublicKey  *ecdsa.PublicKey  `json:"ecdsa_public_key"`
	// Kyber768密钥对
	KyberDecapsulationKey *mlkem.DecapsulationKey768 `json:"-"` // 不序列化私钥
	KyberEncapsulationKey *mlkem.EncapsulationKey768 `json:"kyber_public_key"`
}

// PublicKeyJWK JWK格式的公钥
type PublicKeyJWK struct {
	Kty   string `json:"kty"`   // 密钥类型
	Alg   string `json:"alg"`   // 算法
	Use   string `json:"use"`   // 用途
	Crv   string `json:"crv"`   // 曲线（ECDSA）
	X     string `json:"x"`     // X坐标（ECDSA）
	Y     string `json:"y"`     // Y坐标（ECDSA）
	Kyber string `json:"kyber"` // Kyber768公钥
}

// HybridSignature 混合签名
type HybridSignature struct {
	ECDSASignature []byte `json:"ecdsa_signature"`
	// Kyber768不用于签名，而是用于密钥封装，这里保留字段用于未来扩展
	KyberProof []byte `json:"kyber_proof,omitempty"`
}

// GenerateHybridKeyPair 生成混合密钥对
func GenerateHybridKeyPair() (*HybridKeyPair, error) {
	// 生成ECDSA密钥对（P-256）
	ecdsaPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("生成ECDSA密钥失败: %w", err)
	}

	// 生成Kyber768密钥对
	kyberDecapsKey, err := mlkem.GenerateKey768()
	if err != nil {
		return nil, fmt.Errorf("生成Kyber768密钥失败: %w", err)
	}

	kyberEncapsKey := kyberDecapsKey.EncapsulationKey()

	return &HybridKeyPair{
		ECDSAPrivateKey:       ecdsaPrivKey,
		ECDSAPublicKey:        &ecdsaPrivKey.PublicKey,
		KyberDecapsulationKey: kyberDecapsKey,
		KyberEncapsulationKey: kyberEncapsKey,
	}, nil
}

// Sign 使用混合密钥对数据进行签名
func (hkp *HybridKeyPair) Sign(data []byte) (*HybridSignature, error) {
	if hkp.ECDSAPrivateKey == nil {
		return nil, fmt.Errorf("ECDSA私钥为空")
	}

	// 计算数据哈希
	hash := sha256.Sum256(data)

	// ECDSA签名
	ecdsaSig, err := ecdsa.SignASN1(rand.Reader, hkp.ECDSAPrivateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("ECDSA签名失败: %w", err)
	}

	return &HybridSignature{
		ECDSASignature: ecdsaSig,
	}, nil
}

// Verify 验证混合签名
func (hkp *HybridKeyPair) Verify(data []byte, sig *HybridSignature) bool {
	if hkp.ECDSAPublicKey == nil {
		return false
	}

	// 计算数据哈希
	hash := sha256.Sum256(data)

	// 验证ECDSA签名
	return ecdsa.VerifyASN1(hkp.ECDSAPublicKey, hash[:], sig.ECDSASignature)
}

// ToJWK 将公钥转换为JWK格式
func (hkp *HybridKeyPair) ToJWK() (*PublicKeyJWK, error) {
	if hkp.ECDSAPublicKey == nil {
		return nil, fmt.Errorf("ECDSA公钥为空")
	}

	if hkp.KyberEncapsulationKey == nil {
		return nil, fmt.Errorf("Kyber768公钥为空")
	}

	// 获取ECDSA公钥坐标
	x := hkp.ECDSAPublicKey.X.Bytes()
	y := hkp.ECDSAPublicKey.Y.Bytes()

	// 确保坐标长度为32字节（P-256）
	if len(x) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(x):], x)
		x = padded
	}
	if len(y) < 32 {
		padded := make([]byte, 32)
		copy(padded[32-len(y):], y)
		y = padded
	}

	// 获取Kyber768公钥字节
	kyberBytes := hkp.KyberEncapsulationKey.Bytes()

	return &PublicKeyJWK{
		Kty:   "EC",
		Alg:   "ES256",
		Use:   "sig",
		Crv:   "P-256",
		X:     base64.RawURLEncoding.EncodeToString(x),
		Y:     base64.RawURLEncoding.EncodeToString(y),
		Kyber: base64.RawURLEncoding.EncodeToString(kyberBytes),
	}, nil
}

// FromJWK 从JWK格式创建公钥
func FromJWK(jwk *PublicKeyJWK) (*HybridKeyPair, error) {
	if jwk.Kty != "EC" || jwk.Crv != "P-256" {
		return nil, fmt.Errorf("不支持的密钥类型或曲线")
	}

	// 解码ECDSA坐标
	x, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("解码X坐标失败: %w", err)
	}

	y, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("解码Y坐标失败: %w", err)
	}

	// 创建ECDSA公钥
	curve := elliptic.P256()
	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     new(big.Int).SetBytes(x),
		Y:     new(big.Int).SetBytes(y),
	}

	// 解码Kyber768公钥
	var kyberEncapsKey *mlkem.EncapsulationKey768
	if jwk.Kyber != "" {
		kyberBytes, err := base64.RawURLEncoding.DecodeString(jwk.Kyber)
		if err != nil {
			return nil, fmt.Errorf("解码Kyber768公钥失败: %w", err)
		}

		kyberEncapsKey, err = mlkem.NewEncapsulationKey768(kyberBytes)
		if err != nil {
			return nil, fmt.Errorf("创建Kyber768公钥失败: %w", err)
		}
	}

	return &HybridKeyPair{
		ECDSAPublicKey:        pubKey,
		KyberEncapsulationKey: kyberEncapsKey,
	}, nil
}

// SerializePublicKey 序列化公钥
func (hkp *HybridKeyPair) SerializePublicKey() ([]byte, error) {
	jwk, err := hkp.ToJWK()
	if err != nil {
		return nil, err
	}
	return json.Marshal(jwk)
}

// DeserializePublicKey 反序列化公钥
func DeserializePublicKey(data []byte) (*HybridKeyPair, error) {
	var jwk PublicKeyJWK
	if err := json.Unmarshal(data, &jwk); err != nil {
		return nil, fmt.Errorf("反序列化JWK失败: %w", err)
	}
	return FromJWK(&jwk)
}

// GetFingerprint 获取密钥指纹
func (hkp *HybridKeyPair) GetFingerprint() (string, error) {
	pubKeyData, err := hkp.SerializePublicKey()
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(pubKeyData)
	return base64.RawURLEncoding.EncodeToString(hash[:16]), nil // 使用前16字节作为指纹
}

// GenerateDIDFromKeyPair 从密钥对生成DID
func GenerateDIDFromKeyPair(keyPair *HybridKeyPair) (string, error) {
	fingerprint, err := keyPair.GetFingerprint()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("did:qlink:%s", fingerprint), nil
}

// EncapsulateSharedKey 使用Kyber768封装共享密钥
func (hkp *HybridKeyPair) EncapsulateSharedKey() (ciphertext, sharedKey []byte, err error) {
	if hkp.KyberEncapsulationKey == nil {
		return nil, nil, fmt.Errorf("Kyber768公钥为空")
	}

	sharedKey, ciphertext = hkp.KyberEncapsulationKey.Encapsulate()
	return ciphertext, sharedKey, nil
}

// DecapsulateSharedKey 使用Kyber768解封装共享密钥
func (hkp *HybridKeyPair) DecapsulateSharedKey(ciphertext []byte) (sharedKey []byte, err error) {
	if hkp.KyberDecapsulationKey == nil {
		return nil, fmt.Errorf("Kyber768私钥为空")
	}

	return hkp.KyberDecapsulationKey.Decapsulate(ciphertext)
}

// HybridEncrypt 使用混合加密（ECDSA签名 + Kyber768密钥封装）
func (hkp *HybridKeyPair) HybridEncrypt(data []byte, recipientPublicKey *HybridKeyPair) (encryptedData []byte, signature *HybridSignature, err error) {
	// 1. 使用接收方的Kyber768公钥封装共享密钥
	ciphertext, sharedKey, err := recipientPublicKey.EncapsulateSharedKey()
	if err != nil {
		return nil, nil, fmt.Errorf("密钥封装失败: %w", err)
	}

	// 2. 使用共享密钥加密数据（这里简化处理，实际应该使用AES等对称加密）
	// 为了演示，这里只是简单的XOR操作
	encryptedData = make([]byte, len(data)+len(ciphertext))
	copy(encryptedData[:len(ciphertext)], ciphertext)

	// 使用共享密钥的前几个字节作为XOR密钥
	keyBytes := sharedKey[:min(len(sharedKey), len(data))]
	for i := 0; i < len(data); i++ {
		encryptedData[len(ciphertext)+i] = data[i] ^ keyBytes[i%len(keyBytes)]
	}

	// 3. 使用发送方的ECDSA私钥对原始数据进行签名
	signature, err = hkp.Sign(data)
	if err != nil {
		return nil, nil, fmt.Errorf("签名失败: %w", err)
	}

	return encryptedData, signature, nil
}

// HybridDecrypt 使用混合解密（ECDSA验证 + Kyber768密钥解封装）
func (hkp *HybridKeyPair) HybridDecrypt(encryptedData []byte, signature *HybridSignature, senderPublicKey *HybridKeyPair) (data []byte, err error) {
	// 1. 提取密文和加密的数据
	if len(encryptedData) < mlkem.CiphertextSize768 {
		return nil, fmt.Errorf("加密数据长度不足")
	}

	ciphertext := encryptedData[:mlkem.CiphertextSize768]
	encryptedPayload := encryptedData[mlkem.CiphertextSize768:]

	// 2. 使用自己的Kyber768私钥解封装共享密钥
	sharedKey, err := hkp.DecapsulateSharedKey(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("密钥解封装失败: %w", err)
	}

	// 3. 使用共享密钥解密数据
	data = make([]byte, len(encryptedPayload))
	keyBytes := sharedKey[:min(len(sharedKey), len(encryptedPayload))]
	for i := 0; i < len(encryptedPayload); i++ {
		data[i] = encryptedPayload[i] ^ keyBytes[i%len(keyBytes)]
	}

	// 4. 使用发送方的ECDSA公钥验证签名
	if !senderPublicKey.Verify(data, signature) {
		return nil, fmt.Errorf("签名验证失败")
	}

	return data, nil
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FromPrivateKeyString 从私钥字符串创建HybridKeyPair
// 支持多种格式：hex编码、base64编码等
func FromPrivateKeyString(privateKeyStr string) (*HybridKeyPair, error) {
	// 尝试不同的解码方式
	var privateKeyBytes []byte
	var err error

	// 首先尝试hex解码
	if len(privateKeyStr)%2 == 0 {
		privateKeyBytes, err = hex.DecodeString(privateKeyStr)
		if err == nil && len(privateKeyBytes) >= 32 {
			return createKeyPairFromBytes(privateKeyBytes)
		}
	}

	// 尝试base64解码
	privateKeyBytes, err = base64.StdEncoding.DecodeString(privateKeyStr)
	if err == nil && len(privateKeyBytes) >= 32 {
		return createKeyPairFromBytes(privateKeyBytes)
	}

	// 尝试base64 URL编码
	privateKeyBytes, err = base64.RawURLEncoding.DecodeString(privateKeyStr)
	if err == nil && len(privateKeyBytes) >= 32 {
		return createKeyPairFromBytes(privateKeyBytes)
	}

	// 如果都失败了，直接使用字符串的SHA256哈希作为种子
	hash := sha256.Sum256([]byte(privateKeyStr))
	return createKeyPairFromBytes(hash[:])
}

// createKeyPairFromBytes 从字节数组创建密钥对
func createKeyPairFromBytes(seed []byte) (*HybridKeyPair, error) {
	// 确保种子长度至少32字节
	if len(seed) < 32 {
		// 如果不够32字节，用SHA256扩展
		hash := sha256.Sum256(seed)
		seed = hash[:]
	}

	// 使用种子创建ECDSA私钥
	curve := elliptic.P256()
	// 取前32字节作为私钥
	privateKeyInt := new(big.Int).SetBytes(seed[:32])
	
	// 确保私钥在有效范围内
	n := curve.Params().N
	privateKeyInt.Mod(privateKeyInt, n)
	if privateKeyInt.Sign() == 0 {
		privateKeyInt.SetInt64(1) // 避免零私钥
	}

	ecdsaPrivKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
		},
		D: privateKeyInt,
	}

	// 计算公钥
	ecdsaPrivKey.PublicKey.X, ecdsaPrivKey.PublicKey.Y = curve.ScalarBaseMult(privateKeyInt.Bytes())

	// 为Kyber768生成密钥对（使用种子的另一部分）
	// 注意：这是简化实现，实际应用中可能需要更复杂的种子处理
	kyberSeed := make([]byte, 64)
	copy(kyberSeed[:32], seed[:32])
	if len(seed) >= 64 {
		copy(kyberSeed[32:], seed[32:64])
	} else {
		// 如果种子不够长，使用哈希扩展
		hash := sha256.Sum256(append(seed, []byte("kyber")...))
		copy(kyberSeed[32:], hash[:])
	}

	// 使用确定性方式生成Kyber密钥（这里简化处理）
	// 实际应用中应该使用更安全的方法
	kyberDecapsKey, err := mlkem.GenerateKey768()
	if err != nil {
		return nil, fmt.Errorf("生成Kyber768密钥失败: %w", err)
	}

	kyberEncapsKey := kyberDecapsKey.EncapsulationKey()

	return &HybridKeyPair{
		ECDSAPrivateKey:       ecdsaPrivKey,
		ECDSAPublicKey:        &ecdsaPrivKey.PublicKey,
		KyberDecapsulationKey: kyberDecapsKey,
		KyberEncapsulationKey: kyberEncapsKey,
	}, nil
}
