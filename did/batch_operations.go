package did

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/utils"
)

// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	Index   int         `json:"index"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   error       `json:"error,omitempty"`
}

// BatchRegisterRequest 批量注册请求
type BatchRegisterRequest struct {
	Requests []RegisterRequest `json:"requests"`
	Options  *BatchOptions     `json:"options,omitempty"`
}

// BatchUpdateRequest 批量更新请求
type BatchUpdateRequest struct {
	Requests []UpdateRequest `json:"requests"`
	Options  *BatchOptions   `json:"options,omitempty"`
}

// BatchRevokeRequest 批量撤销请求
type BatchRevokeRequest struct {
	DIDs    []string      `json:"dids"`
	Proof   *Proof        `json:"proof"`
	Options *BatchOptions `json:"options,omitempty"`
}

// BatchOptions 批量操作选项
type BatchOptions struct {
	MaxConcurrency int           `json:"max_concurrency"`
	Timeout        time.Duration `json:"timeout"`
	FailFast       bool          `json:"fail_fast"`
	RetryCount     int           `json:"retry_count"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// DefaultBatchOptions 默认批量操作选项
func DefaultBatchOptions() *BatchOptions {
	return &BatchOptions{
		MaxConcurrency: 10,
		Timeout:        time.Minute * 5,
		FailFast:       false,
		RetryCount:     3,
		RetryDelay:     time.Second,
	}
}

// BatchDIDRegistry 批量DID注册表
type BatchDIDRegistry struct {
	registry *DIDRegistry
	metrics  *Metrics
}

// NewBatchDIDRegistry 创建批量DID注册表
func NewBatchDIDRegistry(cfg *config.Config, blockchain interface{}) *BatchDIDRegistry {
	return &BatchDIDRegistry{
		registry: NewDIDRegistry(cfg, blockchain),
		metrics:  NewMetrics(),
	}
}

// BatchRegister 批量注册DID
func (br *BatchDIDRegistry) BatchRegister(ctx context.Context, req *BatchRegisterRequest) ([]*BatchOperationResult, error) {
	if req.Options == nil {
		req.Options = DefaultBatchOptions()
	}

	results := make([]*BatchOperationResult, len(req.Requests))
	semaphore := make(chan struct{}, req.Options.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	start := time.Now()
	defer func() {
		br.metrics.RecordRegister(time.Since(start), true)
	}()

	for i, registerReq := range req.Requests {
		wg.Add(1)
		go func(index int, request RegisterRequest) {
			defer wg.Done()

			// 获取信号量
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mu.Lock()
				results[index] = &BatchOperationResult{
					Index:   index,
					Success: false,
					Error:   ctx.Err(),
				}
				mu.Unlock()
				return
			}

			// 执行注册操作
			result := br.executeWithRetry(ctx, func() (interface{}, error) {
				return br.registry.Register(&request)
			}, req.Options)

			mu.Lock()
			results[index] = &BatchOperationResult{
				Index:   index,
				Success: result.Error == nil,
				Data:    result.Data,
				Error:   result.Error,
			}
			mu.Unlock()

			// 如果启用快速失败且有错误
			if req.Options.FailFast && result.Error != nil {
				// 这里可以实现取消其他操作的逻辑
			}
		}(i, registerReq)
	}

	// 等待所有操作完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return results, nil
	case <-ctx.Done():
		return results, ctx.Err()
	case <-time.After(req.Options.Timeout):
		return results, utils.NewErrorWithDetails(utils.ErrorTypeTimeout, "BATCH_TIMEOUT", "批量操作超时", fmt.Sprintf("timeout: %v, count: %d", req.Options.Timeout, len(req.Requests)))
	}
}

// BatchResolve 批量解析DID
func (br *BatchDIDRegistry) BatchResolve(ctx context.Context, dids []string, options *BatchOptions) ([]*BatchOperationResult, error) {
	if options == nil {
		options = DefaultBatchOptions()
	}

	results := make([]*BatchOperationResult, len(dids))
	semaphore := make(chan struct{}, options.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	start := time.Now()
	defer func() {
		br.metrics.RecordResolve(time.Since(start), true)
	}()

	for i, did := range dids {
		wg.Add(1)
		go func(index int, didStr string) {
			defer wg.Done()

			// 获取信号量
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mu.Lock()
				results[index] = &BatchOperationResult{
					Index:   index,
					Success: false,
					Error:   ctx.Err(),
				}
				mu.Unlock()
				return
			}

			// 执行解析操作
			result := br.executeWithRetry(ctx, func() (interface{}, error) {
				return br.registry.Resolve(didStr)
			}, options)

			mu.Lock()
			results[index] = &BatchOperationResult{
				Index:   index,
				Success: result.Error == nil,
				Data:    result.Data,
				Error:   result.Error,
			}
			mu.Unlock()
		}(i, did)
	}

	// 等待所有操作完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return results, nil
	case <-ctx.Done():
		return results, ctx.Err()
	case <-time.After(options.Timeout):
		return results, utils.NewErrorWithDetails(utils.ErrorTypeTimeout, "BATCH_TIMEOUT", "批量解析超时", fmt.Sprintf("timeout: %v, count: %d", options.Timeout, len(dids)))
	}
}

// BatchUpdate 批量更新DID
func (br *BatchDIDRegistry) BatchUpdate(ctx context.Context, req *BatchUpdateRequest) ([]*BatchOperationResult, error) {
	if req.Options == nil {
		req.Options = DefaultBatchOptions()
	}

	results := make([]*BatchOperationResult, len(req.Requests))
	semaphore := make(chan struct{}, req.Options.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	start := time.Now()
	defer func() {
		br.metrics.RecordUpdate(time.Since(start), true)
	}()

	for i, updateReq := range req.Requests {
		wg.Add(1)
		go func(index int, request UpdateRequest) {
			defer wg.Done()

			// 获取信号量
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mu.Lock()
				results[index] = &BatchOperationResult{
					Index:   index,
					Success: false,
					Error:   ctx.Err(),
				}
				mu.Unlock()
				return
			}

			// 执行更新操作
			result := br.executeWithRetry(ctx, func() (interface{}, error) {
				return br.registry.Update(&request)
			}, req.Options)

			mu.Lock()
			results[index] = &BatchOperationResult{
				Index:   index,
				Success: result.Error == nil,
				Data:    result.Data,
				Error:   result.Error,
			}
			mu.Unlock()
		}(i, updateReq)
	}

	// 等待所有操作完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return results, nil
	case <-ctx.Done():
		return results, ctx.Err()
	case <-time.After(req.Options.Timeout):
		return results, utils.NewErrorWithDetails(utils.ErrorTypeTimeout, "BATCH_TIMEOUT", "批量更新超时", fmt.Sprintf("timeout: %v, count: %d", req.Options.Timeout, len(req.Requests)))
	}
}

// BatchRevoke 批量撤销DID
func (br *BatchDIDRegistry) BatchRevoke(ctx context.Context, req *BatchRevokeRequest) ([]*BatchOperationResult, error) {
	if req.Options == nil {
		req.Options = DefaultBatchOptions()
	}

	results := make([]*BatchOperationResult, len(req.DIDs))
	semaphore := make(chan struct{}, req.Options.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	start := time.Now()
	defer func() {
		br.metrics.RecordRevoke(time.Since(start), true)
	}()

	for i, did := range req.DIDs {
		wg.Add(1)
		go func(index int, didStr string) {
			defer wg.Done()

			// 获取信号量
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				mu.Lock()
				results[index] = &BatchOperationResult{
					Index:   index,
					Success: false,
					Error:   ctx.Err(),
				}
				mu.Unlock()
				return
			}

			// 执行撤销操作
			result := br.executeWithRetry(ctx, func() (interface{}, error) {
				return nil, br.registry.Revoke(didStr, req.Proof)
			}, req.Options)

			mu.Lock()
			results[index] = &BatchOperationResult{
				Index:   index,
				Success: result.Error == nil,
				Data:    result.Data,
				Error:   result.Error,
			}
			mu.Unlock()
		}(i, did)
	}

	// 等待所有操作完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return results, nil
	case <-ctx.Done():
		return results, ctx.Err()
	case <-time.After(req.Options.Timeout):
		return results, utils.NewErrorWithDetails(utils.ErrorTypeTimeout, "BATCH_TIMEOUT", "批量撤销超时", fmt.Sprintf("timeout: %v, count: %d", req.Options.Timeout, len(req.DIDs)))
	}
}

// executeWithRetry 带重试的执行函数
func (br *BatchDIDRegistry) executeWithRetry(ctx context.Context, fn func() (interface{}, error), options *BatchOptions) *BatchOperationResult {
	var lastErr error
	var result interface{}

	for attempt := 0; attempt <= options.RetryCount; attempt++ {
		select {
		case <-ctx.Done():
			return &BatchOperationResult{
				Success: false,
				Error:   ctx.Err(),
			}
		default:
		}

		result, lastErr = fn()
		if lastErr == nil {
			return &BatchOperationResult{
				Success: true,
				Data:    result,
			}
		}

		// 如果不是最后一次尝试，等待重试延迟
		if attempt < options.RetryCount {
			select {
			case <-time.After(options.RetryDelay):
			case <-ctx.Done():
				return &BatchOperationResult{
					Success: false,
					Error:   ctx.Err(),
				}
			}
		}
	}

	return &BatchOperationResult{
		Success: false,
		Error:   lastErr,
	}
}

// GetBatchMetrics 获取批量操作指标
func (br *BatchDIDRegistry) GetBatchMetrics() map[string]interface{} {
	return map[string]interface{}{
		"batch_operations": br.metrics.GetSnapshot(),
	}
}

// BatchOperationStats 批量操作统计
type BatchOperationStats struct {
	TotalOperations   int           `json:"total_operations"`
	SuccessfulOps     int           `json:"successful_ops"`
	FailedOps         int           `json:"failed_ops"`
	AverageTime       time.Duration `json:"average_time"`
	TotalTime         time.Duration `json:"total_time"`
	ConcurrencyLevel  int           `json:"concurrency_level"`
	RetryCount        int           `json:"retry_count"`
}

// AnalyzeBatchResults 分析批量操作结果
func AnalyzeBatchResults(results []*BatchOperationResult, startTime time.Time) *BatchOperationStats {
	stats := &BatchOperationStats{
		TotalOperations: len(results),
		TotalTime:       time.Since(startTime),
	}

	for _, result := range results {
		if result.Success {
			stats.SuccessfulOps++
		} else {
			stats.FailedOps++
		}
	}

	if stats.TotalOperations > 0 {
		stats.AverageTime = stats.TotalTime / time.Duration(stats.TotalOperations)
	}

	return stats
}

// PrintBatchResults 打印批量操作结果
func PrintBatchResults(results []*BatchOperationResult) {
	successCount := 0
	failCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failCount++
			fmt.Printf("操作 %d 失败: %v\n", result.Index, result.Error)
		}
	}

	fmt.Printf("批量操作完成: 成功 %d, 失败 %d, 总计 %d\n", successCount, failCount, len(results))
}