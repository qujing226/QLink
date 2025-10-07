package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	data   map[string][]byte
	mu     sync.RWMutex
	stats  interfaces.StorageStats
	closed bool
}

// NewMemoryStorage 创建新的内存存储实例
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
		stats: interfaces.StorageStats{
			LastUpdated: time.Now(),
		},
	}
}

// Get 获取数据
func (ms *MemoryStorage) Get(key []byte) ([]byte, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.closed {
		return nil, fmt.Errorf("存储已关闭")
	}

	start := time.Now()
	defer func() {
		ms.updateStats(func(stats *interfaces.StorageStats) {
			stats.ReadCount++
			stats.AvgReadTime = (stats.AvgReadTime + time.Since(start)) / 2
		})
	}()

	value, exists := ms.data[string(key)]
	if !exists {
		return nil, fmt.Errorf("键不存在: %s", string(key))
	}

	return value, nil
}

// Put 存储数据
func (ms *MemoryStorage) Put(key, value []byte) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.closed {
		return fmt.Errorf("存储已关闭")
	}

	start := time.Now()
	defer func() {
		ms.updateStats(func(stats *interfaces.StorageStats) {
			stats.WriteCount++
			stats.AvgWriteTime = (stats.AvgWriteTime + time.Since(start)) / 2
		})
	}()

	ms.data[string(key)] = value
	return nil
}

// Delete 删除数据
func (ms *MemoryStorage) Delete(key []byte) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.closed {
		return fmt.Errorf("存储已关闭")
	}

	defer func() {
		ms.updateStats(func(stats *interfaces.StorageStats) {
			stats.DeleteCount++
		})
	}()

	delete(ms.data, string(key))
	return nil
}

// Has 检查键是否存在
func (ms *MemoryStorage) Has(key []byte) (bool, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if ms.closed {
		return false, fmt.Errorf("存储已关闭")
	}

	_, exists := ms.data[string(key)]
	return exists, nil
}

// Batch 创建批量操作
func (ms *MemoryStorage) Batch() interfaces.Batch {
	return &MemoryBatch{
		storage: ms,
		ops:     make([]batchOp, 0),
	}
}

// Iterator 创建迭代器
func (ms *MemoryStorage) Iterator(prefix []byte) interfaces.Iterator {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	keys := make([]string, 0)
	prefixStr := string(prefix)

	for key := range ms.data {
		if len(prefix) == 0 || (len(key) >= len(prefixStr) && key[:len(prefixStr)] == prefixStr) {
			keys = append(keys, key)
		}
	}

	return &MemoryIterator{
		storage: ms,
		keys:    keys,
		index:   -1,
	}
}

// NewTransaction 创建事务
func (ms *MemoryStorage) NewTransaction() (interfaces.Transaction, error) {
	return &MemoryTransaction{
		storage: ms,
		ops:     make(map[string][]byte),
		deleted: make(map[string]bool),
	}, nil
}

// Open 打开存储
func (ms *MemoryStorage) Open() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.closed = false
	return nil
}

// Close 关闭存储
func (ms *MemoryStorage) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.closed = true
	return nil
}

// Stats 获取统计信息
func (ms *MemoryStorage) Stats() interfaces.StorageStats {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	stats := ms.stats
	stats.KeyCount = int64(len(ms.data))
	stats.LastUpdated = time.Now()

	// 计算总大小
	var totalSize int64
	for _, value := range ms.data {
		totalSize += int64(len(value))
	}
	stats.TotalSize = totalSize
	stats.UsedSize = totalSize

	return stats
}

// updateStats 更新统计信息
func (ms *MemoryStorage) updateStats(updater func(*interfaces.StorageStats)) {
	updater(&ms.stats)
	ms.stats.LastUpdated = time.Now()
}

// MemoryBatch 内存批量操作
type MemoryBatch struct {
	storage *MemoryStorage
	ops     []batchOp
}

type batchOp struct {
	key    string
	value  []byte
	delete bool
}

func (mb *MemoryBatch) Put(key, value []byte) error {
	mb.ops = append(mb.ops, batchOp{
		key:   string(key),
		value: value,
	})
	return nil
}

func (mb *MemoryBatch) Delete(key []byte) error {
	mb.ops = append(mb.ops, batchOp{
		key:    string(key),
		delete: true,
	})
	return nil
}

func (mb *MemoryBatch) Write() error {
	mb.storage.mu.Lock()
	defer mb.storage.mu.Unlock()

	if mb.storage.closed {
		return fmt.Errorf("存储已关闭")
	}

	for _, op := range mb.ops {
		if op.delete {
			delete(mb.storage.data, op.key)
		} else {
			mb.storage.data[op.key] = op.value
		}
	}

	return nil
}

func (mb *MemoryBatch) Reset() {
	mb.ops = mb.ops[:0]
}

func (mb *MemoryBatch) Size() int {
	return len(mb.ops)
}

// MemoryIterator 内存迭代器
type MemoryIterator struct {
	storage *MemoryStorage
	keys    []string
	index   int
}

func (mi *MemoryIterator) Valid() bool {
	return mi.index >= 0 && mi.index < len(mi.keys)
}

func (mi *MemoryIterator) Key() []byte {
	if !mi.Valid() {
		return nil
	}
	return []byte(mi.keys[mi.index])
}

func (mi *MemoryIterator) Value() []byte {
	if !mi.Valid() {
		return nil
	}
	mi.storage.mu.RLock()
	defer mi.storage.mu.RUnlock()
	return mi.storage.data[mi.keys[mi.index]]
}

func (mi *MemoryIterator) Next() {
	mi.index++
}

func (mi *MemoryIterator) Prev() {
	mi.index--
}

func (mi *MemoryIterator) Seek(key []byte) {
	keyStr := string(key)
	for i, k := range mi.keys {
		if k >= keyStr {
			mi.index = i
			return
		}
	}
	mi.index = len(mi.keys)
}

func (mi *MemoryIterator) First() {
	mi.index = 0
}

func (mi *MemoryIterator) Last() {
	mi.index = len(mi.keys) - 1
}

func (mi *MemoryIterator) Close() error {
	return nil
}

// MemoryTransaction 内存事务
type MemoryTransaction struct {
	storage *MemoryStorage
	ops     map[string][]byte
	deleted map[string]bool
}

func (mt *MemoryTransaction) Get(key []byte) ([]byte, error) {
	keyStr := string(key)

	// 检查事务中的操作
	if mt.deleted[keyStr] {
		return nil, fmt.Errorf("键不存在: %s", keyStr)
	}

	if value, exists := mt.ops[keyStr]; exists {
		return value, nil
	}

	// 从存储中获取
	return mt.storage.Get(key)
}

func (mt *MemoryTransaction) Put(key, value []byte) error {
	keyStr := string(key)
	mt.ops[keyStr] = value
	delete(mt.deleted, keyStr)
	return nil
}

func (mt *MemoryTransaction) Delete(key []byte) error {
	keyStr := string(key)
	mt.deleted[keyStr] = true
	delete(mt.ops, keyStr)
	return nil
}

func (mt *MemoryTransaction) Commit() error {
	mt.storage.mu.Lock()
	defer mt.storage.mu.Unlock()

	if mt.storage.closed {
		return fmt.Errorf("存储已关闭")
	}

	// 应用删除操作
	for key := range mt.deleted {
		delete(mt.storage.data, key)
	}

	// 应用写入操作
	for key, value := range mt.ops {
		mt.storage.data[key] = value
	}

	return nil
}

func (mt *MemoryTransaction) Rollback() error {
	// 清空事务操作
	mt.ops = make(map[string][]byte)
	mt.deleted = make(map[string]bool)
	return nil
}
