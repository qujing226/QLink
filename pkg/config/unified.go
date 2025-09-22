package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// UnifiedConfig 统一配置结构，扩展现有Config
type UnifiedConfig struct {
	*Config

	// 存储配置
	Storage *StorageConfig `json:"storage" yaml:"storage"`

	// 同步配置
	Sync *SyncConfig `json:"sync" yaml:"sync"`
}

// ExtendedNodeConfig 扩展节点配置
type ExtendedNodeConfig struct {
	*NodeConfig
	Type string `json:"type" yaml:"type"` // "primary", "secondary", "authority", "peer"
	Role string `json:"role" yaml:"role"` // 兼容旧配置
	Port int    `json:"port" yaml:"port"` // 兼容旧配置
}

// ExtendedNetworkConfig 扩展网络配置
type ExtendedNetworkConfig struct {
	*NetworkConfig
	ListenAddr       string   `json:"listen_addr" yaml:"listen_addr"` // 兼容旧配置
	HTTPAddr         string   `json:"http_addr" yaml:"http_addr"`
	MetricsAddr      string   `json:"metrics_addr" yaml:"metrics_addr"`
	DiscoveryEnabled bool     `json:"discovery_enabled" yaml:"discovery_enabled"`
	BootstrapPeers   []string `json:"bootstrap_peers" yaml:"bootstrap_peers"` // 兼容旧配置
}

// ExtendedConsensusConfig 扩展共识配置
type ExtendedConsensusConfig struct {
	*ConsensusConfig
	Type        string      `json:"type" yaml:"type"` // "raft", "poa", "pbft"
	BlockTime   int         `json:"block_time" yaml:"block_time"`
	Authorities []string    `json:"authorities" yaml:"authorities"`
	Raft        *RaftConfig `json:"raft,omitempty" yaml:"raft,omitempty"`
}

// RaftConfig Raft共识特定配置
type RaftConfig struct {
	Port             int           `json:"port" yaml:"port"`
	DataDir          string        `json:"data_dir" yaml:"data_dir"`
	SnapshotInterval time.Duration `json:"snapshot_interval" yaml:"snapshot_interval"`
	HeartbeatTimeout time.Duration `json:"heartbeat_timeout" yaml:"heartbeat_timeout"`
	ElectionTimeout  time.Duration `json:"election_timeout" yaml:"election_timeout"`
}

// ExtendedClusterConfig 扩展集群配置
type ExtendedClusterConfig struct {
	*ClusterConfig
	Enabled bool     `json:"enabled" yaml:"enabled"`
	Peers   []string `json:"peers" yaml:"peers"`
}

// ExtendedDIDConfig 扩展DID配置
type ExtendedDIDConfig struct {
	*DIDConfig
	ChainID         string          `json:"chain_id" yaml:"chain_id"`
	RegistryAddress string          `json:"registry_address" yaml:"registry_address"`
	RegistryFile    string          `json:"registry_file" yaml:"registry_file"` // 兼容旧配置
	Resolver        *ResolverConfig `json:"resolver,omitempty" yaml:"resolver,omitempty"`
}

// ResolverConfig 解析器配置
type ResolverConfig struct {
	CacheTTL     time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	MaxCacheSize int           `json:"max_cache_size" yaml:"max_cache_size"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type string `json:"type" yaml:"type"` // "local", "leveldb", "ipfs"
	Path string `json:"path" yaml:"path"`
	Sync bool   `json:"sync" yaml:"sync"`

	// 本地存储配置
	Local *LocalStorageConfig `json:"local,omitempty" yaml:"local,omitempty"`

	// IPFS存储配置
	IPFS *IPFSConfig `json:"ipfs,omitempty" yaml:"ipfs,omitempty"`
}

// LocalStorageConfig 本地存储配置
type LocalStorageConfig struct {
	Path string `json:"path" yaml:"path"`
}

// IPFSConfig IPFS配置
type IPFSConfig struct {
	Gateway string `json:"gateway" yaml:"gateway"`
}

// ExtendedAPIConfig 扩展API配置
type ExtendedAPIConfig struct {
	*APIConfig
	Enabled     bool             `json:"enabled" yaml:"enabled"`
	Host        string           `json:"host" yaml:"host"`
	Port        int              `json:"port" yaml:"port"`
	CORSEnabled bool             `json:"cors_enabled" yaml:"cors_enabled"`
	Debug       bool             `json:"debug" yaml:"debug"`
	CORS        *CORSConfig      `json:"cors,omitempty" yaml:"cors,omitempty"`
	RateLimit   *RateLimitConfig `json:"rate_limit,omitempty" yaml:"rate_limit,omitempty"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	Enabled        bool     `json:"enabled" yaml:"enabled"`
	AllowedOrigins []string `json:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers" yaml:"allowed_headers"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled           bool `json:"enabled" yaml:"enabled"`
	RequestsPerMinute int  `json:"requests_per_minute" yaml:"requests_per_minute"`
}

// ExtendedLoggingConfig 扩展日志配置
type ExtendedLoggingConfig struct {
	*LoggingConfig
	File string `json:"file" yaml:"file"` // 兼容旧配置
}

// SyncConfig 同步配置
type SyncConfig struct {
	SyncInterval       time.Duration `json:"sync_interval" yaml:"sync_interval"`
	BatchSize          int           `json:"batch_size" yaml:"batch_size"`
	MaxRetries         int           `json:"max_retries" yaml:"max_retries"`
	ConflictResolution string        `json:"conflict_resolution" yaml:"conflict_resolution"`
}

// LoadUnifiedConfig 加载统一配置
func LoadUnifiedConfig(configPath string) (*UnifiedConfig, error) {
	// 检查文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %s", configPath)
	}

	// 读取文件内容
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config UnifiedConfig

	// 根据文件扩展名选择解析方式
	ext := filepath.Ext(configPath)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析YAML配置文件失败: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("解析JSON配置文件失败: %w", err)
		}
	default:
		return nil, fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	// 应用默认值
	config.applyDefaults()

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// applyDefaults 应用默认值
func (c *UnifiedConfig) applyDefaults() {
	if c.Config == nil {
		c.Config = DefaultConfig()
	}

	if c.Storage == nil {
		c.Storage = &StorageConfig{
			Type: "local",
			Path: "./storage",
			Sync: true,
		}
	}

	if c.Sync == nil {
		c.Sync = &SyncConfig{
			SyncInterval:       30 * time.Second,
			BatchSize:          100,
			MaxRetries:         3,
			ConflictResolution: "latest",
		}
	}
}

// SaveUnifiedConfig 保存统一配置
func SaveUnifiedConfig(config *UnifiedConfig, configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	var data []byte
	var err error

	// 根据文件扩展名选择序列化方式
	ext := filepath.Ext(configPath)
	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("序列化YAML配置失败: %w", err)
		}
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("序列化JSON配置失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的配置文件格式: %s", ext)
	}

	// 写入文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// MergeConfigs 合并配置
func MergeConfigs(base, override *UnifiedConfig) *UnifiedConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := &UnifiedConfig{
		Config: MergeConfig(base.Config, override.Config),
	}

	// 合并存储配置
	if override.Storage != nil {
		result.Storage = override.Storage
	} else {
		result.Storage = base.Storage
	}

	// 合并同步配置
	if override.Sync != nil {
		result.Sync = override.Sync
	} else {
		result.Sync = base.Sync
	}

	return result
}
