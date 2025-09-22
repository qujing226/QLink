package consensus

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qujing226/QLink/did/network"
)

// ExampleUsage 演示如何使用共识管理器
func ExampleUsage() {
	// 创建网络实例（这里使用模拟实例）
	p2pNetwork := &network.P2PNetwork{}
	
	// 创建管理器配置
	config := &ManagerConfig{
		NodeID:      "node-001",
		Authorities: []string{"node-001", "node-002", "node-003"},
		DefaultConsensus: ConsensusTypeRaft,
		MonitorConfig: &MonitorConfig{
			MonitorInterval:         5 * time.Second,
			MaxLatency:             1 * time.Second,
			MinThroughput:          10.0,
			MaxFailureRate:         0.1,
			FailureDetectionWindow: 30 * time.Second,
			MaxConsecutiveFailures: 3,
			RecoveryTimeout:        60 * time.Second,
			MaxRecoveryAttempts:    3,
			EnableAlerts:           true,
		},
		SwitcherConfig: &SwitcherConfig{
			SwitchStrategy:       SwitchStrategyGraceful,
			SwitchTimeout:        60 * time.Second,
			DataSyncTimeout:      30 * time.Second,
			EnableAutoSwitch:     true,
			AutoSwitchThreshold:  0.8,
			AutoSwitchCooldown:   5 * time.Minute,
			RequireConfirmation:  false,
			BackupBeforeSwitch:   true,
			EnableRollback:       true,
			RollbackTimeout:      30 * time.Second,
			MaxRollbackDepth:     3,
		},
		RaftConfig: &RaftConfig{
			ElectionTimeout:   300 * time.Millisecond,
			HeartbeatInterval: 50 * time.Millisecond,
		},
		PoAConfig: &PoAConfig{
			BlockTime:     5 * time.Second,
			VoteThreshold: 0.67,
		},
	}
	
	// 创建共识管理器
	manager := NewConsensusManager(config, p2pNetwork)
	
	// 初始化管理器
	if err := manager.Initialize(); err != nil {
		log.Fatalf("初始化共识管理器失败: %v", err)
	}
	
	// 启动管理器
	ctx := context.Background()
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("启动共识管理器失败: %v", err)
	}
	
	// 等待一段时间让系统稳定
	time.Sleep(2 * time.Second)
	
	// 提交一些提案
	for i := 0; i < 5; i++ {
		proposal := map[string]interface{}{
			"id":   fmt.Sprintf("proposal-%d", i),
			"data": fmt.Sprintf("test data %d", i),
		}
		
		if err := manager.Submit(proposal); err != nil {
			log.Printf("提交提案失败: %v", err)
		} else {
			log.Printf("提交提案成功: %s", proposal["id"])
		}
		
		time.Sleep(1 * time.Second)
	}
	
	// 获取状态信息
	status := manager.GetStatus()
	log.Printf("管理器状态: %+v", status)
	
	// 获取监控指标
	metrics := manager.GetMetrics()
	if metrics != nil {
		log.Printf("监控指标: 总错误数=%d, 成功率=%.2f", 
			metrics.TotalErrors, metrics.SuccessRate)
	}
	
	// 演示共识算法切换
	log.Printf("当前共识算法: %s", getConsensusTypeName(manager.GetCurrentConsensusType()))
	
	// 切换到PoA
	if manager.GetCurrentConsensusType() == ConsensusTypeRaft {
		log.Printf("切换到PoA算法...")
		if err := manager.SwitchConsensus(ConsensusTypePoA); err != nil {
			log.Printf("切换失败: %v", err)
		} else {
			log.Printf("切换成功，当前算法: %s", 
				getConsensusTypeName(manager.GetCurrentConsensusType()))
		}
	}
	
	// 等待一段时间观察切换后的行为
	time.Sleep(5 * time.Second)
	
	// 再次获取状态
	status = manager.GetStatus()
	log.Printf("切换后状态: %+v", status)
	
	// 停止管理器
	if err := manager.Stop(); err != nil {
		log.Printf("停止管理器失败: %v", err)
	} else {
		log.Printf("管理器已停止")
	}
}

// getConsensusTypeName 获取共识算法类型名称
func getConsensusTypeName(consensusType ConsensusType) string {
	switch consensusType {
	case ConsensusTypeRaft:
		return "Raft"
	case ConsensusTypePoA:
		return "PoA"
	case ConsensusTypePBFT:
		return "PBFT"
	case ConsensusTypePoS:
		return "PoS"
	default:
		return "Unknown"
	}
}

// ExampleMonitoring 演示监控功能
func ExampleMonitoring() {
	// 创建监控器
	monitor := NewConsensusMonitor(nil) // 使用默认配置
	
	// 设置回调函数
	monitor.SetFailureCallback(func(failure *FailureEvent) {
		log.Printf("检测到故障: %s - %s (严重程度: %d)", 
			failure.ID, failure.Description, failure.Severity)
	})
	
	monitor.SetRecoveryCallback(func(recovery *RecoveryEvent) {
		log.Printf("开始恢复: %s (策略: %d)", recovery.ID, recovery.Strategy)
	})
	
	// 启动监控
	ctx := context.Background()
	if err := monitor.Start(ctx); err != nil {
		log.Fatalf("启动监控器失败: %v", err)
	}
	
	// 记录操作（通过监控器收集指标）
	// 注意：实际的操作记录会在监控循环中自动收集
	log.Printf("操作已记录到监控系统")
	log.Printf("模拟操作执行中...")
	for i := 0; i < 10; i++ {
		// 模拟成功操作
		log.Printf("执行操作 %d", i+1)
		time.Sleep(500 * time.Millisecond)
	}
	
	// 模拟一些失败场景
	for i := 0; i < 3; i++ {
		log.Printf("模拟失败场景 %d", i+1)
		time.Sleep(200 * time.Millisecond)
	}
	
	// 获取指标
	metrics := monitor.GetMetrics()
	log.Printf("监控指标: %+v", metrics)
	
	// 获取故障历史
	failures := monitor.GetFailureHistory()
	log.Printf("故障历史: %d 个故障", len(failures))
	
	// 停止监控
	monitor.Stop()
	log.Printf("监控器已停止")
}

// ExampleSwitching 演示切换功能
func ExampleSwitching() {
	// 创建模拟的共识算法实例
	p2pNetwork := &network.P2PNetwork{}
	raftNode := NewRaftNode("node-001", p2pNetwork)
	poaNode := NewPoANode("node-001", []string{"node-001", "node-002"}, p2pNetwork)
	
	// 创建切换器
	switcher := NewConsensusSwitcher(nil) // 使用默认配置
	
	// 创建监控器
	monitor := NewConsensusMonitor(nil)
	
	// 初始化切换器
	if err := switcher.Initialize(raftNode, poaNode, monitor); err != nil {
		log.Fatalf("初始化切换器失败: %v", err)
	}
	
	// 设置回调函数
	switcher.SetSwitchStartedCallback(func(from, to ConsensusType) {
		log.Printf("开始切换: %s -> %s", 
			getConsensusTypeName(from), getConsensusTypeName(to))
	})
	
	switcher.SetSwitchCompletedCallback(func(from, to ConsensusType, success bool) {
		if success {
			log.Printf("切换成功: %s -> %s", 
				getConsensusTypeName(from), getConsensusTypeName(to))
		} else {
			log.Printf("切换失败: %s -> %s", 
				getConsensusTypeName(from), getConsensusTypeName(to))
		}
	})
	
	// 获取当前状态
	log.Printf("当前算法: %s", getConsensusTypeName(switcher.GetCurrentType()))
	log.Printf("支持的算法: %v", switcher.GetSupportedTypes())
	
	// 执行切换
	if err := switcher.SwitchTo(ConsensusTypePoA); err != nil {
		log.Printf("切换失败: %v", err)
	}
	
	// 等待切换完成
	time.Sleep(2 * time.Second)
	
	// 获取切换状态
	switchState := switcher.GetSwitchState()
	if switchState != nil {
		log.Printf("切换状态: %+v", switchState)
	}
	
	// 获取状态
	status := switcher.GetStatus()
	log.Printf("切换器状态: %+v", status)
	
	log.Printf("切换演示完成")
}

// ExampleIntegration 演示完整集成
func ExampleIntegration() {
	log.Printf("=== 共识算法集成演示 ===")
	
	log.Printf("\n1. 基本使用演示:")
	ExampleUsage()
	
	log.Printf("\n2. 监控功能演示:")
	ExampleMonitoring()
	
	log.Printf("\n3. 切换功能演示:")
	ExampleSwitching()
	
	log.Printf("\n=== 演示完成 ===")
}