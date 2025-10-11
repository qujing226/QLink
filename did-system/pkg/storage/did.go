package storage

import (
    "encoding/json"
    "fmt"
    "sync"

    "github.com/qujing226/QLink/pkg/interfaces"
)

// DIDStorage 仅使用底层持久化存储的实现
type DIDStorage struct {
    interfaces.Storage
    mu sync.RWMutex
}

// NewDIDStorage 创建新的DID存储实例（仅持久化）
func NewDIDStorage(baseStorage interfaces.Storage) *DIDStorage {
    return &DIDStorage{
        Storage: baseStorage,
    }
}

// GetDIDDocument 获取DID文档（从持久化存储）
func (ds *DIDStorage) GetDIDDocument(did string) (interface{}, error) {
    ds.mu.RLock()
    defer ds.mu.RUnlock()

    key := []byte(fmt.Sprintf("did:%s", did))
    data, err := ds.Storage.Get(key)
    if err != nil {
        return nil, fmt.Errorf("DID文档不存在: %s", did)
    }

    var doc interface{}
    if err := json.Unmarshal(data, &doc); err != nil {
        return nil, fmt.Errorf("解析DID文档失败: %w", err)
    }
    return doc, nil
}

// PutDIDDocument 存储DID文档（写入持久化存储）
func (ds *DIDStorage) PutDIDDocument(did string, doc interface{}) error {
    ds.mu.Lock()
    defer ds.mu.Unlock()

    key := []byte(fmt.Sprintf("did:%s", did))
    data, err := json.Marshal(doc)
    if err != nil {
        return fmt.Errorf("序列化DID文档失败: %w", err)
    }
    return ds.Storage.Put(key, data)
}

// DeleteDIDDocument 删除DID文档（从持久化存储删除）
func (ds *DIDStorage) DeleteDIDDocument(did string) error {
    ds.mu.Lock()
    defer ds.mu.Unlock()

    key := []byte(fmt.Sprintf("did:%s", did))
    return ds.Storage.Delete(key)
}

// GetDIDHistory 获取DID历史记录（从持久化存储）
func (ds *DIDStorage) GetDIDHistory(did string) ([]interface{}, error) {
    ds.mu.RLock()
    defer ds.mu.RUnlock()

    key := []byte(fmt.Sprintf("history:%s", did))
    data, err := ds.Storage.Get(key)
    if err != nil {
        // 不存在则返回空列表
        return []interface{}{}, nil
    }

    var history []interface{}
    if err := json.Unmarshal(data, &history); err != nil {
        return nil, fmt.Errorf("解析DID历史记录失败: %w", err)
    }
    return history, nil
}

// PutDIDHistory 存储DID历史记录（写入持久化存储）
func (ds *DIDStorage) PutDIDHistory(did string, history interface{}) error {
    ds.mu.Lock()
    defer ds.mu.Unlock()

    // 如果传入的是单条记录，则合并到现有历史
    var merged []interface{}
    switch h := history.(type) {
    case []interface{}:
        merged = h
    default:
        existing, _ := ds.GetDIDHistory(did)
        merged = append(existing, h)
    }

    key := []byte(fmt.Sprintf("history:%s", did))
    data, err := json.Marshal(merged)
    if err != nil {
        return fmt.Errorf("序列化DID历史记录失败: %w", err)
    }
    return ds.Storage.Put(key, data)
}

// QueryDIDs 遍历持久化存储，根据查询条件筛选 DID
func (ds *DIDStorage) QueryDIDs(query interface{}) ([]string, error) {
    ds.mu.RLock()
    defer ds.mu.RUnlock()

    queryMap, ok := query.(map[string]interface{})
    if !ok {
        queryMap = map[string]interface{}{}
    }

    it := ds.Storage.Iterator([]byte("did:"))
    defer it.Close()

    var results []string
    for it.First(); it.Valid(); it.Next() {
        key := string(it.Key())
        if len(key) <= 4 {
            continue
        }
        did := key[4:]

        // 如果没有筛选条件，直接加入
        if len(queryMap) == 0 {
            results = append(results, did)
            continue
        }

        // 有筛选条件时解析文档做过滤
        var doc map[string]interface{}
        if err := json.Unmarshal(it.Value(), &doc); err != nil {
            continue
        }

        // controller 过滤
        if controller, ok := queryMap["controller"].(string); ok && controller != "" {
            ctrlVal, _ := doc["controller"].(string)
            if ctrlVal != controller {
                continue
            }
        }

        // status 过滤
        if status, ok := queryMap["status"].(string); ok && status != "" {
            stVal, _ := doc["status"].(string)
            if stVal != status {
                continue
            }
        }

        results = append(results, did)
    }

    return results, nil
}

// GetDIDCount 统计持久化存储中的 DID 数量
func (ds *DIDStorage) GetDIDCount() (int64, error) {
    ds.mu.RLock()
    defer ds.mu.RUnlock()

    it := ds.Storage.Iterator([]byte("did:"))
    defer it.Close()

    var count int64
    for it.First(); it.Valid(); it.Next() {
        count++
    }
    return count, nil
}

// GetDIDsByController 遍历持久化存储按控制器筛选
func (ds *DIDStorage) GetDIDsByController(controller string) ([]string, error) {
    ds.mu.RLock()
    defer ds.mu.RUnlock()

    if controller == "" {
        return []string{}, nil
    }

    it := ds.Storage.Iterator([]byte("did:"))
    defer it.Close()

    var results []string
    for it.First(); it.Valid(); it.Next() {
        key := string(it.Key())
        if len(key) <= 4 {
            continue
        }
        did := key[4:]
        var doc map[string]interface{}
        if err := json.Unmarshal(it.Value(), &doc); err != nil {
            continue
        }
        if ctrlVal, _ := doc["controller"].(string); ctrlVal == controller {
            results = append(results, did)
        }
    }
    return results, nil
}
