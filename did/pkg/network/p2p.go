package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/config"
)

// P2PNetwork P2P网络管理器
type P2PNetwork struct {
	nodeID     string
	address    string
	port       int
	peers      map[string]*Peer
	peersMutex sync.RWMutex

	// 网络监听器
	listener net.Listener

	// 消息处理
	messageHandlers map[MessageType]MessageHandler
	handlersMutex   sync.RWMutex

	// 控制通道
	stopCh chan struct{}

	// 配置
	config *config.NetworkConfig
}

// Peer 对等节点
type Peer struct {
	ID       string     `json:"id"`
	Address  string     `json:"address"`
	Port     int        `json:"port"`
	Conn     net.Conn   `json:"-"`
	Status   PeerStatus `json:"status"`
	LastSeen time.Time  `json:"last_seen"`

	// 发送队列
	sendQueue chan *Message

	// 控制
	stopCh chan struct{}
}

// PeerStatus 节点状态
type PeerStatus int

const (
	PeerConnecting PeerStatus = iota
	PeerConnected
	PeerDisconnected
	PeerFailed
)

// MessageType 消息类型
type MessageType int

const (
	MessageTypeHeartbeat MessageType = iota
	MessageTypeSync
	MessageTypeDIDOperation
	MessageTypeConsensus
	MessageTypeDiscovery
)

// Message 网络消息
type Message struct {
	Type      MessageType `json:"type"`
	From      string      `json:"from"`
	To        string      `json:"to"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// MessageHandler 消息处理器
type MessageHandler func(peer *Peer, msg *Message) error

// NewP2PNetwork 创建新的P2P网络实例
func NewP2PNetwork(nodeID, address string, port int, cfg *config.NetworkConfig) *P2PNetwork {
	if cfg == nil {
		cfg = &config.NetworkConfig{
			MaxPeers:          50,
			DialTimeout:       30 * time.Second,
			HeartbeatInterval: 10 * time.Second,
			ReconnectInterval: 5 * time.Second,
		}
	}

	return &P2PNetwork{
		nodeID:          nodeID,
		address:         address,
		port:            port,
		peers:           make(map[string]*Peer),
		messageHandlers: make(map[MessageType]MessageHandler),
		stopCh:          make(chan struct{}),
		config:          cfg,
	}
}

// Start 启动P2P网络
func (p2p *P2PNetwork) Start(ctx context.Context) error {
	// 启动网络监听
	addr := fmt.Sprintf("%s:%d", p2p.address, p2p.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("启动网络监听失败: %w", err)
	}

	p2p.listener = listener
	log.Printf("P2P网络启动，监听地址: %s", addr)

	// 注册默认消息处理器
	p2p.registerDefaultHandlers()

	// 启动接受连接的goroutine
	go p2p.acceptConnections(ctx)

	// 启动心跳检查
	go p2p.heartbeatLoop(ctx)

	return nil
}

// Stop 停止P2P网络
func (p2p *P2PNetwork) Stop() error {
	close(p2p.stopCh)

	if p2p.listener != nil {
		p2p.listener.Close()
	}

	// 关闭所有peer连接
	p2p.peersMutex.Lock()
	for _, peer := range p2p.peers {
		p2p.disconnectPeer(peer)
	}
	p2p.peersMutex.Unlock()

	log.Printf("P2P网络已停止")
	return nil
}

// AddPeer 添加对等节点
func (p2p *P2PNetwork) AddPeer(id, address string, port int) error {
	p2p.peersMutex.Lock()
	defer p2p.peersMutex.Unlock()

	if _, exists := p2p.peers[id]; exists {
		return fmt.Errorf("节点已存在: %s", id)
	}

	peer := &Peer{
		ID:        id,
		Address:   address,
		Port:      port,
		Status:    PeerDisconnected,
		LastSeen:  time.Now(),
		sendQueue: make(chan *Message, 100),
		stopCh:    make(chan struct{}),
	}

	p2p.peers[id] = peer

	// 尝试连接
	go p2p.connectToPeer(peer)

	log.Printf("添加对等节点: %s (%s:%d)", id, address, port)
	return nil
}

// RemovePeer 移除对等节点
func (p2p *P2PNetwork) RemovePeer(id string) error {
	p2p.peersMutex.Lock()
	defer p2p.peersMutex.Unlock()

	peer, exists := p2p.peers[id]
	if !exists {
		return fmt.Errorf("节点不存在: %s", id)
	}

	p2p.disconnectPeer(peer)
	delete(p2p.peers, id)

	log.Printf("移除对等节点: %s", id)
	return nil
}

// SendMessage 发送消息
func (p2p *P2PNetwork) SendMessage(peerID string, msgType MessageType, data interface{}) error {
	// 输入验证
	if peerID == "" {
		return fmt.Errorf("节点ID不能为空")
	}

	if len(peerID) > 64 {
		return fmt.Errorf("节点ID长度不能超过64字符")
	}

	// 验证消息类型
	if msgType < MessageTypeHeartbeat || msgType > MessageTypeDiscovery {
		return fmt.Errorf("无效的消息类型: %d", msgType)
	}

	if data == nil {
		return fmt.Errorf("消息数据不能为空")
	}

	p2p.peersMutex.RLock()
	peer, exists := p2p.peers[peerID]
	p2p.peersMutex.RUnlock()

	if !exists {
		return fmt.Errorf("节点不存在: %s", peerID)
	}

	if peer.Status != PeerConnected {
		return fmt.Errorf("节点未连接: %s", peerID)
	}

	msg := &Message{
		Type:      msgType,
		From:      p2p.nodeID,
		To:        peerID,
		Timestamp: time.Now(),
		Data:      data,
	}

	select {
	case peer.sendQueue <- msg:
		return nil
	default:
		return fmt.Errorf("发送队列已满: %s", peerID)
	}
}

// BroadcastMessage 广播消息
func (p2p *P2PNetwork) BroadcastMessage(msgType MessageType, data interface{}) {
	p2p.peersMutex.RLock()
	defer p2p.peersMutex.RUnlock()

	for peerID := range p2p.peers {
		go p2p.SendMessage(peerID, msgType, data)
	}
}

// RegisterMessageHandler 注册消息处理器
func (p2p *P2PNetwork) RegisterMessageHandler(msgType MessageType, handler MessageHandler) {
	p2p.handlersMutex.Lock()
	defer p2p.handlersMutex.Unlock()

	// 初始化messageHandlers map如果为nil
	if p2p.messageHandlers == nil {
		p2p.messageHandlers = make(map[MessageType]MessageHandler)
	}

	p2p.messageHandlers[msgType] = handler
}

// GetPeers 获取对等节点列表
func (p2p *P2PNetwork) GetPeers() map[string]*Peer {
	p2p.peersMutex.RLock()
	defer p2p.peersMutex.RUnlock()

	peers := make(map[string]*Peer)
	for id, peer := range p2p.peers {
		peers[id] = &Peer{
			ID:       peer.ID,
			Address:  peer.Address,
			Port:     peer.Port,
			Status:   peer.Status,
			LastSeen: peer.LastSeen,
		}
	}
	return peers
}

// GetConnectedPeers 获取已连接的节点数量
func (p2p *P2PNetwork) GetConnectedPeers() int {
	p2p.peersMutex.RLock()
	defer p2p.peersMutex.RUnlock()

	count := 0
	for _, peer := range p2p.peers {
		if peer.Status == PeerConnected {
			count++
		}
	}
	return count
}

// acceptConnections 接受连接
func (p2p *P2PNetwork) acceptConnections(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p2p.stopCh:
			return
		default:
			conn, err := p2p.listener.Accept()
			if err != nil {
				log.Printf("接受连接失败: %v", err)
				continue
			}

			go p2p.handleIncomingConnection(conn)
		}
	}
}

// handleIncomingConnection 处理传入连接
func (p2p *P2PNetwork) handleIncomingConnection(conn net.Conn) {
	defer conn.Close()

	// 这里应该实现握手协议来识别对等节点
	// 简化实现，直接处理消息
	log.Printf("接受来自 %s 的连接", conn.RemoteAddr())

	// 读取和处理消息
	decoder := json.NewDecoder(conn)
	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			log.Printf("解码消息失败: %v", err)
			break
		}

		p2p.handleMessage(nil, &msg) // 这里peer为nil，需要改进
	}
}

// connectToPeer 连接到对等节点
func (p2p *P2PNetwork) connectToPeer(peer *Peer) {
	peer.Status = PeerConnecting

	addr := net.JoinHostPort(peer.Address, fmt.Sprintf("%d", peer.Port))
	conn, err := net.DialTimeout("tcp", addr, p2p.config.DialTimeout)
	if err != nil {
		log.Printf("连接节点失败 %s (%s): %v", peer.ID, addr, err)
		peer.Status = PeerFailed
		// 记录连接失败的详细信息
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				log.Printf("连接超时: %s", peer.ID)
			} else if netErr.Temporary() {
				log.Printf("临时网络错误: %s", peer.ID)
			}
		}
		return
	}

	peer.Conn = conn
	peer.Status = PeerConnected
	peer.LastSeen = time.Now()

	log.Printf("成功连接到节点: %s (%s)", peer.ID, addr)

	// 启动消息处理goroutines
	go p2p.handlePeerMessages(peer)
	go p2p.handlePeerSending(peer)
}

// disconnectPeer 断开节点连接
func (p2p *P2PNetwork) disconnectPeer(peer *Peer) {
	// 安全关闭stopCh，避免重复关闭
	select {
	case <-peer.stopCh:
		// 通道已关闭
	default:
		close(peer.stopCh)
	}

	if peer.Conn != nil {
		peer.Conn.Close()
	}

	// 安全关闭sendQueue
	select {
	case <-peer.sendQueue:
		// 清空队列
	default:
		close(peer.sendQueue)
	}

	peer.Status = PeerDisconnected
}

// handlePeerMessages 处理节点消息
func (p2p *P2PNetwork) handlePeerMessages(peer *Peer) {
	defer p2p.disconnectPeer(peer)

	decoder := json.NewDecoder(peer.Conn)
	for {
		select {
		case <-peer.stopCh:
			return
		default:
			var msg Message
			if err := decoder.Decode(&msg); err != nil {
				log.Printf("从节点 %s 读取消息失败: %v", peer.ID, err)
				return
			}

			peer.LastSeen = time.Now()
			p2p.handleMessage(peer, &msg)
		}
	}
}

// handlePeerSending 处理节点发送
func (p2p *P2PNetwork) handlePeerSending(peer *Peer) {
	encoder := json.NewEncoder(peer.Conn)

	for {
		select {
		case <-peer.stopCh:
			return
		case msg := <-peer.sendQueue:
			if err := encoder.Encode(msg); err != nil {
				log.Printf("向节点 %s 发送消息失败: %v", peer.ID, err)
				return
			}
		}
	}
}

// handleMessage 处理消息
func (p2p *P2PNetwork) handleMessage(peer *Peer, msg *Message) {
	// 输入验证
	if peer == nil {
		log.Printf("对等节点不能为空")
		return
	}

	if msg == nil {
		log.Printf("消息不能为空")
		return
	}

	if msg.From == "" {
		log.Printf("消息发送者不能为空")
		return
	}

	if msg.To == "" {
		log.Printf("消息接收者不能为空")
		return
	}

	// 验证消息时间戳
	if msg.Timestamp.IsZero() {
		log.Printf("消息时间戳无效")
		return
	}

	// 检查消息是否过期（超过5分钟）
	if time.Since(msg.Timestamp) > 5*time.Minute {
		log.Printf("消息已过期，发送时间: %v", msg.Timestamp)
		return
	}

	p2p.handlersMutex.RLock()
	handler, exists := p2p.messageHandlers[msg.Type]
	p2p.handlersMutex.RUnlock()

	if !exists {
		log.Printf("未找到消息类型 %d 的处理器", msg.Type)
		return
	}

	if err := handler(peer, msg); err != nil {
		log.Printf("处理消息失败: %v", err)
	}
}

// heartbeatLoop 心跳循环
func (p2p *P2PNetwork) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(p2p.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p2p.stopCh:
			return
		case <-ticker.C:
			p2p.sendHeartbeats()
			p2p.checkPeerHealth()
		}
	}
}

// sendHeartbeats 发送心跳
func (p2p *P2PNetwork) sendHeartbeats() {
	p2p.BroadcastMessage(MessageTypeHeartbeat, map[string]interface{}{
		"timestamp": time.Now(),
		"node_id":   p2p.nodeID,
	})
}

// checkPeerHealth 检查节点健康状态
func (p2p *P2PNetwork) checkPeerHealth() {
	p2p.peersMutex.Lock()
	defer p2p.peersMutex.Unlock()

	now := time.Now()
	timeout := p2p.config.HeartbeatInterval * 3 // 3倍心跳间隔作为超时

	for _, peer := range p2p.peers {
		if peer.Status == PeerConnected && now.Sub(peer.LastSeen) > timeout {
			log.Printf("节点 %s 心跳超时，断开连接", peer.ID)
			go p2p.disconnectPeer(peer)
		}
	}
}

// registerDefaultHandlers 注册默认消息处理器
func (p2p *P2PNetwork) registerDefaultHandlers() {
	// 心跳处理器
	p2p.RegisterMessageHandler(MessageTypeHeartbeat, func(peer *Peer, msg *Message) error {
		log.Printf("收到来自 %s 的心跳", msg.From)
		return nil
	})

	// 发现处理器
	p2p.RegisterMessageHandler(MessageTypeDiscovery, func(peer *Peer, msg *Message) error {
		log.Printf("收到来自 %s 的发现消息", msg.From)
		return nil
	})
}

// GetNetworkStatus 获取网络状态
func (p2p *P2PNetwork) GetNetworkStatus() map[string]interface{} {
	p2p.peersMutex.RLock()
	defer p2p.peersMutex.RUnlock()

	connected := 0
	disconnected := 0
	failed := 0

	for _, peer := range p2p.peers {
		switch peer.Status {
		case PeerConnected:
			connected++
		case PeerDisconnected:
			disconnected++
		case PeerFailed:
			failed++
		}
	}

	return map[string]interface{}{
		"node_id":            p2p.nodeID,
		"listening_address":  fmt.Sprintf("%s:%d", p2p.address, p2p.port),
		"total_peers":        len(p2p.peers),
		"connected_peers":    connected,
		"disconnected_peers": disconnected,
		"failed_peers":       failed,
		"max_peers":          p2p.config.MaxPeers,
	}
}
