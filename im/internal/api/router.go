package api

import (
	"net/http"
	"qlink-im/internal/middleware"
	"qlink-im/internal/service"
	"qlink-im/internal/websocket"

	"github.com/gin-gonic/gin"
)

func NewRouter(authService service.AuthService, friendService service.FriendService, messageService service.MessageService, keyExchangeService service.KeyExchangeService, encryptionService service.EncryptionService, wsManager *websocket.Manager) *gin.Engine {
	router := gin.New()

	// 全局中间件
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.ErrorHandler())

	// WebSocket管理器
	go wsManager.Run()

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API路由组
	v1 := router.Group("/api/v1")
	{
		// 认证相关路由
		authHandler := NewAuthHandler(authService)
		v1.POST("/auth/login", authHandler.Login)
		v1.POST("/auth/challenge", authHandler.CreateChallenge)
		v1.POST("/auth/verify", authHandler.VerifyChallenge)
		v1.GET("/auth/lattice-pubkey/:did", authHandler.GetLatticePublicKey)

		// WebSocket连接
		v1.GET("/ws", func(c *gin.Context) {
			userDID := c.Query("did")
			if userDID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "DID parameter required"})
				return
			}
			wsManager.HandleWebSocket(c.Writer, c.Request, userDID)
		})

		// 需要认证的路由
		protected := v1.Group("/")
		protected.Use(middleware.Auth(authService))
		{
			// 好友管理
			friendHandler := NewFriendHandler(friendService)
			protected.POST("/friends/add", friendHandler.AddFriend)
			protected.POST("/friends/accept", friendHandler.AcceptFriend)
			protected.POST("/friends/reject", friendHandler.RejectFriend)
			protected.GET("/friends", friendHandler.GetFriends)
			protected.GET("/friends/requests", friendHandler.GetFriendRequests)
			protected.POST("/friends/block", friendHandler.BlockFriend)
			protected.GET("/users/search", friendHandler.SearchUsers)

			// 消息管理
			messageHandler := NewMessageHandler(messageService, encryptionService)
			protected.POST("/messages/send", messageHandler.SendMessage)
			protected.GET("/messages", messageHandler.GetMessages)
			protected.PUT("/messages/:id/read", messageHandler.MarkAsRead)

			// 会话管理
			sessionHandler := NewSessionHandler(messageService)
			protected.POST("/sessions", sessionHandler.CreateSession)
			protected.GET("/sessions/:friend_did", sessionHandler.GetSession)

			// 密钥交换
			keyExchangeHandler := NewKeyExchangeHandler(keyExchangeService)
			protected.POST("/key-exchange/initiate", keyExchangeHandler.InitiateKeyExchange)
			protected.POST("/key-exchange/:id/complete", keyExchangeHandler.CompleteKeyExchange)
			protected.GET("/key-exchange/pending", keyExchangeHandler.GetPendingKeyExchanges)
		}
	}

	return router
}