package consensus

import (
	"context"
	"testing"
	"time"

	"github.com/qujing226/QLink/pkg/interfaces"
	"github.com/qujing226/QLink/pkg/network"
)

// TestRaftAdapter 测试Raft适配器
func TestRaftAdapter(t *testing.T) {
	// 创建模拟的P2P网络
	p2pNetwork := &network.P2PNetwork{}
	
	// 创建Raft适配器
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	// 测试基本接口
	testConsensusAlgorithm(t, adapter, "Raft")
}

// TestPoAAdapter 测试PoA适配器
func TestPoAAdapter(t *testing.T) {
	// 创建模拟的P2P网络
	p2pNetwork := &network.P2PNetwork{}
	
	// 创建PoA适配器
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	// 测试基本接口
	testConsensusAlgorithm(t, adapter, "Proof of Authority")
}

// testConsensusAlgorithm 测试共识算法接口
func testConsensusAlgorithm(t *testing.T, algorithm interfaces.ConsensusAlgorithm, expectedName string) {
	ctx := context.Background()
	
	// 测试类型和名称
	if adapter, ok := algorithm.(*RaftAdapter); ok {
		if adapter.GetType() != interfaces.ConsensusTypeRaft {
			t.Errorf("Expected Raft type, got %v", adapter.GetType())
		}
		if adapter.GetName() != expectedName {
			t.Errorf("Expected name %s, got %s", expectedName, adapter.GetName())
		}
	}
	
	if adapter, ok := algorithm.(*PoAAdapter); ok {
		if adapter.GetType() != interfaces.ConsensusTypePoA {
			t.Errorf("Expected PoA type, got %v", adapter.GetType())
		}
		if adapter.GetName() != expectedName {
			t.Errorf("Expected name %s, got %s", expectedName, adapter.GetName())
		}
	}
	
	// 测试初始状态
	status := algorithm.GetStatus()
	if status == nil {
		t.Error("Status should not be nil")
	}
	
	// 测试节点列表
	nodes := algorithm.GetNodes()
	if len(nodes) == 0 {
		t.Error("Nodes list should not be empty")
	}
	
	// 测试启动和停止
	err := algorithm.Start(ctx)
	if err != nil {
		t.Logf("Start failed (expected in test environment): %v", err)
	}
	
	// 测试停止
	err = algorithm.Stop()
	if err != nil {
		t.Logf("Stop failed (expected in test environment): %v", err)
	}
}

// TestConsensusSwitcherAdapter 测试共识切换器适配器
func TestConsensusSwitcherAdapter(t *testing.T) {
	// 创建切换器配置
	config := &SwitcherAdapterConfig{
		SwitchStrategy:      SwitchStrategyGraceful,
		SwitchTimeout:       10 * time.Second,
		DataSyncTimeout:     5 * time.Second,
		EnableAutoSwitch:    false,
		RequireConfirmation: false,
		BackupBeforeSwitch:  false,
		EnableRollback:      false,
	}
	
	// 创建切换器适配器
	switcher := NewConsensusSwitcherAdapter(config)
	
	// 创建模拟的P2P网络
	p2pNetwork := &network.P2PNetwork{}
	
	// 初始化切换器
	err := switcher.Initialize("node1", []string{"node2", "node3"}, []string{"node1", "node2", "node3"}, p2pNetwork, nil)
	if err != nil {
		t.Fatalf("Failed to initialize switcher: %v", err)
	}
	
	// 测试当前类型
	currentType := switcher.GetCurrentType()
	if currentType != interfaces.ConsensusTypeRaft {
		t.Errorf("Expected initial type to be Raft, got %v", currentType)
	}
	
	// 测试支持的类型
	supportedTypes := switcher.GetSupportedTypes()
	if len(supportedTypes) != 2 {
		t.Errorf("Expected 2 supported types, got %d", len(supportedTypes))
	}
	
	// 测试是否支持特定类型
	if !switcher.IsSupported(interfaces.ConsensusTypeRaft) {
		t.Error("Should support Raft")
	}
	
	if !switcher.IsSupported(interfaces.ConsensusTypePoA) {
		t.Error("Should support PoA")
	}
	
	if switcher.IsSupported(interfaces.ConsensusTypePBFT) {
		t.Error("Should not support PBFT")
	}
	
	// 测试状态获取
	status := switcher.GetStatus()
	if status == nil {
		t.Error("Status should not be nil")
	}
	
	// 测试切换状态
	switchState := switcher.GetSwitchState()
	if switchState == nil {
		t.Error("Switch state should not be nil")
	}
	
	if switchState.InProgress {
		t.Error("Should not be switching initially")
	}
}

// TestConsensusTypeString 测试共识类型字符串转换
func TestConsensusTypeString(t *testing.T) {
	tests := []struct {
		consensusType interfaces.ConsensusType
		expected      string
	}{
		{interfaces.ConsensusTypeRaft, "Raft"},
		{interfaces.ConsensusTypePoA, "PoA"},
		{interfaces.ConsensusTypePBFT, "PBFT"},
		{interfaces.ConsensusTypePoS, "PoS"},
	}
	
	for _, test := range tests {
		result := test.consensusType.String()
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

// TestRaftAdapterMetrics 测试Raft适配器指标
func TestRaftAdapterMetrics(t *testing.T) {
	p2pNetwork := &network.P2PNetwork{}
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	metrics := adapter.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}
	
	// 检查必要的指标字段
	expectedFields := []string{"current_term", "log_entries", "commit_index", "last_applied", "peer_count", "state"}
	for _, field := range expectedFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Metrics should contain field: %s", field)
		}
	}
}

// TestPoAAdapterMetrics 测试PoA适配器指标
func TestPoAAdapterMetrics(t *testing.T) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	metrics := adapter.GetMetrics()
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}
	
	// 检查必要的指标字段
	expectedFields := []string{"block_height", "authority_count", "is_authority", "proposal_count"}
	for _, field := range expectedFields {
		if _, exists := metrics[field]; !exists {
			t.Errorf("Metrics should contain field: %s", field)
		}
	}
}

// TestRaftAdapterValidation 测试Raft适配器验证功能
func TestRaftAdapterValidation(t *testing.T) {
	p2pNetwork := &network.P2PNetwork{}
	adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
	
	// 测试区块验证
	err := adapter.ValidateBlock(nil)
	if err == nil {
		t.Error("Should reject nil block")
	}
	
	err = adapter.ValidateBlock("valid_block")
	if err != nil {
		t.Errorf("Should accept valid block: %v", err)
	}
	
	// 测试提案者验证
	err = adapter.ValidateProposer("unknown_node", 1)
	if err == nil {
		t.Error("Should reject unknown proposer")
	}
	
	// 测试权威节点检查
	if !adapter.IsAuthority("node1") {
		t.Error("node1 should be authority")
	}
	
	if adapter.IsAuthority("unknown_node") {
		t.Error("unknown_node should not be authority")
	}
	
	// 测试权威节点列表
	authorities := adapter.GetAuthorities()
	if len(authorities) == 0 {
		t.Error("Should have authorities")
	}
}

// TestPoAAdapterValidation 测试PoA适配器验证功能
func TestPoAAdapterValidation(t *testing.T) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
	
	// 测试区块验证
	err := adapter.ValidateBlock(nil)
	if err == nil {
		t.Error("Should reject nil block")
	}
	
	// 测试提案者验证
	err = adapter.ValidateProposer("node1", 1)
	if err != nil {
		t.Errorf("Should accept authority proposer: %v", err)
	}
	
	err = adapter.ValidateProposer("unknown_node", 1)
	if err == nil {
		t.Error("Should reject non-authority proposer")
	}
	
	// 测试下一个提案者
	nextProposer := adapter.GetNextProposer(0)
	if nextProposer != "node1" {
		t.Errorf("Expected node1 as next proposer for block 0, got %s", nextProposer)
	}
	
	nextProposer = adapter.GetNextProposer(1)
	if nextProposer != "node2" {
		t.Errorf("Expected node2 as next proposer for block 1, got %s", nextProposer)
	}
	
	// 测试权威节点管理
	err = adapter.AddAuthority("node4")
	if err != nil {
		t.Errorf("Should be able to add new authority: %v", err)
	}
	
	err = adapter.AddAuthority("node1")
	if err == nil {
		t.Error("Should not be able to add existing authority")
	}
	
	err = adapter.RemoveAuthority("node4")
	if err != nil {
		t.Errorf("Should be able to remove authority: %v", err)
	}
	
	err = adapter.RemoveAuthority("unknown_node")
	if err == nil {
		t.Error("Should not be able to remove non-existent authority")
	}
}

// BenchmarkRaftAdapterStart 基准测试Raft适配器启动
func BenchmarkRaftAdapterStart(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := NewRaftAdapter("node1", []string{"node2", "node3"}, p2pNetwork)
		ctx := context.Background()
		adapter.Start(ctx)
		adapter.Stop()
	}
}

// BenchmarkPoAAdapterStart 基准测试PoA适配器启动
func BenchmarkPoAAdapterStart(b *testing.B) {
	p2pNetwork := &network.P2PNetwork{}
	authorities := []string{"node1", "node2", "node3"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter := NewPoAAdapter("node1", authorities, p2pNetwork)
		ctx := context.Background()
		adapter.Start(ctx)
		adapter.Stop()
	}
}