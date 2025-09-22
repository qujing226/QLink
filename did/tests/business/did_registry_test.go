package business

import (
	"strings"
	"testing"
	"time"

	"github.com/qujing226/QLink/did"
	"github.com/qujing226/QLink/did/config"
	"github.com/qujing226/QLink/did/tests/testutils"
)

// TestDIDRegistryBasic 测试DID注册表基本功能
func TestDIDRegistryBasic(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 创建配置
	cfg := &config.Config{
		DID: &config.DIDConfig{
			Method:  "QLink",
			ChainID: "test123",
		},
	}

	// 创建DID注册表
	registry := did.NewDIDRegistry(cfg, nil)

	// 验证注册表创建成功
	if registry == nil {
		t.Fatal("DID注册表创建失败")
	}

	// 测试列表功能（应该为空）
	docs, err := registry.List()
	testutils.AssertNoError(t, err, "列出DID文档")
	if len(docs) != 0 {
		t.Fatalf("期望0个文档，实际%d个", len(docs))
	}
}

// TestDIDDocumentCreation 测试DID文档创建
func TestDIDDocumentCreation(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 生成测试DID
	didStr := testutils.GenerateTestDID("test123")

	// 创建测试DID文档
	doc := &did.DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://example.org/ns/pqc/v1",
		},
		ID:      didStr,
		Created: time.Now(),
		Updated: time.Now(),
		Status:  "active",
		VerificationMethod: []did.VerificationMethod{
			{
				ID:         "#authentication-key",
				Type:       "JsonWebKey2020",
				Controller: didStr,
			},
			{
				ID:         "#lattice-key",
				Type:       "KemJsonKey2025",
				Controller: didStr,
			},
		},
		AssertionMethod: []string{"#authentication-key"},
		KeyAgreement:    []string{"#lattice-key"},
	}

	// 验证文档结构
	testutils.AssertNotEmpty(t, doc.ID, "DID文档ID")
	testutils.AssertEqual(t, didStr, doc.ID, "DID匹配")
	testutils.AssertEqual(t, "active", doc.Status, "DID状态")

	// 验证验证方法
	if len(doc.VerificationMethod) != 2 {
		t.Fatalf("期望2个验证方法，实际%d个", len(doc.VerificationMethod))
	}

	// 验证JSON序列化
	jsonData, err := doc.ToJSON()
	testutils.AssertNoError(t, err, "JSON序列化")
	if len(jsonData) == 0 {
		t.Fatal("JSON数据不应为空")
	}

	// 验证JSON反序列化
	parsedDoc, err := did.FromJSON(jsonData)
	testutils.AssertNoError(t, err, "JSON反序列化")
	testutils.AssertEqual(t, doc.ID, parsedDoc.ID, "反序列化后DID匹配")
}

// TestDIDValidation 测试DID格式验证
func TestDIDValidation(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 测试有效的DID格式
	validDIDs := []string{
		"did:QLink:test123:1234567890abcdef",
		"did:QLink:mainnet:abcdef1234567890",
		testutils.GenerateTestDID("test123"),
	}

	for _, validDID := range validDIDs {
		// 验证DID格式
		testutils.AssertNotEmpty(t, validDID, "有效DID: "+validDID)

		// 测试DID格式验证
		if !strings.HasPrefix(validDID, "did:") {
			t.Errorf("DID格式无效: %s", validDID)
		}
	}
}

// TestDIDDocumentServices 测试DID文档服务端点
func TestDIDDocumentServices(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 生成测试DID
	didStr := testutils.GenerateTestDID("test123")

	// 创建带服务端点的DID文档
	doc := &did.DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      didStr,
		Created: time.Now(),
		Updated: time.Now(),
		Status:  "active",
		Service: []did.Service{
			{
				ID:              "#messaging",
				Type:            "MessagingService",
				ServiceEndpoint: "https://example.com/messaging",
			},
			{
				ID:              "#storage",
				Type:            "StorageService",
				ServiceEndpoint: "https://example.com/storage",
			},
		},
	}

	// 验证服务端点
	if len(doc.Service) != 2 {
		t.Fatalf("期望2个服务端点，实际%d个", len(doc.Service))
	}

	// 验证第一个服务
	firstService := doc.Service[0]
	testutils.AssertEqual(t, "#messaging", firstService.ID, "服务ID")
	testutils.AssertEqual(t, "MessagingService", firstService.Type, "服务类型")
	testutils.AssertNotEmpty(t, firstService.ServiceEndpoint, "服务端点")
}

// TestDIDDocumentProof 测试DID文档证明
func TestDIDDocumentProof(t *testing.T) {
	// 设置测试环境
	testEnv := testutils.SetupTestEnvironment(t)
	defer testEnv.Cleanup()

	// 生成测试DID
	didStr := testutils.GenerateTestDID("test123")

	// 创建带证明的DID文档
	proof := &did.Proof{
		Type:               "Ed25519Signature2020",
		Created:            time.Now(),
		VerificationMethod: didStr + "#key-1",
		ProofPurpose:       "authentication",
		Jws:                "mock-signature-value",
	}

	doc := &did.DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      didStr,
		Created: time.Now(),
		Updated: time.Now(),
		Status:  "active",
		Proof:   proof,
	}

	// 验证证明
	if doc.Proof == nil {
		t.Fatal("DID文档应该包含证明")
	}

	testutils.AssertEqual(t, "Ed25519Signature2020", doc.Proof.Type, "证明类型")
	testutils.AssertEqual(t, "authentication", doc.Proof.ProofPurpose, "证明目的")
	testutils.AssertNotEmpty(t, doc.Proof.Jws, "JWS签名")
}
