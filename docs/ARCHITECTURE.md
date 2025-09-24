# QLink 架构文档

## 项目概述

QLink 是一个基于区块链的去中心化身份(DID)系统，支持多种共识算法和灵活的插件架构。

## 核心架构

### 1. 模块化设计

项目采用模块化设计，主要包含以下核心模块：

```
QLink/
├── cmd/                    # 命令行工具
│   ├── qlink-cli/         # 客户端工具
│   └── qlink-node/        # 节点程序
├── pkg/                   # 核心包
│   ├── api/              # API服务
│   ├── blockchain/       # 区块链实现
│   ├── config/           # 配置管理
│   ├── consensus/        # 共识算法
│   ├── interfaces/       # 统一接口定义
│   ├── network/          # 网络通信
│   ├── sync/             # 数据同步
│   ├── types/            # 通用类型
│   └── utils/            # 工具函数
├── did/                  # DID相关实现
└── config/               # 配置文件
```

### 2. 接口设计

#### 2.1 共识接口 (pkg/interfaces/consensus.go)

- **ConsensusAlgorithm**: 统一的共识算法接口
- **ConsensusEngine**: 区块链共识引擎接口
- **ConsensusFactory**: 共识算法工厂接口
- **ConsensusAdapter**: 共识适配器接口

#### 2.2 存储接口 (pkg/interfaces/storage.go)

- **Storage**: 通用存储接口
- **BlockchainStorage**: 区块链专用存储
- **DIDStorage**: DID专用存储
- **CacheStorage**: 缓存存储接口

#### 2.3 插件接口 (pkg/interfaces/plugin.go)

- **Plugin**: 通用插件接口
- **DIDPlugin**: DID插件接口
- **NetworkPlugin**: 网络插件接口
- **CryptoProvider**: 加密服务提供者接口

#### 2.4 适配器接口 (pkg/interfaces/adapter.go)

- **ConsensusAdapter**: 共识适配器
- **BlockchainAdapter**: 区块链适配器
- **NetworkAdapter**: 网络适配器
- **ServiceAdapter**: 服务适配器

### 3. 共识算法架构

#### 3.1 支持的共识算法

- **Raft**: 适用于小规模网络的强一致性算法
- **PoA (Proof of Authority)**: 权威证明，适用于联盟链
- **PBFT (Practical Byzantine Fault Tolerance)**: 拜占庭容错算法

#### 3.2 共识切换机制

系统支持动态共识算法切换，包括：

- **优雅切换**: 等待当前操作完成后切换
- **立即切换**: 立即停止当前算法并启动新算法
- **滚动切换**: 逐步迁移节点到新算法
- **蓝绿切换**: 并行运行两套系统后切换

#### 3.3 监控和恢复

- **ConsensusMonitor**: 实时监控共识状态
- **FailureDetector**: 故障检测机制
- **RecoveryManager**: 自动恢复管理

### 4. DID系统架构

#### 4.1 核心组件

- **DIDRegistry**: DID注册表
- **DIDResolver**: DID解析器
- **DocumentManager**: 文档管理器
- **CryptoProvider**: 加密服务提供者

#### 4.2 区块链集成

- **BlockchainInterface**: 统一的区块链接口
- **MockBlockchain**: 测试用模拟区块链
- **TransactionManager**: 交易管理器

### 5. 网络架构

#### 5.1 P2P网络

- **P2PNetwork**: 点对点网络实现
- **PeerManager**: 节点管理
- **MessageRouter**: 消息路由

#### 5.2 集群管理

- **ClusterManager**: 集群管理器
- **NodeDiscovery**: 节点发现
- **LoadBalancer**: 负载均衡

### 6. 配置管理

#### 6.1 配置结构

```go
type Config struct {
    Node      *NodeConfig      // 节点配置
    Network   *NetworkConfig   // 网络配置
    Consensus *ConsensusConfig // 共识配置
    API       *APIConfig       // API配置
    Storage   *StorageConfig   // 存储配置
    DID       *DIDConfig       // DID配置
}
```

#### 6.2 配置文件类型

- **统一配置**: unified.yaml
- **节点配置**: node1.yaml, node2.yaml, node3.yaml
- **共识配置**: consensus_node1.yaml, consensus_node2.yaml, consensus_node3.yaml
- **网关配置**: gateway_node.yaml

### 7. API架构

#### 7.1 REST API

- **DID操作**: 创建、更新、查询、撤销DID
- **共识管理**: 状态查询、算法切换、监控指标
- **网络管理**: 节点管理、连接状态、网络统计

#### 7.2 中间件

- **认证中间件**: JWT认证
- **限流中间件**: 请求频率限制
- **日志中间件**: 请求日志记录
- **CORS中间件**: 跨域请求支持

### 8. 存储架构

#### 8.1 存储层次

- **持久化存储**: LevelDB/BadgerDB
- **缓存层**: 内存缓存
- **分布式存储**: 支持集群存储

#### 8.2 数据模型

- **区块数据**: 区块头、交易列表、状态根
- **DID数据**: DID文档、验证方法、服务端点
- **共识数据**: 日志条目、快照、状态机

### 9. 安全架构

#### 9.1 加密算法

- **对称加密**: AES-256
- **非对称加密**: RSA-2048, ECDSA
- **哈希算法**: SHA-256, SHA-3
- **后量子加密**: Kyber, Dilithium

#### 9.2 安全机制

- **数字签名**: 交易和消息签名
- **证书管理**: X.509证书链
- **密钥管理**: 分层确定性钱包
- **访问控制**: 基于角色的权限管理

### 10. 部署架构

#### 10.1 节点类型

- **网关节点**: 提供API服务和负载均衡
- **共识节点**: 参与共识算法的核心节点
- **验证节点**: 验证交易和区块的节点
- **完整节点**: 包含所有功能的节点

#### 10.2 部署模式

- **单机部署**: 开发和测试环境
- **集群部署**: 生产环境多节点部署
- **容器化部署**: Docker和Kubernetes支持
- **云原生部署**: 支持主流云平台

## 设计原则

### 1. 模块化

- 清晰的模块边界
- 松耦合设计
- 高内聚实现

### 2. 可扩展性

- 插件化架构
- 接口驱动设计
- 动态配置支持

### 3. 高可用性

- 故障检测和恢复
- 负载均衡
- 数据备份和恢复

### 4. 安全性

- 端到端加密
- 身份认证和授权
- 审计日志

### 5. 性能优化

- 异步处理
- 缓存机制
- 批量操作

## 未来规划

### 1. 功能扩展

- 支持更多共识算法
- 增强DID功能
- 跨链互操作性

### 2. 性能优化

- 并行处理优化
- 存储性能提升
- 网络通信优化

### 3. 生态建设

- SDK开发
- 开发者工具
- 社区建设