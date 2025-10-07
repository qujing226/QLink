package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Config 主配置结构
type Config struct {
	// 节点配置
	Node *NodeConfig `json:"node" yaml:"node"`

	// 网络配置
	Network *NetworkConfig `json:"network" yaml:"network"`

	// 共识配置
	Consensus *ConsensusConfig `json:"consensus" yaml:"consensus"`

	// 集群配置
	Cluster *ClusterConfig `json:"cluster" yaml:"cluster"`

	// DID配置
	DID *DIDConfig `json:"did" yaml:"did"`

	// API配置
	API *APIConfig `json:"api" yaml:"api"`

	// 日志配置
	Logging *LoggingConfig `json:"logging" yaml:"logging"`

	// 存储配置
	Storage *StorageConfig `json:"storage" yaml:"storage"`

	// 同步配置
	Sync *SyncConfig `json:"sync" yaml:"sync"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	ID           string            `json:"id" yaml:"id"`
	Name         string            `json:"name" yaml:"name"`
	Version      string            `json:"version" yaml:"version"`
	DataDir      string            `json:"data_dir" yaml:"data_dir"`
	Role         string            `json:"role" yaml:"role"`
	Type         string            `json:"type" yaml:"type"` // "primary", "secondary", "authority", "peer"
	Port         int               `json:"port" yaml:"port"`
	Capabilities []string          `json:"capabilities" yaml:"capabilities"`
	Metadata     map[string]string `json:"metadata" yaml:"metadata"`
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	ListenAddress     string        `json:"listen_address" yaml:"listen_address"`
	ListenPort        int           `json:"listen_port" yaml:"listen_port"`
	HTTPAddr          string        `json:"http_addr" yaml:"http_addr"`
	MetricsAddr       string        `json:"metrics_addr" yaml:"metrics_addr"`
	MaxPeers          int           `json:"max_peers" yaml:"max_peers"`
	DialTimeout       time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval"`
	ReconnectInterval time.Duration `json:"reconnect_interval" yaml:"reconnect_interval"`
	MessageTimeout    time.Duration `json:"message_timeout" yaml:"message_timeout"`
	BufferSize        int           `json:"buffer_size" yaml:"buffer_size"`
	EnableTLS         bool          `json:"enable_tls" yaml:"enable_tls"`
	TLSCertFile       string        `json:"tls_cert_file" yaml:"tls_cert_file"`
	TLSKeyFile        string        `json:"tls_key_file" yaml:"tls_key_file"`
	DiscoveryEnabled  bool          `json:"discovery_enabled" yaml:"discovery_enabled"`
	BootstrapPeers    []string      `json:"bootstrap_peers" yaml:"bootstrap_peers"`
}

// ConsensusConfig 共识配置
type ConsensusConfig struct {
	Algorithm           string        `json:"algorithm" yaml:"algorithm"`
	Type                string        `json:"type" yaml:"type"` // "raft", "poa", "pbft"
	ElectionTimeout     time.Duration `json:"election_timeout" yaml:"election_timeout"`
	HeartbeatTimeout    time.Duration `json:"heartbeat_timeout" yaml:"heartbeat_timeout"`
	LogRetention        int           `json:"log_retention" yaml:"log_retention"`
	SnapshotInterval    int           `json:"snapshot_interval" yaml:"snapshot_interval"`
	MaxLogEntries       int           `json:"max_log_entries" yaml:"max_log_entries"`
	Authorities         []string      `json:"authorities" yaml:"authorities"`
	BlockTime           int           `json:"block_time" yaml:"block_time"`
	ProposalTimeout     time.Duration `json:"proposal_timeout" yaml:"proposal_timeout"`
	CommitTimeout       time.Duration `json:"commit_timeout" yaml:"commit_timeout"`
	MaxPendingProposals int           `json:"max_pending_proposals" yaml:"max_pending_proposals"`
	BatchSize           int           `json:"batch_size" yaml:"batch_size"`
	Raft                *RaftConfig   `json:"raft,omitempty" yaml:"raft,omitempty"`
}

// RaftConfig Raft共识配置
type RaftConfig struct {
	Port             int           `json:"port" yaml:"port"`
	DataDir          string        `json:"data_dir" yaml:"data_dir"`
	SnapshotInterval time.Duration `json:"snapshot_interval" yaml:"snapshot_interval"`
	HeartbeatTimeout time.Duration `json:"heartbeat_timeout" yaml:"heartbeat_timeout"`
	ElectionTimeout  time.Duration `json:"election_timeout" yaml:"election_timeout"`
}

// ClusterConfig 集群配置
type ClusterConfig struct {
	ID                  string        `json:"id" yaml:"id"`
	MaxNodes            int           `json:"max_nodes" yaml:"max_nodes"`
	MinNodes            int           `json:"min_nodes" yaml:"min_nodes"`
	JoinTimeout         time.Duration `json:"join_timeout" yaml:"join_timeout"`
	SyncInterval        time.Duration `json:"sync_interval" yaml:"sync_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	HeartbeatInterval   time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval"`
	ElectionTimeout     time.Duration `json:"election_timeout" yaml:"election_timeout"`
	AutoJoin            bool          `json:"auto_join" yaml:"auto_join"`
	BootstrapNodes      []string      `json:"bootstrap_nodes" yaml:"bootstrap_nodes"`
	Enabled             bool          `json:"enabled" yaml:"enabled"`
	Peers               []string      `json:"peers" yaml:"peers"`
}

// DIDConfig DID配置
type DIDConfig struct {
	Method          string          `json:"method" yaml:"method"`
	Network         string          `json:"network" yaml:"network"`
	StorageType     string          `json:"storage_type" yaml:"storage_type"`
	StoragePath     string          `json:"storage_path" yaml:"storage_path"`
	CacheSize       int             `json:"cache_size" yaml:"cache_size"`
	EnableCache     bool            `json:"enable_cache" yaml:"enable_cache"`
	ValidationLevel string          `json:"validation_level" yaml:"validation_level"`
	ChainID         string          `json:"chain_id" yaml:"chain_id"`
	RegistryAddress string          `json:"registry_address" yaml:"registry_address"`
	RegistryFile    string          `json:"registry_file" yaml:"registry_file"` // 兼容旧配置
	Resolver        *ResolverConfig `json:"resolver,omitempty" yaml:"resolver,omitempty"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type  string              `json:"type" yaml:"type"` // "local", "leveldb", "ipfs", "memory"
	Path  string              `json:"path" yaml:"path"`
	Sync  bool                `json:"sync" yaml:"sync"`
	Local *LocalStorageConfig `json:"local,omitempty" yaml:"local,omitempty"`
	IPFS  *IPFSStorageConfig  `json:"ipfs,omitempty" yaml:"ipfs,omitempty"`
}

// LocalStorageConfig 本地存储配置
type LocalStorageConfig struct {
	Path string `json:"path" yaml:"path"`
}

// IPFSStorageConfig IPFS存储配置
type IPFSStorageConfig struct {
	Gateway string `json:"gateway" yaml:"gateway"`
}

// SyncConfig 同步配置
type SyncConfig struct {
	SyncInterval       time.Duration `json:"sync_interval" yaml:"sync_interval"`
	BatchSize          int           `json:"batch_size" yaml:"batch_size"`
	MaxRetries         int           `json:"max_retries" yaml:"max_retries"`
	ConflictResolution string        `json:"conflict_resolution" yaml:"conflict_resolution"`
}

// ResolverConfig 解析器配置
type ResolverConfig struct {
	CacheTTL     time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	MaxCacheSize int           `json:"max_cache_size" yaml:"max_cache_size"`
}

// APIConfig API配置
type APIConfig struct {
	Enable          bool          `json:"enable" yaml:"enable"`
	ListenAddress   string        `json:"listen_address" yaml:"listen_address"`
	ListenPort      int           `json:"listen_port" yaml:"listen_port"`
	Host            string        `json:"host" yaml:"host"`
	Port            int           `json:"port" yaml:"port"`
	Debug           bool          `json:"debug" yaml:"debug"`
	EnableCORS      bool          `json:"enable_cors" yaml:"enable_cors"`
	EnableMetrics   bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableRateLimit bool          `json:"enable_rate_limit" yaml:"enable_rate_limit"`
	RateLimit       int           `json:"rate_limit" yaml:"rate_limit"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout"`
	MaxRequestSize  int64         `json:"max_request_size" yaml:"max_request_size"`
	EnableTLS       bool          `json:"enable_tls" yaml:"enable_tls"`
	TLSCertFile     string        `json:"tls_cert_file" yaml:"tls_cert_file"`
	TLSKeyFile      string        `json:"tls_key_file" yaml:"tls_key_file"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
	Compress   bool   `json:"compress"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Node: &NodeConfig{
			ID:           generateNodeID(),
			Name:         "qlink-node",
			Version:      "1.0.0",
			DataDir:      "./data",
			Role:         "peer",
			Type:         "peer",
			Port:         8080,
			Capabilities: []string{"did", "consensus", "storage"},
			Metadata:     make(map[string]string),
		},
		Network: &NetworkConfig{
			ListenAddress:     "0.0.0.0",
			ListenPort:        9000,
			HTTPAddr:          "0.0.0.0:8080",
			MetricsAddr:       "0.0.0.0:9090",
			MaxPeers:          50,
			DialTimeout:       10 * time.Second,
			HeartbeatInterval: 30 * time.Second,
			ReconnectInterval: 5 * time.Second,
			MessageTimeout:    30 * time.Second,
			BufferSize:        1024,
			EnableTLS:         false,
			TLSCertFile:       "",
			TLSKeyFile:        "",
			DiscoveryEnabled:  true,
			BootstrapPeers:    []string{},
		},
		Consensus: &ConsensusConfig{
			Algorithm:        "raft",
			Type:             "raft",
			ElectionTimeout:  5 * time.Second,
			HeartbeatTimeout: 1 * time.Second,
			LogRetention:     1000,
			SnapshotInterval: 100,
			MaxLogEntries:    10000,
			Authorities:      []string{},
			BlockTime:        5,
			Raft: &RaftConfig{
				Port:             9001,
				DataDir:          "./data/raft",
				SnapshotInterval: 100 * time.Second,
				HeartbeatTimeout: 1 * time.Second,
				ElectionTimeout:  5 * time.Second,
			},
		},
		Cluster: &ClusterConfig{
			ID:                  generateClusterID(),
			MaxNodes:            10,
			MinNodes:            1,
			JoinTimeout:         30 * time.Second,
			SyncInterval:        60 * time.Second,
			HealthCheckInterval: 10 * time.Second,
			HeartbeatInterval:   30 * time.Second,
			ElectionTimeout:     5 * time.Second,
			AutoJoin:            false,
			BootstrapNodes:      []string{},
			Enabled:             true,
			Peers:               []string{},
		},
		DID: &DIDConfig{
			Method:          "qlink",
			Network:         "mainnet",
			StorageType:     "local",
			StoragePath:     "./data/did",
			CacheSize:       1000,
			EnableCache:     true,
			ValidationLevel: "strict",
			ChainID:         "qlink-mainnet",
			RegistryAddress: "",
			RegistryFile:    "./data/registry.json",
			Resolver: &ResolverConfig{
				CacheTTL:     300 * time.Second,
				MaxCacheSize: 10000,
			},
		},
		API: &APIConfig{
			Enable:          true,
			ListenAddress:   "0.0.0.0",
			ListenPort:      8080,
			Host:            "0.0.0.0",
			Port:            8080,
			Debug:           false,
			EnableCORS:      true,
			EnableMetrics:   true,
			EnableRateLimit: true,
			RateLimit:       100,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			MaxRequestSize:  1024 * 1024, // 1MB
			EnableTLS:       false,
			TLSCertFile:     "",
			TLSKeyFile:      "",
		},
		Logging: &LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		},
		Storage: &StorageConfig{
			Type: "local",
			Path: "./data/storage",
			Sync: true,
			Local: &LocalStorageConfig{
				Path: "./data/storage",
			},
		},
		Sync: &SyncConfig{
			SyncInterval:       30 * time.Second,
			BatchSize:          100,
			MaxRetries:         3,
			ConflictResolution: "timestamp",
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return DefaultConfig(), nil
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	config := &Config{}
	ext := strings.ToLower(filepath.Ext(configPath))

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %v", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %v", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	// 填充默认值
	defaultCfg := DefaultConfig()
	if config.Node == nil {
		config.Node = defaultCfg.Node
	}
	if config.Network == nil {
		config.Network = defaultCfg.Network
	}
	if config.Consensus == nil {
		config.Consensus = defaultCfg.Consensus
	}
	if config.Cluster == nil {
		config.Cluster = defaultCfg.Cluster
	}
	if config.DID == nil {
		config.DID = defaultCfg.DID
	}
	if config.API == nil {
		config.API = defaultCfg.API
	}
	if config.Logging == nil {
		config.Logging = defaultCfg.Logging
	}
	if config.Storage == nil {
		config.Storage = defaultCfg.Storage
	}
	if config.Sync == nil {
		config.Sync = defaultCfg.Sync
	}

	return config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, configPath string) error {
	ext := strings.ToLower(filepath.Ext(configPath))
	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	return ioutil.WriteFile(configPath, data, 0644)
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Node == nil {
		return fmt.Errorf("node config is required")
	}
	if c.Node.ID == "" {
		return fmt.Errorf("node ID is required")
	}
	if c.Node.DataDir == "" {
		return fmt.Errorf("node data directory is required")
	}

	if c.Network == nil {
		return fmt.Errorf("network config is required")
	}
	if c.Network.ListenPort <= 0 {
		return fmt.Errorf("invalid network listen port")
	}

	if c.Consensus == nil {
		return fmt.Errorf("consensus config is required")
	}
	if c.Consensus.Algorithm == "" {
		return fmt.Errorf("consensus algorithm is required")
	}

	if c.DID == nil {
		return fmt.Errorf("DID config is required")
	}
	if c.DID.Method == "" {
		return fmt.Errorf("DID method is required")
	}

	return nil
}

// GetNodeID 获取节点ID
func (c *Config) GetNodeID() string {
	if c.Node != nil {
		return c.Node.ID
	}
	return ""
}

// GetClusterID 获取集群ID
func (c *Config) GetClusterID() string {
	if c.Cluster != nil {
		return c.Cluster.ID
	}
	return ""
}

// IsBootstrapNode 判断是否为引导节点
func (c *Config) IsBootstrapNode() bool {
	return c.Cluster != nil && len(c.Cluster.BootstrapNodes) == 0
}

// GetBootstrapNodes 获取引导节点列表
func (c *Config) GetBootstrapNodes() []string {
	if c.Cluster != nil {
		return c.Cluster.BootstrapNodes
	}
	return []string{}
}

// generateNodeID 生成节点ID
func generateNodeID() string {
	return fmt.Sprintf("node-%d", time.Now().UnixNano())
}

// generateClusterID 生成集群ID
func generateClusterID() string {
	return fmt.Sprintf("cluster-%d", time.Now().UnixNano())
}

// MergeConfig 合并配置
func MergeConfig(base, override *Config) *Config {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := *base

	if override.Node != nil {
		result.Node = override.Node
	}
	if override.Network != nil {
		result.Network = override.Network
	}
	if override.Consensus != nil {
		result.Consensus = override.Consensus
	}
	if override.Cluster != nil {
		result.Cluster = override.Cluster
	}
	if override.DID != nil {
		result.DID = override.DID
	}
	if override.API != nil {
		result.API = override.API
	}
	if override.Logging != nil {
		result.Logging = override.Logging
	}
	if override.Storage != nil {
		result.Storage = override.Storage
	}
	if override.Sync != nil {
		result.Sync = override.Sync
	}

	return &result
}

// ToJSON 转换为JSON字符串
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	return string(data), err
}

// FromJSON 从JSON字符串创建配置
func FromJSON(jsonStr string) (*Config, error) {
	config := &Config{}
	err := json.Unmarshal([]byte(jsonStr), config)
	return config, err
}
