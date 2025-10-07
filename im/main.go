package main

import (
    "log"
    "qlink-im/internal/api"
    "qlink-im/internal/config"
    "qlink-im/internal/service"
    "qlink-im/internal/storage"
    "qlink-im/internal/websocket"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化存储
	storage, err := storage.NewStorage(cfg.Database)
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

    // 初始化服务
    authService := service.NewAuthService(storage, cfg.DID, cfg.Security)
    friendService := service.NewFriendService(storage)
    messageService := service.NewMessageService(storage)
    keyExchangeService := service.NewKeyExchangeService(storage)
    encryptionService := service.NewEncryptionService(storage)

	// 初始化WebSocket管理器
	wsManager := websocket.NewManager()

	// 初始化API路由
	router := api.NewRouter(authService, friendService, messageService, keyExchangeService, encryptionService, wsManager)

	// 启动服务器
	log.Printf("Starting server on port %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}