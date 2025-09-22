package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// Config 主配置结构
type Config struct {
	// 节点配置
	Node *NodeConfig `json:"node"`

	// 网络配置
	Network *NetworkConfig `json:"network"`

	// 共识配置
	Consensus *ConsensusConfig `json:"consensus"`

	// 集群配置
	Cluster *ClusterConfig `json:"cluster"`

	// DID配置
	DID *DIDConfig `json:"did"`

	// API配置
	API *APIConfig `json:"api"`

	// 日志配置
	Logging *LoggingConfig `json:"logging"`
}

// NodeConfig 节点配置
type NodeConfig struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	DataDir      string            `json:"data_dir"`
	Role         string            `json:"role"`
	Port         int               `json:"port"`
	Capabilities []string          `json:"capabilities"`
	Metadata     map[string]string `json:"metadata"`
}

// NetworkConfig 网络配置
type NetworkConfig struct {
	ListenAddress     string        `json:"listen_address"`
	ListenPort        int           `json:"listen_port"`
	MaxPeers          int           `json:"max_peers"`
	DialTimeout       time.Duration `json:"dial_timeout"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	ReconnectInterval time.Duration `json:"reconnect_interval"`
	MessageTimeout    time.Duration `json:"message_timeout"`
	BufferSize        int           `json:"buffer_size"`
	EnableTLS         bool          `json:"enable_tls"`
	TLSCertFile       string        `json:"tls_cert_file"`
	TLSKeyFile        string        `json:"tls_key_file"`
}

// ConsensusConfig 共识配置
type ConsensusConfig struct {
	Algorithm        string        `json:"algorithm"`
	ElectionTimeout  time.Duration `json:"election_timeout"`
	HeartbeatTimeout time.Duration `json:"heartbeat_timeout"`
	LogRetention     int           `json:"log_retention"`
	SnapshotInterval int           `json:"snapshot_interval"`
	MaxLogEntries    int           `json:"max_log_entries"`
}

// ClusterConfig 集群配置
type ClusterConfig struct {
	ID                  string        `json:"id"`
	MaxNodes            int           `json:"max_nodes"`
	MinNodes            int           `json:"min_nodes"`
	JoinTimeout         time.Duration `json:"join_timeout"`
	SyncInterval        time.Duration `json:"sync_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	AutoJoin            bool          `json:"auto_join"`
	BootstrapNodes      []string      `json:"bootstrap_nodes"`
}

// DIDConfig DID配置
type DIDConfig struct {
	Method          string `json:"method"`
	Network         string `json:"network"`
	StorageType     string `json:"storage_type"`
	StoragePath     string `json:"storage_path"`
	CacheSize       int    `json:"cache_size"`
	EnableCache     bool   `json:"enable_cache"`
	ValidationLevel string `json:"validation_level"`
}

// APIConfig API配置
type APIConfig struct {
	ListenAddress   string        `json:"listen_address"`
	ListenPort      int           `json:"listen_port"`
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Debug           bool          `json:"debug"`
	EnableCORS      bool          `json:"enable_cors"`
	EnableMetrics   bool          `json:"enable_metrics"`
	EnableRateLimit bool          `json:"enable_rate_limit"`
	RateLimit       int           `json:"rate_limit"`
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	MaxRequestSize  int64         `json:"max_request_size"`
	EnableTLS       bool          `json:"enable_tls"`
	TLSCertFile     string        `json:"tls_cert_file"`
	TLSKeyFile      string        `json:"tls_key_file"`
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
			Capabilities: []string{"did", "consensus", "sync"},
			Metadata:     make(map[string]string),
		},
		Network: &NetworkConfig{
			ListenAddress:     "0.0.0.0",
			ListenPort:        8080,
			MaxPeers:          50,
			DialTimeout:       10 * time.Second,
			HeartbeatInterval: 5 * time.Second,
			ReconnectInterval: 30 * time.Second,
			MessageTimeout:    30 * time.Second,
			BufferSize:        1024,
			EnableTLS:         false,
		},
		Consensus: &ConsensusConfig{
			Algorithm:        "raft",
			ElectionTimeout:  15 * time.Second,
			HeartbeatTimeout: 5 * time.Second,
			LogRetention:     1000,
			SnapshotInterval: 100,
			MaxLogEntries:    10000,
		},
		Cluster: &ClusterConfig{
			ID:                  generateClusterID(),
			MaxNodes:            10,
			MinNodes:            1,
			JoinTimeout:         30 * time.Second,
			SyncInterval:        10 * time.Second,
			HealthCheckInterval: 5 * time.Second,
			AutoJoin:            false,
			BootstrapNodes:      []string{},
		},
		DID: &DIDConfig{
			Method:          "qlink",
			Network:         "mainnet",
			StorageType:     "file",
			StoragePath:     "./data/did",
			CacheSize:       1000,
			EnableCache:     true,
			ValidationLevel: "strict",
		},
		API: &APIConfig{
			ListenAddress:   "0.0.0.0",
			ListenPort:      8081,
			EnableCORS:      true,
			EnableMetrics:   true,
			EnableRateLimit: true,
			RateLimit:       100,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			MaxRequestSize:  1024 * 1024, // 1MB
			EnableTLS:       false,
		},
		Logging: &LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := SaveConfig(config, configPath); err != nil {
			return nil, fmt.Errorf("创建默认配置文件失败: %w", err)
		}
		return config, nil
	}

	// 读取配置文件
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入文件
	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Node == nil {
		return fmt.Errorf("节点配置不能为空")
	}

	if c.Node.ID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	if c.Network == nil {
		return fmt.Errorf("网络配置不能为空")
	}

	if c.Network.ListenPort <= 0 || c.Network.ListenPort > 65535 {
		return fmt.Errorf("网络监听端口无效: %d", c.Network.ListenPort)
	}

	if c.API == nil {
		return fmt.Errorf("API配置不能为空")
	}

	if c.API.ListenPort <= 0 || c.API.ListenPort > 65535 {
		return fmt.Errorf("API监听端口无效: %d", c.API.ListenPort)
	}

	if c.Cluster == nil {
		return fmt.Errorf("集群配置不能为空")
	}

	if c.Cluster.MaxNodes <= 0 {
		return fmt.Errorf("集群最大节点数必须大于0")
	}

	if c.Cluster.MinNodes <= 0 {
		return fmt.Errorf("集群最小节点数必须大于0")
	}

	if c.Cluster.MinNodes > c.Cluster.MaxNodes {
		return fmt.Errorf("集群最小节点数不能大于最大节点数")
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

// IsBootstrapNode 检查是否为引导节点
func (c *Config) IsBootstrapNode() bool {
	if c.Cluster == nil {
		return false
	}
	return len(c.Cluster.BootstrapNodes) == 0
}

// GetBootstrapNodes 获取引导节点列表
func (c *Config) GetBootstrapNodes() []string {
	if c.Cluster == nil {
		return []string{}
	}
	return c.Cluster.BootstrapNodes
}

// generateNodeID 生成节点ID
func generateNodeID() string {
	// 简化实现，实际应该生成唯一ID
	return fmt.Sprintf("node-%d", time.Now().Unix())
}

// generateClusterID 生成集群ID
func generateClusterID() string {
	// 简化实现，实际应该生成唯一ID
	return fmt.Sprintf("cluster-%d", time.Now().Unix())
}

// MergeConfig 合并配置
func MergeConfig(base, override *Config) *Config {
	if override == nil {
		return base
	}

	if base == nil {
		return override
	}

	// 简化实现，实际应该深度合并
	merged := *base

	if override.Node != nil {
		merged.Node = override.Node
	}

	if override.Network != nil {
		merged.Network = override.Network
	}

	if override.Consensus != nil {
		merged.Consensus = override.Consensus
	}

	if override.Cluster != nil {
		merged.Cluster = override.Cluster
	}

	if override.DID != nil {
		merged.DID = override.DID
	}

	if override.API != nil {
		merged.API = override.API
	}

	if override.Logging != nil {
		merged.Logging = override.Logging
	}

	return &merged
}

// ToJSON 转换为JSON字符串
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON 从JSON字符串解析
func FromJSON(jsonStr string) (*Config, error) {
	var config Config
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, err
	}
	return &config, nil
}