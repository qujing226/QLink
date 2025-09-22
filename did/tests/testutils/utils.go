package testutils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/blockchain"
	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/crypto"
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

// GenerateTestDID 生成测试用DID
func GenerateTestDID(chainID string) string {
	buf := make([]byte, 16)
	rand.Read(buf)
	uniqueID := hex.EncodeToString(buf)
	return fmt.Sprintf("did:QLink:%s:%s", chainID, uniqueID)
}

// GenerateTestKeyPair 生成测试密钥对
func GenerateTestKeyPair(t *testing.T) (*crypto.HybridKeyPair, error) {
	return crypto.GenerateHybridKeyPair()
}

// CreateTestDIDDocument 创建测试DID文档
func CreateTestDIDDocument(t *testing.T, didStr string, keyPair *crypto.HybridKeyPair) *did.DIDDocument {
	doc := &did.DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://example.org/ns/pqc/v1",
		},
		ID:      didStr,
		Created: time.Now(),
		Updated: time.Now(),
		Status:  "active",
	}

	// 添加P-256密钥
	p256Method := did.VerificationMethod{
		ID:         "#authentication-key",
		Type:       "JsonWebKey2020",
		Controller: didStr,
	}

	// 添加Kyber768密钥
	kyberMethod := did.VerificationMethod{
		ID:         "#lattice-key",
		Type:       "KemJsonKey2025",
		Controller: didStr,
	}

	doc.VerificationMethod = []did.VerificationMethod{p256Method, kyberMethod}
	doc.AssertionMethod = []string{"#authentication-key"}
	doc.KeyAgreement = []string{"#lattice-key"}

	return doc
}

// SetupTestStorage 设置测试存储
func SetupTestStorage(t *testing.T, tempDir string) blockchain.Storage {
	cfg := &config.Config{
		Storage: &config.StorageConfig{
			Local: &config.LocalStorageConfig{
				Path: filepath.Join(tempDir, "storage"),
			},
		},
	}

	storage, err := blockchain.NewLocalStorage(cfg)
	if err != nil {
		t.Fatalf("创建测试存储失败: %v", err)
	}

	return storage
}

// AssertNoError 断言无错误
func AssertNoError(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError 断言有错误
func AssertError(t *testing.T, err error, msg string) {
	if err == nil {
		t.Fatalf("%s: 期望有错误但没有", msg)
	}
}

// AssertEqual 断言相等
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	if expected != actual {
		t.Fatalf("%s: 期望 %v, 实际 %v", msg, expected, actual)
	}
}

// AssertNotEmpty 断言非空
func AssertNotEmpty(t *testing.T, value string, msg string) {
	if value == "" {
		t.Fatalf("%s: 值不应为空", msg)
	}
}
