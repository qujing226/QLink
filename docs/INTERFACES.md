# QLink 接口设计文档

## 概述

QLink 项目采用接口驱动的设计模式，通过统一的接口定义实现模块间的松耦合。所有核心接口定义在 `pkg/interfaces/` 目录下。

## 接口分类

### 1. 共识接口 (consensus.go)

#### 1.1 ConsensusType 枚举

```go
type ConsensusType int

const (
    ConsensusTypeRaft ConsensusType = iota
    ConsensusTypePoA
    ConsensusTypePBFT
)
```

支持的共识算法类型：
- **Raft**: 适用于小规模网络的强一致性算法
- **PoA**: 权威证明，适用于联盟链
- **PBFT**: 拜占庭容错算法

#### 1.2 ConsensusAlgorithm 接口

```go
type ConsensusAlgorithm interface {
    Start() error
    Stop() error
    Submit(proposal interface{}) error
    GetStatus() map[string]interface{}
    GetLeader() string
    GetNodes() []string
}
```

**用途**: 定义共识算法的基本操作
**实现**: 各种共识算法的具体实现

#### 1.3 ConsensusEngine 接口

```go
type ConsensusEngine interface {
    ValidateBlock(block interface{}) error
    ValidateProposer(proposer string, blockNumber uint64) error
    GetNextProposer(blockNumber uint64) string
    IsAuthority(address string) bool
    GetAuthorities() []string
    Start() error
    Stop() error
}
```

**用途**: 区块链共识引擎的核心功能
**实现**: 区块链层面的共识逻辑

#### 1.4 ConsensusFactory 接口

```go
type ConsensusFactory interface {
    CreateConsensus(consensusType ConsensusType, config map[string]interface{}) (ConsensusAlgorithm, error)
    GetSupportedTypes() []ConsensusType
    ValidateConfig(consensusType ConsensusType, config map[string]interface{}) error
}
```

**用途**: 工厂模式创建共识算法实例
**实现**: 共识算法的工厂类

### 2. 存储接口 (storage.go)

#### 2.1 Storage 基础接口

```go
type Storage interface {
    Get(key []byte) ([]byte, error)
    Put(key, value []byte) error
    Delete(key []byte) error
    Has(key []byte) (bool, error)
    Close() error
    NewBatch() Batch
    NewIterator(prefix []byte) Iterator
}
```

**用途**: 通用的键值存储接口
**实现**: LevelDB, BadgerDB, 内存存储等

#### 2.2 BlockchainStorage 接口

```go
type BlockchainStorage interface {
    Storage
    
    // 区块操作
    GetBlock(hash []byte) (interface{}, error)
    PutBlock(hash []byte, block interface{}) error
    GetBlockByNumber(number uint64) (interface{}, error)
    
    // 交易操作
    GetTransaction(hash []byte) (interface{}, error)
    PutTransaction(hash []byte, tx interface{}) error
    
    // 状态操作
    GetState(key []byte) ([]byte, error)
    PutState(key, value []byte) error
    
    // 元数据
    GetLatestBlockNumber() (uint64, error)
    SetLatestBlockNumber(number uint64) error
}
```

**用途**: 区块链专用存储操作
**实现**: 区块链数据的持久化存储

#### 2.3 DIDStorage 接口

```go
type DIDStorage interface {
    Storage
    
    // DID文档操作
    GetDIDDocument(did string) (interface{}, error)
    PutDIDDocument(did string, doc interface{}) error
    DeleteDIDDocument(did string) error
    
    // DID状态管理
    GetDIDStatus(did string) (string, error)
    SetDIDStatus(did string, status string) error
    
    // 查询操作
    ListDIDs(limit int, offset int) ([]string, error)
    SearchDIDs(query map[string]interface{}) ([]string, error)
}
```

**用途**: DID专用存储操作
**实现**: DID文档和状态的存储管理

### 3. 插件接口 (plugin.go)

#### 3.1 Plugin 基础接口

```go
type Plugin interface {
    GetName() string
    GetVersion() string
    Initialize(config map[string]interface{}) error
    Start() error
    Stop() error
    GetStatus() PluginStatus
    GetMetrics() map[string]interface{}
}
```

**用途**: 所有插件的基础接口
**实现**: 各种功能插件的基类

#### 3.2 DIDPlugin 接口

```go
type DIDPlugin interface {
    Plugin
    
    CreateDID(method string, options map[string]interface{}) (string, error)
    ResolveDID(did string) (interface{}, error)
    UpdateDID(did string, updates map[string]interface{}) error
    DeactivateDID(did string) error
    
    ValidateDID(did string) error
    GetSupportedMethods() []string
}
```

**用途**: DID相关功能的插件接口
**实现**: 不同DID方法的插件实现

#### 3.3 NetworkPlugin 接口

```go
type NetworkPlugin interface {
    Plugin
    
    Connect(address string) error
    Disconnect(nodeID string) error
    Broadcast(message interface{}) error
    SendTo(nodeID string, message interface{}) error
    
    GetPeers() []string
    GetNetworkStats() NetworkStats
}
```

**用途**: 网络通信功能的插件接口
**实现**: P2P网络、HTTP通信等插件

#### 3.4 CryptoProvider 接口

```go
type CryptoProvider interface {
    Plugin
    
    GenerateKeyPair(algorithm string) (publicKey interface{}, privateKey interface{}, err error)
    Sign(privateKey interface{}, data []byte) ([]byte, error)
    Verify(publicKey interface{}, data, signature []byte) error
    
    Encrypt(publicKey interface{}, plaintext []byte) ([]byte, error)
    Decrypt(privateKey interface{}, ciphertext []byte) ([]byte, error)
    
    Hash(data []byte, algorithm string) ([]byte, error)
    GetSupportedAlgorithms() []string
}
```

**用途**: 加密服务提供者接口
**实现**: 各种加密算法的插件实现

### 4. 适配器接口 (adapter.go)

#### 4.1 ConsensusAdapter 接口

```go
type ConsensusAdapter interface {
    GetType() ConsensusType
    GetName() string
    
    StartConsensus() error
    StopConsensus() error
    Submit(proposal interface{}) error
    GetStatus() map[string]interface{}
    GetLeader() string
    GetNodes() []string
    
    ValidateBlock(block interface{}) error
    ValidateProposer(proposer string, blockNumber uint64) error
    GetNextProposer(blockNumber uint64) string
    IsAuthority(address string) bool
    GetAuthorities() []string
}
```

**用途**: 统一不同共识实现的适配器
**实现**: 各种共识算法的适配器实现

#### 4.2 BlockchainAdapter 接口

```go
type BlockchainAdapter interface {
    Connect() error
    Disconnect() error
    IsConnected() bool
    
    GetLatestBlock() (interface{}, error)
    GetBlock(hash string) (interface{}, error)
    GetBlockHeight() (int64, error)
    
    SubmitTransaction(tx interface{}) error
    GetTransaction(hash string) (interface{}, error)
    GetTransactionStatus(hash string) (string, error)
}
```

**用途**: 统一不同区块链实现的适配器
**实现**: 各种区块链的适配器实现

## 接口使用指南

### 1. 实现接口

```go
// 实现共识算法接口
type RaftConsensus struct {
    // 内部状态
}

func (r *RaftConsensus) Start() error {
    // 启动Raft共识
    return nil
}

func (r *RaftConsensus) Stop() error {
    // 停止Raft共识
    return nil
}

// 实现其他方法...
```

### 2. 使用工厂模式

```go
// 创建共识算法实例
factory := &ConsensusFactoryImpl{}
consensus, err := factory.CreateConsensus(ConsensusTypeRaft, config)
if err != nil {
    return err
}

// 启动共识
err = consensus.Start()
```

### 3. 插件注册

```go
// 注册插件
pluginManager := &PluginManagerImpl{}
didPlugin := &MyDIDPlugin{}

err := pluginManager.RegisterPlugin("my-did", didPlugin)
if err != nil {
    return err
}

// 启动插件
err = pluginManager.StartPlugin("my-did")
```

### 4. 适配器模式

```go
// 使用适配器统一接口
adapter := &EthereumAdapter{
    client: ethClient,
}

// 统一的区块链操作
block, err := adapter.GetLatestBlock()
if err != nil {
    return err
}
```

## 接口扩展

### 1. 添加新的共识算法

1. 实现 `ConsensusAlgorithm` 接口
2. 在 `ConsensusType` 中添加新类型
3. 在工厂类中添加创建逻辑
4. 编写单元测试

### 2. 添加新的存储后端

1. 实现 `Storage` 接口
2. 根据需要实现专用存储接口
3. 在配置中添加新的存储类型
4. 编写集成测试

### 3. 开发新插件

1. 实现 `Plugin` 基础接口
2. 根据功能实现专用插件接口
3. 注册到插件管理器
4. 编写插件文档

## 最佳实践

### 1. 接口设计原则

- **单一职责**: 每个接口只负责一个功能领域
- **接口隔离**: 客户端不应依赖不需要的接口
- **依赖倒置**: 依赖抽象而不是具体实现

### 2. 错误处理

- 使用Go标准的错误处理模式
- 定义特定的错误类型
- 提供详细的错误信息

### 3. 配置管理

- 使用map[string]interface{}传递配置
- 提供配置验证方法
- 支持配置的动态更新

### 4. 测试策略

- 为每个接口编写mock实现
- 使用接口进行单元测试
- 编写集成测试验证接口协作

### 5. 文档维护

- 保持接口文档的及时更新
- 提供使用示例
- 记录接口变更历史

## 接口版本管理

### 1. 版本策略

- 使用语义化版本控制
- 向后兼容的变更为小版本更新
- 破坏性变更为大版本更新

### 2. 废弃策略

- 标记废弃的接口和方法
- 提供迁移指南
- 保持至少一个大版本的兼容性

### 3. 变更通知

- 在变更日志中记录接口变更
- 通过文档和注释说明变更原因
- 提供升级指南和示例代码