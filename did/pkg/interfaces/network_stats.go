package interfaces

import "time"

// NetworkStats 定义网络状态统计的最小结构，以满足适配器接口返回类型。
type NetworkStats struct {
    NodeID           string    `json:"node_id"`
    ListeningAddress string    `json:"listening_address"`
    TotalPeers       int       `json:"total_peers"`
    ConnectedPeers   int       `json:"connected_peers"`
    DisconnectedPeers int      `json:"disconnected_peers"`
    FailedPeers      int       `json:"failed_peers"`
    MaxPeers         int       `json:"max_peers"`
    LastUpdate       time.Time `json:"last_update"`
}