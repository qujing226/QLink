package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/pkg/interfaces"
	"github.com/qujing226/QLink/pkg/types"
)

// SimpleDIDPlugin 简化版DID插件实现
type SimpleDIDPlugin struct {
	name        string
	version     string
	description string
	config      map[string]interface{}
	status      interfaces.PluginStatus
	
	// DID相关组件
	registry *did.DIDRegistry
	
	// 状态管理
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	
	// 统计信息
	stats *SimpleDIDStats
}

// SimpleDIDStats 简化版DID插件统计信息
type SimpleDIDStats struct {
	RegisteredDIDs int64     `json:"registered_dids"`
	UpdatedDIDs    int64     `json:"updated_dids"`
	RevokedDIDs    int64     `json:"revoked_dids"`
	ResolvedDIDs   int64     `json:"resolved_dids"`
	StartTime      time.Time `json:"start_time"`
	LastActivity   time.Time `json:"last_activity"`
}

// NewSimpleDIDPlugin 创建新的简化版DID插件
func NewSimpleDIDPlugin() *SimpleDIDPlugin {
	return &SimpleDIDPlugin{
		name:        "simple-did-plugin",
		version:     "1.0.0",
		description: "QLink简化版DID管理插件，提供基础DID功能",
		config:      make(map[string]interface{}),
		status:      interfaces.PluginStatusStopped,
		stats: &SimpleDIDStats{
			StartTime: time.Now(),
		},
	}
}

// 实现Plugin接口
func (dp *SimpleDIDPlugin) Name() string {
	return dp.name
}

func (dp *SimpleDIDPlugin) Version() string {
	return dp.version
}

func (dp *SimpleDIDPlugin) Description() string {
	return dp.description
}

func (dp *SimpleDIDPlugin) Initialize(config map[string]interface{}) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	dp.config = config
	
	// 初始化DID注册表
	dp.registry = did.NewDIDRegistry(nil) // 不使用区块链接口
	
	dp.status = interfaces.PluginStatusStopped
	return nil
}

func (dp *SimpleDIDPlugin) Start(ctx context.Context) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	if dp.started {
		return fmt.Errorf("插件已经启动")
	}
	
	dp.status = interfaces.PluginStatusStarting
	dp.ctx, dp.cancel = context.WithCancel(ctx)
	
	dp.started = true
	dp.status = interfaces.PluginStatusRunning
	dp.stats.StartTime = time.Now()
	
	return nil
}

func (dp *SimpleDIDPlugin) Stop() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	
	if !dp.started {
		return nil
	}
	
	dp.status = interfaces.PluginStatusStopping
	
	// 取消上下文
	if dp.cancel != nil {
		dp.cancel()
	}
	
	dp.started = false
	dp.status = interfaces.PluginStatusStopped
	
	return nil
}

func (dp *SimpleDIDPlugin) Status() interfaces.PluginStatus {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.status
}

func (dp *SimpleDIDPlugin) Config() map[string]interface{} {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	// 返回配置的副本
	config := make(map[string]interface{})
	for k, v := range dp.config {
		config[k] = v
	}
	return config
}

// 实现DIDPlugin接口
func (dp *SimpleDIDPlugin) RegisterDID(didStr string, document interface{}) error {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	if !dp.started {
		return fmt.Errorf("插件未启动")
	}
	
	// 创建注册请求
	req := &did.RegisterRequest{
		DID: didStr,
	}
	
	// 注册DID
	_, err := dp.registry.Register(req)
	if err != nil {
		return fmt.Errorf("注册DID失败: %w", err)
	}
	
	// 更新统计信息
	dp.stats.RegisteredDIDs++
	dp.stats.LastActivity = time.Now()
	
	return nil
}

func (dp *SimpleDIDPlugin) UpdateDID(didStr string, document interface{}) error {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	if !dp.started {
		return fmt.Errorf("插件未启动")
	}
	
	// 创建更新请求
	req := &did.UpdateRequest{
		DID: didStr,
	}
	
	// 更新DID
	_, err := dp.registry.Update(req)
	if err != nil {
		return fmt.Errorf("更新DID失败: %w", err)
	}
	
	// 更新统计信息
	dp.stats.UpdatedDIDs++
	dp.stats.LastActivity = time.Now()
	
	return nil
}

func (dp *SimpleDIDPlugin) RevokeDID(didStr string) error {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	if !dp.started {
		return fmt.Errorf("插件未启动")
	}
	
	// 撤销DID
	err := dp.registry.Revoke(didStr, nil)
	if err != nil {
		return fmt.Errorf("撤销DID失败: %w", err)
	}
	
	// 更新统计信息
	dp.stats.RevokedDIDs++
	dp.stats.LastActivity = time.Now()
	
	return nil
}

func (dp *SimpleDIDPlugin) ResolveDID(didStr string) (interface{}, error) {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	if !dp.started {
		return nil, fmt.Errorf("插件未启动")
	}
	
	// 解析DID
	document, err := dp.registry.Resolve(didStr)
	if err != nil {
		return nil, fmt.Errorf("解析DID失败: %w", err)
	}
	
	// 更新统计信息
	dp.stats.ResolvedDIDs++
	dp.stats.LastActivity = time.Now()
	
	return document, nil
}

func (dp *SimpleDIDPlugin) ValidateDocument(document interface{}) error {
	// 类型断言检查文档类型
	didDoc, ok := document.(*types.DIDDocument)
	if !ok {
		return fmt.Errorf("无效的DID文档类型")
	}
	
	// 基础验证
	if didDoc.ID == "" {
		return fmt.Errorf("DID文档缺少ID")
	}
	
	return nil
}

// 插件特有的方法
func (dp *SimpleDIDPlugin) GetStats() *SimpleDIDStats {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	
	// 返回统计信息的副本
	return &SimpleDIDStats{
		RegisteredDIDs: dp.stats.RegisteredDIDs,
		UpdatedDIDs:    dp.stats.UpdatedDIDs,
		RevokedDIDs:    dp.stats.RevokedDIDs,
		ResolvedDIDs:   dp.stats.ResolvedDIDs,
		StartTime:      dp.stats.StartTime,
		LastActivity:   dp.stats.LastActivity,
	}
}

func (dp *SimpleDIDPlugin) CreateDIDDocument() (*types.DIDDocument, error) {
	// 创建DID文档构建器
	builder, err := did.NewDIDDocumentBuilder()
	if err != nil {
		return nil, fmt.Errorf("创建DID文档构建器失败: %w", err)
	}
	
	// 构建DID文档
	document, err := builder.BuildDocument()
	if err != nil {
		return nil, fmt.Errorf("构建DID文档失败: %w", err)
	}
	
	return document, nil
}

func (dp *SimpleDIDPlugin) GetRegistry() *did.DIDRegistry {
	dp.mu.RLock()
	defer dp.mu.RUnlock()
	return dp.registry
}

// 确保SimpleDIDPlugin实现了DIDPlugin接口
var _ interfaces.DIDPlugin = (*SimpleDIDPlugin)(nil)