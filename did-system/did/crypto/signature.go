package crypto

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/qujing226/QLink/pkg/types"
	"github.com/qujing226/QLink/pkg/utils"
)

// SignatureVerifier 签名验证器
type SignatureVerifier struct{}

// NewSignatureVerifier 创建签名验证器实例
func NewSignatureVerifier() *SignatureVerifier {
	return &SignatureVerifier{}
}

// VerifyProof 验证DID文档的证明
func (sv *SignatureVerifier) VerifyProof(document interface{}, proof *types.Proof, verificationMethod *types.VerificationMethod) error {
	if proof == nil {
		return utils.NewError(utils.ErrorTypeValidation, "PROOF_REQUIRED", "证明不能为空")
	}

	if verificationMethod == nil {
		return utils.NewError(utils.ErrorTypeValidation, "VERIFICATION_METHOD_REQUIRED", "验证方法不能为空")
	}

	// 检查证明类型
	if proof.Type != "Ed25519Signature2020" && proof.Type != "JsonWebSignature2020" {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "UNSUPPORTED_PROOF_TYPE",
			"不支持的证明类型", proof.Type)
	}

	// 检查证明时间
	if err := sv.validateProofTime(proof); err != nil {
		return err
	}

	// 验证签名
	switch proof.Type {
	case "Ed25519Signature2020":
		return sv.verifyEd25519Signature(document, proof, verificationMethod)
	case "JsonWebSignature2020":
		return sv.verifyJWSSignature(document, proof, verificationMethod)
	default:
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "UNSUPPORTED_PROOF_TYPE",
			"不支持的证明类型", proof.Type)
	}
}

// VerifyController 验证控制者权限
func (sv *SignatureVerifier) VerifyController(didStr string, verificationMethod *types.VerificationMethod) error {
	if verificationMethod == nil {
		return utils.NewError(utils.ErrorTypeValidation, "VERIFICATION_METHOD_REQUIRED", "验证方法不能为空")
	}

	// 检查控制者是否匹配
	if verificationMethod.Controller != didStr {
		return utils.NewErrorWithDetails(utils.ErrorTypeUnauthorized, "CONTROLLER_MISMATCH",
			"控制者不匹配", fmt.Sprintf("期望: %s, 实际: %s", didStr, verificationMethod.Controller))
	}

	return nil
}

// VerifyUpdatePermission 验证更新权限
func (sv *SignatureVerifier) VerifyUpdatePermission(didStr string, proof *types.Proof, verificationMethods []types.VerificationMethod) error {
	if proof == nil {
		return utils.NewError(utils.ErrorTypeValidation, "PROOF_REQUIRED", "更新操作需要提供证明")
	}

	// 查找对应的验证方法
	var verificationMethod *types.VerificationMethod
	for _, vm := range verificationMethods {
		if vm.ID == proof.VerificationMethod {
			verificationMethod = &vm
			break
		}
	}

	if verificationMethod == nil {
		return utils.NewErrorWithDetails(utils.ErrorTypeNotFound, "VERIFICATION_METHOD_NOT_FOUND",
			"验证方法不存在", proof.VerificationMethod)
	}

	// 验证控制者权限
	if err := sv.VerifyController(didStr, verificationMethod); err != nil {
		return err
	}

	// 验证证明目的
	if proof.ProofPurpose != "assertionMethod" && proof.ProofPurpose != "authentication" {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "INVALID_PROOF_PURPOSE",
			"无效的证明目的", proof.ProofPurpose)
	}

	return nil
}

// VerifyRevokePermission 验证撤销权限
func (sv *SignatureVerifier) VerifyRevokePermission(didStr string, proof *types.Proof, verificationMethods []types.VerificationMethod) error {
	if proof == nil {
		return utils.NewError(utils.ErrorTypeValidation, "PROOF_REQUIRED", "撤销操作需要提供证明")
	}

	// 查找对应的验证方法
	var verificationMethod *types.VerificationMethod
	for _, vm := range verificationMethods {
		if vm.ID == proof.VerificationMethod {
			verificationMethod = &vm
			break
		}
	}

	if verificationMethod == nil {
		return utils.NewErrorWithDetails(utils.ErrorTypeNotFound, "VERIFICATION_METHOD_NOT_FOUND",
			"验证方法不存在", proof.VerificationMethod)
	}

	// 验证控制者权限
	if err := sv.VerifyController(didStr, verificationMethod); err != nil {
		return err
	}

	// 验证证明目的（撤销需要更高权限）
	if proof.ProofPurpose != "authentication" {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "INVALID_PROOF_PURPOSE",
			"撤销操作需要authentication权限", proof.ProofPurpose)
	}

	return nil
}

// 私有方法

// validateProofTime 验证证明时间
func (sv *SignatureVerifier) validateProofTime(proof *types.Proof) error {
	now := time.Now()

	// 检查证明是否过期（24小时内有效）
	if now.Sub(proof.Created) > 24*time.Hour {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "PROOF_EXPIRED",
			"证明已过期", fmt.Sprintf("创建时间: %s", proof.Created.Format(time.RFC3339)))
	}

	// 检查证明是否来自未来
	if proof.Created.After(now.Add(5 * time.Minute)) {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "PROOF_FROM_FUTURE",
			"证明时间不能来自未来", fmt.Sprintf("创建时间: %s", proof.Created.Format(time.RFC3339)))
	}

	return nil
}

// verifyEd25519Signature 验证Ed25519签名
func (sv *SignatureVerifier) verifyEd25519Signature(document interface{}, proof *types.Proof, verificationMethod *types.VerificationMethod) error {
	// 获取公钥
	publicKey, err := sv.extractEd25519PublicKey(verificationMethod)
	if err != nil {
		return err
	}

	// 创建签名数据
	signatureData, err := sv.createSignatureData(document, proof)
	if err != nil {
		return err
	}

	// 解码签名
	signature, err := base64.StdEncoding.DecodeString(proof.ProofValue)
	if err != nil {
		return utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_SIGNATURE_FORMAT",
			"签名格式无效", err)
	}

	// 验证签名
	if !ed25519.Verify(publicKey, signatureData, signature) {
		return utils.NewError(utils.ErrorTypeUnauthorized, "SIGNATURE_VERIFICATION_FAILED", "签名验证失败")
	}

	return nil
}

// verifyJWSSignature 验证JWS签名
func (sv *SignatureVerifier) verifyJWSSignature(document interface{}, proof *types.Proof, verificationMethod *types.VerificationMethod) error {
	// 解析JWS格式的签名
	jwsSignature := proof.ProofValue
	if jwsSignature == "" {
		return utils.NewError(utils.ErrorTypeValidation, "EMPTY_JWS_SIGNATURE", "JWS签名不能为空")
	}

	// JWS格式: header.payload.signature
	parts := strings.Split(jwsSignature, ".")
	if len(parts) != 3 {
		return utils.NewError(utils.ErrorTypeValidation, "INVALID_JWS_FORMAT", "无效的JWS格式，应为header.payload.signature")
	}

	headerB64, payloadB64, signatureB64 := parts[0], parts[1], parts[2]

	// 解码header
	headerBytes, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_JWS_HEADER", "无效的JWS头部", err)
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
		Kid string `json:"kid,omitempty"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_JWS_HEADER_JSON", "JWS头部JSON解析失败", err)
	}

	// 验证算法类型
	if header.Alg != "ES256" && header.Alg != "EdDSA" {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "UNSUPPORTED_JWS_ALGORITHM",
			"不支持的JWS算法", header.Alg)
	}

	// 解码payload（验证payload格式但不需要使用内容）
	_, err = base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_JWS_PAYLOAD", "无效的JWS载荷", err)
	}

	// 解码签名
	signatureBytes, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_JWS_SIGNATURE", "无效的JWS签名", err)
	}

	// 创建签名数据 (header.payload)
	signingInput := headerB64 + "." + payloadB64

	// 根据算法验证签名
	switch header.Alg {
	case "ES256":
		return sv.verifyES256Signature([]byte(signingInput), signatureBytes, verificationMethod)
	case "EdDSA":
		return sv.verifyEdDSASignature([]byte(signingInput), signatureBytes, verificationMethod)
	default:
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "UNSUPPORTED_JWS_ALGORITHM",
			"不支持的JWS算法", header.Alg)
	}
}

// verifyES256Signature 验证ES256签名（ECDSA P-256 + SHA256）
func (sv *SignatureVerifier) verifyES256Signature(signingInput, signature []byte, verificationMethod *types.VerificationMethod) error {
	// 从验证方法中提取ECDSA公钥
	publicKey, err := sv.extractECDSAPublicKey(verificationMethod)
	if err != nil {
		return err
	}

	// 计算签名输入的SHA256哈希
	hash := sha256.Sum256(signingInput)

	// 验证ECDSA签名
	if !ecdsa.VerifyASN1(publicKey, hash[:], signature) {
		return utils.NewError(utils.ErrorTypeValidation, "INVALID_ES256_SIGNATURE", "ES256签名验证失败")
	}

	return nil
}

// verifyEdDSASignature 验证EdDSA签名（Ed25519）
func (sv *SignatureVerifier) verifyEdDSASignature(signingInput, signature []byte, verificationMethod *types.VerificationMethod) error {
	// 从验证方法中提取Ed25519公钥
	publicKey, err := sv.extractEd25519PublicKey(verificationMethod)
	if err != nil {
		return err
	}

	// 验证Ed25519签名
	if !ed25519.Verify(publicKey, signingInput, signature) {
		return utils.NewError(utils.ErrorTypeValidation, "INVALID_EDDSA_SIGNATURE", "EdDSA签名验证失败")
	}

	return nil
}

// extractECDSAPublicKey 从验证方法中提取ECDSA公钥
func (sv *SignatureVerifier) extractECDSAPublicKey(verificationMethod *types.VerificationMethod) (*ecdsa.PublicKey, error) {
	if verificationMethod.Type != "JsonWebKey2020" {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeValidation, "UNSUPPORTED_KEY_TYPE",
			"ES256签名需要JsonWebKey2020类型的验证方法", verificationMethod.Type)
	}

	// 解析JWK格式的公钥
	jwkData, ok := verificationMethod.PublicKeyJwk.(map[string]interface{})
	if !ok {
		return nil, utils.NewError(utils.ErrorTypeValidation, "INVALID_JWK_FORMAT", "无效的JWK格式")
	}

	// 检查密钥类型
	kty, ok := jwkData["kty"].(string)
	if !ok || kty != "EC" {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeValidation, "INVALID_KEY_TYPE",
			"ES256签名需要EC类型的密钥", kty)
	}

	// 检查曲线类型
	crv, ok := jwkData["crv"].(string)
	if !ok || crv != "P-256" {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeValidation, "INVALID_CURVE",
			"ES256签名需要P-256曲线", crv)
	}

	// 提取X和Y坐标
	xStr, ok := jwkData["x"].(string)
	if !ok {
		return nil, utils.NewError(utils.ErrorTypeValidation, "MISSING_X_COORDINATE", "缺少X坐标")
	}

	yStr, ok := jwkData["y"].(string)
	if !ok {
		return nil, utils.NewError(utils.ErrorTypeValidation, "MISSING_Y_COORDINATE", "缺少Y坐标")
	}

	// 解码坐标
	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_X_COORDINATE", "无效的X坐标", err)
	}

	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_Y_COORDINATE", "无效的Y坐标", err)
	}

	// 构造ECDSA公钥
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	// 验证公钥是否在曲线上
	if !publicKey.Curve.IsOnCurve(x, y) {
		return nil, utils.NewError(utils.ErrorTypeValidation, "INVALID_PUBLIC_KEY", "公钥不在P-256曲线上")
	}

	return publicKey, nil
}

// extractEd25519PublicKey 提取Ed25519公钥
func (sv *SignatureVerifier) extractEd25519PublicKey(verificationMethod *types.VerificationMethod) (ed25519.PublicKey, error) {
	if verificationMethod.Type != "Ed25519VerificationKey2020" {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeValidation, "UNSUPPORTED_KEY_TYPE",
			"不支持的密钥类型", verificationMethod.Type)
	}

	if verificationMethod.PublicKeyMultibase == "" {
		return nil, utils.NewError(utils.ErrorTypeValidation, "PUBLIC_KEY_REQUIRED", "公钥不能为空")
	}

	// 解码multibase格式的公钥
	// 假设使用base58编码，前缀为'z'
	if !strings.HasPrefix(verificationMethod.PublicKeyMultibase, "z") {
		return nil, utils.NewError(utils.ErrorTypeValidation, "INVALID_MULTIBASE_FORMAT", "无效的multibase格式")
	}

	// 简化处理：直接使用base64解码（实际应该使用multibase库）
	keyData := strings.TrimPrefix(verificationMethod.PublicKeyMultibase, "z")
	publicKey, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return nil, utils.NewErrorWithCause(utils.ErrorTypeValidation, "INVALID_PUBLIC_KEY_FORMAT",
			"公钥格式无效", err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeValidation, "INVALID_PUBLIC_KEY_SIZE",
			"公钥长度无效", fmt.Sprintf("期望: %d, 实际: %d", ed25519.PublicKeySize, len(publicKey)))
	}

	return ed25519.PublicKey(publicKey), nil
}

// createSignatureData 创建签名数据
func (sv *SignatureVerifier) createSignatureData(document interface{}, proof *types.Proof) ([]byte, error) {
	// 创建规范化的文档副本（移除proof字段）
	docBytes, err := json.Marshal(document)
	if err != nil {
		return nil, utils.NewErrorWithCause(utils.ErrorTypeInternal, "DOCUMENT_SERIALIZATION_FAILED",
			"文档序列化失败", err)
	}

	// 创建证明副本（移除proofValue字段）
	proofCopy := *proof
	proofCopy.ProofValue = ""

	proofBytes, err := json.Marshal(proofCopy)
	if err != nil {
		return nil, utils.NewErrorWithCause(utils.ErrorTypeInternal, "PROOF_SERIALIZATION_FAILED",
			"证明序列化失败", err)
	}

	// 组合数据并计算哈希
	combinedData := append(docBytes, proofBytes...)
	hash := sha256.Sum256(combinedData)

	return hash[:], nil
}
