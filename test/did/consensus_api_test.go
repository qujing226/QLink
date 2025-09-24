package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/pkg/api"
	"github.com/qujing226/QLink/pkg/config"
	"github.com/qujing226/QLink/pkg/consensus"
	"github.com/qujing226/QLink/pkg/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConsensusAPI 测试共识API功能
type TestConsensusAPI struct {
	router *gin.Engine
	api    *api.ConsensusAPI
}

// SetupTestConsensusAPI 设置测试环境
func SetupTestConsensusAPI(t *testing.T) *TestConsensusAPI {
	gin.SetMode(gin.TestMode)

	// 创建模拟依赖
	p2pNetwork := &network.P2PNetwork{}                          // 简化的模拟对象
	raftNode := consensus.NewRaftNode("test-node-1", p2pNetwork) // 使用NewRaftNode创建完整实例
	didRegistry := &did.DIDRegistry{}                            // 简化的模拟对象

	// 创建共识配置
	config := &config.ConsensusConfig{
		ProposalTimeout:     5 * time.Second,
		CommitTimeout:       3 * time.Second,
		MaxPendingProposals: 100,
		BatchSize:           10,
	}

	// 创建共识集成实例
	consensusIntegration := consensus.NewConsensusIntegration(
		"test-node-1",
		raftNode,
		didRegistry,
		p2pNetwork,
		config,
	)

	// 启动共识集成器（测试模式）
	// 设置Raft节点为Leader状态以允许提议操作
	// 通过反射直接设置RaftNode的内部状态为Leader
	raftNodeValue := reflect.ValueOf(raftNode).Elem()
	stateField := raftNodeValue.FieldByName("State")
	if stateField.IsValid() && stateField.CanSet() {
		stateField.Set(reflect.ValueOf(consensus.Leader))
	}
	termField := raftNodeValue.FieldByName("term")
	if termField.IsValid() && termField.CanSet() {
		termField.SetInt(1)
	}

	// 创建API实例
	consensusAPI := api.NewConsensusAPI(consensusIntegration)

	// 设置路由
	router := gin.New()
	consensusAPI.RegisterRoutes(router)

	return &TestConsensusAPI{
		router: router,
		api:    consensusAPI,
	}
}

// TestProposeOperation 测试提议操作
func TestProposeOperation(t *testing.T) {
	testAPI := SetupTestConsensusAPI(t)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "有效的DID操作",
			requestBody: map[string]interface{}{
				"type": "did_operation",
				"data": map[string]interface{}{
					"operation": "register",
					"did":       "did:qlink:test123",
					"document": map[string]interface{}{
						"id": "did:qlink:test123",
						"publicKey": []map[string]interface{}{
							{
								"id":              "key1",
								"type":            "Ed25519VerificationKey2018",
								"publicKeyBase58": "H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV",
							},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "无效的操作类型",
			requestBody: map[string]interface{}{
				"type": "invalid_operation",
				"data": map[string]interface{}{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "不支持的操作类型",
		},
		{
			name: "缺少数据字段",
			requestBody: map[string]interface{}{
				"type": "did_operation",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/consensus/propose", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			testAPI.router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

// TestGetConsensusStatus 测试获取共识状态
func TestGetConsensusStatus(t *testing.T) {
	testAPI := SetupTestConsensusAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/consensus/status", nil)
	w := httptest.NewRecorder()

	testAPI.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var status map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &status)
	require.NoError(t, err)

	// 验证状态字段
	assert.Contains(t, status, "current_term")
	assert.Contains(t, status, "status")
	assert.Contains(t, status, "last_commit_index")
}

// TestGetNodes 测试获取节点列表
func TestGetNodes(t *testing.T) {
	testAPI := SetupTestConsensusAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/consensus/nodes", nil)
	w := httptest.NewRecorder()

	testAPI.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "nodes")
	assert.Contains(t, response, "count")

	nodes, ok := response["nodes"].([]interface{})
	assert.True(t, ok)
	count, ok := response["count"].(float64)
	assert.True(t, ok)
	assert.Equal(t, float64(len(nodes)), count)
}

// TestGetLeader 测试获取领导者
func TestGetLeader(t *testing.T) {
	testAPI := SetupTestConsensusAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/consensus/leader", nil)
	w := httptest.NewRecorder()

	testAPI.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "leader")
}

// TestHealthCheck 测试健康检查
func TestHealthCheck(t *testing.T) {
	testAPI := SetupTestConsensusAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/consensus/health", nil)
	w := httptest.NewRecorder()

	testAPI.router.ServeHTTP(w, req)

	// 健康检查应该返回200或503
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")

	status := response["status"].(string)
	assert.Contains(t, []string{"healthy", "unhealthy"}, status)
}

// TestGetMetrics 测试获取指标
func TestGetMetrics(t *testing.T) {
	testAPI := SetupTestConsensusAPI(t)

	req := httptest.NewRequest(http.MethodGet, "/consensus/metrics", nil)
	w := httptest.NewRecorder()

	testAPI.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var metrics api.ConsensusMetrics
	err := json.Unmarshal(w.Body.Bytes(), &metrics)
	require.NoError(t, err)

	// 验证指标字段
	assert.GreaterOrEqual(t, metrics.TotalProposals, int64(0))
	assert.GreaterOrEqual(t, metrics.SuccessfulCommits, int64(0))
	assert.GreaterOrEqual(t, metrics.FailedCommits, int64(0))
	assert.GreaterOrEqual(t, metrics.AverageLatency, float64(0))
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	testAPI := SetupTestConsensusAPI(t)

	const numOperations = 10
	results := make(chan int, numOperations)

	// 并发发送多个提议
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			requestBody := map[string]interface{}{
				"type": "did_operation",
				"data": map[string]interface{}{
					"operation": "register",
					"did":       fmt.Sprintf("did:qlink:test%d", id),
					"document": map[string]interface{}{
						"id": fmt.Sprintf("did:qlink:test%d", id),
					},
				},
			}

			body, _ := json.Marshal(requestBody)
			req := httptest.NewRequest(http.MethodPost, "/consensus/propose", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			testAPI.router.ServeHTTP(w, req)

			results <- w.Code
		}(i)
	}

	// 收集结果
	successCount := 0
	for i := 0; i < numOperations; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		}
	}

	// 至少应该有一些操作成功
	assert.Greater(t, successCount, 0)
	t.Logf("成功操作数: %d/%d", successCount, numOperations)
}

// BenchmarkProposeOperation 性能测试
func BenchmarkProposeOperation(b *testing.B) {
	testAPI := SetupTestConsensusAPI(&testing.T{})

	requestBody := map[string]interface{}{
		"type": "did_operation",
		"data": map[string]interface{}{
			"operation": "register",
			"did":       "did:qlink:benchmark",
			"document": map[string]interface{}{
				"id": "did:qlink:benchmark",
			},
		},
	}

	body, _ := json.Marshal(requestBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/consensus/propose", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		testAPI.router.ServeHTTP(w, req)
	}
}
