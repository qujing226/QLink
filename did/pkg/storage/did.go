package storage

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// DIDStorage DID存储实现
type DIDStorage struct {
	interfaces.Storage
	mu sync.RWMutex

	// DID文档存储
	documents map[string]interface{}

	// DID历史记录
	history map[string][]interface{}

	// 控制器索引
	controllerIndex map[string][]string

	// 状态索引
	statusIndex map[string][]string

	// 计数器
	totalCount int64
}

// NewDIDStorage 创建新的DID存储实例
func NewDIDStorage(baseStorage interfaces.Storage) *DIDStorage {
	return &DIDStorage{
		Storage:         baseStorage,
		documents:       make(map[string]interface{}),
		history:         make(map[string][]interface{}),
		controllerIndex: make(map[string][]string),
		statusIndex:     make(map[string][]string),
	}
}

// GetDIDDocument 获取DID文档
func (ds *DIDStorage) GetDIDDocument(did string) (interface{}, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	doc, exists := ds.documents[did]
	if !exists {
		return nil, fmt.Errorf("DID文档不存在: %s", did)
	}

	return doc, nil
}

// PutDIDDocument 存储DID文档
func (ds *DIDStorage) PutDIDDocument(did string, doc interface{}) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// 检查是否是新文档
	isNew := false
	if _, exists := ds.documents[did]; !exists {
		isNew = true
		ds.totalCount++
	}

	// 存储文档
	ds.documents[did] = doc

	// 更新索引
	if err := ds.updateIndexes(did, doc, isNew); err != nil {
		return fmt.Errorf("更新索引失败: %w", err)
	}

	// 持久化到底层存储
	key := []byte(fmt.Sprintf("did:%s", did))
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("序列化DID文档失败: %w", err)
	}

	return ds.Storage.Put(key, data)
}

// DeleteDIDDocument 删除DID文档
func (ds *DIDStorage) DeleteDIDDocument(did string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// 检查文档是否存在
	if _, exists := ds.documents[did]; !exists {
		return fmt.Errorf("DID文档不存在: %s", did)
	}

	// 删除文档
	delete(ds.documents, did)
	ds.totalCount--

	// 清理索引
	ds.cleanupIndexes(did)

	// 从底层存储删除
	key := []byte(fmt.Sprintf("did:%s", did))
	return ds.Storage.Delete(key)
}

// GetDIDHistory 获取DID历史记录
func (ds *DIDStorage) GetDIDHistory(did string) ([]interface{}, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	history, exists := ds.history[did]
	if !exists {
		return []interface{}{}, nil
	}

	return history, nil
}

// PutDIDHistory 存储DID历史记录
func (ds *DIDStorage) PutDIDHistory(did string, historyEntry interface{}) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// 添加到历史记录
	if ds.history[did] == nil {
		ds.history[did] = make([]interface{}, 0)
	}
	ds.history[did] = append(ds.history[did], historyEntry)

	// 持久化到底层存储
	key := []byte(fmt.Sprintf("history:%s", did))
	data, err := json.Marshal(ds.history[did])
	if err != nil {
		return fmt.Errorf("序列化DID历史记录失败: %w", err)
	}

	return ds.Storage.Put(key, data)
}

// QueryDIDs 查询DID
func (ds *DIDStorage) QueryDIDs(query interface{}) ([]string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	queryMap, ok := query.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的查询格式")
	}

	var results []string

	// 根据控制器查询
	if controller, exists := queryMap["controller"]; exists {
		if controllerStr, ok := controller.(string); ok {
			if dids, exists := ds.controllerIndex[controllerStr]; exists {
				results = append(results, dids...)
			}
		}
	}

	// 根据状态查询
	if status, exists := queryMap["status"]; exists {
		if statusStr, ok := status.(string); ok {
			if dids, exists := ds.statusIndex[statusStr]; exists {
				if len(results) == 0 {
					results = append(results, dids...)
				} else {
					// 取交集
					results = ds.intersect(results, dids)
				}
			}
		}
	}

	// 如果没有指定查询条件，返回所有DID
	if len(results) == 0 && len(queryMap) == 0 {
		for did := range ds.documents {
			results = append(results, did)
		}
	}

	return results, nil
}

// GetDIDCount 获取DID数量
func (ds *DIDStorage) GetDIDCount() (int64, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return ds.totalCount, nil
}

// GetDIDsByController 根据控制器获取DID列表
func (ds *DIDStorage) GetDIDsByController(controller string) ([]string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	dids, exists := ds.controllerIndex[controller]
	if !exists {
		return []string{}, nil
	}

	return dids, nil
}

// LoadFromStorage 从底层存储加载数据
func (ds *DIDStorage) LoadFromStorage() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// 加载DID文档
	iter := ds.Storage.Iterator([]byte("did:"))
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		if len(key) > 4 { // "did:" prefix
			did := key[4:]

			var doc interface{}
			if err := json.Unmarshal(iter.Value(), &doc); err != nil {
				continue
			}

			ds.documents[did] = doc
			ds.totalCount++

			// 重建索引
			ds.updateIndexes(did, doc, true)
		}
	}

	// 加载历史记录
	historyIter := ds.Storage.Iterator([]byte("history:"))
	defer historyIter.Close()

	for historyIter.First(); historyIter.Valid(); historyIter.Next() {
		key := string(historyIter.Key())
		if len(key) > 8 { // "history:" prefix
			did := key[8:]

			var history []interface{}
			if err := json.Unmarshal(historyIter.Value(), &history); err != nil {
				continue
			}

			ds.history[did] = history
		}
	}

	return nil
}

// updateIndexes 更新索引
func (ds *DIDStorage) updateIndexes(did string, doc interface{}, isNew bool) error {
	docMap, ok := doc.(map[string]interface{})
	if !ok {
		return nil
	}

	// 如果不是新文档，先清理旧索引
	if !isNew {
		ds.cleanupIndexes(did)
	}

	// 更新控制器索引
	if controller, exists := docMap["controller"]; exists {
		if controllerStr, ok := controller.(string); ok {
			if ds.controllerIndex[controllerStr] == nil {
				ds.controllerIndex[controllerStr] = make([]string, 0)
			}
			ds.controllerIndex[controllerStr] = append(ds.controllerIndex[controllerStr], did)
		}
	}

	// 更新状态索引
	if status, exists := docMap["status"]; exists {
		if statusStr, ok := status.(string); ok {
			if ds.statusIndex[statusStr] == nil {
				ds.statusIndex[statusStr] = make([]string, 0)
			}
			ds.statusIndex[statusStr] = append(ds.statusIndex[statusStr], did)
		}
	}

	return nil
}

// cleanupIndexes 清理索引
func (ds *DIDStorage) cleanupIndexes(did string) {
	// 清理控制器索引
	for controller, dids := range ds.controllerIndex {
		ds.controllerIndex[controller] = ds.removeFromSlice(dids, did)
		if len(ds.controllerIndex[controller]) == 0 {
			delete(ds.controllerIndex, controller)
		}
	}

	// 清理状态索引
	for status, dids := range ds.statusIndex {
		ds.statusIndex[status] = ds.removeFromSlice(dids, did)
		if len(ds.statusIndex[status]) == 0 {
			delete(ds.statusIndex, status)
		}
	}
}

// removeFromSlice 从切片中移除指定元素
func (ds *DIDStorage) removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

// intersect 计算两个字符串切片的交集
func (ds *DIDStorage) intersect(slice1, slice2 []string) []string {
	set := make(map[string]bool)
	for _, item := range slice1 {
		set[item] = true
	}

	var result []string
	for _, item := range slice2 {
		if set[item] {
			result = append(result, item)
		}
	}

	return result
}

// GetDIDsByStatus 根据状态获取DID列表
func (ds *DIDStorage) GetDIDsByStatus(status string) ([]string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	dids, exists := ds.statusIndex[status]
	if !exists {
		return []string{}, nil
	}

	return dids, nil
}

// SearchDIDs 搜索DID文档
func (ds *DIDStorage) SearchDIDs(keyword string) ([]string, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var results []string
	keyword = strings.ToLower(keyword)

	for did, doc := range ds.documents {
		// 在DID标识符中搜索
		if strings.Contains(strings.ToLower(did), keyword) {
			results = append(results, did)
			continue
		}

		// 在文档内容中搜索
		if docMap, ok := doc.(map[string]interface{}); ok {
			docJSON, _ := json.Marshal(docMap)
			if strings.Contains(strings.ToLower(string(docJSON)), keyword) {
				results = append(results, did)
			}
		}
	}

	return results, nil
}

// GetDIDDocumentWithHistory 获取DID文档及其历史记录
func (ds *DIDStorage) GetDIDDocumentWithHistory(did string) (map[string]interface{}, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	doc, exists := ds.documents[did]
	if !exists {
		return nil, fmt.Errorf("DID文档不存在: %s", did)
	}

	history, _ := ds.history[did]

	result := map[string]interface{}{
		"document": doc,
		"history":  history,
		"metadata": map[string]interface{}{
			"did":         did,
			"last_update": time.Now(),
		},
	}

	return result, nil
}
