package interfaces

import (
	"context"
)

// ConsensusAlgorithm 统一的共识算法接口
type ConsensusAlgorithm interface {
	// 启动共识算法
	Start(ctx context.Context) error
	// 停止共识算法
	Stop() error
	// 提交提案
	Submit(proposal interface{}) error
	// 获取状态信息
	GetStatus() map[string]interface{}
	// 获取当前领导者
	GetLeader() string
	// 获取节点列表
	GetNodes() []string
}

// ConsensusEngine 区块链共识引擎接口
type ConsensusEngine interface {
	// 验证区块
	ValidateBlock(block interface{}) error
	// 验证提议者
	ValidateProposer(proposer string, blockNumber uint64) error
	// 获取下一个提议者
	GetNextProposer(blockNumber uint64) string
	// 是否为权威节点
	IsAuthority(address string) bool
	// 获取权威节点列表
	GetAuthorities() []string
	// 启动共识
	Start() error
	// 停止共识
	Stop() error
}

// ConsensusType 共识算法类型
type ConsensusType int

const (
	ConsensusTypeRaft ConsensusType = iota
	ConsensusTypePoA
	ConsensusTypePBFT
	ConsensusTypePoS
)

// String 返回共识类型的字符串表示
func (ct ConsensusType) String() string {
	switch ct {
	case ConsensusTypeRaft:
		return "Raft"
	case ConsensusTypePoA:
		return "PoA"
	case ConsensusTypePBFT:
		return "PBFT"
	case ConsensusTypePoS:
		return "PoS"
	default:
		return "unknown"
	}
}

// ConsensusFactory 共识算法工厂接口
type ConsensusFactory interface {
	// 创建共识算法实例
	CreateConsensus(consensusType ConsensusType, config interface{}) (ConsensusAlgorithm, error)
	// 获取支持的共识类型
	GetSupportedTypes() []ConsensusType
	// 验证配置
	ValidateConfig(consensusType ConsensusType, config interface{}) error
}