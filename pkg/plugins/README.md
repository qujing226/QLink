# QLink 插件系统

QLink 插件系统提供了一个灵活、可扩展的架构，支持动态加载、卸载和热重载插件。

## 功能特性

### 🔌 插件管理
- **插件注册与卸载**: 动态注册和卸载插件
- **生命周期管理**: 统一的插件启动、停止和状态管理
- **配置管理**: 灵活的插件配置系统
- **状态监控**: 实时监控插件运行状态

### 🔥 热加载支持
- **文件监控**: 自动监控插件文件变化
- **热重载**: 无需重启系统即可更新插件
- **延迟加载**: 防止文件写入冲突的智能延迟机制
- **重试机制**: 自动重试失败的加载操作

### 📊 插件类型
- **DID插件**: 分布式身份标识管理
- **网络插件**: 网络连接和监控功能
- **加密插件**: 加密服务提供者
- **自定义插件**: 支持用户自定义插件类型

## 快速开始

### 1. 创建插件管理器

```go
package main

import (
    "context"
    "log"
    
    "github.com/qujing226/QLink/pkg/plugins"
)

func main() {
    // 创建插件管理器
    manager := plugins.NewPluginManager()
    
    // 创建热加载器
    hotReloader := plugins.NewHotReloader(manager)
    
    // 添加监控路径
    if err := hotReloader.AddWatchPath("./plugins"); err != nil {
        log.Fatal(err)
    }
    
    // 启动系统
    ctx := context.Background()
    if err := hotReloader.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer hotReloader.Stop()
    
    if err := manager.StartAll(ctx); err != nil {
        log.Fatal(err)
    }
    defer manager.StopAll()
}
```

### 2. 注册插件

```go
// 注册DID插件
didPlugin := plugins.NewSimpleDIDPlugin()
if err := manager.RegisterPlugin(didPlugin); err != nil {
    log.Printf("注册DID插件失败: %v", err)
}

// 注册网络监控插件
networkPlugin := plugins.NewNetworkMonitorPlugin()
if err := manager.RegisterPlugin(networkPlugin); err != nil {
    log.Printf("注册网络监控插件失败: %v", err)
}
```

### 3. 插件管理操作

```go
// 获取插件信息
info, err := manager.GetPluginInfo("simple-did")
if err != nil {
    log.Printf("获取插件信息失败: %v", err)
}

// 重启插件
if err := manager.RestartPlugin("network-monitor"); err != nil {
    log.Printf("重启插件失败: %v", err)
}

// 获取插件状态
status, err := manager.GetPluginStatus("simple-did")
if err != nil {
    log.Printf("获取插件状态失败: %v", err)
}

// 列出所有插件
plugins := manager.ListPlugins()
for _, plugin := range plugins {
    fmt.Printf("插件: %s, 状态: %s\n", plugin["name"], plugin["status"])
}
```

## 插件开发

### 实现插件接口

所有插件都必须实现 `interfaces.Plugin` 接口：

```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Initialize(config map[string]interface{}) error
    Start(ctx context.Context) error
    Stop() error
    Status() PluginStatus
    Config() map[string]interface{}
}
```

### 示例插件实现

```go
package main

import (
    "context"
    "github.com/qujing226/QLink/pkg/interfaces"
)

type MyPlugin struct {
    name   string
    status interfaces.PluginStatus
    config map[string]interface{}
}

func (p *MyPlugin) Name() string {
    return p.name
}

func (p *MyPlugin) Version() string {
    return "1.0.0"
}

func (p *MyPlugin) Description() string {
    return "My custom plugin"
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    p.status = interfaces.PluginStatusStopped
    return nil
}

func (p *MyPlugin) Start(ctx context.Context) error {
    p.status = interfaces.PluginStatusRunning
    return nil
}

func (p *MyPlugin) Stop() error {
    p.status = interfaces.PluginStatusStopped
    return nil
}

func (p *MyPlugin) Status() interfaces.PluginStatus {
    return p.status
}

func (p *MyPlugin) Config() map[string]interface{} {
    return p.config
}

// 插件工厂函数（用于动态加载）
func NewPlugin() interfaces.Plugin {
    return &MyPlugin{
        name:   "my-plugin",
        config: make(map[string]interface{}),
    }
}
```

### 专用插件接口

#### DID插件

实现 `interfaces.DIDPlugin` 接口：

```go
type DIDPlugin interface {
    Plugin
    RegisterDID(did string, document interface{}) error
    UpdateDID(did string, document interface{}) error
    RevokeDID(did string) error
    ResolveDID(did string) (interface{}, error)
    ValidateDocument(document interface{}) error
}
```

#### 网络插件

实现 `interfaces.NetworkPlugin` 接口：

```go
type NetworkPlugin interface {
    Plugin
    Connect(address string) error
    Disconnect(address string) error
    SendMessage(address string, message interface{}) error
    BroadcastMessage(message interface{}) error
    GetConnectedPeers() []string
    GetNetworkStats() NetworkStats
}
```

## 热加载配置

### 配置选项

```go
type HotReloadConfig struct {
    // 监控间隔
    WatchInterval time.Duration
    // 插件文件扩展名
    PluginExtension string
    // 是否自动重载
    AutoReload bool
    // 重载延迟
    ReloadDelay time.Duration
    // 最大重试次数
    MaxRetries int
}
```

### 设置配置

```go
hotReloader.SetConfig(plugins.HotReloadConfig{
    WatchInterval:   2 * time.Second,
    PluginExtension: ".so",
    AutoReload:      true,
    ReloadDelay:     1 * time.Second,
    MaxRetries:      3,
})
```

## 动态插件加载

### 编译动态插件

```bash
# 编译为动态库
go build -buildmode=plugin -o myplugin.so myplugin.go
```

### 加载动态插件

```go
// 手动加载插件
if err := hotReloader.LoadPlugin("./plugins/myplugin.so"); err != nil {
    log.Printf("加载插件失败: %v", err)
}

// 重载插件
if err := hotReloader.ReloadPlugin("./plugins/myplugin.so"); err != nil {
    log.Printf("重载插件失败: %v", err)
}
```

## 监控和指标

### 获取插件指标

```go
metrics := plugins.GetPluginMetrics(manager)
fmt.Printf("插件总数: %d\n", metrics["total_plugins"])
fmt.Printf("状态分布: %v\n", metrics["status_distribution"])
```

### 网络插件统计

```go
if networkPlugin, err := manager.GetPlugin("network-monitor"); err == nil {
    if np, ok := networkPlugin.(*plugins.NetworkMonitorPlugin); ok {
        stats := np.GetNetworkStats()
        fmt.Printf("连接节点数: %d\n", stats.ConnectedPeers)
        fmt.Printf("发送消息数: %d\n", stats.MessagesSent)
        fmt.Printf("运行时间: %d秒\n", stats.Uptime)
    }
}
```

## 最佳实践

### 1. 错误处理
- 始终检查插件操作的返回错误
- 实现适当的错误恢复机制
- 记录详细的错误信息

### 2. 资源管理
- 在插件的 `Stop()` 方法中清理所有资源
- 避免在插件中创建全局状态
- 使用上下文进行优雅关闭

### 3. 配置管理
- 验证插件配置的有效性
- 提供合理的默认配置
- 支持配置热更新

### 4. 并发安全
- 使用适当的同步机制保护共享状态
- 避免在插件间共享可变状态
- 实现线程安全的插件接口

### 5. 性能优化
- 避免在插件初始化时执行耗时操作
- 使用异步处理长时间运行的任务
- 实现适当的缓存机制

## 故障排除

### 常见问题

1. **插件加载失败**
   - 检查插件文件是否存在
   - 验证插件文件格式是否正确
   - 确认插件实现了必需的接口

2. **热重载不工作**
   - 检查监控路径是否正确
   - 验证文件权限
   - 确认热加载器已启动

3. **插件启动失败**
   - 检查插件配置是否有效
   - 验证依赖项是否满足
   - 查看详细的错误日志

### 调试技巧

- 启用详细日志记录
- 使用插件状态监控
- 检查插件指标和统计信息
- 验证插件配置

## 示例代码

完整的使用示例请参考 `example_usage.go` 文件，其中包含了：
- 插件管理器的创建和配置
- 插件的注册和管理
- 热加载功能的使用
- 性能指标的获取

## 贡献

欢迎提交问题报告和功能请求。在开发新插件时，请遵循现有的代码风格和最佳实践。