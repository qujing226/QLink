package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestConfig 测试配置
type TestConfig struct {
	TempDir    string
	ConfigPath string
	Cleanup    func()
}

// SetupTestEnvironment 设置测试环境
func SetupTestEnvironment(t *testing.T) *TestConfig {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "qlink-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 配置文件路径
	configPath := filepath.Join(tempDir, "config.yaml")

	return &TestConfig{
		TempDir:    tempDir,
		ConfigPath: configPath,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// GenerateTestID 生成测试用ID
func GenerateTestID(prefix string) string {
	buf := make([]byte, 16)
	rand.Read(buf)
	uniqueID := hex.EncodeToString(buf)
	if prefix != "" {
		return fmt.Sprintf("%s:%s", prefix, uniqueID)
	}
	return uniqueID
}

// GenerateTestDID 生成测试用DID
func GenerateTestDID(chainID string) string {
	buf := make([]byte, 16)
	rand.Read(buf)
	uniqueID := hex.EncodeToString(buf)
	return fmt.Sprintf("did:QLink:%s:%s", chainID, uniqueID)
}

// CreateTestHTTPRequest 创建测试HTTP请求
func CreateTestHTTPRequest(method, url string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, url, strings.NewReader(string(jsonBody)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	return req
}

// AssertNoError 断言无错误
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError 断言有错误
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 期望有错误但没有", msg)
	}
}

// AssertEqual 断言相等
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: 期望 %v, 实际 %v", msg, expected, actual)
	}
}

// AssertNotEqual 断言不相等
func AssertNotEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected == actual {
		t.Fatalf("%s: 期望不等于 %v, 但实际相等", msg, expected)
	}
}

// AssertNotEmpty 断言非空
func AssertNotEmpty(t *testing.T, value string, msg string) {
	t.Helper()
	if value == "" {
		t.Fatalf("%s: 值不应为空", msg)
	}
}

// AssertEmpty 断言为空
func AssertEmpty(t *testing.T, value string, msg string) {
	t.Helper()
	if value != "" {
		t.Fatalf("%s: 值应为空，实际为: %s", msg, value)
	}
}

// AssertTrue 断言为真
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Fatalf("%s: 条件应为真", msg)
	}
}

// AssertFalse 断言为假
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Fatalf("%s: 条件应为假", msg)
	}
}

// AssertContains 断言包含
func AssertContains(t *testing.T, haystack, needle string, msg string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("%s: '%s' 应包含 '%s'", msg, haystack, needle)
	}
}

// AssertNotContains 断言不包含
func AssertNotContains(t *testing.T, haystack, needle string, msg string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("%s: '%s' 不应包含 '%s'", msg, haystack, needle)
	}
}

// AssertHTTPStatus 断言HTTP状态码
func AssertHTTPStatus(t *testing.T, expected, actual int, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: 期望状态码 %d, 实际 %d", msg, expected, actual)
	}
}

// AssertGreater 断言大于
func AssertGreater(t *testing.T, actual, threshold int, msg string) {
	t.Helper()
	if actual <= threshold {
		t.Fatalf("%s: %d 应大于 %d", msg, actual, threshold)
	}
}

// AssertLess 断言小于
func AssertLess(t *testing.T, actual, threshold int, msg string) {
	t.Helper()
	if actual >= threshold {
		t.Fatalf("%s: %d 应小于 %d", msg, actual, threshold)
	}
}

// AssertWithinDuration 断言时间在指定范围内
func AssertWithinDuration(t *testing.T, expected, actual time.Time, delta time.Duration, msg string) {
	t.Helper()
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	if diff > delta {
		t.Fatalf("%s: 时间差 %v 超过允许范围 %v", msg, diff, delta)
	}
}

// RunConcurrentTest 运行并发测试
func RunConcurrentTest(t *testing.T, concurrency int, testFunc func(int) error) {
	t.Helper()
	done := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			done <- testFunc(index)
		}(i)
	}

	for i := 0; i < concurrency; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("并发测试失败: %v", err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("并发测试超时")
		}
	}
}

// WaitForCondition 等待条件满足
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timeoutCh:
			t.Fatalf("%s: 等待条件超时", msg)
		}
	}
}

// CreateTempFile 创建临时文件
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "qlink-test-*")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}

	if content != "" {
		_, err = tmpFile.WriteString(content)
		if err != nil {
			t.Fatalf("写入临时文件失败: %v", err)
		}
	}

	tmpFile.Close()
	return tmpFile.Name()
}

// CleanupTempFile 清理临时文件
func CleanupTempFile(t *testing.T, filepath string) {
	t.Helper()
	if err := os.Remove(filepath); err != nil {
		t.Logf("清理临时文件失败: %v", err)
	}
}
