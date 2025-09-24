package storage

import (
	"fmt"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeMemory     StorageType = "memory"
	StorageTypeBlockchain StorageType = "blockchain"
	StorageTypeDID        StorageType = "did"
)

// StorageFactory 存储工厂实现
type StorageFactory struct{}

// NewStorageFactory 创建新的存储工厂
func NewStorageFactory() *StorageFactory {
	return &StorageFactory{}
}

// CreateStorage 创建存储实例
func (sf *StorageFactory) CreateStorage(storageType StorageType, config interfaces.StorageConfig) (interfaces.Storage, error) {
	switch storageType {
	case StorageTypeMemory:
		return sf.createMemoryStorage(config)
	case StorageTypeBlockchain:
		return sf.createBlockchainStorage(config)
	case StorageTypeDID:
		return sf.createDIDStorage(config)
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", storageType)
	}
}

// createMemoryStorage 创建内存存储
func (sf *StorageFactory) createMemoryStorage(config interfaces.StorageConfig) (interfaces.Storage, error) {
	return NewMemoryStorage(), nil
}

// createBlockchainStorage 创建区块链存储
func (sf *StorageFactory) createBlockchainStorage(config interfaces.StorageConfig) (interfaces.Storage, error) {
	// 创建底层存储
	baseStorage := NewMemoryStorage()
	
	// 创建区块链存储
	blockchainStorage := NewBlockchainStorage(baseStorage)
	
	return blockchainStorage, nil
}

// createDIDStorage 创建DID存储
func (sf *StorageFactory) createDIDStorage(config interfaces.StorageConfig) (interfaces.Storage, error) {
	// 创建底层存储
	baseStorage := NewMemoryStorage()
	
	// 创建DID存储
	didStorage := NewDIDStorage(baseStorage)
	
	return didStorage, nil
}

// GetSupportedTypes 获取支持的存储类型
func (sf *StorageFactory) GetSupportedTypes() []StorageType {
	return []StorageType{
		StorageTypeMemory,
		StorageTypeBlockchain,
		StorageTypeDID,
	}
}

// ValidateConfig 验证存储配置
func (sf *StorageFactory) ValidateConfig(storageType StorageType, config interfaces.StorageConfig) error {
	switch storageType {
	case StorageTypeMemory:
		return sf.validateMemoryConfig(config)
	case StorageTypeBlockchain:
		return sf.validateBlockchainConfig(config)
	case StorageTypeDID:
		return sf.validateDIDConfig(config)
	default:
		return fmt.Errorf("不支持的存储类型: %s", storageType)
	}
}

// validateMemoryConfig 验证内存存储配置
func (sf *StorageFactory) validateMemoryConfig(config interfaces.StorageConfig) error {
	// 内存存储不需要特殊配置验证
	return nil
}

// validateBlockchainConfig 验证区块链存储配置
func (sf *StorageFactory) validateBlockchainConfig(config interfaces.StorageConfig) error {
	// 区块链存储配置验证
	return nil
}

// validateDIDConfig 验证DID存储配置
func (sf *StorageFactory) validateDIDConfig(config interfaces.StorageConfig) error {
	// DID存储配置验证
	return nil
}

// CreateStorageWithDefaults 使用默认配置创建存储
func (sf *StorageFactory) CreateStorageWithDefaults(storageType StorageType) (interfaces.Storage, error) {
	// 创建默认配置
	config := NewDefaultStorageConfig(string(storageType))
	
	return sf.CreateStorage(storageType, config)
}

// DefaultStorageConfig 默认存储配置
type DefaultStorageConfig struct {
	storageType string
}

// NewDefaultStorageConfig 创建默认存储配置
func NewDefaultStorageConfig(storageType string) *DefaultStorageConfig {
	return &DefaultStorageConfig{
		storageType: storageType,
	}
}

// GetType 获取存储类型
func (c *DefaultStorageConfig) GetType() string {
	return c.storageType
}

// GetPath 获取存储路径
func (c *DefaultStorageConfig) GetPath() string {
	return ""
}

// GetConfig 获取配置映射
func (c *DefaultStorageConfig) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"type":               c.storageType,
		"path":               c.GetPath(),
		"cache_size":         1000,
		"compression":        false,
		"encryption":         false,
		"backup_enabled":     false,
		"sync_enabled":       false,
	}
}

// Validate 验证配置
func (c *DefaultStorageConfig) Validate() error {
	return nil
}

// BatchCreateStorages 批量创建存储实例
func (sf *StorageFactory) BatchCreateStorages(configs map[string]FactoryStorageConfig) (map[string]interfaces.Storage, error) {
	storages := make(map[string]interfaces.Storage)
	
	for name, config := range configs {
		storage, err := sf.CreateStorage(config.Type, config.Config)
		if err != nil {
			// 清理已创建的存储
			for _, s := range storages {
				if closer, ok := s.(interface{ Close() error }); ok {
					closer.Close()
				}
			}
			return nil, fmt.Errorf("创建存储 %s 失败: %w", name, err)
		}
		
		storages[name] = storage
	}
	
	return storages, nil
}

// StorageConfig 存储配置结构
type FactoryStorageConfig struct {
	Type   StorageType
	Config interfaces.StorageConfig
}

// CreateStorageManager 创建配置好的存储管理器
func (sf *StorageFactory) CreateStorageManager(configs map[string]FactoryStorageConfig) (*StorageManager, error) {
	manager := NewStorageManager()
	
	for name, config := range configs {
		storage, err := sf.CreateStorage(config.Type, config.Config)
		if err != nil {
			return nil, fmt.Errorf("创建存储 %s 失败: %w", name, err)
		}
		
		if err := manager.RegisterStorage(name, storage); err != nil {
			storage.Close()
			return nil, fmt.Errorf("注册存储 %s 失败: %w", name, err)
		}
	}
	
	return manager, nil
}

// CreateDefaultStorageManager 创建默认存储管理器
func (sf *StorageFactory) CreateDefaultStorageManager() (*StorageManager, error) {
	// 创建默认存储配置
	configs := map[string]FactoryStorageConfig{
		"memory": {
			Type:   StorageTypeMemory,
			Config: NewDefaultStorageConfig(string(StorageTypeMemory)),
		},
		"blockchain": {
			Type:   StorageTypeBlockchain,
			Config: NewDefaultStorageConfig(string(StorageTypeBlockchain)),
		},
		"did": {
			Type:   StorageTypeDID,
			Config: NewDefaultStorageConfig(string(StorageTypeDID)),
		},
	}
	
	return sf.CreateStorageManager(configs)
}