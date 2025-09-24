package did

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/qujing226/QLink/did/blockchain"
	"github.com/qujing226/QLink/pkg/config"
	"github.com/qujing226/QLink/pkg/types"
	"github.com/qujing226/QLink/pkg/utils"
)

// DIDResolver DID解析器
type DIDResolver struct {
	config   *config.Config
	registry *DIDRegistry
	storage  *blockchain.StorageManager
}

// ResolutionResult DID解析结果
type ResolutionResult struct {
	DIDDocument           *types.DIDDocument  `json:"didDocument,omitempty"`
	DIDResolutionMetadata *ResolutionMetadata `json:"didResolutionMetadata"`
	DIDDocumentMetadata   *DocumentMetadata   `json:"didDocumentMetadata"`
}

// ResolutionMetadata 解析元数据
type ResolutionMetadata struct {
	ContentType string `json:"contentType"`
	Error       string `json:"error,omitempty"`
}

// DocumentMetadata 文档元数据
type DocumentMetadata struct {
	Created     string `json:"created,omitempty"`
	Updated     string `json:"updated,omitempty"`
	Deactivated bool   `json:"deactivated,omitempty"`
}

// NewDIDResolver 创建DID解析器
func NewDIDResolver(cfg *config.Config, reg *DIDRegistry, storageManager *blockchain.StorageManager) *DIDResolver {
	return &DIDResolver{
		config:   cfg,
		registry: reg,
		storage:  storageManager,
	}
}

// Resolve 解析DID
func (r *DIDResolver) Resolve(didStr string) (*ResolutionResult, error) {
	log.Printf("解析DID: %s", didStr)

	// 验证DID格式
	if err := r.validateDIDFormat(didStr); err != nil {
		return &ResolutionResult{
			DIDResolutionMetadata: &ResolutionMetadata{
				ContentType: "application/did+ld+json",
				Error:       "invalidDid",
			},
		}, nil
	}

	// 解析DID方法
	method := r.extractMethod(didStr)
	switch method {
	case "qlink":
		return r.resolveQlinkDID(didStr)
	default:
		return &ResolutionResult{
			DIDResolutionMetadata: &ResolutionMetadata{
				ContentType: "application/did+ld+json",
				Error:       "methodNotSupported",
			},
		}, nil
	}
}

// resolveQlinkDID 解析QLink DID
func (r *DIDResolver) resolveQlinkDID(didStr string) (*ResolutionResult, error) {
	// 首先尝试从链上解析
	doc, err := r.registry.Resolve(didStr)
	if err == nil {
		return &ResolutionResult{
			DIDDocument: doc,
			DIDResolutionMetadata: &ResolutionMetadata{
				ContentType: "application/did+ld+json",
			},
			DIDDocumentMetadata: &DocumentMetadata{
				Created:     doc.Created.Format("2006-01-02T15:04:05Z"),
				Updated:     doc.Updated.Format("2006-01-02T15:04:05Z"),
				Deactivated: doc.Status == "revoked",
			},
		}, nil
	}

	// 如果链上没有，尝试从链下存储解析
	offchainDoc, err := r.resolveFromOffchain(didStr)
	if err == nil && offchainDoc != nil {
		return &ResolutionResult{
			DIDDocument: offchainDoc,
			DIDResolutionMetadata: &ResolutionMetadata{
				ContentType: "application/did+ld+json",
			},
			DIDDocumentMetadata: &DocumentMetadata{
				Created:     offchainDoc.Created.Format("2006-01-02T15:04:05Z"),
				Updated:     offchainDoc.Updated.Format("2006-01-02T15:04:05Z"),
				Deactivated: offchainDoc.Status == "revoked",
			},
		}, nil
	}

	// DID不存在
	return &ResolutionResult{
		DIDResolutionMetadata: &ResolutionMetadata{
			ContentType: "application/did+ld+json",
			Error:       "notFound",
		},
	}, nil
}

// validateDIDFormat 验证DID格式
func (r *DIDResolver) validateDIDFormat(didStr string) error {
	if !strings.HasPrefix(didStr, "did:") {
		return utils.WrapValidationError(fmt.Errorf("DID必须以'did:'开头"), didStr)
	}

	parts := strings.Split(didStr, ":")
	if len(parts) < 3 {
		return utils.WrapValidationError(fmt.Errorf("DID格式无效，至少需要3个部分"), didStr)
	}

	return nil
}

// extractMethod 提取DID方法
func (r *DIDResolver) extractMethod(didStr string) string {
	parts := strings.Split(didStr, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// ResolveVerificationMethod 解析验证方法
func (r *DIDResolver) ResolveVerificationMethod(didURL string) (*types.VerificationMethod, error) {
	// 解析DID URL
	parts := strings.Split(didURL, "#")
	if len(parts) != 2 {
		return nil, utils.WrapValidationError(fmt.Errorf("无效的DID URL格式"), didURL)
	}

	didStr := parts[0]
	fragment := parts[1]

	// 解析DID文档
	result, err := r.Resolve(didStr)
	if err != nil {
		return nil, err
	}

	if result.DIDDocument == nil {
		return nil, utils.NewErrorWithDetails(utils.ErrorTypeNotFound, "DID_DOCUMENT_NOT_FOUND",
			"DID文档不存在", didStr)
	}

	// 查找验证方法
	for _, vm := range result.DIDDocument.VerificationMethod {
		if strings.HasSuffix(vm.ID, "#"+fragment) {
			return &vm, nil
		}
	}

	return nil, utils.NewErrorWithDetails(utils.ErrorTypeNotFound, "VERIFICATION_METHOD_NOT_FOUND",
		"验证方法不存在", fragment)
}

// GetSupportedMethods 获取支持的DID方法
func (r *DIDResolver) GetSupportedMethods() []string {
	return []string{"qlink"}
}

// IsSupported 检查是否支持指定的DID方法
func (r *DIDResolver) IsSupported(method string) bool {
	supportedMethods := r.GetSupportedMethods()
	for _, supported := range supportedMethods {
		if supported == method {
			return true
		}
	}
	return false
}

// resolveFromOffchain 从链下存储解析
func (r *DIDResolver) resolveFromOffchain(didStr string) (*types.DIDDocument, error) {
	if r.storage == nil {
		return nil, fmt.Errorf("存储管理器未初始化")
	}

	// 构造存储键
	storageKey := fmt.Sprintf("did:%s", didStr)

	// 从存储中获取DID文档
	data, err := r.storage.Get(storageKey)
	if err != nil {
		return nil, fmt.Errorf("从链下存储获取DID文档失败: %w", err)
	}

	// 反序列化DID文档
	var doc types.DIDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("反序列化DID文档失败: %w", err)
	}

	return &doc, nil
}
