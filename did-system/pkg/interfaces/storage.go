package interfaces

import (
	"context"
	"time"
)

// Storage 通用存储接口
type Storage interface {
	// 基本操作
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
	Has(key []byte) (bool, error)

	// 批量操作
	Batch() Batch

	// 迭代器
	Iterator(prefix []byte) Iterator

	// 事务支持
	NewTransaction() (Transaction, error)

	// 生命周期管理
	Open() error
	Close() error

	// 统计信息
	Stats() StorageStats
}

// Batch 批量操作接口
type Batch interface {
	Put(key, value []byte) error
	Delete(key []byte) error
	Write() error
	Reset()
	Size() int
}

// Iterator 迭代器接口
type Iterator interface {
	Valid() bool
	Key() []byte
	Value() []byte
	Next()
	Prev()
	Seek(key []byte)
	First()
	Last()
	Close() error
}

// Transaction 事务接口
type Transaction interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
	Delete(key []byte) error
	Commit() error
	Rollback() error
}

// BlockchainStorage 区块链存储接口
type BlockchainStorage interface {
	Storage

	// 区块操作
	GetBlock(height uint64) (interface{}, error)
	PutBlock(height uint64, block interface{}) error
	GetLatestBlock() (interface{}, error)
	GetBlockHeight() (uint64, error)

	// 交易操作
	GetTransaction(hash string) (interface{}, error)
	PutTransaction(hash string, tx interface{}) error

	// 状态操作
	GetState(key string) (interface{}, error)
	PutState(key string, value interface{}) error

	// 索引操作
	CreateIndex(name string, fields []string) error
	QueryByIndex(name string, query interface{}) ([]interface{}, error)
}

// DIDStorage DID存储接口
type DIDStorage interface {
	Storage

	// DID文档操作
	GetDIDDocument(did string) (interface{}, error)
	PutDIDDocument(did string, doc interface{}) error
	DeleteDIDDocument(did string) error

	// DID历史记录
	GetDIDHistory(did string) ([]interface{}, error)
	PutDIDHistory(did string, history interface{}) error

	// DID查询
	QueryDIDs(query interface{}) ([]string, error)

	// DID统计
	GetDIDCount() (int64, error)
	GetDIDsByController(controller string) ([]string, error)
}

// CacheStorage 缓存存储接口
type CacheStorage interface {
	// 基本缓存操作
	Get(key string) (interface{}, error)
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	Exists(key string) bool

	// 批量操作
	MGet(keys []string) (map[string]interface{}, error)
	MSet(items map[string]interface{}, ttl time.Duration) error
	MDelete(keys []string) error

	// 过期管理
	Expire(key string, ttl time.Duration) error
	TTL(key string) (time.Duration, error)

	// 清理操作
	Clear() error
	Flush() error

	// 统计信息
	Size() int64
	Keys(pattern string) ([]string, error)
}

// StorageStats 存储统计信息
type StorageStats struct {
	// 基本统计
	KeyCount  int64 `json:"key_count"`
	TotalSize int64 `json:"total_size_bytes"`
	UsedSize  int64 `json:"used_size_bytes"`
	FreeSize  int64 `json:"free_size_bytes"`

	// 操作统计
	ReadCount   int64 `json:"read_count"`
	WriteCount  int64 `json:"write_count"`
	DeleteCount int64 `json:"delete_count"`

	// 性能统计
	AvgReadTime  time.Duration `json:"avg_read_time"`
	AvgWriteTime time.Duration `json:"avg_write_time"`

	// 错误统计
	ErrorCount int64  `json:"error_count"`
	LastError  string `json:"last_error,omitempty"`

	// 时间戳
	LastUpdated time.Time `json:"last_updated"`
}

// StorageConfig 存储配置接口
type StorageConfig interface {
	// 获取存储类型
	GetType() string
	// 获取存储路径
	GetPath() string
	// 获取配置参数
	GetConfig() map[string]interface{}
	// 验证配置
	Validate() error
}

// StorageFactory 存储工厂接口
type StorageFactory interface {
	// 创建存储实例
	CreateStorage(config StorageConfig) (Storage, error)
	// 获取支持的存储类型
	GetSupportedTypes() []string
	// 注册存储类型
	RegisterType(storageType string, creator func(StorageConfig) (Storage, error)) error
}

// StorageManager 存储管理器接口
type StorageManager interface {
	// 获取存储实例
	GetStorage(name string) (Storage, error)
	// 注册存储实例
	RegisterStorage(name string, storage Storage) error
	// 卸载存储实例
	UnregisterStorage(name string) error
	// 获取所有存储实例
	GetAllStorages() map[string]Storage
	// 启动所有存储
	StartAll(ctx context.Context) error
	// 停止所有存储
	StopAll() error
	// 健康检查
	HealthCheck() map[string]bool
}
