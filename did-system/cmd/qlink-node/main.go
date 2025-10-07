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
    "github.com/qujing226/QLink/pkg/blockchain"
    "github.com/qujing226/QLink/pkg/config"
)

// 去除冗余适配器，直接使用 MockBlockchain 作为 BlockchainInterface

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

	// 加载配置 - 使用统一配置加载器
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 转换配置格式
	didCfg := convertToDidConfig(cfg)

	// 初始化存储管理器
	sm, err := didblockchain.NewStorageManager(didCfg)
	if err != nil {
		log.Fatalf("初始化存储管理器失败: %v", err)
	}
	defer sm.Close()

	fmt.Println("节点初始化完成")
}

func convertToDidConfig(cfg *config.Config) *config.Config {
	// 直接返回统一的配置，不需要转换
	return cfg
}

func startNode(configPath string) {
	fmt.Println("启动 DID-QLink 节点...")

	// 加载配置 - 使用统一配置加载器
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 转换配置格式
	didCfg := convertToDidConfig(cfg)

	// 初始化存储管理器
	sm, err := didblockchain.NewStorageManager(didCfg)
	if err != nil {
		log.Fatalf("初始化区块链失败: %v", err)
	}
	defer sm.Close()

    // 初始化 DID Registry，直接使用 MockBlockchain（已实现 BlockchainInterface）
    mockBlockchain := didblockchain.NewMockBlockchain(nil)
    registry := did.NewDIDRegistry(mockBlockchain)

	// 初始化 DID Resolver
	resolver := did.NewDIDResolver(didCfg, registry, sm)

	// 创建区块链实例
	bc := &blockchain.Blockchain{} // 创建一个简单的区块链实例

	// 启动 HTTP API 服务
	log.Printf("检查API配置: didCfg.API=%p", didCfg.API)
	if didCfg.API != nil {
		log.Printf("didCfg API配置存在，Port=%d", didCfg.API.Port)
	}

	if didCfg.API != nil && didCfg.API.Port > 0 {
		log.Printf("准备启动API服务器...")
		apiServer := api.NewServer(didCfg, sm, registry, resolver, bc)
		if apiServer == nil {
			log.Printf("API服务器创建失败")
		} else {
			log.Printf("API服务器创建成功，准备启动...")
			go func() {
				if err := apiServer.Start(); err != nil {
					log.Printf("API 服务启动失败: %v", err)
				} else {
					log.Printf("API 服务启动成功")
				}
			}()
		}
	} else {
		log.Printf("API服务未启动: didCfg.API=%p, Port=%d", didCfg.API, func() int {
			if didCfg.API != nil {
				return didCfg.API.Port
			}
			return 0
		}())
	}

	// 存储管理器已启动，无需额外启动

	fmt.Printf("DID-QLink 节点已启动 (端口: %d)\n", didCfg.Node.Port)
	if didCfg.API != nil {
		fmt.Printf("API 服务地址: http://%s:%d\n", didCfg.API.Host, didCfg.API.Port)
	}

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
