package app

import (
    "context"
    "fmt"
    "log"
    "sync"

    "github.com/qujing226/QLink/did"
    didblockchain "github.com/qujing226/QLink/did/blockchain"
    "github.com/qujing226/QLink/pkg/api"
    "github.com/qujing226/QLink/pkg/config"
    "github.com/qujing226/QLink/pkg/consensus"
    "github.com/qujing226/QLink/pkg/network"
    "github.com/qujing226/QLink/pkg/storage"
    syncpkg "github.com/qujing226/QLink/pkg/sync"
)

// Application 应用程序主结构
type Application struct {
    config           *config.Config
    mu               sync.RWMutex
    storageManager   *storage.StorageManager
    didRegistry      *did.DIDRegistry
    didResolver      *did.DIDResolver
    blockchain       didblockchain.BlockchainInterface
    p2pNetwork       *network.P2PNetwork
    consensusManager *consensus.ConsensusManager
    synchronizer     *syncpkg.Synchronizer
    apiServer        *api.Server
    started          bool
    ctx              context.Context
    cancel           context.CancelFunc
}

// NewApplication 创建新的应用程序实例
func NewApplication(cfg *config.Config) *Application {
	return &Application{
		config: cfg,
	}
}

// Initialize 初始化应用程序
func (app *Application) Initialize() error {
    log.Println("开始初始化应用程序...")

    // 1. 初始化存储管理器
    var err error
    sf := storage.NewStorageFactory()
    app.storageManager, err = sf.CreateDefaultStorageManager()
    if err != nil {
        return fmt.Errorf("初始化存储管理器失败: %v", err)
    }
    // 启动所有存储并加载DID文档
    if err := app.storageManager.StartAll(context.Background()); err != nil {
        return fmt.Errorf("启动存储失败: %v", err)
    }

	// 2. 初始化区块链接口
	blockchainConfig := &didblockchain.BlockchainConfig{
		Type: "mock",
	}
	app.blockchain, err = didblockchain.NewBlockchainFactory().CreateBlockchain(blockchainConfig)
	if err != nil {
		return fmt.Errorf("初始化区块链失败: %v", err)
	}

    // 3. 初始化DID注册表和解析器
    app.didRegistry = did.NewDIDRegistry(app.blockchain, app.storageManager)
    app.didResolver = did.NewDIDResolver(app.config, app.didRegistry, app.storageManager)

	// 4. 初始化网络组件
	if app.config.Network != nil {
		app.p2pNetwork = network.NewP2PNetwork(
			app.config.GetNodeID(),
			app.config.Network.ListenAddress,
			app.config.Network.ListenPort,
			app.config.Network,
		)
	}

	// 5. 初始化共识管理器
	if app.config.Consensus != nil {
		consensusConfig := &consensus.ManagerConfig{
			NodeID:           app.config.GetNodeID(),
			DefaultConsensus: consensus.ConsensusTypeRaft,
		}
		app.consensusManager = consensus.NewConsensusManager(consensusConfig, app.p2pNetwork)
	}

	// 6. 初始化同步器
	app.synchronizer = syncpkg.NewSynchronizer(
		app.config.GetNodeID(),
		app.didRegistry,
		app.p2pNetwork,
		app.config.Sync,
	)

    // 7. 初始化API服务器
    if app.config.API != nil {
        app.apiServer = api.NewServer(
            app.config,
            app.storageManager,
            app.didRegistry,
            app.didResolver,
            nil, // 暂时传nil
        )
    }

	log.Println("应用程序初始化完成")
	return nil
}

// Start 启动应用程序
func (app *Application) Start(ctx context.Context) error {
	log.Println("启动应用程序...")

	// 启动网络组件
	if app.p2pNetwork != nil {
		if err := app.p2pNetwork.Start(ctx); err != nil {
			return fmt.Errorf("启动P2P网络失败: %v", err)
		}
	}

	// 启动共识管理器
	if app.consensusManager != nil {
		if err := app.consensusManager.Start(ctx); err != nil {
			return fmt.Errorf("启动共识管理器失败: %v", err)
		}
	}

	// 启动同步器
	if app.synchronizer != nil {
		if err := app.synchronizer.Start(ctx); err != nil {
			return fmt.Errorf("启动同步器失败: %v", err)
		}
	}

	// 启动API服务器
	if app.apiServer != nil {
		if err := app.apiServer.Start(); err != nil {
			return fmt.Errorf("启动API服务器失败: %v", err)
		}
	}

	log.Println("应用程序启动完成")
	return nil
}

// Stop 停止应用程序
func (app *Application) Stop() error {
	log.Println("停止应用程序...")

	// 停止API服务器
	if app.apiServer != nil {
		if err := app.apiServer.Stop(); err != nil {
			log.Printf("停止API服务器失败: %v", err)
		}
	}

	// 停止同步器
	if app.synchronizer != nil {
		if err := app.synchronizer.Stop(); err != nil {
			log.Printf("停止同步器失败: %v", err)
		}
	}

	// 停止共识管理器
	if app.consensusManager != nil {
		if err := app.consensusManager.Stop(); err != nil {
			log.Printf("停止共识管理器失败: %v", err)
		}
	}

	// 停止网络组件
	if app.p2pNetwork != nil {
		if err := app.p2pNetwork.Stop(); err != nil {
			log.Printf("停止P2P网络失败: %v", err)
		}
	}

	// 断开区块链连接
	if app.blockchain != nil {
		if err := app.blockchain.Disconnect(); err != nil {
			log.Printf("断开区块链连接失败: %v", err)
		}
	}

	log.Println("应用程序停止完成")
	return nil
}


// GetNodeID 获取节点ID
func (app *Application) GetNodeID() string {
	app.mu.RLock()
	defer app.mu.RUnlock()

	if app.config != nil && app.config.Node != nil {
		return app.config.Node.ID
	}
	return "unknown"
}

// GetAPIAddress 获取API地址
func (app *Application) GetAPIAddress() string {
	app.mu.RLock()
	defer app.mu.RUnlock()

	if app.config != nil && app.config.API != nil {
		return fmt.Sprintf("%s:%d", app.config.API.Host, app.config.API.Port)
	}
	return "unknown"
}

// GetP2PAddress 获取P2P地址
func (app *Application) GetP2PAddress() string {
	app.mu.RLock()
	defer app.mu.RUnlock()

	if app.config != nil && app.config.Network != nil {
		return fmt.Sprintf("%s:%d", app.config.Network.ListenAddress, app.config.Network.ListenPort)
	}
	return "unknown"
}
