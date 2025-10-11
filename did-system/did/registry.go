package did

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "strings"
    "sync"
    "time"

    "github.com/qujing226/QLink/pkg/storage"
    "github.com/qujing226/QLink/pkg/types"
)

// DIDRegistry DID注册表
type DIDRegistry struct {
    blockchain BlockchainInterface           // 区块链接口
    storeMgr   *storage.StorageManager        // 持久化存储管理器
    mu         sync.RWMutex
}

// RegisterRequest DID注册请求
type RegisterRequest struct {
    DID                string                     `json:"did"`
    VerificationMethod []types.VerificationMethod `json:"verificationMethod"`
    Service            []types.Service            `json:"service,omitempty"`
}

// UpdateRequest DID更新请求
type UpdateRequest struct {
    DID                string                     `json:"did"`
    VerificationMethod []types.VerificationMethod `json:"verificationMethod,omitempty"`
    Service            []types.Service            `json:"service,omitempty"`
    Proof              *types.Proof               `json:"proof"`
}

// NewDIDRegistry 创建DID注册表实例
func NewDIDRegistry(blockchain BlockchainInterface, storeMgr *storage.StorageManager) *DIDRegistry {
    return &DIDRegistry{
        blockchain: blockchain,
        storeMgr:   storeMgr,
    }
}

// Register 注册DID
func (r *DIDRegistry) Register(req *RegisterRequest) (*types.DIDDocument, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

	// 验证DID格式
	if err := r.validateDID(req.DID); err != nil {
		return nil, err
	}

    // 检查DID是否已存在（持久化存储）
    if r.storeMgr != nil {
        if didStore, err := r.storeMgr.GetDIDStorage(); err == nil {
            if _, gErr := didStore.GetDIDDocument(req.DID); gErr == nil {
                return nil, &DIDError{
                    Type:    ErrorTypeConflict,
                    Code:    "DID_EXISTS",
                    Message: "DID已存在",
                    Details: req.DID,
                }
            }
        }
    }

	// 创建DID文档
	now := time.Now()
	doc := &types.DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/suites/jws-2020/v1",
		},
		ID:                 req.DID,
		VerificationMethod: req.VerificationMethod,
		Service:            req.Service,
		Created:            &now,
		Updated:            &now,
		Status:             "active",
	}

	// 设置认证和断言方法
	for _, vm := range req.VerificationMethod {
		doc.Authentication = append(doc.Authentication, vm.ID)
		doc.AssertionMethod = append(doc.AssertionMethod, vm.ID)
	}

    // 持久化到DID存储
    if r.storeMgr != nil {
        if didStore, err := r.storeMgr.GetDIDStorage(); err == nil {
            if err := didStore.PutDIDDocument(req.DID, doc); err != nil {
                log.Printf("持久化DID文档失败: %v", err)
            }
        } else {
            log.Printf("获取DID存储失败: %v", err)
        }
    }

	// 提交DID注册交易到区块链
	if r.blockchain != nil {
		tx, err := r.blockchain.RegisterDID(context.Background(), doc)
		if err != nil {
			// 区块链注册失败时，记录错误但不阻止DID注册
			log.Printf("区块链注册失败，但DID已在内存中注册: %s, 错误: %v", doc.ID, err)
		} else {
			log.Printf("DID注册交易已提交到区块链: %s, 交易哈希: %s", doc.ID, tx.Hash)
		}
    }

	return doc, nil
}

// Resolve 解析DID
func (r *DIDRegistry) Resolve(didStr string) (*types.DIDDocument, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    if r.storeMgr != nil {
        didStore, err := r.storeMgr.GetDIDStorage()
        if err != nil {
            return nil, fmt.Errorf("获取DID存储失败: %w", err)
        }
        iface, err := didStore.GetDIDDocument(didStr)
        if err == nil && iface != nil {
            raw, mErr := json.Marshal(iface)
            if mErr == nil {
                var doc types.DIDDocument
                if uErr := json.Unmarshal(raw, &doc); uErr == nil {
                    return &doc, nil
                }
            }
        }
    }

    return nil, &DIDError{
        Type:    ErrorTypeNotFound,
        Code:    "DID_NOT_FOUND",
        Message: "DID不存在",
        Details: didStr,
    }
}

// Update 更新DID
func (r *DIDRegistry) Update(req *UpdateRequest) (*types.DIDDocument, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

    var doc *types.DIDDocument
    if r.storeMgr != nil {
        didStore, err := r.storeMgr.GetDIDStorage()
        if err != nil {
            return nil, fmt.Errorf("获取DID存储失败: %w", err)
        }
        iface, gErr := didStore.GetDIDDocument(req.DID)
        if gErr != nil || iface == nil {
            return nil, &DIDError{
                Type:    ErrorTypeNotFound,
                Code:    "DID_NOT_FOUND",
                Message: "DID不存在",
                Details: req.DID,
            }
        }
        raw, mErr := json.Marshal(iface)
        if mErr != nil {
            return nil, fmt.Errorf("序列化DID文档失败: %w", mErr)
        }
        var loaded types.DIDDocument
        if uErr := json.Unmarshal(raw, &loaded); uErr != nil {
            return nil, fmt.Errorf("反序列化DID文档失败: %w", uErr)
        }
        doc = &loaded
    } else {
        return nil, fmt.Errorf("存储管理器未初始化")
    }

	if doc.Status == "revoked" {
		return nil, &DIDError{
			Type:    ErrorTypeValidation,
			Code:    "DID_REVOKED",
			Message: "DID已被撤销",
			Details: req.DID,
		}
	}

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

	now := time.Now()
	doc.Updated = &now
	doc.Proof = req.Proof

    // 提交DID更新交易到区块链
    if r.blockchain != nil {
        tx, err := r.blockchain.UpdateDID(context.Background(), req.DID, doc, req.Proof)
        if err != nil {
            log.Printf("区块链更新失败，但继续内存更新: %v", err)
        } else {
            log.Printf("DID更新交易已提交到区块链: %s, 交易哈希: %s", req.DID, tx.Hash)
        }
    }

    // 持久化到DID存储
    if r.storeMgr != nil {
        if didStore, err := r.storeMgr.GetDIDStorage(); err == nil {
            if err := didStore.PutDIDDocument(req.DID, doc); err != nil {
                log.Printf("持久化更新DID失败: %v", err)
            }
        } else {
            log.Printf("获取DID存储失败: %v", err)
        }
    }

	return doc, nil
}

// Revoke 撤销DID
func (r *DIDRegistry) Revoke(didStr string, proof *types.Proof) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    var doc *types.DIDDocument
    if r.storeMgr != nil {
        didStore, err := r.storeMgr.GetDIDStorage()
        if err != nil {
            return fmt.Errorf("获取DID存储失败: %w", err)
        }
        iface, gErr := didStore.GetDIDDocument(didStr)
        if gErr != nil || iface == nil {
            return &DIDError{
                Type:    ErrorTypeNotFound,
                Code:    "DID_NOT_FOUND",
                Message: "DID不存在",
                Details: didStr,
            }
        }
        raw, mErr := json.Marshal(iface)
        if mErr != nil {
            return fmt.Errorf("序列化DID文档失败: %w", mErr)
        }
        var loaded types.DIDDocument
        if uErr := json.Unmarshal(raw, &loaded); uErr != nil {
            return fmt.Errorf("反序列化DID文档失败: %w", uErr)
        }
        doc = &loaded
    } else {
        return fmt.Errorf("存储管理器未初始化")
    }

	if doc.Status == "revoked" {
		return &DIDError{
			Type:    ErrorTypeValidation,
			Code:    "DID_ALREADY_REVOKED",
			Message: "DID已被撤销",
			Details: didStr,
		}
	}

	// 更新状态
	doc.Status = "revoked"
	now := time.Now()
	doc.Updated = &now
	doc.Proof = proof

    // 提交DID撤销交易到区块链
    if r.blockchain != nil {
        tx, err := r.blockchain.RevokeDID(context.Background(), didStr, proof)
        if err != nil {
            log.Printf("区块链撤销失败，但继续内存撤销: %v", err)
        } else {
            log.Printf("DID撤销交易已提交到区块链: %s, 交易哈希: %s", didStr, tx.Hash)
        }
    }

    // 持久化更新撤销状态
    if r.storeMgr != nil {
        if didStore, err := r.storeMgr.GetDIDStorage(); err == nil {
            if err := didStore.PutDIDDocument(didStr, doc); err != nil {
                log.Printf("持久化撤销DID失败: %v", err)
            }
        }
    }

	return nil
}

// List 列出所有DID
func (r *DIDRegistry) List() ([]*types.DIDDocument, error) {
    r.mu.RLock()
    r.mu.RUnlock()

    if r.storeMgr != nil {
        didStore, err := r.storeMgr.GetDIDStorage()
        if err != nil {
            return nil, fmt.Errorf("获取DID存储失败: %w", err)
        }
        ids, err := didStore.QueryDIDs(map[string]interface{}{})
        if err != nil {
            return nil, fmt.Errorf("查询DID失败: %w", err)
        }
        var docs []*types.DIDDocument
        for _, id := range ids {
            iface, gErr := didStore.GetDIDDocument(id)
            if gErr != nil || iface == nil {
                continue
            }
            raw, mErr := json.Marshal(iface)
            if mErr != nil {
                continue
            }
            var doc types.DIDDocument
            if uErr := json.Unmarshal(raw, &doc); uErr != nil {
                continue
            }
            docs = append(docs, &doc)
        }
        return docs, nil
    }

    return []*types.DIDDocument{}, nil
}

// validateDID 验证DID格式
func (r *DIDRegistry) validateDID(didStr string) error {
	if didStr == "" {
		return &DIDError{
			Type:    ErrorTypeValidation,
			Code:    "INVALID_DID_FORMAT",
			Message: "DID不能为空",
		}
	}

	if !strings.HasPrefix(didStr, "did:") {
		return &DIDError{
			Type:    ErrorTypeValidation,
			Code:    "INVALID_DID_FORMAT",
			Message: "DID必须以'did:'开头",
		}
	}

	parts := strings.Split(didStr, ":")
	if len(parts) < 3 {
		return &DIDError{
			Type:    ErrorTypeValidation,
			Code:    "INVALID_DID_FORMAT",
			Message: "DID格式无效，应为 did:method:identifier",
		}
	}

	return nil
}

// FromJSON 从JSON创建DID文档
func FromJSON(data []byte) (*types.DIDDocument, error) {
	var doc types.DIDDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

// DIDError DID错误类型
type DIDError struct {
	Type    string      `json:"type"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func (e *DIDError) Error() string {
	return e.Message
}

// 错误类型常量
const (
	ErrorTypeValidation = "validation"
	ErrorTypeNotFound   = "not_found"
	ErrorTypeConflict   = "conflict"
	ErrorTypeBlockchain = "blockchain"
)
