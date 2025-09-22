package did

import (
	"testing"
	"time"
)

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()
	
	// 测试记录操作
	metrics.RecordRegister(100*time.Millisecond, true)
	metrics.RecordResolve(50*time.Millisecond, true)
	metrics.RecordUpdate(75*time.Millisecond, false)
	metrics.RecordRevoke(25*time.Millisecond, true)
	
	// 验证计数器
	if metrics.RegisterCount != 1 {
		t.Errorf("期望RegisterCount为1，实际为%d", metrics.RegisterCount)
	}
	if metrics.ResolveCount != 1 {
		t.Errorf("期望ResolveCount为1，实际为%d", metrics.ResolveCount)
	}
	if metrics.UpdateCount != 1 {
		t.Errorf("期望UpdateCount为1，实际为%d", metrics.UpdateCount)
	}
	if metrics.RevokeCount != 1 {
		t.Errorf("期望RevokeCount为1，实际为%d", metrics.RevokeCount)
	}
	
	// 验证错误计数器
	if metrics.RegisterErrors != 0 {
		t.Errorf("期望RegisterErrors为0，实际为%d", metrics.RegisterErrors)
	}
	if metrics.UpdateErrors != 1 {
		t.Errorf("期望UpdateErrors为1，实际为%d", metrics.UpdateErrors)
	}
	
	// 验证平均时间
	if metrics.AvgRegisterTime != 100*time.Millisecond {
		t.Errorf("期望AvgRegisterTime为100ms，实际为%v", metrics.AvgRegisterTime)
	}
	if metrics.AvgResolveTime != 50*time.Millisecond {
		t.Errorf("期望AvgResolveTime为50ms，实际为%v", metrics.AvgResolveTime)
	}
}

func TestMetricsCache(t *testing.T) {
	metrics := NewMetrics()
	
	// 记录缓存操作
	metrics.RecordCacheHit()
	metrics.RecordCacheHit()
	metrics.RecordCacheMiss()
	metrics.UpdateCacheSize(100)
	
	// 验证缓存指标
	if metrics.CacheHits != 2 {
		t.Errorf("期望CacheHits为2，实际为%d", metrics.CacheHits)
	}
	if metrics.CacheMisses != 1 {
		t.Errorf("期望CacheMisses为1，实际为%d", metrics.CacheMisses)
	}
	if metrics.CacheSize != 100 {
		t.Errorf("期望CacheSize为100，实际为%d", metrics.CacheSize)
	}
	
	// 验证缓存命中率
	hitRate := metrics.GetCacheHitRate()
	expectedRate := 2.0 / 3.0
	if hitRate != expectedRate {
		t.Errorf("期望缓存命中率为%.2f，实际为%.2f", expectedRate, hitRate)
	}
}

func TestMetricsSuccessRate(t *testing.T) {
	metrics := NewMetrics()
	
	// 记录多个操作
	metrics.RecordRegister(10*time.Millisecond, true)
	metrics.RecordRegister(15*time.Millisecond, true)
	metrics.RecordRegister(20*time.Millisecond, false)
	
	metrics.RecordResolve(5*time.Millisecond, true)
	metrics.RecordResolve(8*time.Millisecond, false)
	
	// 获取成功率
	rates := metrics.GetSuccessRate()
	
	// 验证注册成功率 (2/3)
	expectedRegisterRate := 2.0 / 3.0
	if rates["register"] != expectedRegisterRate {
		t.Errorf("期望注册成功率为%.2f，实际为%.2f", expectedRegisterRate, rates["register"])
	}
	
	// 验证解析成功率 (1/2)
	expectedResolveRate := 1.0 / 2.0
	if rates["resolve"] != expectedResolveRate {
		t.Errorf("期望解析成功率为%.2f，实际为%.2f", expectedResolveRate, rates["resolve"])
	}
}

func TestMetricsSnapshot(t *testing.T) {
	metrics := NewMetrics()
	
	// 记录一些操作
	metrics.RecordRegister(100*time.Millisecond, true)
	metrics.RecordResolve(50*time.Millisecond, false)
	metrics.RecordCacheHit()
	metrics.UpdateDIDCounts(10, 2)
	
	// 获取快照
	snapshot := metrics.GetSnapshot()
	
	// 验证快照数据
	if snapshot.RegisterCount != 1 {
		t.Errorf("快照中RegisterCount期望为1，实际为%d", snapshot.RegisterCount)
	}
	if snapshot.ResolveCount != 1 {
		t.Errorf("快照中ResolveCount期望为1，实际为%d", snapshot.ResolveCount)
	}
	if snapshot.ResolveErrors != 1 {
		t.Errorf("快照中ResolveErrors期望为1，实际为%d", snapshot.ResolveErrors)
	}
	if snapshot.CacheHits != 1 {
		t.Errorf("快照中CacheHits期望为1，实际为%d", snapshot.CacheHits)
	}
	if snapshot.ActiveDIDs != 10 {
		t.Errorf("快照中ActiveDIDs期望为10，实际为%d", snapshot.ActiveDIDs)
	}
	if snapshot.RevokedDIDs != 2 {
		t.Errorf("快照中RevokedDIDs期望为2，实际为%d", snapshot.RevokedDIDs)
	}
}

func TestMetricsReset(t *testing.T) {
	metrics := NewMetrics()
	
	// 记录一些操作
	metrics.RecordRegister(100*time.Millisecond, true)
	metrics.RecordResolve(50*time.Millisecond, false)
	metrics.RecordCacheHit()
	metrics.UpdateDIDCounts(10, 2)
	
	// 重置指标
	metrics.Reset()
	
	// 验证所有指标都被重置
	if metrics.RegisterCount != 0 {
		t.Errorf("重置后RegisterCount期望为0，实际为%d", metrics.RegisterCount)
	}
	if metrics.ResolveCount != 0 {
		t.Errorf("重置后ResolveCount期望为0，实际为%d", metrics.ResolveCount)
	}
	if metrics.ResolveErrors != 0 {
		t.Errorf("重置后ResolveErrors期望为0，实际为%d", metrics.ResolveErrors)
	}
	if metrics.CacheHits != 0 {
		t.Errorf("重置后CacheHits期望为0，实际为%d", metrics.CacheHits)
	}
	if metrics.ActiveDIDs != 0 {
		t.Errorf("重置后ActiveDIDs期望为0，实际为%d", metrics.ActiveDIDs)
	}
	if metrics.AvgRegisterTime != 0 {
		t.Errorf("重置后AvgRegisterTime期望为0，实际为%v", metrics.AvgRegisterTime)
	}
}

func TestMetricsAverageCalculation(t *testing.T) {
	metrics := NewMetrics()
	
	// 记录多个注册操作
	durations := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}
	
	for _, d := range durations {
		metrics.RecordRegister(d, true)
	}
	
	// 计算期望的平均值
	expectedAvg := (100 + 200 + 300) * time.Millisecond / 3
	
	if metrics.AvgRegisterTime != expectedAvg {
		t.Errorf("期望平均注册时间为%v，实际为%v", expectedAvg, metrics.AvgRegisterTime)
	}
}

func TestMetricsCollector(t *testing.T) {
	metrics := NewMetrics()
	collector := NewMetricsCollector(metrics, 10*time.Millisecond)
	
	// 启动收集器
	collector.Start()
	
	// 等待一小段时间
	time.Sleep(50 * time.Millisecond)
	
	// 停止收集器
	collector.Stop()
	
	// 验证收集器可以正常启动和停止
	// 这里主要是测试没有panic或死锁
}

func BenchmarkMetricsRecordRegister(b *testing.B) {
	metrics := NewMetrics()
	duration := 100 * time.Millisecond
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordRegister(duration, true)
	}
}

func BenchmarkMetricsGetSnapshot(b *testing.B) {
	metrics := NewMetrics()
	
	// 预先记录一些数据
	for i := 0; i < 100; i++ {
		metrics.RecordRegister(time.Duration(i)*time.Millisecond, true)
		metrics.RecordResolve(time.Duration(i)*time.Millisecond, true)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetSnapshot()
	}
}

func BenchmarkMetricsGetSuccessRate(b *testing.B) {
	metrics := NewMetrics()
	
	// 预先记录一些数据
	for i := 0; i < 100; i++ {
		metrics.RecordRegister(time.Duration(i)*time.Millisecond, i%2 == 0)
		metrics.RecordResolve(time.Duration(i)*time.Millisecond, i%3 == 0)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = metrics.GetSuccessRate()
	}
}