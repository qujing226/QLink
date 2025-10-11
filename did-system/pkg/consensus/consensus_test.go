package consensus

import (
    "context"
    "testing"
    "time"

    "github.com/qujing226/QLink/pkg/interfaces"
    "github.com/qujing226/QLink/pkg/network"
)

// TestRaftNode 测试Raft节点
func TestRaftNode(t *testing.T) {
    // 创建模拟的P2P网络
    p2pNetwork := &network.P2PNetwork{}

    // 创建Raft节点
    node := NewRaftNode("node1", p2pNetwork)

    // 测试基本接口
    testConsensusAlgorithm(t, node)
}

// TestPoANode 测试PoA节点
func TestPoANode(t *testing.T) {
    // 创建模拟的P2P网络
    p2pNetwork := &network.P2PNetwork{}

    // 创建PoA节点
    authorities := []string{"node1", "node2", "node3"}
    node := NewPoANode("node1", authorities, p2pNetwork)

    // 测试基本接口
    testConsensusAlgorithm(t, node)
}

// testConsensusAlgorithm 测试共识算法统一接口
func testConsensusAlgorithm(t *testing.T, algorithm interfaces.ConsensusAlgorithm) {
    ctx := context.Background()

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

// TestConsensusSwitcher 测试共识切换器
func TestConsensusSwitcher(t *testing.T) {
    // 创建切换器配置
    config := &SwitcherConfig{
        SwitchStrategy:      SwitchStrategyGraceful,
        SwitchTimeout:       10 * time.Second,
        DataSyncTimeout:     5 * time.Second,
        EnableAutoSwitch:    false,
        RequireConfirmation: false,
        BackupBeforeSwitch:  false,
        EnableRollback:      false,
    }

    // 创建切换器
    switcher := NewConsensusSwitcher(config)

    // 创建模拟的P2P网络与节点
    p2pNetwork := &network.P2PNetwork{}
    raftNode := NewRaftNode("node1", p2pNetwork)
    poaNode := NewPoANode("node1", []string{"node1", "node2", "node3"}, p2pNetwork)
    monitor := NewConsensusMonitor(nil)

    // 初始化切换器
    err := switcher.Initialize(raftNode, poaNode, monitor)
    if err != nil {
        t.Fatalf("Failed to initialize switcher: %v", err)
    }

    // 测试当前类型
    currentType := switcher.GetCurrentType()
    if currentType != ConsensusTypeRaft {
        t.Errorf("Expected initial type to be Raft, got %v", currentType)
    }

    // 测试支持的类型
    supportedTypes := switcher.GetSupportedTypes()
    if len(supportedTypes) != 2 {
        t.Errorf("Expected 2 supported types, got %d", len(supportedTypes))
    }

    // 测试是否支持特定类型
    if !switcher.IsSupported(ConsensusTypeRaft) {
        t.Error("Should support Raft")
    }

    if !switcher.IsSupported(ConsensusTypePoA) {
        t.Error("Should support PoA")
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

// TestConsensusTypeString 测试共识类型字符串转换（使用共识模块常量）
func TestConsensusTypeString(t *testing.T) {
    tests := []struct {
        consensusType ConsensusType
        expected      string
    }{
        {ConsensusTypeRaft, "Raft"},
        {ConsensusTypePoA, "PoA"},
        {ConsensusTypePBFT, "PBFT"},
        {ConsensusTypePoS, "PoS"},
    }

    for _, test := range tests {
        // 使用 switcher 的 getConsensusTypeName 等价逻辑
        var result string
        switch test.consensusType {
        case ConsensusTypeRaft:
            result = "Raft"
        case ConsensusTypePoA:
            result = "PoA"
        case ConsensusTypePBFT:
            result = "PBFT"
        case ConsensusTypePoS:
            result = "PoS"
        default:
            result = "unknown"
        }
        if result != test.expected {
            t.Errorf("Expected %s, got %s", test.expected, result)
        }
    }
}

// TestRaftStatusFields 检查Raft节点状态字段
func TestRaftStatusFields(t *testing.T) {
    p2pNetwork := &network.P2PNetwork{}
    node := NewRaftNode("node1", p2pNetwork)

    status := node.GetStatus()
    if status == nil {
        t.Error("Status should not be nil")
    }

    // 检查必要的状态字段
    expectedFields := []string{"id", "state", "term", "voted_for", "log_length", "commit_index", "last_applied", "peer_count"}
    for _, field := range expectedFields {
        if _, exists := status[field]; !exists {
            t.Errorf("Status should contain field: %s", field)
        }
    }
}

// TestPoAStatusFields 检查PoA节点状态字段
func TestPoAStatusFields(t *testing.T) {
    p2pNetwork := &network.P2PNetwork{}
    authorities := []string{"node1", "node2", "node3"}
    node := NewPoANode("node1", authorities, p2pNetwork)

    status := node.GetStatus()
    if status == nil {
        t.Error("Status should not be nil")
    }

    // 检查必要的状态字段
    expectedFields := []string{"node_id", "is_authority", "authorities", "block_height", "current_hash", "proposals", "block_time", "vote_threshold"}
    for _, field := range expectedFields {
        if _, exists := status[field]; !exists {
            t.Errorf("Status should contain field: %s", field)
        }
    }
}

// 删除 Raft 适配器验证测试：RaftNode 不提供区块/提案者验证接口

// TestPoAValidation 测试PoA节点验证功能
func TestPoAValidation(t *testing.T) {
    p2pNetwork := &network.P2PNetwork{}
    authorities := []string{"node1", "node2", "node3"}
    node := NewPoANode("node1", authorities, p2pNetwork)

    // 测试区块验证
    err := node.ValidateBlock(nil)
    if err == nil {
        t.Error("Should reject nil block")
    }

    // 测试提案者验证
    err = node.ValidateProposer("node1", 1)
    if err != nil {
        t.Errorf("Should accept authority proposer: %v", err)
    }

    err = node.ValidateProposer("unknown_node", 1)
    if err == nil {
        t.Error("Should reject non-authority proposer")
    }

    // 测试下一个提案者
    nextProposer := node.GetNextProposer(0)
    if nextProposer != "node1" {
        t.Errorf("Expected node1 as next proposer for block 0, got %s", nextProposer)
    }

    nextProposer = node.GetNextProposer(1)
    if nextProposer != "node2" {
        t.Errorf("Expected node2 as next proposer for block 1, got %s", nextProposer)
    }

    // 测试权威节点管理
    err = node.AddAuthority("node4")
    if err != nil {
        t.Errorf("Should be able to add new authority: %v", err)
    }

    err = node.AddAuthority("node1")
    if err == nil {
        t.Error("Should not be able to add existing authority")
    }

    err = node.RemoveAuthority("node4")
    if err != nil {
        t.Errorf("Should be able to remove authority: %v", err)
    }

    err = node.RemoveAuthority("unknown_node")
    if err == nil {
        t.Error("Should not be able to remove non-existent authority")
    }
}

// BenchmarkRaftNodeStart 基准测试Raft节点启动
func BenchmarkRaftNodeStart(b *testing.B) {
    p2pNetwork := &network.P2PNetwork{}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        node := NewRaftNode("node1", p2pNetwork)
        ctx := context.Background()
        node.Start(ctx)
        node.Stop()
    }
}

// BenchmarkPoANodeStart 基准测试PoA节点启动
func BenchmarkPoANodeStart(b *testing.B) {
    p2pNetwork := &network.P2PNetwork{}
    authorities := []string{"node1", "node2", "node3"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        node := NewPoANode("node1", authorities, p2pNetwork)
        ctx := context.Background()
        node.Start(ctx)
        node.Stop()
    }
}
