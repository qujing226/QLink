package service

import (
    "bytes"
    "crypto/hmac"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "math/big"
    "net/http"
    "qlink-im/internal/config"
    "qlink-im/internal/errors"
    "qlink-im/internal/models"
    "qlink-im/internal/storage"
    "strings"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

type AuthService interface {
    Login(req *models.LoginRequest) (*models.LoginResponse, error)
    ValidateToken(tokenString string) (*jwt.Token, error)
    GetUserFromToken(token *jwt.Token) (*models.User, error)
    CreateChallenge(from, to string) (*models.Challenge, error)
    VerifyChallenge(challengeID uint, signature string) error
    VerifyChallengeByNonce(nonce, signature string) error
    GetChallenge(did string) (*BlockchainChallengeResponse, error)
    VerifyWithBlockchain(did, signature, challengeID string) (*BlockchainLoginResponse, error)
    GetLatticePublicKey(did string) (string, error)
}

type authService struct {
	storage       storage.Storage
	didConfig     config.DIDConfig
	jwtSecret     []byte
	blockchainURL string
}

type Claims struct {
	DID string `json:"did"`
	jwt.RegisteredClaims
}

// 区块链系统的请求和响应结构
type BlockchainChallengeRequest struct {
	DID string `json:"did"`
}

type BlockchainChallengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	Challenge   string    `json:"challenge"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type BlockchainLoginRequest struct {
	DID         string `json:"did"`
	Signature   string `json:"signature"`
	ChallengeID string `json:"challenge_id"`
}

type BlockchainLoginResponse struct {
	Token     string    `json:"token"`
	DID       string    `json:"did"`
	LoginTime time.Time `json:"login_time"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewAuthService(storage storage.Storage, didConfig config.DIDConfig, securityConfig config.SecurityConfig) AuthService {
    return &authService{
        storage:       storage,
        didConfig:     didConfig,
        jwtSecret:     []byte(securityConfig.JWTSecret),
        blockchainURL: didConfig.NodeURL,
    }
}

// Login 用户登录
func (a *authService) Login(req *models.LoginRequest) (*models.LoginResponse, error) {
	// 验证DID格式
	if !a.validateDIDFormat(req.DID) {
		return nil, errors.New(errors.ErrInvalidDID, "Invalid DID format")
	}

	// 从区块链解析DID Document
	didDoc, err := a.resolveDIDDocument(req.DID)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternalServer, "Failed to resolve DID document: %v", err)
	}

	// 检查用户是否已存在
	user, err := a.storage.GetUserByDID(req.DID)
	if err != nil {
		// 用户不存在，创建新用户
		user = &models.User{
			DID:       req.DID,
			PublicKey: didDoc.PublicKey,
			Status:    "online",
		}
		if err := a.storage.CreateUser(user); err != nil {
			return nil, errors.WrapDatabaseError(err, "create user")
		}
	} else {
		// 更新用户状态为在线
		user.Status = "online"
		if err := a.storage.UpdateUser(user); err != nil {
			return nil, errors.WrapDatabaseError(err, "update user status")
		}
	}

	// 生成JWT token
	expiresAt := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		DID: req.DID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return nil, errors.Newf(errors.ErrInternalServer, "Failed to generate token: %v", err)
	}

	return &models.LoginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Unix(),
		User:      *user,
	}, nil
}

// ValidateToken 验证JWT token
func (a *authService) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

// GetUserFromToken 从token中获取用户信息
func (a *authService) GetUserFromToken(token *jwt.Token) (*models.User, error) {
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return a.storage.GetUserByDID(claims.DID)
}

// CreateChallenge 创建认证质询
func (a *authService) CreateChallenge(from, to string) (*models.Challenge, error) {
	// 生成随机nonce
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	challenge := &models.Challenge{
		From:      from,
		To:        to,
		Nonce:     hex.EncodeToString(nonce),
		Status:    "pending",
		ExpiresAt: time.Now().Add(30 * time.Minute), // 延长到30分钟过期
	}

	if err := a.storage.CreateChallenge(challenge); err != nil {
		return nil, fmt.Errorf("failed to create challenge: %w", err)
	}

	return challenge, nil
}

// VerifyChallenge 验证质询签名
func (a *authService) VerifyChallenge(challengeID uint, signature string) error {
	challenge, err := a.storage.GetChallenge(challengeID)
	if err != nil {
		return fmt.Errorf("challenge not found: %w", err)
	}

	if challenge.Status != "pending" {
		return fmt.Errorf("challenge already processed")
	}

	if time.Now().After(challenge.ExpiresAt) {
		return fmt.Errorf("challenge expired")
	}

    // 检查用户是否存在（不使用返回值，仅用于校验存在性）
    _, err = a.storage.GetUserByDID(challenge.From)
    if err != nil {
        return fmt.Errorf("user not found: %w", err)
    }

    // 使用与网关一致的 HMAC-SHA256 校验方案
    fmt.Printf("DEBUG: Verifying signature - nonce: %s, signature: %s, did: %s\n", challenge.Nonce, signature, challenge.From)
    if !a.verifySignature(challenge.Nonce, signature, challenge.From) {
        challenge.Status = "failed"
        a.storage.UpdateChallenge(challenge)
        fmt.Printf("DEBUG: Signature verification failed\n")
        return fmt.Errorf("invalid signature")
    }
	fmt.Printf("DEBUG: Signature verification successful\n")

	// 更新质询状态
	challenge.Status = "completed"
	challenge.Signature = signature
	if err := a.storage.UpdateChallenge(challenge); err != nil {
		return fmt.Errorf("failed to update challenge: %w", err)
	}

	return nil
}

// VerifyChallengeByNonce 根据nonce验证质询签名
func (a *authService) VerifyChallengeByNonce(nonce, signature string) error {
	challenge, err := a.storage.GetChallengeByNonce(nonce)
	if err != nil {
		return fmt.Errorf("challenge not found: %w", err)
	}

	if challenge.Status != "pending" {
		return fmt.Errorf("challenge already processed")
	}

	if time.Now().After(challenge.ExpiresAt) {
		return fmt.Errorf("challenge expired")
	}

    // 获取用户，如果用户不存在则创建用户
    _, err = a.storage.GetUserByDID(challenge.From)
    if err != nil {
        // 如果用户不存在，创建新用户
        if err.Error() == "record not found" {
            // 解析DID Document获取公钥
            didDoc, err := a.resolveDIDDocument(challenge.From)
            if err != nil {
                return fmt.Errorf("failed to parse DID document: %w", err)
            }

            // 创建新用户
            newUser := &models.User{
                DID:       challenge.From,
                PublicKey: didDoc.PublicKey,
                Status:    "online",
            }

            if err := a.storage.CreateUser(newUser); err != nil {
                return fmt.Errorf("failed to create user: %w", err)
            }
        } else {
            return fmt.Errorf("user not found: %w", err)
        }
    }

    // 使用与网关一致的 HMAC-SHA256 校验方案
    if !a.verifySignature(challenge.Nonce, signature, challenge.From) {
        challenge.Status = "failed"
        a.storage.UpdateChallenge(challenge)
        return fmt.Errorf("invalid signature")
    }

	// 更新质询状态
	challenge.Status = "completed"
	challenge.Signature = signature
	if err := a.storage.UpdateChallenge(challenge); err != nil {
		return fmt.Errorf("failed to update challenge: %w", err)
	}

	return nil
}

// GetChallenge 从区块链系统获取质询
func (a *authService) GetChallenge(did string) (*BlockchainChallengeResponse, error) {
	// 验证DID格式
	if !a.validateDIDFormat(did) {
		return nil, errors.New(errors.ErrInvalidDID, "invalid DID format")
	}

	// 向区块链系统请求质询
	reqBody := BlockchainChallengeRequest{DID: did}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(a.blockchainURL+"/api/v1/auth/challenge", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to request challenge from blockchain: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("blockchain returned error: %s", string(body))
	}

	var challengeResp BlockchainChallengeResponse
	if err := json.NewDecoder(resp.Body).Decode(&challengeResp); err != nil {
		return nil, fmt.Errorf("failed to decode challenge response: %v", err)
	}

	return &challengeResp, nil
}

// VerifyWithBlockchain 向区块链系统验证签名
func (a *authService) VerifyWithBlockchain(did, signature, challengeID string) (*BlockchainLoginResponse, error) {
	// 向区块链系统发送登录请求
	reqBody := BlockchainLoginRequest{
		DID:         did,
		Signature:   signature,
		ChallengeID: challengeID,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %v", err)
	}

	resp, err := http.Post(a.blockchainURL+"/api/v1/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to verify with blockchain: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("blockchain verification failed: %s", string(body))
	}

	var loginResp BlockchainLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %v", err)
	}

	return &loginResp, nil
}
func (a *authService) validateDIDFormat(did string) bool {
	// 验证DID格式：did:qlink:<identifier>
	if !strings.HasPrefix(did, "did:qlink:") {
		return false
	}
	
	// 检查长度
	if len(did) < 15 || len(did) > 256 {
		return false
	}
	
	// 提取标识符部分
	parts := strings.Split(did, ":")
	if len(parts) != 3 {
		return false
	}
	
	identifier := parts[2]
	// 标识符不能为空
	if len(identifier) == 0 {
		return false
	}
	
	return true
}

// validateDIDPrivateKeyPair 验证DID和私钥是否匹配
func (a *authService) validateDIDPrivateKeyPair(did, privateKey string) bool {
	// 生成预期的DID
	expectedDID := a.generateDIDFromPrivateKey(privateKey)
	return did == expectedDID
}

// generateDIDFromPrivateKey 从私钥生成DID
func (a *authService) generateDIDFromPrivateKey(privateKey string) string {
	// 使用SHA256哈希私钥来生成唯一标识符
	hash := sha256.Sum256([]byte(privateKey))
	identifier := hex.EncodeToString(hash[:])[:32] // 取前32个字符
	return fmt.Sprintf("did:qlink:%s", identifier)
}

// DIDDocument 简化的DID文档结构
type DIDDocument struct {
    ID        string `json:"id"`
    PublicKey string `json:"publicKey"`
}

// resolveDIDDocument 从区块链解析DID文档（简化实现）
func (a *authService) resolveDIDDocument(did string) (*DIDDocument, error) {
    // 验证DID格式
    if !a.validateDIDFormat(did) {
        return nil, fmt.Errorf("invalid DID format")
    }

    // 提取不带前缀的DID标识符
    id := strings.TrimPrefix(did, a.didConfig.Prefix)
    if id == did { // 未带前缀时尝试按冒号分割
        parts := strings.Split(did, ":")
        if len(parts) == 3 {
            id = parts[2]
        }
    }

    // 调用区块链DID网关解析接口
    url := fmt.Sprintf("%s/api/v1/did/resolve/%s", a.blockchainURL, id)
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve DID via gateway: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("gateway returned error: %s", string(body))
    }

    var resolveResp struct {
        DIDDocument map[string]interface{} `json:"did_document"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&resolveResp); err != nil {
        return nil, fmt.Errorf("failed to decode resolve response: %w")
    }

    // 提取可用于ECDSA验证的公钥
    pubKey, err := a.extractPublicKeyFromDoc(resolveResp.DIDDocument)
    if err != nil || pubKey == "" {
        // 兜底：基于DID派生一个确定性公钥（兼容旧逻辑）
        derived := fmt.Sprintf("%x", []byte(id+"public"))
        if len(derived) > 32 {
            pubKey = derived[:32]
        } else {
            pubKey = derived
        }
    }

    return &DIDDocument{
        ID:        did,
        PublicKey: pubKey,
    }, nil
}

// fetchRawDIDDocument 直接获取区块链DID文档的原始结构
func (a *authService) fetchRawDIDDocument(did string) (map[string]interface{}, error) {
    // 验证DID格式
    if !a.validateDIDFormat(did) {
        return nil, fmt.Errorf("invalid DID format")
    }

    // 提取不带前缀的DID标识符
    id := strings.TrimPrefix(did, a.didConfig.Prefix)
    if id == did {
        parts := strings.Split(did, ":")
        if len(parts) == 3 {
            id = parts[2]
        }
    }

    url := fmt.Sprintf("%s/api/v1/did/resolve/%s", a.blockchainURL, id)
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve DID via gateway: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("gateway returned error: %s", string(body))
    }

    var resolveResp struct {
        DIDDocument map[string]interface{} `json:"did_document"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&resolveResp); err != nil {
        return nil, fmt.Errorf("failed to decode resolve response: %w", err)
    }
    return resolveResp.DIDDocument, nil
}

// extractLatticePublicKeyFromDoc 从DID文档提取格加密公钥（当前版本不再使用，返回空）
func (a *authService) extractLatticePublicKeyFromDoc(doc map[string]interface{}) (string, error) {
    return "", fmt.Errorf("lattice public key not used in current version")
}

// GetLatticePublicKey 获取DID的格加密公钥（占位实现）
func (a *authService) GetLatticePublicKey(did string) (string, error) {
    // 当前版本不提供格加密公钥，返回占位符（基于 DID 派生的哈希）
    seed := did + ":lattice"
    sum := sha256.Sum256([]byte(seed))
    return hex.EncodeToString(sum[:]), nil
}

// extractPublicKeyFromDoc 从DID文档提取ECDSA公钥（十六进制未压缩格式）
func (a *authService) extractPublicKeyFromDoc(doc map[string]interface{}) (string, error) {
    if doc == nil {
        return "", fmt.Errorf("empty DID document")
    }

    // 直接字段
    if pk, ok := doc["publicKey"].(string); ok && pk != "" {
        return pk, nil
    }

    // verificationMethod 数组
    if vmData, ok := doc["verificationMethod"]; ok {
        if vmArr, ok := vmData.([]interface{}); ok {
            for _, v := range vmArr {
                if vm, ok := v.(map[string]interface{}); ok {
                    // JWK -> 组装未压缩04||x||y 并返回hex
                    if jwkAny, ok := vm["publicKeyJwk"]; ok {
                        if jwk, ok := jwkAny.(map[string]interface{}); ok {
                            xStr, _ := jwk["x"].(string)
                            yStr, _ := jwk["y"].(string)
                            if xStr != "" && yStr != "" {
                                xBytes, errX := base64.RawURLEncoding.DecodeString(xStr)
                                yBytes, errY := base64.RawURLEncoding.DecodeString(yStr)
                                if errX == nil && errY == nil && len(xBytes) == 32 && len(yBytes) == 32 {
                                    uncompressed := append([]byte{0x04}, append(xBytes, yBytes...)...)
                                    return hex.EncodeToString(uncompressed), nil
                                }
                            }
                        }
                    }

                    // publicKeyHex 已有十六进制公钥
                    if pkHex, ok := vm["publicKeyHex"].(string); ok && pkHex != "" {
                        return pkHex, nil
                    }

                    // publicKeyMultibase（可选）：尝试作为base64解码再hex（若是base58则跳过）
                    if pkm, ok := vm["publicKeyMultibase"].(string); ok && pkm != "" {
                        // 常见multibase前缀 'm'（base64）或 'z'（base58），这里只处理base64路径
                        if len(pkm) > 1 && (pkm[0] == 'm' || pkm[0] == 'M') {
                            decoded, err := base64.StdEncoding.DecodeString(pkm[1:])
                            if err == nil && len(decoded) > 0 {
                                return hex.EncodeToString(decoded), nil
                            }
                        }
                    }
                }
            }
        }
    }

    return "", fmt.Errorf("no suitable public key found")
}

// verifySignature 使用 HMAC-SHA256 验证签名（与网关保持一致）
func (a *authService) verifySignature(nonce, signature, did string) bool {
    key := a.generatePublicKeyFromDIDFallback(did)
    h := hmac.New(sha256.New, []byte(key))
    h.Write([]byte(nonce))
    expected := hex.EncodeToString(h.Sum(nil))
    return strings.EqualFold(signature, expected)
}

// parseECDSAPublicKey 解析ECDSA公钥
func (a *authService) parseECDSAPublicKey(publicKeyStr string) (*ecdsa.PublicKey, error) {
	// 移除可能的前缀
	publicKeyStr = strings.TrimPrefix(publicKeyStr, "0x")
	
	// 解码十六进制公钥
	pubKeyBytes, err := hex.DecodeString(publicKeyStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex public key: %w", err)
	}
	
	// 如果是未压缩格式（65字节，以0x04开头）
	if len(pubKeyBytes) == 65 && pubKeyBytes[0] == 0x04 {
		x := new(big.Int).SetBytes(pubKeyBytes[1:33])
		y := new(big.Int).SetBytes(pubKeyBytes[33:65])
		
		return &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}, nil
	}
	
	// 如果是压缩格式（33字节）
	if len(pubKeyBytes) == 33 {
		x, y := elliptic.UnmarshalCompressed(elliptic.P256(), pubKeyBytes)
		if x == nil {
			return nil, fmt.Errorf("invalid compressed public key")
		}
		
		return &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}, nil
	}
	
	// 如果是坐标格式（64字节，x和y各32字节）
	if len(pubKeyBytes) == 64 {
		x := new(big.Int).SetBytes(pubKeyBytes[:32])
		y := new(big.Int).SetBytes(pubKeyBytes[32:])
		
		return &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     x,
			Y:     y,
		}, nil
	}
	
    return nil, fmt.Errorf("unsupported public key format, length: %d", len(pubKeyBytes))
}

// generatePublicKeyFromDIDFallback 基于 DID 派生用于 HMAC 的密钥（与网关相同逻辑）
func (a *authService) generatePublicKeyFromDIDFallback(did string) string {
    parts := strings.Split(did, ":")
    if len(parts) >= 3 {
        identifier := parts[2]
        if len(identifier) >= 32 {
            return identifier[:32]
        }
        return identifier + "default-private-key"[:32-len(identifier)]
    }
    return "default-private-key"
}