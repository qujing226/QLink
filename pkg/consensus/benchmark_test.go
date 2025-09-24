package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
	"github.com/qujing226/QLink/pkg/network"
)

// BenchmarkRaftAdapter 基准测试Raft适配器
func BenchmarkRaftAdapter(b *testing.B) {
	b.Run("Start", benchmarkRaftStart)
	b.Run("Submit", benchmarkRaftSubmit)
	b.Run("GetStatus", benchmarkRaftGetStatus)
	b.Run("GetMetrics", benchmarkRaftGetMetrics)
	b.Run("ValidateBlock", benchmarkRaftValidateBlock)
}

// benchmarkRaftStart 基准测试Raft启动
func benchmarkRaftStart(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
		ctx := context.Background()
		
		// 启动
		adapter.Start(ctx)
		
		// 停止
		adapter.Stop()
	}
}

// benchmarkRaftSubmit 基准测试Raft提案提交
func benchmarkRaftSubmit(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	proposal := map[string]interface{}{
		"type": "benchmark_test",
		"data": "test_data",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Submit(proposal)
	}
}

// benchmarkRaftGetStatus 基准测试Raft状态获取
func benchmarkRaftGetStatus(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.GetStatus()
	}
}

// benchmarkRaftGetMetrics 基准测试Raft指标获取
func benchmarkRaftGetMetrics(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.GetMetrics()
	}
}

// benchmarkRaftValidateBlock 基准测试Raft区块验证
func benchmarkRaftValidateBlock(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	block := map[string]interface{}{
		"height": 1,
		"hash":   "test_hash",
		"data":   "test_data",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.ValidateBlock(block)
	}
}

// BenchmarkPoAAdapter 基准测试PoA适配器
func BenchmarkPoAAdapter(b *testing.B) {
	b.Run("Start", benchmarkPoAStart)
	b.Run("Submit", benchmarkPoASubmit)
	b.Run("GetStatus", benchmarkPoAGetStatus)
	b.Run("ValidateProposer", benchmarkPoAValidateProposer)
	b.Run("GetNextProposer", benchmarkPoAGetNextProposer)
}

// benchmarkPoAStart 基准测试PoA启动
func benchmarkPoAStart(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
		ctx := context.Background()
		
		// 启动
		adapter.Start(ctx)
		
		// 停止
		adapter.Stop()
	}
}

// benchmarkPoASubmit 基准测试PoA提案提交
func benchmarkPoASubmit(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	proposal := map[string]interface{}{
		"type": "benchmark_test",
		"data": "test_data",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.Submit(proposal)
	}
}

// benchmarkPoAGetStatus 基准测试PoA状态获取
func benchmarkPoAGetStatus(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.GetStatus()
	}
}

// benchmarkPoAValidateProposer 基准测试PoA提案者验证
func benchmarkPoAValidateProposer(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.ValidateProposer("node1", uint64(i))
	}
}

// benchmarkPoAGetNextProposer 基准测试PoA下一个提案者获取
func benchmarkPoAGetNextProposer(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.GetNextProposer(uint64(i))
	}
}

// BenchmarkConsensusSwitcher 基准测试共识切换器
func BenchmarkConsensusSwitcher(b *testing.B) {
	b.Run("SwitchTo", benchmarkConsensusSwitchTo)
	b.Run("GetCurrentType", benchmarkConsensusGetCurrentType)
	b.Run("GetStatus", benchmarkConsensusSwitcherGetStatus)
}

// benchmarkConsensusSwitchTo 基准测试共识切换
func benchmarkConsensusSwitchTo(b *testing.B) {
	config := &SwitcherAdapterConfig{
		SwitchStrategy:      SwitchStrategyImmediate,
		SwitchTimeout:       5 * time.Second,
		DataSyncTimeout:     2 * time.Second,
		EnableAutoSwitch:    false,
		RequireConfirmation: false,
		BackupBeforeSwitch:  false,
		EnableRollback:      false,
	}
	
	switcher := NewConsensusSwitcherAdapter(config)
	p2pNetwork := &network.P2PNetwork{}
	
	err := switcher.Initialize("node1", []string{"node2", "node3"}, []string{"node1", "node2", "node3"}, p2pNetwork, nil)
	if err != nil {
		b.Fatalf("Failed to initialize switcher: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 在Raft和PoA之间切换
		if i%2 == 0 {
			switcher.SwitchTo(interfaces.ConsensusTypePoA)
		} else {
			switcher.SwitchTo(interfaces.ConsensusTypeRaft)
		}
	}
}

// benchmarkConsensusGetCurrentType 基准测试获取当前共识类型
func benchmarkConsensusGetCurrentType(b *testing.B) {
	config := &SwitcherAdapterConfig{
		SwitchStrategy:      SwitchStrategyImmediate,
		SwitchTimeout:       5 * time.Second,
		DataSyncTimeout:     2 * time.Second,
		EnableAutoSwitch:    false,
		RequireConfirmation: false,
		BackupBeforeSwitch:  false,
		EnableRollback:      false,
	}
	
	switcher := NewConsensusSwitcherAdapter(config)
	p2pNetwork := &network.P2PNetwork{}
	
	err := switcher.Initialize("node1", []string{"node2", "node3"}, []string{"node1", "node2", "node3"}, p2pNetwork, nil)
	if err != nil {
		b.Fatalf("Failed to initialize switcher: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switcher.GetCurrentType()
	}
}

// benchmarkConsensusSwitcherGetStatus 基准测试切换器状态获取
func benchmarkConsensusSwitcherGetStatus(b *testing.B) {
	config := &SwitcherAdapterConfig{
		SwitchStrategy:      SwitchStrategyImmediate,
		SwitchTimeout:       5 * time.Second,
		DataSyncTimeout:     2 * time.Second,
		EnableAutoSwitch:    false,
		RequireConfirmation: false,
		BackupBeforeSwitch:  false,
		EnableRollback:      false,
	}
	
	switcher := NewConsensusSwitcherAdapter(config)
	p2pNetwork := &network.P2PNetwork{}
	
	err := switcher.Initialize("node1", []string{"node2", "node3"}, []string{"node1", "node2", "node3"}, p2pNetwork, nil)
	if err != nil {
		b.Fatalf("Failed to initialize switcher: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switcher.GetStatus()
	}
}

// BenchmarkMetricsCollector 基准测试指标收集器
func BenchmarkMetricsCollector(b *testing.B) {
	b.Run("RegisterConsensus", benchmarkMetricsRegister)
	b.Run("GetMetrics", benchmarkMetricsGet)
	b.Run("GetSummary", benchmarkMetricsGetSummary)
	b.Run("UpdateProposal", benchmarkMetricsUpdateProposal)
}

// benchmarkMetricsRegister 基准测试指标注册
func benchmarkMetricsRegister(b *testing.B) {
	collector := NewMetricsCollector()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := NewConsensusMetrics()
		collector.RegisterConsensus("test_consensus", metrics)
		collector.UnregisterConsensus("test_consensus")
	}
}

// benchmarkMetricsGet 基准测试指标获取
func benchmarkMetricsGet(b *testing.B) {
	collector := NewMetricsCollector()
	metrics := NewConsensusMetrics()
	collector.RegisterConsensus("test_consensus", metrics)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetMetrics("test_consensus")
	}
}

// benchmarkMetricsGetSummary 基准测试指标摘要获取
func benchmarkMetricsGetSummary(b *testing.B) {
	collector := NewMetricsCollector()
	
	// 注册多个指标
	for i := 0; i < 10; i++ {
		metrics := NewConsensusMetrics()
		collector.RegisterConsensus("test_consensus_"+string(rune('0'+i)), metrics)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.GetSummary()
	}
}

// benchmarkMetricsUpdateProposal 基准测试指标更新
func benchmarkMetricsUpdateProposal(b *testing.B) {
	metrics := NewConsensusMetrics()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.UpdateProposal(true, time.Millisecond)
	}
}

// BenchmarkConcurrentOperations 基准测试并发操作
func BenchmarkConcurrentOperations(b *testing.B) {
	b.Run("ConcurrentRaftOperations", benchmarkConcurrentRaftOperations)
	b.Run("ConcurrentPoAOperations", benchmarkConcurrentPoAOperations)
	b.Run("ConcurrentMetricsOperations", benchmarkConcurrentMetricsOperations)
}

// benchmarkConcurrentRaftOperations 基准测试并发Raft操作
func benchmarkConcurrentRaftOperations(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 并发执行多种操作
			go adapter.GetStatus()
			go adapter.GetMetrics()
			go adapter.GetNodes()
			go adapter.Submit(map[string]interface{}{
				"type": "concurrent_test",
				"data": "test_data",
			})
		}
	})
}

// benchmarkConcurrentPoAOperations 基准测试并发PoA操作
func benchmarkConcurrentPoAOperations(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	ctx := context.Background()
	adapter.Start(ctx)
	defer adapter.Stop()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 并发执行多种操作
			go adapter.GetStatus()
			go adapter.ValidateProposer("node1", 1)
			go adapter.GetNextProposer(1)
			go adapter.GetAuthorities()
		}
	})
}

// benchmarkConcurrentMetricsOperations 基准测试并发指标操作
func benchmarkConcurrentMetricsOperations(b *testing.B) {
	collector := NewMetricsCollector()
	
	// 注册多个指标
	for i := 0; i < 10; i++ {
		metrics := NewConsensusMetrics()
		collector.RegisterConsensus("test_consensus_"+string(rune('0'+i)), metrics)
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 并发执行多种操作
			go collector.GetSummary()
			go collector.GetAllMetrics()
			go collector.GetMetrics("test_consensus_0")
		}
	})
}

// BenchmarkMemoryUsage 基准测试内存使用
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("RaftAdapterMemory", benchmarkRaftAdapterMemory)
	b.Run("PoAAdapterMemory", benchmarkPoAAdapterMemory)
	b.Run("MetricsMemory", benchmarkMetricsMemory)
}

// benchmarkRaftAdapterMemory 基准测试Raft适配器内存使用
func benchmarkRaftAdapterMemory(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
		
		// 模拟一些操作
		ctx := context.Background()
		adapter.Start(ctx)
		adapter.GetStatus()
		adapter.GetMetrics()
		adapter.Stop()
	}
}

// benchmarkPoAAdapterMemory 基准测试PoA适配器内存使用
func benchmarkPoAAdapterMemory(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
		
		// 模拟一些操作
		ctx := context.Background()
		adapter.Start(ctx)
		adapter.GetStatus()
		adapter.ValidateProposer("node1", 1)
		adapter.Stop()
	}
}

// benchmarkMetricsMemory 基准测试指标内存使用
func benchmarkMetricsMemory(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics := NewConsensusMetrics()
		
		// 模拟指标更新
		metrics.UpdateProposal(true, time.Millisecond)
		metrics.UpdateNetworkMetrics(3, 3, time.Millisecond)
		metrics.UpdateStatus(1, 1, true, true)
		metrics.SetCustomMetric("test", "value")
		metrics.GetSnapshot()
	}
}