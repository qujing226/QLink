package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/qujing226/QLink/did/crypto"
)

// Client QLink客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	keyPair    *crypto.HybridKeyPair
}

// NewClient 创建新的客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithKeyPair 使用密钥对创建客户端
func NewClientWithKeyPair(baseURL string, keyPair *crypto.HybridKeyPair) *Client {
	c := NewClient(baseURL)
	c.keyPair = keyPair
	return c
}

// GenerateKeyPair 生成新的密钥对
func (c *Client) GenerateKeyPair() error {
	keyPair, err := crypto.GenerateHybridKeyPair()
	if err != nil {
		return fmt.Errorf("生成密钥对失败: %w", err)
	}
	c.keyPair = keyPair
	return nil
}

// GetKeyPair 获取当前密钥对
func (c *Client) GetKeyPair() *crypto.HybridKeyPair {
	return c.keyPair
}

// RegisterDIDRequest 注册DID请求
type RegisterDIDRequest struct {
	DID       string                 `json:"did"`
	Document  map[string]interface{} `json:"document"`
	Signature string                 `json:"signature"`
}

// RegisterDIDResponse 注册DID响应
type RegisterDIDResponse struct {
	Message  string      `json:"message"`
	DID      string      `json:"did"`
	Document interface{} `json:"document"`
}

// RegisterDID 注册DID
func (c *Client) RegisterDID(did string, document map[string]interface{}) (*RegisterDIDResponse, error) {
	if c.keyPair == nil {
		return nil, fmt.Errorf("密钥对未初始化")
	}

	// 序列化文档用于签名
	docBytes, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("序列化文档失败: %w", err)
	}

	// 签名
	signature, err := c.keyPair.Sign(docBytes)
	if err != nil {
		return nil, fmt.Errorf("签名失败: %w", err)
	}

	// 构造请求
	req := RegisterDIDRequest{
		DID:       did,
		Document:  document,
		Signature: fmt.Sprintf("%x", signature),
	}

	// 发送HTTP请求
	var resp RegisterDIDResponse
	err = c.post("/api/v1/did/register", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("注册DID失败: %w", err)
	}

	return &resp, nil
}

// ResolveDIDResponse 解析DID响应
type ResolveDIDResponse struct {
	DIDDocument           interface{} `json:"did_document"`
	DIDResolutionMetadata interface{} `json:"did_resolution_metadata"`
	DIDDocumentMetadata   interface{} `json:"did_document_metadata"`
}

// ResolveDID 解析DID
func (c *Client) ResolveDID(did string) (*ResolveDIDResponse, error) {
	// 提取DID ID部分
	didID := did
	if len(did) > 10 && did[:10] == "did:qlink:" {
		didID = did[10:]
	}

	var resp ResolveDIDResponse
	err := c.get(fmt.Sprintf("/api/v1/did/%s", didID), &resp)
	if err != nil {
		return nil, fmt.Errorf("解析DID失败: %w", err)
	}

	return &resp, nil
}

// UpdateDIDRequest 更新DID请求
type UpdateDIDRequest struct {
	Document  map[string]interface{} `json:"document"`
	Signature string                 `json:"signature"`
}

// UpdateDIDResponse 更新DID响应
type UpdateDIDResponse struct {
	Message  string      `json:"message"`
	DID      string      `json:"did"`
	Document interface{} `json:"document"`
}

// UpdateDID 更新DID
func (c *Client) UpdateDID(did string, document map[string]interface{}) (*UpdateDIDResponse, error) {
	if c.keyPair == nil {
		return nil, fmt.Errorf("密钥对未初始化")
	}

	// 序列化文档用于签名
	docBytes, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("序列化文档失败: %w", err)
	}

	// 签名
	signature, err := c.keyPair.Sign(docBytes)
	if err != nil {
		return nil, fmt.Errorf("签名失败: %w", err)
	}

	// 提取DID ID部分
	didID := did
	if len(did) > 10 && did[:10] == "did:qlink:" {
		didID = did[10:]
	}

	// 构造请求
	req := UpdateDIDRequest{
		Document:  document,
		Signature: fmt.Sprintf("%x", signature),
	}

	// 发送HTTP请求
	var resp UpdateDIDResponse
	err = c.put(fmt.Sprintf("/api/v1/did/%s", didID), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("更新DID失败: %w", err)
	}

	return &resp, nil
}

// RevokeDIDRequest 撤销DID请求
type RevokeDIDRequest struct {
	Signature string `json:"signature"`
	Reason    string `json:"reason,omitempty"`
}

// RevokeDIDResponse 撤销DID响应
type RevokeDIDResponse struct {
	Message string `json:"message"`
	DID     string `json:"did"`
	Reason  string `json:"reason"`
}

// RevokeDID 撤销DID
func (c *Client) RevokeDID(did string, reason string) (*RevokeDIDResponse, error) {
	if c.keyPair == nil {
		return nil, fmt.Errorf("密钥对未初始化")
	}

	// 签名撤销请求
	signData := []byte(fmt.Sprintf("revoke:%s:%s", did, reason))
	signature, err := c.keyPair.Sign(signData)
	if err != nil {
		return nil, fmt.Errorf("签名失败: %w", err)
	}

	// 提取DID ID部分
	didID := did
	if len(did) > 10 && did[:10] == "did:qlink:" {
		didID = did[10:]
	}

	// 构造请求
	req := RevokeDIDRequest{
		Signature: fmt.Sprintf("%x", signature),
		Reason:    reason,
	}

	// 发送HTTP请求
	var resp RevokeDIDResponse
	err = c.delete(fmt.Sprintf("/api/v1/did/%s", didID), req, &resp)
	if err != nil {
		return nil, fmt.Errorf("撤销DID失败: %w", err)
	}

	return &resp, nil
}

// GenerateDIDResponse 生成DID响应
type GenerateDIDResponse struct {
	DID     string `json:"did"`
	Message string `json:"message"`
}

// GenerateDID 生成新的DID
func (c *Client) GenerateDID() (*GenerateDIDResponse, error) {
	var resp GenerateDIDResponse
	err := c.post("/api/v1/did/generate", nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("生成DID失败: %w", err)
	}

	return &resp, nil
}

// GetDIDDocument 获取DID文档
func (c *Client) GetDIDDocument(did string) (interface{}, error) {
	// 提取DID ID部分
	didID := did
	if len(did) > 10 && did[:10] == "did:qlink:" {
		didID = did[10:]
	}

	var resp struct {
		Context            []string      `json:"@context"`
		ID                 string        `json:"id"`
		VerificationMethod []interface{} `json:"verificationMethod,omitempty"`
		Authentication     []string      `json:"authentication,omitempty"`
		AssertionMethod    []string      `json:"assertionMethod,omitempty"`
		KeyAgreement       []string      `json:"keyAgreement,omitempty"`
		Service            []interface{} `json:"service,omitempty"`
		Created            string        `json:"created"`
		Updated            string        `json:"updated"`
		Proof              interface{}   `json:"proof,omitempty"`
		Status             string        `json:"status"`
	}
	err := c.get(fmt.Sprintf("/api/v1/did/%s/document", didID), &resp)
	if err != nil {
		return nil, fmt.Errorf("获取DID文档失败: %w", err)
	}

	return &resp, nil
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Version   string `json:"version"`
}

// Health 健康检查
func (c *Client) Health() (*HealthResponse, error) {
	var resp HealthResponse
	err := c.get("/api/v1/health", &resp)
	if err != nil {
		return nil, fmt.Errorf("健康检查失败: %w", err)
	}

	return &resp, nil
}

// NodeInfoResponse 节点信息响应
type NodeInfoResponse struct {
	NodeID    string   `json:"node_id"`
	Version   string   `json:"version"`
	ChainID   string   `json:"chain_id"`
	BlockHash string   `json:"block_hash"`
	Peers     []string `json:"peers"`
}

// GetNodeInfo 获取节点信息
func (c *Client) GetNodeInfo() (*NodeInfoResponse, error) {
	var resp NodeInfoResponse
	err := c.get("/api/v1/node/info", &resp)
	if err != nil {
		return nil, fmt.Errorf("获取节点信息失败: %w", err)
	}

	return &resp, nil
}

// HTTP辅助方法

// get 发送GET请求
func (c *Client) get(path string, result interface{}) error {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// post 发送POST请求
func (c *Client) post(path string, data interface{}, result interface{}) error {
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(jsonData)
	}

	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// put 发送PUT请求
func (c *Client) put(path string, data interface{}, result interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// delete 发送DELETE请求
func (c *Client) delete(path string, data interface{}, result interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// SetTimeout 设置HTTP超时
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// SetBaseURL 设置基础URL
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}
