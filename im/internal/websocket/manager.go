package websocket

import (
	"encoding/json"
	"net/http"
	"qlink-im/internal/logger"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 在生产环境中应该检查origin
	},
}

type Manager struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mutex      sync.RWMutex
}

type Client struct {
	ID     string
	DID    string
	Conn   *websocket.Conn
	Send   chan []byte
	Manager *Manager
}

type Message struct {
	Type    string      `json:"type"`
	From    string      `json:"from,omitempty"`
	To      string      `json:"to,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Timestamp int64     `json:"timestamp"`
}

func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
	}
}

func (m *Manager) Run() {
	for {
		select {
		case client := <-m.register:
			m.mutex.Lock()
			m.clients[client.DID] = client
			m.mutex.Unlock()
			logger.Info("Client %s connected", client.DID)

		case client := <-m.unregister:
			m.mutex.Lock()
			if _, ok := m.clients[client.DID]; ok {
				delete(m.clients, client.DID)
				close(client.Send)
			}
			m.mutex.Unlock()
			logger.Info("Client %s disconnected", client.DID)

		case message := <-m.broadcast:
			m.mutex.RLock()
			for _, client := range m.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(m.clients, client.DID)
				}
			}
			m.mutex.RUnlock()
		}
	}
}

func (m *Manager) HandleWebSocket(w http.ResponseWriter, r *http.Request, userDID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		DID:     userDID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Manager: m,
	}

	m.register <- client

	go client.writePump()
	go client.readPump()
}

func (m *Manager) SendToUser(userDID string, message *Message) error {
	m.mutex.RLock()
	client, exists := m.clients[userDID]
	m.mutex.RUnlock()

	if !exists {
		return nil // 用户不在线
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case client.Send <- data:
		return nil
	default:
		// 客户端发送缓冲区已满，断开连接
		m.unregister <- client
		return nil
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket error: %v", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(messageData, &msg); err != nil {
			logger.Error("Invalid message format: %v", err)
			continue
		}

		// 处理接收到的消息
		c.handleMessage(&msg)
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("WebSocket write error: %v", err)
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case "ping":
		// 响应心跳
		response := &Message{
			Type: "pong",
			Timestamp: msg.Timestamp,
		}
		data, _ := json.Marshal(response)
		c.Send <- data

	case "message":
		// 转发消息给目标用户
		if msg.To != "" {
			c.Manager.SendToUser(msg.To, msg)
		}

	default:
		logger.Warn("Unknown message type: %s", msg.Type)
	}
}