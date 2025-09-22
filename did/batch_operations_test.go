package did

import (
	"context"
	"testing"
	"time"

	"github.com/qujing226/QLink/did/config"
)

func TestBatchDIDRegistry(t *testing.T) {
	cfg := &config.Config{
		DID: &config.DIDConfig{
			Method:          "qlink",
			ChainID:         "testnet",
			RegistryAddress: "0x1234567890123456789012345678901234567890",
		},
	}

	batchRegistry := NewBatchDIDRegistry(cfg, nil)
	if batchRegistry == nil {
		t.Fatal("Failed to create batch DID registry")
	}
}

func TestDefaultBatchOptions(t *testing.T) {
	options := DefaultBatchOptions()
	
	if options.MaxConcurrency <= 0 {
		t.Error("MaxConcurrency should be positive")
	}
	
	if options.Timeout <= 0 {
		t.Error("Timeout should be positive")
	}
	
	if options.RetryCount < 0 {
		t.Error("RetryCount should be non-negative")
	}
}

func TestAnalyzeBatchResults(t *testing.T) {
	results := []*BatchOperationResult{
		{Index: 0, Success: true, Data: "test1"},
		{Index: 1, Success: false, Error: context.DeadlineExceeded},
		{Index: 2, Success: true, Data: "test2"},
	}
	
	startTime := time.Now().Add(-time.Second)
	stats := AnalyzeBatchResults(results, startTime)
	
	if stats.TotalOperations != 3 {
		t.Errorf("Expected 3 total operations, got %d", stats.TotalOperations)
	}
	
	if stats.SuccessfulOps != 2 {
		t.Errorf("Expected 2 successful operations, got %d", stats.SuccessfulOps)
	}
	
	if stats.FailedOps != 1 {
		t.Errorf("Expected 1 failed operation, got %d", stats.FailedOps)
	}
}

func TestBatchOperationResult(t *testing.T) {
	result := &BatchOperationResult{
		Index:   0,
		Success: true,
		Data:    "test data",
		Error:   nil,
	}
	
	if result.Index != 0 {
		t.Errorf("Expected index 0, got %d", result.Index)
	}
	
	if !result.Success {
		t.Error("Expected success to be true")
	}
	
	if result.Data != "test data" {
		t.Errorf("Expected data 'test data', got %v", result.Data)
	}
}

// 基准测试
func BenchmarkBatchOperationResult(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := &BatchOperationResult{
			Index:   i,
			Success: true,
			Data:    "benchmark data",
			Error:   nil,
		}
		_ = result
	}
}