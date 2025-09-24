package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/qujing226/QLink/pkg/app"
	"github.com/qujing226/QLink/pkg/config"
)

var (
	configPath = flag.String("config", "configs/unified.yaml", "配置文件路径")
	mode       = flag.String("mode", "node", "运行模式: node, cli, demo")
	version    = flag.Bool("version", false, "显示版本信息")
	initNode   = flag.Bool("init", false, "初始化节点")
)

func main() {
	flag.Parse()

	if *version {
		printVersion()
		return
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	// 创建应用实例
	application := app.NewApplication(cfg)

	// 根据模式执行不同操作
	switch *mode {
	case "node":
		if *initNode {
			if err := application.Initialize(); err != nil {
				log.Fatalf("初始化节点失败: %v", err)
			}
			fmt.Println("节点初始化完成")
			return
		}
		runNode(application)
	case "cli":
		runCLI(application, flag.Args())
	case "demo":
		runDemo(application)
	default:
		log.Fatalf("不支持的运行模式: %s", *mode)
	}
}

func runNode(app *app.Application) {
	fmt.Println("🚀 启动 QLink 节点...")

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动应用
	if err := app.Start(ctx); err != nil {
		log.Fatalf("启动应用失败: %v", err)
	}
	defer app.Stop()

	fmt.Printf("✅ QLink 节点已启动\n")
	fmt.Printf("📡 节点ID: %s\n", app.GetNodeID())
	fmt.Printf("🌐 API地址: %s\n", app.GetAPIAddress())
	fmt.Printf("🔗 P2P地址: %s\n", app.GetP2PAddress())

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("🛑 正在关闭节点...")
}

func runCLI(app *app.Application, args []string) {
	fmt.Println("🔧 QLink CLI 模式")
	
	if len(args) == 0 {
		fmt.Println("使用方法:")
		fmt.Println("  qlink -mode=cli generate-did")
		fmt.Println("  qlink -mode=cli register-did <did-document>")
		fmt.Println("  qlink -mode=cli resolve-did <did>")
		return
	}

	// 初始化CLI客户端
	client := app.GetCLIClient()
	
	switch args[0] {
	case "generate-did":
		if err := client.GenerateDID(); err != nil {
			log.Fatalf("生成DID失败: %v", err)
		}
	case "register-did":
		if len(args) < 2 {
			log.Fatal("请提供DID文档")
		}
		if err := client.RegisterDID(args[1]); err != nil {
			log.Fatalf("注册DID失败: %v", err)
		}
	case "resolve-did":
		if len(args) < 2 {
			log.Fatal("请提供DID")
		}
		if err := client.ResolveDID(args[1]); err != nil {
			log.Fatalf("解析DID失败: %v", err)
		}
	default:
		log.Fatalf("不支持的命令: %s", args[0])
	}
}

func runDemo(app *app.Application) {
	fmt.Println("🎮 QLink 演示模式")
	
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动演示
	demo := app.GetDemo()
	if err := demo.Run(ctx); err != nil {
		log.Fatalf("运行演示失败: %v", err)
	}
}

func printVersion() {
	fmt.Println("QLink v2.0.0")
	fmt.Println("基于多共识算法的去中心化身份区块链系统")
	fmt.Println("支持 Raft、PoA、PBFT 共识算法")
	fmt.Println("插件化架构，支持热加载")
}