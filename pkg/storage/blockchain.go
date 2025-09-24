package storage

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// BlockchainStorage 区块链存储实现
type BlockchainStorage struct {
	interfaces.Storage
	mu sync.RWMutex

	// 区块存储
	blocks       map[uint64]interface{}
	blocksByHash map[string]interface{}
	latestHeight uint64

	// 交易存储
	transactions map[string]interface{}

	// 状态存储
	states map[string]interface{}

	// 索引
	indexes map[string]map[string][]interface{}
}

// NewBlockchainStorage 创建新的区块链存储实例
func NewBlockchainStorage(baseStorage interfaces.Storage) *BlockchainStorage {
	return &BlockchainStorage{
		Storage:      baseStorage,
		blocks:       make(map[uint64]interface{}),
		blocksByHash: make(map[string]interface{}),
		transactions: make(map[string]interface{}),
		states:       make(map[string]interface{}),
		indexes:      make(map[string]map[string][]interface{}),
	}
}

// GetBlock 根据高度获取区块
func (bs *BlockchainStorage) GetBlock(height uint64) (interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	block, exists := bs.blocks[height]
	if !exists {
		return nil, fmt.Errorf("区块不存在，高度: %d", height)
	}

	return block, nil
}

// PutBlock 存储区块
func (bs *BlockchainStorage) PutBlock(height uint64, block interface{}) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// 存储区块
	bs.blocks[height] = block
	
	// 如果区块有哈希字段，也按哈希存储
	if blockMap, ok := block.(map[string]interface{}); ok {
		if hash, exists := blockMap["hash"]; exists {
			if hashStr, ok := hash.(string); ok {
				bs.blocksByHash[hashStr] = block
			}
		}
	}

	// 更新最新高度
	if height > bs.latestHeight {
		bs.latestHeight = height
	}

	// 持久化到底层存储
	key := []byte(fmt.Sprintf("block:%d", height))
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("序列化区块失败: %w", err)
	}

	return bs.Storage.Put(key, data)
}

// GetLatestBlock 获取最新区块
func (bs *BlockchainStorage) GetLatestBlock() (interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if bs.latestHeight == 0 {
		return nil, fmt.Errorf("没有区块")
	}

	return bs.blocks[bs.latestHeight], nil
}

// GetBlockHeight 获取区块链高度
func (bs *BlockchainStorage) GetBlockHeight() (uint64, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	return bs.latestHeight, nil
}

// GetTransaction 获取交易
func (bs *BlockchainStorage) GetTransaction(hash string) (interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	tx, exists := bs.transactions[hash]
	if !exists {
		return nil, fmt.Errorf("交易不存在: %s", hash)
	}

	return tx, nil
}

// PutTransaction 存储交易
func (bs *BlockchainStorage) PutTransaction(hash string, tx interface{}) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// 存储交易
	bs.transactions[hash] = tx

	// 持久化到底层存储
	key := []byte(fmt.Sprintf("tx:%s", hash))
	data, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("序列化交易失败: %w", err)
	}

	return bs.Storage.Put(key, data)
}

// GetState 获取状态
func (bs *BlockchainStorage) GetState(key string) (interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	state, exists := bs.states[key]
	if !exists {
		return nil, fmt.Errorf("状态不存在: %s", key)
	}

	return state, nil
}

// PutState 存储状态
func (bs *BlockchainStorage) PutState(key string, value interface{}) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// 存储状态
	bs.states[key] = value

	// 持久化到底层存储
	stateKey := []byte(fmt.Sprintf("state:%s", key))
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化状态失败: %w", err)
	}

	return bs.Storage.Put(stateKey, data)
}

// CreateIndex 创建索引
func (bs *BlockchainStorage) CreateIndex(name string, fields []string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.indexes[name] == nil {
		bs.indexes[name] = make(map[string][]interface{})
	}

	return nil
}

// QueryByIndex 根据索引查询
func (bs *BlockchainStorage) QueryByIndex(name string, query interface{}) ([]interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	index, exists := bs.indexes[name]
	if !exists {
		return nil, fmt.Errorf("索引不存在: %s", name)
	}

	queryStr := fmt.Sprintf("%v", query)
	results, exists := index[queryStr]
	if !exists {
		return []interface{}{}, nil
	}

	return results, nil
}

// LoadFromStorage 从底层存储加载数据
func (bs *BlockchainStorage) LoadFromStorage() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// 加载区块
	iter := bs.Storage.Iterator([]byte("block:"))
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := string(iter.Key())
		if len(key) > 6 { // "block:" prefix
			heightStr := key[6:]
			height, err := strconv.ParseUint(heightStr, 10, 64)
			if err != nil {
				continue
			}

			var block interface{}
			if err := json.Unmarshal(iter.Value(), &block); err != nil {
				continue
			}

			bs.blocks[height] = block
			if height > bs.latestHeight {
				bs.latestHeight = height
			}

			// 如果区块有哈希，也存储到哈希映射中
			if blockMap, ok := block.(map[string]interface{}); ok {
				if hash, exists := blockMap["hash"]; exists {
					if hashStr, ok := hash.(string); ok {
						bs.blocksByHash[hashStr] = block
					}
				}
			}
		}
	}

	// 加载交易
	txIter := bs.Storage.Iterator([]byte("tx:"))
	defer txIter.Close()

	for txIter.First(); txIter.Valid(); txIter.Next() {
		key := string(txIter.Key())
		if len(key) > 3 { // "tx:" prefix
			hash := key[3:]

			var tx interface{}
			if err := json.Unmarshal(txIter.Value(), &tx); err != nil {
				continue
			}

			bs.transactions[hash] = tx
		}
	}

	// 加载状态
	stateIter := bs.Storage.Iterator([]byte("state:"))
	defer stateIter.Close()

	for stateIter.First(); stateIter.Valid(); stateIter.Next() {
		key := string(stateIter.Key())
		if len(key) > 6 { // "state:" prefix
			stateKey := key[6:]

			var state interface{}
			if err := json.Unmarshal(stateIter.Value(), &state); err != nil {
				continue
			}

			bs.states[stateKey] = state
		}
	}

	return nil
}

// GetBlockByHash 根据哈希获取区块
func (bs *BlockchainStorage) GetBlockByHash(hash string) (interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	block, exists := bs.blocksByHash[hash]
	if !exists {
		return nil, fmt.Errorf("区块不存在，哈希: %s", hash)
	}

	return block, nil
}

// GetAllBlocks 获取所有区块
func (bs *BlockchainStorage) GetAllBlocks() (map[uint64]interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// 创建副本以避免并发问题
	blocks := make(map[uint64]interface{})
	for height, block := range bs.blocks {
		blocks[height] = block
	}

	return blocks, nil
}

// GetAllTransactions 获取所有交易
func (bs *BlockchainStorage) GetAllTransactions() (map[string]interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// 创建副本以避免并发问题
	transactions := make(map[string]interface{})
	for hash, tx := range bs.transactions {
		transactions[hash] = tx
	}

	return transactions, nil
}

// GetAllStates 获取所有状态
func (bs *BlockchainStorage) GetAllStates() (map[string]interface{}, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	// 创建副本以避免并发问题
	states := make(map[string]interface{})
	for key, state := range bs.states {
		states[key] = state
	}

	return states, nil
}