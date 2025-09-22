package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/qujing226/QLink/did"
	didblockchain "github.com/qujing226/QLink/did/blockchain"
	"github.com/qujing226/QLink/pkg/api"

	"github.com/qujing226/QLink/did/config"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config.yaml", "配置文件路径")
	command := flag.String("cmd", "start", "命令: start, init, version")
	flag.Parse()

	switch *command {
	case "init":
		initNode(*configPath)
	case "version":
		printVersion()
	case "start":
		startNode(*configPath)
	default:
		fmt.Printf("未知命令: %s\n", *command)
		os.Exit(1)
	}
}

func initNode(configPath string) {
	fmt.Println("初始化 DID-QLink 节点...")

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化存储管理器
	sm, err := didblockchain.NewStorageManager(cfg)
	if err != nil {
		log.Fatalf("初始化区块链失败: %v", err)
	}
	defer sm.Close()

	fmt.Println("节点初始化完成")
}

func startNode(configPath string) {
	fmt.Println("启动 DID-QLink 节点...")

	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化存储管理器
	sm, err := didblockchain.NewStorageManager(cfg)
	if err != nil {
		log.Fatalf("初始化区块链失败: %v", err)
	}
	defer sm.Close()

	// 初始化 DID Registry
	registry := did.NewDIDRegistry(cfg, sm)

	// 初始化 DID Resolver
	resolver := did.NewDIDResolver(cfg, registry, sm)

	// 启动 HTTP API 服务
	if cfg.API.Enabled {
		apiServer := api.NewServer(cfg, sm, registry, resolver)
		go func() {
			if err := apiServer.Start(); err != nil {
				log.Printf("API 服务启动失败: %v", err)
			}
		}()
	}

	// 存储管理器已启动，无需额外启动

	fmt.Printf("DID-QLink 节点已启动 (端口: %d)\n", cfg.Node.Port)
	fmt.Printf("API 服务地址: http://%s:%d\n", cfg.API.Host, cfg.API.Port)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("正在关闭节点...")
}

func printVersion() {
	fmt.Println("DID-QLink v1.0.0")
	fmt.Println("基于 PoA 共识的去中心化身份区块链")
}
