package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/blockchain"
	didconfig "github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/cluster"
	"github.com/qujing226/QLink/did/consensus"
	"github.com/qujing226/QLink/did/network"
	syncpkg "github.com/qujing226/QLink/did/sync"
	"github.com/qujing226/QLink/pkg/config"
	"github.com/qujing226/QLink/pkg/api"
	blockchainPkg "github.com/qujing226/QLink/pkg/blockchain"
)

// QLinkNode QLink节点
type QLinkNode struct {
	config         *config.Config
	didRegistry    *did.DIDRegistry
	didResolver    *did.DIDResolver
	storageManager *blockchain.StorageManager
	p2pNetwork     *network.P2PNetwork
	consensus      *consensus.RaftNode
	synchronizer   *syncpkg.Synchronizer
	clusterManager *cluster.ClusterManager
	blockchain     *blockchainPkg.Blockchain // 添加区块链字段
	apiServer      *api.Server

	// 控制通道
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewQLinkNode 创建新的QLink节点
func NewQLinkNode(configPath string) (*QLinkNode, error) {
	// 加载配置
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &QLinkNode{
		config: cfg,
		stopCh: make(chan struct{}),
	}, nil
}

// initComponents 初始化所有组件
func (n *QLinkNode) initComponents() error {
	log.Println("开始初始化组件...")

	// 1. 初始化存储管理器
	n.storageManager = &blockchain.StorageManager{} // 简化初始化

	// 2. 初始化DID配置
	didCfg := &didconfig.Config{
		Node: &didconfig.NodeConfig{
			ID:      n.config.Node.ID,
			DataDir: n.config.Node.DataDir,
		},
		DID: &didconfig.DIDConfig{
			Method:          n.config.DID.Method,
			ChainID:         n.config.DID.Network,
			RegistryAddress: n.config.DID.StoragePath,
		},
	}

	// 3. 初始化DID注册表
	n.didRegistry = did.NewDIDRegistry(didCfg, nil)

	// 4. 初始化DID解析器
	n.didResolver = did.NewDIDResolver(didCfg, n.didRegistry, n.storageManager)

	// 5. 初始化P2P网络
	networkConfig := &network.NetworkConfig{
		MaxPeers:          n.config.Network.MaxPeers,
		ConnectionTimeout: 30 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		ReconnectInterval: 5 * time.Second,
	}
	n.p2pNetwork = network.NewP2PNetwork(
		n.config.Node.ID,
		n.config.Network.ListenAddress,
		n.config.Network.ListenPort,
		networkConfig,
	)

	// 6. 初始化共识模块
	n.consensus = consensus.NewRaftNode(n.config.Node.ID, n.p2pNetwork)

	// 7. 初始化同步器
	syncConfig := &syncpkg.SyncConfig{
		SyncInterval:       n.config.Consensus.ElectionTimeout,
		BatchSize:          100,
		MaxRetries:         3,
		ConflictResolution: "timestamp",
	}
	n.synchronizer = syncpkg.NewSynchronizer(
		n.config.Node.ID,
		n.didRegistry,
		n.p2pNetwork,
		syncConfig,
	)

	// 8. 初始化区块链
	var err error
	n.blockchain, err = blockchainPkg.NewBlockchain(didCfg)
	if err != nil {
		return fmt.Errorf("初始化区块链失败: %w", err)
	}

	// 9. 初始化集群管理器
	clusterConfig := &cluster.ClusterConfig{
		MaxNodes:          n.config.Cluster.MaxNodes,
		HeartbeatInterval: 5 * time.Second,
		ElectionTimeout:   15 * time.Second,
		JoinTimeout:       30 * time.Second,
		SyncInterval:      10 * time.Second,
	}
	n.clusterManager = cluster.NewClusterManager(
		n.config.Node.ID,
		n.config.Cluster.ID,
		n.p2pNetwork,
		n.consensus,
		n.synchronizer,
		clusterConfig,
	)

	// 10. 初始化API服务器
	n.apiServer = api.NewServer(
		didCfg,
		n.storageManager,
		n.didRegistry,
		n.didResolver,
		n.blockchain, // 添加区块链参数
	)

	log.Println("组件初始化完成")
	return nil
}

// Start 启动节点
func (n *QLinkNode) Start(ctx context.Context) error {
	log.Printf("启动QLink节点: %s", n.config.Node.ID)

	// 初始化组件
	if err := n.initComponents(); err != nil {
		return fmt.Errorf("failed to initialize components: %w", err)
	}

	// 启动各个组件
	// 启动区块链
	if err := n.blockchain.Start(); err != nil {
		return fmt.Errorf("启动区块链失败: %w", err)
	}

	// 启动API服务器
	if err := n.apiServer.Start(); err != nil {
		return fmt.Errorf("启动API服务器失败: %w", err)
	}

	n.wg.Add(1)
	go func() {
		defer n.wg.Done()
		n.run(ctx)
	}()

	log.Printf("QLink节点启动成功")
	return nil
}

// run 运行主循环
func (n *QLinkNode) run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("收到停止信号，正在关闭节点...")
			return
		case <-n.stopCh:
			log.Println("节点停止通道关闭")
			return
		case <-ticker.C:
			// 定期健康检查
			log.Printf("节点 %s 运行正常", n.config.Node.ID)
		}
	}
}

// Stop 停止节点
func (n *QLinkNode) Stop() error {
	log.Println("正在停止QLink节点...")

	// 停止API服务器
	if n.apiServer != nil {
		log.Printf("Stopping API server...")
		n.apiServer.Stop()
	}

	// 停止区块链
	if n.blockchain != nil {
		log.Printf("Stopping blockchain...")
		n.blockchain.Stop()
	}

	// 关闭停止通道
	close(n.stopCh)

	// 等待所有goroutine完成
	n.wg.Wait()

	log.Println("QLink节点已停止")
	return nil
}

func main() {
	// 设置默认配置路径
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("配置文件 %s 不存在，创建默认配置", configPath)
		defaultConfig := config.DefaultConfig()
		if err := config.SaveConfig(defaultConfig, configPath); err != nil {
			log.Fatalf("无法创建默认配置文件: %v", err)
		}
		log.Printf("默认配置已保存到 %s", configPath)
	}

	// 创建节点
	node, err := NewQLinkNode(configPath)
	if err != nil {
		log.Fatalf("无法创建QLink节点: %v", err)
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动节点
	if err := node.Start(ctx); err != nil {
		log.Fatalf("无法启动QLink节点: %v", err)
	}

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	sig := <-sigCh
	log.Printf("收到信号 %v，开始优雅关闭...", sig)

	// 取消上下文
	cancel()

	// 停止节点
	if err := node.Stop(); err != nil {
		log.Printf("停止节点时出错: %v", err)
	}

	log.Println("程序退出")
}