package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
)

// HotReloader 插件热加载器
type HotReloader struct {
	mu           sync.RWMutex
	manager      *PluginManagerImpl
	watchPaths   []string
	watchers     map[string]*FileWatcher
	ctx          context.Context
	cancel       context.CancelFunc
	started      bool
	reloadConfig HotReloadConfig
}

// HotReloadConfig 热加载配置
type HotReloadConfig struct {
	// 监控间隔
	WatchInterval time.Duration
	// 插件文件扩展名
	PluginExtension string
	// 是否自动重载
	AutoReload bool
	// 重载延迟（防止文件正在写入时重载）
	ReloadDelay time.Duration
	// 最大重试次数
	MaxRetries int
}

// FileWatcher 文件监控器
type FileWatcher struct {
	path     string
	lastMod  time.Time
	size     int64
	checksum string
}

// PluginFactory 插件工厂函数类型
type PluginFactory func() interfaces.Plugin

// NewHotReloader 创建新的热加载器
func NewHotReloader(manager *PluginManagerImpl) *HotReloader {
	return &HotReloader{
		manager:    manager,
		watchPaths: []string{},
		watchers:   make(map[string]*FileWatcher),
		reloadConfig: HotReloadConfig{
			WatchInterval:   2 * time.Second,
			PluginExtension: ".so",
			AutoReload:      true,
			ReloadDelay:     1 * time.Second,
			MaxRetries:      3,
		},
	}
}

// SetConfig 设置热加载配置
func (hr *HotReloader) SetConfig(config HotReloadConfig) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	hr.reloadConfig = config
}

// AddWatchPath 添加监控路径
func (hr *HotReloader) AddWatchPath(path string) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	
	// 检查路径是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("监控路径不存在: %s", path)
	}
	
	// 检查是否已经在监控
	for _, watchPath := range hr.watchPaths {
		if watchPath == path {
			return nil // 已经在监控
		}
	}
	
	hr.watchPaths = append(hr.watchPaths, path)
	
	// 如果热加载器已启动，立即开始监控这个路径
	if hr.started {
		go hr.watchPath(path)
	}
	
	return nil
}

// RemoveWatchPath 移除监控路径
func (hr *HotReloader) RemoveWatchPath(path string) {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	
	// 从监控路径列表中移除
	for i, watchPath := range hr.watchPaths {
		if watchPath == path {
			hr.watchPaths = append(hr.watchPaths[:i], hr.watchPaths[i+1:]...)
			break
		}
	}
	
	// 移除文件监控器
	delete(hr.watchers, path)
}

// Start 启动热加载器
func (hr *HotReloader) Start(ctx context.Context) error {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	
	if hr.started {
		return fmt.Errorf("热加载器已经启动")
	}
	
	hr.ctx, hr.cancel = context.WithCancel(ctx)
	hr.started = true
	
	// 为每个监控路径启动监控协程
	for _, path := range hr.watchPaths {
		go hr.watchPath(path)
	}
	
	return nil
}

// Stop 停止热加载器
func (hr *HotReloader) Stop() error {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	
	if !hr.started {
		return nil
	}
	
	if hr.cancel != nil {
		hr.cancel()
	}
	
	hr.started = false
	hr.watchers = make(map[string]*FileWatcher)
	
	return nil
}

// LoadPlugin 加载插件
func (hr *HotReloader) LoadPlugin(pluginPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("插件文件不存在: %s", pluginPath)
	}
	
	// 加载动态库
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("加载插件失败: %w", err)
	}
	
	// 查找插件工厂函数
	factorySymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("插件中未找到NewPlugin函数: %w", err)
	}
	
	// 类型断言
	factory, ok := factorySymbol.(func() interfaces.Plugin)
	if !ok {
		return fmt.Errorf("NewPlugin函数签名不正确")
	}
	
	// 创建插件实例
	pluginInstance := factory()
	if pluginInstance == nil {
		return fmt.Errorf("插件工厂返回nil")
	}
	
	// 注册插件
	return hr.manager.RegisterPlugin(pluginInstance)
}

// ReloadPlugin 重载插件
func (hr *HotReloader) ReloadPlugin(pluginPath string) error {
	// 获取插件名称（从文件名推断）
	pluginName := filepath.Base(pluginPath)
	pluginName = pluginName[:len(pluginName)-len(filepath.Ext(pluginName))]
	
	// 检查插件是否存在
	if _, err := hr.manager.GetPlugin(pluginName); err != nil {
		// 插件不存在，直接加载
		return hr.LoadPlugin(pluginPath)
	}
	
	// 加载新插件
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("重载插件失败: %w", err)
	}
	
	// 查找插件工厂函数
	factorySymbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("插件中未找到NewPlugin函数: %w", err)
	}
	
	// 类型断言
	factory, ok := factorySymbol.(func() interfaces.Plugin)
	if !ok {
		return fmt.Errorf("NewPlugin函数签名不正确")
	}
	
	// 创建新插件实例
	newPluginInstance := factory()
	if newPluginInstance == nil {
		return fmt.Errorf("插件工厂返回nil")
	}
	
	// 热重载插件
	return hr.manager.HotReloadPlugin(pluginName, newPluginInstance)
}

// watchPath 监控指定路径
func (hr *HotReloader) watchPath(path string) {
	ticker := time.NewTicker(hr.reloadConfig.WatchInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-hr.ctx.Done():
			return
		case <-ticker.C:
			hr.checkPathForChanges(path)
		}
	}
}

// checkPathForChanges 检查路径中的文件变化
func (hr *HotReloader) checkPathForChanges(path string) {
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 只处理插件文件
		if !info.IsDir() && filepath.Ext(filePath) == hr.reloadConfig.PluginExtension {
			hr.checkFileForChanges(filePath, info)
		}
		
		return nil
	})
	
	if err != nil {
		// 记录错误但不中断监控
		fmt.Printf("监控路径 %s 时发生错误: %v\n", path, err)
	}
}

// checkFileForChanges 检查单个文件的变化
func (hr *HotReloader) checkFileForChanges(filePath string, info os.FileInfo) {
	hr.mu.Lock()
	watcher, exists := hr.watchers[filePath]
	hr.mu.Unlock()
	
	if !exists {
		// 新文件，创建监控器
		watcher = &FileWatcher{
			path:    filePath,
			lastMod: info.ModTime(),
			size:    info.Size(),
		}
		
		hr.mu.Lock()
		hr.watchers[filePath] = watcher
		hr.mu.Unlock()
		
		// 如果启用自动重载，加载新插件
		if hr.reloadConfig.AutoReload {
			go hr.delayedReload(filePath)
		}
		
		return
	}
	
	// 检查文件是否有变化
	if info.ModTime().After(watcher.lastMod) || info.Size() != watcher.size {
		// 文件已更改
		watcher.lastMod = info.ModTime()
		watcher.size = info.Size()
		
		// 如果启用自动重载，重载插件
		if hr.reloadConfig.AutoReload {
			go hr.delayedReload(filePath)
		}
	}
}

// delayedReload 延迟重载插件（防止文件正在写入）
func (hr *HotReloader) delayedReload(pluginPath string) {
	// 等待一段时间确保文件写入完成
	time.Sleep(hr.reloadConfig.ReloadDelay)
	
	// 重试机制
	var lastErr error
	for i := 0; i < hr.reloadConfig.MaxRetries; i++ {
		if err := hr.ReloadPlugin(pluginPath); err != nil {
			lastErr = err
			time.Sleep(time.Duration(i+1) * time.Second) // 递增延迟
			continue
		}
		
		fmt.Printf("成功重载插件: %s\n", pluginPath)
		return
	}
	
	fmt.Printf("重载插件失败 %s (重试%d次): %v\n", pluginPath, hr.reloadConfig.MaxRetries, lastErr)
}

// GetWatchedFiles 获取正在监控的文件列表
func (hr *HotReloader) GetWatchedFiles() []string {
	hr.mu.RLock()
	defer hr.mu.RUnlock()
	
	var files []string
	for filePath := range hr.watchers {
		files = append(files, filePath)
	}
	
	return files
}

// GetWatchPaths 获取监控路径列表
func (hr *HotReloader) GetWatchPaths() []string {
	hr.mu.RLock()
	defer hr.mu.RUnlock()
	
	paths := make([]string, len(hr.watchPaths))
	copy(paths, hr.watchPaths)
	return paths
}