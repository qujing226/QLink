package interfaces

import "context"

// ConsensusAdapter 共识适配器接口，用于统一不同共识实现
type ConsensusAdapter interface {
	// 适配器基础方法
	GetType() ConsensusType
	GetName() string

	// 共识算法接口（统一使用context）
	Start(ctx context.Context) error
	Stop() error
	Submit(proposal interface{}) error
	GetStatus() map[string]interface{}
	GetLeader() string
	GetNodes() []string

	// 共识引擎接口
	ValidateBlock(block interface{}) error
	ValidateProposer(proposer string, blockNumber uint64) error
	GetNextProposer(blockNumber uint64) string
	IsAuthority(address string) bool
	GetAuthorities() []string
}


// NetworkAdapter 网络适配器接口
type NetworkAdapter interface {
	// 网络连接
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool

	// 节点管理
	AddPeer(address string) error
	RemovePeer(nodeID string) error
	GetPeers() []string

	// 消息传递
	Broadcast(message interface{}) error
	SendTo(nodeID string, message interface{}) error

	// 网络状态
	GetNetworkStats() NetworkStats
}

// ServiceAdapter 服务适配器接口
type ServiceAdapter interface {
	// 服务生命周期
	Initialize() error
	Start(ctx context.Context) error
	Stop() error
	Shutdown() error

	// 服务状态
	IsHealthy() bool
	GetStatus() map[string]interface{}
	GetMetrics() map[string]interface{}
}
