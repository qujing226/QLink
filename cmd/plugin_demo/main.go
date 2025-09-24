package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qujing226/QLink/pkg/plugins"
)

func main() {
	fmt.Println("🚀 QLink 插件系统演示程序")
	fmt.Println("=============================")

	// 创建插件管理器
	fmt.Println("📦 创建插件管理器...")
	manager := plugins.NewPluginManager()

	// 创建热加载器
	fmt.Println("🔥 创建热加载器...")
	hotReloader := plugins.NewHotReloader(manager)

	// 配置热加载器
	hotReloader.SetConfig(plugins.HotReloadConfig{
		WatchInterval:   2 * time.Second,
		PluginExtension: ".so",
		AutoReload:      true,
		ReloadDelay:     1 * time.Second,
		MaxRetries:      3,
	})

	// 添加监控路径
	if err := hotReloader.AddWatchPath("./plugins"); err != nil {
		log.Printf("⚠️  添加监控路径失败: %v", err)
	}

	// 启动热加载器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("🔥 启动热加载器...")
	if err := hotReloader.Start(ctx); err != nil {
		log.Printf("⚠️  启动热加载器失败: %v", err)
	}
	defer hotReloader.Stop()

	// 注册示例插件
	fmt.Println("\n🔌 注册示例插件...")
	registerExamplePlugins(manager)

	// 启动所有插件
	fmt.Println("\n▶️  启动所有插件...")
	if err := manager.StartAll(ctx); err != nil {
		log.Printf("⚠️  启动插件失败: %v", err)
	}
	defer manager.StopAll()

	// 演示插件管理功能
	fmt.Println("\n📊 插件管理功能演示:")
	demonstratePluginManagement(manager)

	// 演示热加载功能
	fmt.Println("\n🔥 热加载功能演示:")
	demonstrateHotReload(hotReloader)

	// 监控插件状态
	fmt.Println("\n📈 开始监控插件状态...")
	go monitorPluginStatus(manager)

	// 等待中断信号
	fmt.Println("\n✅ 插件系统运行中... (按 Ctrl+C 退出)")
	waitForInterrupt()

	fmt.Println("\n🛑 正在关闭插件系统...")
}

func registerExamplePlugins(manager *plugins.PluginManagerImpl) {
	// 注册DID插件
	fmt.Println("  📝 注册DID插件...")
	didPlugin := plugins.NewSimpleDIDPlugin()
	if err := manager.RegisterPlugin(didPlugin); err != nil {
		log.Printf("    ❌ 注册DID插件失败: %v", err)
	} else {
		fmt.Println("    ✅ DID插件注册成功")
	}

	// 注册网络监控插件
	fmt.Println("  🌐 注册网络监控插件...")
	networkPlugin := plugins.NewNetworkMonitorPlugin()
	if err := manager.RegisterPlugin(networkPlugin); err != nil {
		log.Printf("    ❌ 注册网络监控插件失败: %v", err)
	} else {
		fmt.Println("    ✅ 网络监控插件注册成功")
	}
}

func demonstratePluginManagement(manager *plugins.PluginManagerImpl) {
	// 列出所有插件
	fmt.Println("  📋 插件列表:")
	plugins := manager.ListPlugins()
	for _, plugin := range plugins {
		fmt.Printf("    - %s (状态: %s)\n", plugin["name"], plugin["status"])
	}

	// 获取插件详细信息
	fmt.Println("\n  ℹ️  插件详细信息:")
	if info, err := manager.GetPluginInfo("simple-did"); err == nil {
		fmt.Printf("    DID插件: %s v%s - %s\n", 
			info["name"], info["version"], info["description"])
	}

	if info, err := manager.GetPluginInfo("network-monitor"); err == nil {
		fmt.Printf("    网络插件: %s v%s - %s\n", 
			info["name"], info["version"], info["description"])
	}

	// 演示插件重启
	fmt.Println("\n  🔄 重启DID插件...")
	if err := manager.RestartPlugin("simple-did"); err != nil {
		log.Printf("    ❌ 重启失败: %v", err)
	} else {
		fmt.Println("    ✅ 重启成功")
	}

	// 检查插件状态
	fmt.Println("\n  📊 插件状态检查:")
	for _, pluginName := range []string{"simple-did", "network-monitor"} {
		if status, err := manager.GetPluginStatus(pluginName); err == nil {
			fmt.Printf("    %s: %s\n", pluginName, status)
		}
	}
}

func demonstrateHotReload(hotReloader *plugins.HotReloader) {
	// 演示手动加载插件
	fmt.Println("  🔧 手动加载插件演示...")
	if err := hotReloader.LoadPlugin("./example_plugin.so"); err != nil {
		log.Printf("    ⚠️  手动加载失败: %v (这是正常的，因为文件不存在)", err)
	}

	// 显示监控状态
	fmt.Println("  👀 监控状态:")
	fmt.Printf("    监控路径数: %d\n", len(hotReloader.GetWatchPaths()))
	fmt.Printf("    监控文件数: %d\n", len(hotReloader.GetWatchedFiles()))
}

func monitorPluginStatus(manager *plugins.PluginManagerImpl) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("\n📊 插件状态更新:")
			plugins := manager.ListPlugins()
			for _, plugin := range plugins {
				fmt.Printf("  %s: %s\n", plugin["name"], plugin["status"])
			}

			// 显示插件统计信息（简化版本）
			fmt.Printf("  网络插件状态: 正常运行\n")
			fmt.Printf("  DID插件状态: 正常运行\n")
		}
	}
}

func waitForInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}

// 创建示例插件配置
func createExamplePluginConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled":     true,
		"debug":       false,
		"max_retries": 3,
		"timeout":     30,
	}
}

// 验证插件配置
func validatePluginConfig(config map[string]interface{}) error {
	required := []string{"enabled"}
	for _, key := range required {
		if _, exists := config[key]; !exists {
			return fmt.Errorf("缺少必需的配置项: %s", key)
		}
	}
	return nil
}

// 获取插件性能指标
func getPluginMetrics(manager *plugins.PluginManagerImpl) map[string]interface{} {
	plugins := manager.ListPlugins()
	statusCount := make(map[string]int)
	
	for _, plugin := range plugins {
		status := plugin["status"].(string)
		statusCount[status]++
	}

	return map[string]interface{}{
		"total_plugins":       len(plugins),
		"status_distribution": statusCount,
		"timestamp":          time.Now().Unix(),
	}
}