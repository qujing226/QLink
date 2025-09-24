package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// PluginManagerImpl 插件管理器实现
type PluginManagerImpl struct {
	mu      sync.RWMutex
	plugins map[string]interfaces.Plugin
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

// NewPluginManager 创建新的插件管理器实例
func NewPluginManager() *PluginManagerImpl {
	return &PluginManagerImpl{
		plugins: make(map[string]interfaces.Plugin),
	}
}

// RegisterPlugin 注册插件
func (pm *PluginManagerImpl) RegisterPlugin(plugin interfaces.Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("插件名称不能为空")
	}
	
	// 检查插件是否已存在
	if _, exists := pm.plugins[name]; exists {
		return fmt.Errorf("插件 %s 已存在", name)
	}
	
	pm.plugins[name] = plugin
	
	// 如果管理器已启动，则自动启动新插件
	if pm.started && pm.ctx != nil {
		if err := plugin.Start(pm.ctx); err != nil {
			// 启动失败，从注册表中移除
			delete(pm.plugins, name)
			return fmt.Errorf("启动插件 %s 失败: %w", name, err)
		}
	}
	
	return nil
}

// UnregisterPlugin 卸载插件
func (pm *PluginManagerImpl) UnregisterPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("插件 %s 不存在", name)
	}
	
	// 停止插件
	if plugin.Status() == interfaces.PluginStatusRunning {
		if err := plugin.Stop(); err != nil {
			return fmt.Errorf("停止插件 %s 失败: %w", name, err)
		}
	}
	
	// 从注册表中移除
	delete(pm.plugins, name)
	
	return nil
}

// GetPlugin 获取插件
func (pm *PluginManagerImpl) GetPlugin(name string) (interfaces.Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("插件 %s 不存在", name)
	}
	
	return plugin, nil
}

// GetAllPlugins 获取所有插件
func (pm *PluginManagerImpl) GetAllPlugins() map[string]interfaces.Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// 返回插件映射的副本
	result := make(map[string]interfaces.Plugin)
	for name, plugin := range pm.plugins {
		result[name] = plugin
	}
	
	return result
}

// StartAll 启动所有插件
func (pm *PluginManagerImpl) StartAll(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if pm.started {
		return fmt.Errorf("插件管理器已经启动")
	}
	
	pm.ctx, pm.cancel = context.WithCancel(ctx)
	pm.started = true
	
	var errors []error
	
	// 启动所有插件
	for name, plugin := range pm.plugins {
		if err := plugin.Start(pm.ctx); err != nil {
			errors = append(errors, fmt.Errorf("启动插件 %s 失败: %w", name, err))
		}
	}
	
	if len(errors) > 0 {
		// 如果有插件启动失败，返回错误信息
		errorMsg := "部分插件启动失败:\n"
		for _, err := range errors {
			errorMsg += fmt.Sprintf("  - %s\n", err.Error())
		}
		return fmt.Errorf("%s", errorMsg)
	}
	
	return nil
}

// StopAll 停止所有插件
func (pm *PluginManagerImpl) StopAll() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if !pm.started {
		return nil
	}
	
	var errors []error
	
	// 停止所有插件
	for name, plugin := range pm.plugins {
		if plugin.Status() == interfaces.PluginStatusRunning {
			if err := plugin.Stop(); err != nil {
				errors = append(errors, fmt.Errorf("停止插件 %s 失败: %w", name, err))
			}
		}
	}
	
	// 取消上下文
	if pm.cancel != nil {
		pm.cancel()
	}
	
	pm.started = false
	
	if len(errors) > 0 {
		// 如果有插件停止失败，返回错误信息
		errorMsg := "部分插件停止失败:\n"
		for _, err := range errors {
			errorMsg += fmt.Sprintf("  - %s\n", err.Error())
		}
		return fmt.Errorf("%s", errorMsg)
	}
	
	return nil
}

// GetPluginStatus 获取插件状态
func (pm *PluginManagerImpl) GetPluginStatus(name string) (interfaces.PluginStatus, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return interfaces.PluginStatusStopped, fmt.Errorf("插件 %s 不存在", name)
	}
	
	return plugin.Status(), nil
}

// HotReloadPlugin 热重载插件
func (pm *PluginManagerImpl) HotReloadPlugin(name string, newPlugin interfaces.Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	oldPlugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("插件 %s 不存在，无法重载", name)
	}
	
	// 停止旧插件
	if oldPlugin.Status() == interfaces.PluginStatusRunning {
		if err := oldPlugin.Stop(); err != nil {
			return fmt.Errorf("停止旧插件 %s 失败: %w", name, err)
		}
	}
	
	// 等待插件完全停止
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			return fmt.Errorf("等待插件 %s 停止超时", name)
		case <-ticker.C:
			if oldPlugin.Status() == interfaces.PluginStatusStopped {
				goto stopped
			}
		}
	}
	
stopped:
	// 替换插件
	pm.plugins[name] = newPlugin
	
	// 如果管理器已启动，则启动新插件
	if pm.started && pm.ctx != nil {
		if err := newPlugin.Start(pm.ctx); err != nil {
			// 启动失败，恢复旧插件
			pm.plugins[name] = oldPlugin
			return fmt.Errorf("启动新插件 %s 失败: %w", name, err)
		}
	}
	
	return nil
}

// GetPluginInfo 获取插件信息
func (pm *PluginManagerImpl) GetPluginInfo(name string) (map[string]interface{}, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, fmt.Errorf("插件 %s 不存在", name)
	}
	
	info := map[string]interface{}{
		"name":        plugin.Name(),
		"version":     plugin.Version(),
		"description": plugin.Description(),
		"status":      plugin.Status().String(),
		"config":      plugin.Config(),
	}
	
	return info, nil
}

// ListPlugins 列出所有插件信息
func (pm *PluginManagerImpl) ListPlugins() []map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	var plugins []map[string]interface{}
	
	for _, plugin := range pm.plugins {
		info := map[string]interface{}{
			"name":        plugin.Name(),
			"version":     plugin.Version(),
			"description": plugin.Description(),
			"status":      plugin.Status().String(),
		}
		plugins = append(plugins, info)
	}
	
	return plugins
}

// RestartPlugin 重启插件
func (pm *PluginManagerImpl) RestartPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	plugin, exists := pm.plugins[name]
	if !exists {
		return fmt.Errorf("插件 %s 不存在", name)
	}
	
	// 停止插件
	if plugin.Status() == interfaces.PluginStatusRunning {
		if err := plugin.Stop(); err != nil {
			return fmt.Errorf("停止插件 %s 失败: %w", name, err)
		}
	}
	
	// 等待插件完全停止
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-timeout:
			return fmt.Errorf("等待插件 %s 停止超时", name)
		case <-ticker.C:
			if plugin.Status() == interfaces.PluginStatusStopped {
				goto stopped
			}
		}
	}
	
stopped:
	// 重新启动插件
	if pm.started && pm.ctx != nil {
		if err := plugin.Start(pm.ctx); err != nil {
			return fmt.Errorf("重启插件 %s 失败: %w", name, err)
		}
	}
	
	return nil
}

// 确保PluginManagerImpl实现了PluginManager接口
var _ interfaces.PluginManager = (*PluginManagerImpl)(nil)