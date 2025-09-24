package interfaces

import (
	"context"
)

// Plugin 通用插件接口
type Plugin interface {
	// 获取插件名称
	Name() string
	// 获取插件版本
	Version() string
	// 获取插件描述
	Description() string
	// 初始化插件
	Initialize(config map[string]interface{}) error
	// 启动插件
	Start(ctx context.Context) error
	// 停止插件
	Stop() error
	// 获取插件状态
	Status() PluginStatus
	// 获取插件配置
	Config() map[string]interface{}
}

// DIDPlugin DID插件接口
type DIDPlugin interface {
	Plugin
	// DID特定方法
	RegisterDID(did string, document interface{}) error
	UpdateDID(did string, document interface{}) error
	RevokeDID(did string) error
	ResolveDID(did string) (interface{}, error)
	// 验证DID文档
	ValidateDocument(document interface{}) error
}

// NetworkPlugin 网络插件接口
type NetworkPlugin interface {
	Plugin
	// 网络特定方法
	Connect(address string) error
	Disconnect(address string) error
	SendMessage(address string, message interface{}) error
	BroadcastMessage(message interface{}) error
	// 获取连接的节点列表
	GetConnectedPeers() []string
	// 获取网络统计信息
	GetNetworkStats() NetworkStats
}

// CryptoProvider 加密服务提供者接口
type CryptoProvider interface {
	Plugin
	// 生成密钥对
	GenerateKeyPair() (publicKey interface{}, privateKey interface{}, err error)
	// 签名
	Sign(data []byte, privateKey interface{}) ([]byte, error)
	// 验证签名
	Verify(data []byte, signature []byte, publicKey interface{}) bool
	// 加密
	Encrypt(data []byte, publicKey interface{}) ([]byte, error)
	// 解密
	Decrypt(encryptedData []byte, privateKey interface{}) ([]byte, error)
	// 生成哈希
	Hash(data []byte) []byte
}

// PluginStatus 插件状态
type PluginStatus int

const (
	PluginStatusStopped PluginStatus = iota
	PluginStatusStarting
	PluginStatusRunning
	PluginStatusStopping
	PluginStatusError
)

// String 返回插件状态的字符串表示
func (ps PluginStatus) String() string {
	switch ps {
	case PluginStatusStopped:
		return "stopped"
	case PluginStatusStarting:
		return "starting"
	case PluginStatusRunning:
		return "running"
	case PluginStatusStopping:
		return "stopping"
	case PluginStatusError:
		return "error"
	default:
		return "unknown"
	}
}

// NetworkStats 网络统计信息
type NetworkStats struct {
	ConnectedPeers   int   `json:"connected_peers"`
	MessagesSent     int64 `json:"messages_sent"`
	MessagesReceived int64 `json:"messages_received"`
	BytesSent        int64 `json:"bytes_sent"`
	BytesReceived    int64 `json:"bytes_received"`
	Uptime           int64 `json:"uptime_seconds"`
}

// PluginManager 插件管理器接口
type PluginManager interface {
	// 注册插件
	RegisterPlugin(plugin Plugin) error
	// 卸载插件
	UnregisterPlugin(name string) error
	// 获取插件
	GetPlugin(name string) (Plugin, error)
	// 获取所有插件
	GetAllPlugins() map[string]Plugin
	// 启动所有插件
	StartAll(ctx context.Context) error
	// 停止所有插件
	StopAll() error
	// 获取插件状态
	GetPluginStatus(name string) (PluginStatus, error)
}