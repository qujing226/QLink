package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// StorageManager 存储管理器实现
type StorageManager struct {
	storages map[string]interfaces.Storage
	mu       sync.RWMutex
	running  bool
}

// NewStorageManager 创建新的存储管理器
func NewStorageManager() *StorageManager {
	return &StorageManager{
		storages: make(map[string]interfaces.Storage),
	}
}

// GetStorage 获取存储实例
func (sm *StorageManager) GetStorage(name string) (interfaces.Storage, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	storage, exists := sm.storages[name]
	if !exists {
		return nil, fmt.Errorf("存储不存在: %s", name)
	}

	return storage, nil
}

// RegisterStorage 注册存储实例
func (sm *StorageManager) RegisterStorage(name string, storage interfaces.Storage) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.storages[name]; exists {
		return fmt.Errorf("存储已存在: %s", name)
	}

	sm.storages[name] = storage
	return nil
}

// UnregisterStorage 注销存储实例
func (sm *StorageManager) UnregisterStorage(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	storage, exists := sm.storages[name]
	if !exists {
		return fmt.Errorf("存储不存在: %s", name)
	}

	// 关闭存储
	if err := storage.Close(); err != nil {
		return fmt.Errorf("关闭存储失败: %w", err)
	}

	delete(sm.storages, name)
	return nil
}

// GetAllStorages 获取所有存储实例
func (sm *StorageManager) GetAllStorages() map[string]interfaces.Storage {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 创建副本以避免并发问题
	storages := make(map[string]interfaces.Storage)
	for name, storage := range sm.storages {
		storages[name] = storage
	}

	return storages
}

// StartAll 启动所有存储
func (sm *StorageManager) StartAll(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("存储管理器已在运行")
	}

	for name, storage := range sm.storages {
		if err := storage.Open(); err != nil {
			return fmt.Errorf("启动存储 %s 失败: %w", name, err)
		}
	}

	sm.running = true
	return nil
}

// StopAll 停止所有存储
func (sm *StorageManager) StopAll() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return nil
	}

	var lastError error
	for name, storage := range sm.storages {
		if err := storage.Close(); err != nil {
			lastError = fmt.Errorf("停止存储 %s 失败: %w", name, err)
		}
	}

	sm.running = false
	return lastError
}

// HealthCheck 检查所有存储的健康状态
func (sm *StorageManager) HealthCheck() map[string]bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	health := make(map[string]bool)
	for name, storage := range sm.storages {
		// 尝试执行一个简单的操作来检查健康状态
		_, err := storage.Has([]byte("health_check"))
		if err != nil {
			// 记录健康检查失败的详细信息
			fmt.Printf("存储 %s 健康检查失败: %v\n", name, err)
			health[name] = false
		} else {
			health[name] = true
		}
	}

	return health
}

// GetStorageStats 获取存储统计信息
func (sm *StorageManager) GetStorageStats() map[string]interfaces.StorageStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]interfaces.StorageStats)
	for name, storage := range sm.storages {
		stats[name] = storage.Stats()
	}

	return stats
}

// CreateDefaultStorages 创建默认存储实例
func (sm *StorageManager) CreateDefaultStorages() error {
    // 创建持久化本地存储作为统一底层
    localStorage := NewLocalStorage(getDataDir())
    if err := sm.RegisterStorage("local", localStorage); err != nil {
        return fmt.Errorf("注册本地存储失败: %w", err)
    }

    // 创建区块链存储
    blockchainStorage := NewBlockchainStorage(localStorage)
    if err := sm.RegisterStorage("blockchain", blockchainStorage); err != nil {
        return fmt.Errorf("注册区块链存储失败: %w", err)
    }

    // 创建DID存储
    didStorage := NewDIDStorage(localStorage)
    if err := sm.RegisterStorage("did", didStorage); err != nil {
        return fmt.Errorf("注册DID存储失败: %w", err)
    }

    return nil
}

// GetBlockchainStorage 获取区块链存储
func (sm *StorageManager) GetBlockchainStorage() (*BlockchainStorage, error) {
	storage, err := sm.GetStorage("blockchain")
	if err != nil {
		return nil, err
	}

	blockchainStorage, ok := storage.(*BlockchainStorage)
	if !ok {
		return nil, fmt.Errorf("存储类型不匹配，期望 BlockchainStorage")
	}

	return blockchainStorage, nil
}

// GetDIDStorage 获取DID存储
func (sm *StorageManager) GetDIDStorage() (*DIDStorage, error) {
	storage, err := sm.GetStorage("did")
	if err != nil {
		return nil, err
	}

	didStorage, ok := storage.(*DIDStorage)
	if !ok {
		return nil, fmt.Errorf("存储类型不匹配，期望 DIDStorage")
	}

	return didStorage, nil
}

// BackupStorage 备份存储数据
func (sm *StorageManager) BackupStorage(name string, backupPath string) error {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    _, exists := sm.storages[name]
    if !exists {
        return fmt.Errorf("存储不存在: %s", name)
    }

    // TODO: 将备份数据直接写入文件系统
    // 暂时不实现内存备份，返回未实现
    return fmt.Errorf("备份功能尚未实现")
}

// RestoreStorage 恢复存储数据
func (sm *StorageManager) RestoreStorage(name string, backupPath string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	storage, exists := sm.storages[name]
	if !exists {
		return fmt.Errorf("存储不存在: %s", name)
	}

	// TODO: 从文件系统加载备份数据
	// 这里可以根据需要实现文件系统恢复逻辑

	_ = storage // 避免未使用变量警告

	return fmt.Errorf("恢复功能尚未实现")
}

// SyncStorages 同步存储数据
func (sm *StorageManager) SyncStorages(source, target string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sourceStorage, exists := sm.storages[source]
	if !exists {
		return fmt.Errorf("源存储不存在: %s", source)
	}

	targetStorage, exists := sm.storages[target]
	if !exists {
		return fmt.Errorf("目标存储不存在: %s", target)
	}

	// 同步数据
	iter := sourceStorage.Iterator([]byte(""))
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		if err := targetStorage.Put(iter.Key(), iter.Value()); err != nil {
			return fmt.Errorf("同步数据失败: %w", err)
		}
	}

	return nil
}

// Close 关闭存储管理器
func (sm *StorageManager) Close() error {
	return sm.StopAll()
}
