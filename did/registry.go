package did

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/utils"
)

// DIDRegistry DID注册表
type DIDRegistry struct {
	config     *config.Config
	blockchain interface{}             // 区块链接口
	storage    map[string]*DIDDocument // 内存存储，实际应该用数据库
	mu         sync.RWMutex
}

// DIDDocument DID文档结构
type DIDDocument struct {
	Context            []string             `json:"@context"`
	ID                 string               `json:"id"`
	VerificationMethod []VerificationMethod `json:"verificationMethod,omitempty"`
	Authentication     []string             `json:"authentication,omitempty"`
	AssertionMethod    []string             `json:"assertionMethod,omitempty"`
	KeyAgreement       []string             `json:"keyAgreement,omitempty"`
	Service            []Service            `json:"service,omitempty"`
	Created            time.Time            `json:"created"`
	Updated            time.Time            `json:"updated"`
	Proof              *Proof               `json:"proof,omitempty"`
	Status             string               `json:"status"` // active, revoked
}

// VerificationMethod 验证方法
type VerificationMethod struct {
	ID                 string      `json:"id"`
	Type               string      `json:"type"`
	Controller         string      `json:"controller"`
	PublicKeyMultibase string      `json:"publicKeyMultibase,omitempty"`
	PublicKeyJwk       interface{} `json:"publicKeyJwk,omitempty"`
}

// Service 服务端点
type Service struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// Proof 证明
type Proof struct {
	Type               string    `json:"type"`
	Created            time.Time `json:"created"`
	VerificationMethod string    `json:"verificationMethod"`
	ProofPurpose       string    `json:"proofPurpose"`
	Jws                string    `json:"jws"`
}

// RegisterRequest DID注册请求
type RegisterRequest struct {
	DID                string               `json:"did"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
	Service            []Service            `json:"service,omitempty"`
}

// UpdateRequest DID更新请求
type UpdateRequest struct {
	DID                string               `json:"did"`
	VerificationMethod []VerificationMethod `json:"verificationMethod,omitempty"`
	Service            []Service            `json:"service,omitempty"`
	Proof              *Proof               `json:"proof"`
}

// NewDIDRegistry 创建DID注册表
func NewDIDRegistry(cfg *config.Config, blockchain interface{}) *DIDRegistry {
	return &DIDRegistry{
		config:     cfg,
		blockchain: blockchain,
		storage:    make(map[string]*DIDDocument),
	}
}

// Register 注册DID
func (r *DIDRegistry) Register(req *RegisterRequest) (*DIDDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 验证DID格式
	if err := r.validateDID(req.DID); err != nil {
		return nil, utils.WrapValidationError(err, "DID")
	}

	// 检查DID是否已存在
	if _, exists := r.storage[req.DID]; exists {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeConflict, "DID_EXISTS", 
			"DID已存在", req.DID)
	}

	// 创建DID文档
	doc := &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/jws-2020/v1",
		},
		ID:                 req.DID,
		VerificationMethod: req.VerificationMethod,
		Service:            req.Service,
		Created:            time.Now(),
		Updated:            time.Now(),
		Status:             "active",
	}

	// 设置认证和断言方法
	for _, vm := range req.VerificationMethod {
		doc.Authentication = append(doc.Authentication, vm.ID)
		doc.AssertionMethod = append(doc.AssertionMethod, vm.ID)
	}

	// 存储DID文档
	r.storage[req.DID] = doc

	// TODO: 将DID注册交易提交到区块链
	log.Printf("注册DID: %s", req.DID)

	return doc, nil
}

// Resolve 解析DID
func (r *DIDRegistry) Resolve(didStr string) (*DIDDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doc, exists := r.storage[didStr]
	if !exists {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeNotFound, "DID_NOT_FOUND", 
			"DID不存在", didStr)
	}

	if doc.Status == "revoked" {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeValidation, "DID_REVOKED", 
			"DID已被撤销", didStr)
	}

	return doc, nil
}

// Update 更新DID
func (r *DIDRegistry) Update(req *UpdateRequest) (*DIDDocument, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, exists := r.storage[req.DID]
	if !exists {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeNotFound, "DID_NOT_FOUND", 
			"DID不存在", req.DID)
	}

	if doc.Status == "revoked" {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeValidation, "DID_REVOKED", 
			"DID已被撤销", req.DID)
	}

	// TODO: 验证更新权限和签名

	// 更新文档
	if len(req.VerificationMethod) > 0 {
		doc.VerificationMethod = req.VerificationMethod
		// 重新设置认证方法
		doc.Authentication = []string{}
		doc.AssertionMethod = []string{}
		for _, vm := range req.VerificationMethod {
			doc.Authentication = append(doc.Authentication, vm.ID)
			doc.AssertionMethod = append(doc.AssertionMethod, vm.ID)
		}
	}

	if len(req.Service) > 0 {
		doc.Service = req.Service
	}

	doc.Updated = time.Now()
	doc.Proof = req.Proof

	// TODO: 将DID更新交易提交到区块链
	log.Printf("更新DID: %s", req.DID)

	return doc, nil
}

// Revoke 撤销DID
func (r *DIDRegistry) Revoke(didStr string, proof *Proof) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	doc, exists := r.storage[didStr]
	if !exists {
		return utils.NewErrorWithDetails(utils.ErrorTypeNotFound, "DID_NOT_FOUND", 
			"DID不存在", didStr)
	}

	if doc.Status == "revoked" {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "DID_REVOKED", 
			"DID已被撤销", didStr)
	}

	// TODO: 验证撤销权限和签名

	// 撤销DID
	doc.Status = "revoked"
	doc.Updated = time.Now()
	doc.Proof = proof

	// TODO: 将DID撤销交易提交到区块链
	log.Printf("撤销DID: %s", didStr)

	return nil
}

// List 列出所有DID
func (r *DIDRegistry) List() ([]*DIDDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	docs := make([]*DIDDocument, 0, len(r.storage))
	for _, doc := range r.storage {
		docs = append(docs, doc)
	}

	return docs, nil
}

// validateDID 验证DID格式
func (r *DIDRegistry) validateDID(didStr string) error {
	if !strings.HasPrefix(didStr, "did:") {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "INVALID_DID_PREFIX", 
			"DID必须以'did:'开头", didStr)
	}

	parts := strings.Split(didStr, ":")
	if len(parts) < 3 {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "INVALID_DID_FORMAT", 
			"DID格式无效，至少需要3个部分", didStr)
	}

	// 验证方法名
	method := parts[1]
	if method != "qlink" {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "UNSUPPORTED_DID_METHOD", 
			"不支持的DID方法", method)
	}

	// 验证标识符
	identifier := parts[2]
	if len(identifier) == 0 {
		return utils.NewErrorWithDetails(utils.ErrorTypeValidation, "EMPTY_DID_IDENTIFIER", 
			"DID标识符不能为空", didStr)
	}

	return nil
}

// ToJSON 将DID文档转换为JSON
func (doc *DIDDocument) ToJSON() ([]byte, error) {
	return json.MarshalIndent(doc, "", "  ")
}

// FromJSON 从JSON创建DID文档
func FromJSON(data []byte) (*DIDDocument, error) {
	var doc DIDDocument
	err := json.Unmarshal(data, &doc)
	return &doc, err
}
