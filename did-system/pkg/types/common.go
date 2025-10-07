package types

import (
	"encoding/json"
	"time"
)

// NodeState 节点状态
type NodeState int

const (
	NodeStateFollower NodeState = iota
	NodeStateCandidate
	NodeStateLeader
	NodeStateInactive
)

// String 返回节点状态的字符串表示
func (ns NodeState) String() string {
	switch ns {
	case NodeStateFollower:
		return "follower"
	case NodeStateCandidate:
		return "candidate"
	case NodeStateLeader:
		return "leader"
	case NodeStateInactive:
		return "inactive"
	default:
		return "unknown"
	}
}

// BaseState 基础状态结构
type BaseState struct {
	Status     string    `json:"status"`
	LastUpdate time.Time `json:"last_update"`
	Version    int64     `json:"version"`
	NodeID     string    `json:"node_id"`
}

// PeerInfo 节点信息
type PeerInfo struct {
	NodeID   string    `json:"node_id"`
	Address  string    `json:"address"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"last_seen"`
	Version  int64     `json:"version"`
	Active   bool      `json:"active"`
}

// LogEntry 日志条目
type LogEntry struct {
	Term      int64       `json:"term"`
	Index     int64       `json:"index"`
	Command   interface{} `json:"command"`
	Timestamp time.Time   `json:"timestamp"`
}

// MessageType 消息类型
type MessageType int

const (
	MessageTypeProposal MessageType = iota
	MessageTypeVote
	MessageTypeCommit
	MessageTypeSync
	MessageTypeHeartbeat
	MessageTypeJoin
	MessageTypeLeave
)

// String 返回消息类型的字符串表示
func (mt MessageType) String() string {
	switch mt {
	case MessageTypeProposal:
		return "proposal"
	case MessageTypeVote:
		return "vote"
	case MessageTypeCommit:
		return "commit"
	case MessageTypeSync:
		return "sync"
	case MessageTypeHeartbeat:
		return "heartbeat"
	case MessageTypeJoin:
		return "join"
	case MessageTypeLeave:
		return "leave"
	default:
		return "unknown"
	}
}

// BaseMessage 基础消息结构
type BaseMessage struct {
	Type      MessageType `json:"type"`
	NodeID    string      `json:"node_id"`
	Timestamp time.Time   `json:"timestamp"`
	Term      int64       `json:"term,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// OperationType 操作类型
type OperationType int

const (
	OperationTypeDIDCreate OperationType = iota
	OperationTypeDIDUpdate
	OperationTypeDIDDeactivate
	OperationTypeNodeJoin
	OperationTypeNodeLeave
	OperationTypeConfigUpdate
)

// String 返回操作类型的字符串表示
func (ot OperationType) String() string {
	switch ot {
	case OperationTypeDIDCreate:
		return "did_create"
	case OperationTypeDIDUpdate:
		return "did_update"
	case OperationTypeDIDDeactivate:
		return "did_deactivate"
	case OperationTypeNodeJoin:
		return "node_join"
	case OperationTypeNodeLeave:
		return "node_leave"
	case OperationTypeConfigUpdate:
		return "config_update"
	default:
		return "unknown"
	}
}

// OperationStatus 操作状态
type OperationStatus int

const (
	OperationStatusPending OperationStatus = iota
	OperationStatusProcessing
	OperationStatusCommitted
	OperationStatusRejected
	OperationStatusTimeout
	OperationStatusFailed
)

// String 返回操作状态的字符串表示
func (os OperationStatus) String() string {
	switch os {
	case OperationStatusPending:
		return "pending"
	case OperationStatusProcessing:
		return "processing"
	case OperationStatusCommitted:
		return "committed"
	case OperationStatusRejected:
		return "rejected"
	case OperationStatusTimeout:
		return "timeout"
	case OperationStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// DIDOperation DID操作
// DIDDocument DID文档结构
type DIDDocument struct {
	Context              []string             `json:"@context"`
	ID                   string               `json:"id"`
	VerificationMethod   []VerificationMethod `json:"verificationMethod,omitempty"`
	Authentication       []string             `json:"authentication,omitempty"`
	AssertionMethod      []string             `json:"assertionMethod,omitempty"`
	KeyAgreement         []string             `json:"keyAgreement,omitempty"`
	CapabilityInvocation []string             `json:"capabilityInvocation,omitempty"`
	CapabilityDelegation []string             `json:"capabilityDelegation,omitempty"`
	Service              []Service            `json:"service,omitempty"`
	Created              *time.Time           `json:"created,omitempty"`
	Updated              *time.Time           `json:"updated,omitempty"`
	Deactivated          bool                 `json:"deactivated,omitempty"`
	Status               string               `json:"status,omitempty"`
	Proof                *Proof               `json:"proof,omitempty"`
}

// ToJSON 将DID文档转换为JSON
func (doc *DIDDocument) ToJSON() ([]byte, error) {
	return json.Marshal(doc)
}

// VerificationMethod 验证方法
type VerificationMethod struct {
	ID                 string                 `json:"id"`
	Type               string                 `json:"type"`
	Controller         string                 `json:"controller"`
	PublicKeyJwk       interface{}            `json:"publicKeyJwk,omitempty"`
	PublicKeyMultibase string                 `json:"publicKeyMultibase,omitempty"`
	PublicKeyLattice   map[string]interface{} `json:"publicKeyLattice,omitempty"`
}

// Service 服务端点
type Service struct {
	ID              string      `json:"id"`
	Type            string      `json:"type"`
	ServiceEndpoint interface{} `json:"serviceEndpoint"`
}

// Proof 证明结构
type Proof struct {
	Type               string    `json:"type"`
	Created            time.Time `json:"created"`
	VerificationMethod string    `json:"verificationMethod"`
	ProofPurpose       string    `json:"proofPurpose"`
	ProofValue         string    `json:"proofValue"`
}

// TransactionType 交易类型
type TransactionType string

const (
	TransactionTypeRegister TransactionType = "register"
	TransactionTypeUpdate   TransactionType = "update"
	TransactionTypeRevoke   TransactionType = "revoke"
)

// TransactionStatus 交易状态
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusConfirmed TransactionStatus = "confirmed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

type DIDOperation struct {
	Operation string       `json:"operation"` // "create", "update", "deactivate"
	DID       string       `json:"did"`
	Document  *DIDDocument `json:"document,omitempty"`
	Proof     *Proof       `json:"proof,omitempty"`
}

// ConflictEntry 冲突条目
type ConflictEntry struct {
	NodeID    string       `json:"node_id"`
	Document  *DIDDocument `json:"document"`
	Timestamp time.Time    `json:"timestamp"`
	Version   int64        `json:"version"`
}

// ConflictData 冲突数据
type ConflictData struct {
	DID       string           `json:"did"`
	Conflicts []*ConflictEntry `json:"conflicts"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy    bool             `json:"healthy"`
	Status     string           `json:"status"`
	LastCheck  time.Time        `json:"last_check"`
	Errors     []string         `json:"errors,omitempty"`
	Metrics    map[string]int64 `json:"metrics,omitempty"`
	Components map[string]bool  `json:"components,omitempty"`
}
