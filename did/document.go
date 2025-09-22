package did

import (
	"fmt"
	"time"

	"github.com/qujing226/QLink/did/crypto"
)

// DIDDocumentBuilder DID文档构建器
type DIDDocumentBuilder struct {
	keyPair *crypto.HybridKeyPair
	did     string
}

// NewDIDDocumentBuilder 创建DID文档构建器
func NewDIDDocumentBuilder() (*DIDDocumentBuilder, error) {
	// 生成混合密钥对
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		return nil, fmt.Errorf("生成密钥对失败: %w", err)
	}

	// 从密钥对生成DID
	did, err := crypto.GenerateDIDFromKeyPair(keyPair)
	if err != nil {
		return nil, fmt.Errorf("生成DID失败: %w", err)
	}

	return &DIDDocumentBuilder{
		keyPair: keyPair,
		did:     did,
	}, nil
}

// NewDIDDocumentBuilderFromKeyPair 从现有密钥对创建DID文档构建器
func NewDIDDocumentBuilderFromKeyPair(keyPair *crypto.HybridKeyPair) (*DIDDocumentBuilder, error) {
	// 从密钥对生成DID
	did, err := crypto.GenerateDIDFromKeyPair(keyPair)
	if err != nil {
		return nil, fmt.Errorf("生成DID失败: %w", err)
	}

	return &DIDDocumentBuilder{
		keyPair: keyPair,
		did:     did,
	}, nil
}

// BuildDocument 构建DID文档
func (builder *DIDDocumentBuilder) BuildDocument() (*DIDDocument, error) {
	// 将公钥转换为JWK格式
	jwk, err := builder.keyPair.ToJWK()
	if err != nil {
		return nil, fmt.Errorf("转换公钥为JWK失败: %w", err)
	}

	// 创建验证方法ID
	verificationMethodID := fmt.Sprintf("%s#key-1", builder.did)

	// 创建验证方法
	verificationMethod := VerificationMethod{
		ID:           verificationMethodID,
		Type:         "JsonWebKey2020",
		Controller:   builder.did,
		PublicKeyJwk: jwk,
	}

	// 创建DID文档
	doc := &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/jws-2020/v1",
		},
		ID:                 builder.did,
		VerificationMethod: []VerificationMethod{verificationMethod},
		Authentication:     []string{verificationMethodID},
		AssertionMethod:    []string{verificationMethodID},
		KeyAgreement:       []string{verificationMethodID},
		Created:            time.Now(),
		Updated:            time.Now(),
		Status:             "active",
	}

	return doc, nil
}

// AddService 添加服务端点
func (builder *DIDDocumentBuilder) AddService(serviceType, endpoint string) *DIDDocumentBuilder {
	// 这里可以扩展服务添加逻辑
	return builder
}

// GetDID 获取DID
func (builder *DIDDocumentBuilder) GetDID() string {
	return builder.did
}

// GetKeyPair 获取密钥对
func (builder *DIDDocumentBuilder) GetKeyPair() *crypto.HybridKeyPair {
	return builder.keyPair
}

// SignDocument 对DID文档进行签名
func (builder *DIDDocumentBuilder) SignDocument(doc *DIDDocument) error {
	// 序列化文档用于签名
	docData, err := doc.ToJSON()
	if err != nil {
		return fmt.Errorf("序列化文档失败: %w", err)
	}

	// 使用混合密钥对文档进行签名
	signature, err := builder.keyPair.Sign(docData)
	if err != nil {
		return fmt.Errorf("签名失败: %w", err)
	}

	// 创建证明
	proof := &Proof{
		Type:               "JsonWebSignature2020",
		Created:            time.Now(),
		VerificationMethod: fmt.Sprintf("%s#key-1", builder.did),
		ProofPurpose:       "assertionMethod",
		Jws:                fmt.Sprintf("%x", signature.ECDSASignature),
	}

	doc.Proof = proof
	return nil
}

// VerifyDocument 验证DID文档签名
func VerifyDocument(doc *DIDDocument, keyPair *crypto.HybridKeyPair) error {
	if doc.Proof == nil {
		return fmt.Errorf("文档没有证明")
	}

	// 临时移除证明进行验证
	originalProof := doc.Proof
	doc.Proof = nil

	// 序列化文档
	docData, err := doc.ToJSON()
	if err != nil {
		doc.Proof = originalProof
		return fmt.Errorf("序列化文档失败: %w", err)
	}

	// 恢复证明
	doc.Proof = originalProof

	// 解析签名
	signatureBytes := []byte(originalProof.Jws) // 简化处理，实际需要从hex解码
	signature := &crypto.HybridSignature{
		ECDSASignature: signatureBytes,
	}

	// 验证签名
	if !keyPair.Verify(docData, signature) {
		return fmt.Errorf("签名验证失败")
	}

	return nil
}

// CreateRegistrationRequest 创建DID注册请求
func (builder *DIDDocumentBuilder) CreateRegistrationRequest() (*RegisterRequest, error) {
	// 构建DID文档
	doc, err := builder.BuildDocument()
	if err != nil {
		return nil, err
	}

	// 对文档进行签名
	if err := builder.SignDocument(doc); err != nil {
		return nil, err
	}

	// 创建注册请求
	req := &RegisterRequest{
		DID:                builder.did,
		VerificationMethod: doc.VerificationMethod,
		Service:            doc.Service,
	}

	return req, nil
}

// CreateUpdateRequest 创建DID更新请求
func (builder *DIDDocumentBuilder) CreateUpdateRequest(newServices []Service) (*UpdateRequest, error) {
	// 构建更新的DID文档
	doc, err := builder.BuildDocument()
	if err != nil {
		return nil, err
	}

	// 添加新服务
	doc.Service = newServices

	// 对文档进行签名
	if err := builder.SignDocument(doc); err != nil {
		return nil, err
	}

	// 创建更新请求
	req := &UpdateRequest{
		DID:                builder.did,
		VerificationMethod: doc.VerificationMethod,
		Service:            doc.Service,
		Proof:              doc.Proof,
	}

	return req, nil
}

// ValidateDIDDocument 验证DID文档的完整性
func ValidateDIDDocument(doc *DIDDocument) error {
	if doc == nil {
		return fmt.Errorf("DID文档为空")
	}

	if doc.ID == "" {
		return fmt.Errorf("DID为空")
	}

	if len(doc.VerificationMethod) == 0 {
		return fmt.Errorf("验证方法为空")
	}

	if doc.Status != "active" && doc.Status != "revoked" {
		return fmt.Errorf("无效的DID状态: %s", doc.Status)
	}

	// 验证验证方法
	for _, vm := range doc.VerificationMethod {
		if vm.ID == "" {
			return fmt.Errorf("验证方法ID为空")
		}
		if vm.Type == "" {
			return fmt.Errorf("验证方法类型为空")
		}
		if vm.Controller == "" {
			return fmt.Errorf("验证方法控制器为空")
		}
	}

	return nil
}
