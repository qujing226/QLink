package plugins

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ExampleUsage 展示插件系统的使用方法
func ExampleUsage() {
	// 创建插件管理器
	manager := NewPluginManager()
	
	// 创建热加载器
	hotReloader := NewHotReloader(manager)
	
	// 配置热加载器
	hotReloader.SetConfig(HotReloadConfig{
		WatchInterval:   2 * time.Second,
		PluginExtension: ".so",
		AutoReload:      true,
		ReloadDelay:     1 * time.Second,
		MaxRetries:      3,
	})
	
	// 添加监控路径
	if err := hotReloader.AddWatchPath("./plugins"); err != nil {
		log.Printf("添加监控路径失败: %v", err)
	}
	
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 启动热加载器
	if err := hotReloader.Start(ctx); err != nil {
		log.Printf("启动热加载器失败: %v", err)
		return
	}
	defer hotReloader.Stop()
	
	// 注册示例插件
	registerExamplePlugins(manager)
	
	// 启动所有插件
	if err := manager.StartAll(ctx); err != nil {
		log.Printf("启动插件失败: %v", err)
	}
	
	// 展示插件管理功能
	demonstratePluginManagement(manager)
	
	// 展示热加载功能
	demonstrateHotReload(hotReloader)
	
	// 停止所有插件
	if err := manager.StopAll(); err != nil {
		log.Printf("停止插件失败: %v", err)
	}
}

// registerExamplePlugins 注册示例插件
func registerExamplePlugins(manager *PluginManagerImpl) {
	// 注册DID插件
	didPlugin := NewSimpleDIDPlugin()
	if err := manager.RegisterPlugin(didPlugin); err != nil {
		log.Printf("注册DID插件失败: %v", err)
	} else {
		fmt.Println("✓ DID插件注册成功")
	}
	
	// 注册网络监控插件
	networkPlugin := NewNetworkMonitorPlugin()
	if err := manager.RegisterPlugin(networkPlugin); err != nil {
		log.Printf("注册网络监控插件失败: %v", err)
	} else {
		fmt.Println("✓ 网络监控插件注册成功")
	}
}

// demonstratePluginManagement 展示插件管理功能
func demonstratePluginManagement(manager *PluginManagerImpl) {
	fmt.Println("\n=== 插件管理功能演示 ===")
	
	// 列出所有插件
	plugins := manager.ListPlugins()
	fmt.Printf("已注册插件数量: %d\n", len(plugins))
	
	for _, plugin := range plugins {
		fmt.Printf("- %s (v%s): %s [状态: %s]\n",
			plugin["name"],
			plugin["version"],
			plugin["description"],
			plugin["status"])
	}
	
	// 获取特定插件信息
	if info, err := manager.GetPluginInfo("simple-did"); err == nil {
		fmt.Printf("\nDID插件详细信息:\n")
		fmt.Printf("  名称: %s\n", info["name"])
		fmt.Printf("  版本: %s\n", info["version"])
		fmt.Printf("  描述: %s\n", info["description"])
		fmt.Printf("  状态: %s\n", info["status"])
	}
	
	// 重启插件
	fmt.Println("\n重启网络监控插件...")
	if err := manager.RestartPlugin("network-monitor"); err != nil {
		log.Printf("重启插件失败: %v", err)
	} else {
		fmt.Println("✓ 插件重启成功")
	}
	
	// 检查插件状态
	if status, err := manager.GetPluginStatus("network-monitor"); err == nil {
		fmt.Printf("网络监控插件状态: %s\n", status.String())
	}
}

// demonstrateHotReload 展示热加载功能
func demonstrateHotReload(hotReloader *HotReloader) {
	fmt.Println("\n=== 热加载功能演示 ===")
	
	// 显示监控路径
	watchPaths := hotReloader.GetWatchPaths()
	fmt.Printf("监控路径: %v\n", watchPaths)
	
	// 显示正在监控的文件
	watchedFiles := hotReloader.GetWatchedFiles()
	if len(watchedFiles) > 0 {
		fmt.Printf("正在监控的插件文件:\n")
		for _, file := range watchedFiles {
			fmt.Printf("  - %s\n", file)
		}
	} else {
		fmt.Println("当前没有监控任何插件文件")
	}
	
	// 模拟手动加载插件
	fmt.Println("\n尝试手动加载插件...")
	if err := hotReloader.LoadPlugin("./plugins/example.so"); err != nil {
		fmt.Printf("手动加载插件失败 (这是正常的，因为文件可能不存在): %v\n", err)
	}
}

// CreateExamplePluginConfig 创建示例插件配置
func CreateExamplePluginConfig() map[string]interface{} {
	return map[string]interface{}{
		"did_plugin": map[string]interface{}{
			"registry_type": "memory",
			"cache_size":    1000,
			"ttl_seconds":   3600,
		},
		"network_plugin": map[string]interface{}{
			"monitor_interval": "30s",
			"targets": []string{
				"8.8.8.8:53",
				"1.1.1.1:53",
			},
		},
		"hot_reload": map[string]interface{}{
			"watch_interval":   "2s",
			"plugin_extension": ".so",
			"auto_reload":      true,
			"reload_delay":     "1s",
			"max_retries":      3,
		},
	}
}

// ValidatePluginConfig 验证插件配置
func ValidatePluginConfig(config map[string]interface{}) error {
	// 验证DID插件配置
	if didConfig, ok := config["did_plugin"].(map[string]interface{}); ok {
		if registryType, exists := didConfig["registry_type"]; exists {
			if registryType != "memory" && registryType != "file" && registryType != "database" {
				return fmt.Errorf("无效的registry_type: %v", registryType)
			}
		}
	}
	
	// 验证网络插件配置
	if networkConfig, ok := config["network_plugin"].(map[string]interface{}); ok {
		if targets, exists := networkConfig["targets"]; exists {
			if targetList, ok := targets.([]string); ok {
				if len(targetList) == 0 {
					return fmt.Errorf("网络插件目标列表不能为空")
				}
			}
		}
	}
	
	return nil
}

// GetPluginMetrics 获取插件性能指标
func GetPluginMetrics(manager *PluginManagerImpl) map[string]interface{} {
	metrics := make(map[string]interface{})
	
	plugins := manager.GetAllPlugins()
	metrics["total_plugins"] = len(plugins)
	
	statusCount := make(map[string]int)
	for _, plugin := range plugins {
		status := plugin.Status().String()
		statusCount[status]++
	}
	metrics["status_distribution"] = statusCount
	
	// 获取网络插件统计信息
	if networkPlugin, err := manager.GetPlugin("network-monitor"); err == nil {
		if np, ok := networkPlugin.(*NetworkMonitorPlugin); ok {
			stats := np.GetNetworkStats()
			metrics["network_stats"] = map[string]interface{}{
				"connected_peers":   stats.ConnectedPeers,
				"messages_sent":     stats.MessagesSent,
				"messages_received": stats.MessagesReceived,
				"bytes_sent":        stats.BytesSent,
				"bytes_received":    stats.BytesReceived,
				"uptime":            stats.Uptime,
			}
		}
	}
	
	return metrics
}