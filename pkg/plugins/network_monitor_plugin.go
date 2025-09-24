package plugins

import (
	"context"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// NetworkMonitorPlugin 网络监控插件实现
type NetworkMonitorPlugin struct {
	mu        sync.RWMutex
	status    interfaces.PluginStatus
	config    map[string]interface{}
	stats     *interfaces.NetworkStats
	startTime time.Time
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewNetworkMonitorPlugin 创建新的网络监控插件实例
func NewNetworkMonitorPlugin() *NetworkMonitorPlugin {
	return &NetworkMonitorPlugin{
		status: interfaces.PluginStatusStopped,
		config: make(map[string]interface{}),
		stats: &interfaces.NetworkStats{
			ConnectedPeers:   0,
			MessagesSent:     0,
			MessagesReceived: 0,
			BytesSent:        0,
			BytesReceived:    0,
			Uptime:           0,
		},
	}
}

// Name 返回插件名称
func (np *NetworkMonitorPlugin) Name() string {
	return "network-monitor"
}

// Version 返回插件版本
func (np *NetworkMonitorPlugin) Version() string {
	return "1.0.0"
}

// Description 返回插件描述
func (np *NetworkMonitorPlugin) Description() string {
	return "Network monitoring plugin for QLink"
}

// Initialize 初始化插件
func (np *NetworkMonitorPlugin) Initialize(config map[string]interface{}) error {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	np.config = config
	return nil
}

// Start 启动插件
func (np *NetworkMonitorPlugin) Start(ctx context.Context) error {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	if np.status == interfaces.PluginStatusRunning {
		return nil
	}
	
	np.status = interfaces.PluginStatusStarting
	np.startTime = time.Now()
	np.ctx, np.cancel = context.WithCancel(ctx)
	
	// 启动监控协程
	go np.monitorLoop()
	
	np.status = interfaces.PluginStatusRunning
	return nil
}

// Stop 停止插件
func (np *NetworkMonitorPlugin) Stop() error {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	if np.status != interfaces.PluginStatusRunning {
		return nil
	}
	
	np.status = interfaces.PluginStatusStopping
	
	if np.cancel != nil {
		np.cancel()
	}
	
	np.status = interfaces.PluginStatusStopped
	return nil
}

// Status 返回插件状态
func (np *NetworkMonitorPlugin) Status() interfaces.PluginStatus {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return np.status
}

// Config 返回插件配置
func (np *NetworkMonitorPlugin) Config() map[string]interface{} {
	np.mu.RLock()
	defer np.mu.RUnlock()
	
	config := make(map[string]interface{})
	for k, v := range np.config {
		config[k] = v
	}
	return config
}

// Connect 连接到指定地址
func (np *NetworkMonitorPlugin) Connect(address string) error {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	// 模拟连接逻辑
	np.stats.ConnectedPeers++
	return nil
}

// Disconnect 断开与指定地址的连接
func (np *NetworkMonitorPlugin) Disconnect(address string) error {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	// 模拟断开连接逻辑
	if np.stats.ConnectedPeers > 0 {
		np.stats.ConnectedPeers--
	}
	return nil
}

// SendMessage 发送消息到指定地址
func (np *NetworkMonitorPlugin) SendMessage(address string, message interface{}) error {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	// 模拟发送消息逻辑
	np.stats.MessagesSent++
	np.stats.BytesSent += 100 // 假设每条消息100字节
	return nil
}

// BroadcastMessage 广播消息
func (np *NetworkMonitorPlugin) BroadcastMessage(message interface{}) error {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	// 模拟广播消息逻辑
	np.stats.MessagesSent += int64(np.stats.ConnectedPeers)
	np.stats.BytesSent += int64(np.stats.ConnectedPeers) * 100
	return nil
}

// GetConnectedPeers 获取已连接的节点列表
func (np *NetworkMonitorPlugin) GetConnectedPeers() []string {
	np.mu.RLock()
	defer np.mu.RUnlock()
	
	// 返回模拟的节点列表
	peers := make([]string, np.stats.ConnectedPeers)
	for i := 0; i < np.stats.ConnectedPeers; i++ {
		peers[i] = "peer-" + string(rune('A'+i))
	}
	return peers
}

// GetNetworkStats 获取网络统计信息
func (np *NetworkMonitorPlugin) GetNetworkStats() interfaces.NetworkStats {
	np.mu.RLock()
	defer np.mu.RUnlock()
	
	uptime := int64(0)
	if !np.startTime.IsZero() {
		uptime = int64(time.Since(np.startTime).Seconds())
	}
	
	return interfaces.NetworkStats{
		ConnectedPeers:   np.stats.ConnectedPeers,
		MessagesSent:     np.stats.MessagesSent,
		MessagesReceived: np.stats.MessagesReceived,
		BytesSent:        np.stats.BytesSent,
		BytesReceived:    np.stats.BytesReceived,
		Uptime:           uptime,
	}
}

// monitorLoop 监控循环
func (np *NetworkMonitorPlugin) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-np.ctx.Done():
			return
		case <-ticker.C:
			np.performMonitoring()
		}
	}
}

// performMonitoring 执行监控任务
func (np *NetworkMonitorPlugin) performMonitoring() {
	np.mu.Lock()
	defer np.mu.Unlock()
	
	// 模拟接收消息
	np.stats.MessagesReceived++
	np.stats.BytesReceived += 100
}