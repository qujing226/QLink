# QLink 共识算法模块

本模块实现了QLink DID系统的共识算法，支持多种共识机制的动态切换和监控。

## 功能特性

### 1. 多共识算法支持
- **Raft算法**: 适用于强一致性要求的场景
- **PoA算法**: 适用于联盟链和许可网络
- **动态切换**: 支持运行时在不同共识算法间切换

### 2. 监控和故障恢复
- **性能监控**: 实时监控延迟、吞吐量、成功率等指标
- **故障检测**: 自动检测网络分区、节点故障等问题
- **自动恢复**: 支持多种恢复策略（重启、选举、网络修复等）

### 3. 配置管理
- **灵活配置**: 支持各种参数的动态配置
- **配置切换**: 支持共识算法配置的热切换
- **状态管理**: 完整的状态备份和恢复机制

## 核心组件

### 文件结构
```
consensus/
├── raft.go              # Raft共识算法实现
├── poa.go               # PoA共识算法实现
├── monitoring.go        # 监控和故障恢复
├── switcher.go          # 共识算法切换器
├── integration.go       # 共识管理器集成
├── example.go           # 使用示例
└── README.md           # 本文档
```

### 主要结构体

#### ConsensusManager
共识管理器，整合所有共识相关功能：
```go
type ConsensusManager struct {
    config   *ManagerConfig
    monitor  *ConsensusMonitor
    switcher *ConsensusSwitcher
    // ...
}
```

#### ConsensusMonitor
监控器，负责性能监控和故障恢复：
```go
type ConsensusMonitor struct {
    config          *MonitorConfig
    metrics         *ConsensusMetrics
    failureDetector *FailureDetector
    recoveryManager *RecoveryManager
    // ...
}
```

#### ConsensusSwitcher
切换器，负责共识算法的动态切换：
```go
type ConsensusSwitcher struct {
    config           *SwitcherConfig
    currentConsensus ConsensusAlgorithm
    algorithms       map[ConsensusType]ConsensusAlgorithm
    // ...
}
```

## 使用方法

### 1. 基本使用
```go
// 创建配置
config := &ManagerConfig{
    NodeID:           "node1",
    DefaultConsensus: ConsensusTypeRaft,
    MonitorConfig: &MonitorConfig{
        MonitorInterval: 5 * time.Second,
        MaxLatency:     1 * time.Second,
    },
    SwitcherConfig: &SwitcherConfig{
        EnableAutoSwitch: true,
        SwitchThreshold:  0.8,
    },
}

// 创建管理器
manager := NewConsensusManager(config)

// 启动
ctx := context.Background()
err := manager.Start(ctx)
if err != nil {
    log.Fatal(err)
}

// 提交提案
proposal := &Proposal{
    Type: ProposalTypeDIDCreate,
    Data: didOperation,
}
err = manager.SubmitProposal(proposal)
```

### 2. 监控使用
```go
// 获取监控指标
metrics := manager.GetMetrics()
log.Printf("延迟: %v, 吞吐量: %.2f, 成功率: %.2f", 
    metrics.Latency, metrics.Throughput, metrics.SuccessRate)

// 获取故障历史
failures := manager.GetFailureHistory()
for _, failure := range failures {
    log.Printf("故障: %s, 类型: %d, 时间: %v", 
        failure.ID, failure.Type, failure.Timestamp)
}
```

### 3. 算法切换
```go
// 手动切换到PoA
err := manager.SwitchConsensus(ConsensusTypePoA)
if err != nil {
    log.Printf("切换失败: %v", err)
}

// 获取当前算法类型
currentType := manager.GetCurrentConsensusType()
log.Printf("当前共识算法: %v", currentType)
```

## 配置说明

### MonitorConfig
监控配置参数：
- `MonitorInterval`: 监控间隔
- `MaxLatency`: 最大延迟阈值
- `MinThroughput`: 最小吞吐量阈值
- `MaxFailureRate`: 最大失败率阈值
- `RecoveryTimeout`: 恢复超时时间

### SwitcherConfig
切换器配置参数：
- `EnableAutoSwitch`: 是否启用自动切换
- `SwitchThreshold`: 切换阈值
- `SwitchCooldown`: 切换冷却时间
- `BackupEnabled`: 是否启用状态备份

## 监控指标

### 性能指标
- **延迟 (Latency)**: 操作完成时间
- **吞吐量 (Throughput)**: 每秒处理的操作数
- **成功率 (SuccessRate)**: 操作成功的比例

### 状态指标
- **活跃节点数 (ActiveNodes)**: 当前活跃的节点数量
- **领导者变更次数 (LeaderChanges)**: 领导者变更的次数
- **网络分区数 (NetworkPartitions)**: 检测到的网络分区数量

### 错误指标
- **总错误数 (TotalErrors)**: 累计错误数量
- **连续错误数 (ConsecutiveErrors)**: 连续发生的错误数量
- **最后错误时间 (LastError)**: 最后一次错误的时间

## 故障恢复策略

### 1. 重启恢复 (RecoveryStrategyRestart)
适用于临时性故障，通过重启节点解决问题。

### 2. 领导者选举 (RecoveryStrategyLeaderElection)
适用于领导者故障，触发新的领导者选举。

### 3. 网络修复 (RecoveryStrategyNetworkRepair)
适用于网络分区问题，尝试修复网络连接。

### 4. 数据同步 (RecoveryStrategyDataSync)
适用于数据不一致问题，重新同步数据。

### 5. 回滚恢复 (RecoveryStrategyRollback)
适用于严重错误，回滚到之前的稳定状态。

### 6. 手动干预 (RecoveryStrategyManualIntervention)
适用于复杂问题，需要人工介入处理。

## 注意事项

1. **线程安全**: 所有公共方法都是线程安全的
2. **资源管理**: 使用完毕后需要调用Stop()方法释放资源
3. **配置验证**: 启动前会验证配置的有效性
4. **状态持久化**: 重要状态会自动持久化到磁盘
5. **日志记录**: 所有重要操作都会记录详细日志

## 示例代码

完整的使用示例请参考 `example.go` 文件，包含：
- 基本使用示例
- 监控功能示例
- 算法切换示例
- 集成使用示例

## 扩展开发

### 添加新的共识算法
1. 实现 `ConsensusAlgorithm` 接口
2. 在 `ConsensusType` 中添加新类型
3. 在切换器中注册新算法
4. 更新配置和文档

### 添加新的监控指标
1. 在 `ConsensusMetrics` 中添加新字段
2. 在监控循环中收集新指标
3. 更新API和文档

### 添加新的恢复策略
1. 在 `RecoveryStrategy` 中添加新策略
2. 实现对应的恢复逻辑
3. 更新策略选择算法