DID‑QLink + 区块链系统 PRD
1. 项目背景与目标
- 背景
  现有 DID 方法多基于传统公钥体系，缺乏对抗量子安全的原生支持，且在链上存储与解析效率、密钥管理等方面存在不足。
- 目标
  设计并实现 DID 方法 did:QLink，原生支持 Kyber768 抗量子密钥，与传统密钥混合使用；结合区块链系统实现 DID 注册、解析、认证与通信加密的一体化方案。

2. 系统总体架构
   [客户端] <—TLS/PQC握手—> [通信服务端]
   |                          |
   |——调用解析API——> [DID Resolver]
   |
   |——链上查询——> [区块链节点]
   |<——链上返回哈希+元数据
   |
   |——链下获取——> [DID Document存储(IPFS/DB)]


- 链上部分：DID Registry 智能合约
- 存储 DID → Document 哈希、版本号、更新时间、撤销标记
- 提供注册、更新、撤销、解析接口
- 链下部分：DID Document 存储
- 存放完整 JSON-LD 文档（含公钥信息）
- 可用 IPFS 或分布式数据库
- 解析器：
- 混合解析（链上取哈希 → 链下取文档）
- 提供 HTTP API（符合 W3C DID Resolution v1.0）

3. DID 方法定义（did:QLink）
- 标识符格式
  did:QLink:<chain-id>:<unique-id>
- 解析规则
- 从链上 DID Registry 获取 Document 哈希
- 从链下存储获取完整 Document
- 校验哈希一致性

4. DID Document 结构（
- 核心字段
- @context：DID v1 / v1.1
- id：DID 唯一标识符
- assertionMethod：可用于签名断言的密钥引用
- verificationMethod：
- P‑256 公钥（传统签名/认证）
- Kyber768 公钥（KEM 会话密钥协商）
- 示例
```json
{
  "@context": [
    "https://www.w3.org/ns/did/v1",
    "https://example.org/ns/pqc/v1"
  ],
  "id": "did:QLink:chain123:4LxQvAd4QGvQsLWL3iK6LdN9Xzvk",
  "updated": "2025-09-10T09:00:00Z",
  "verificationMethod": [
    {
      "id": "#authentication-key",
      "type": "JsonWebKey2020",
      "controller": "did:QLink:chain123:4LxQvAd4QGvQsLWL3iK6LdN9Xzvk",
      "usage": ["authentication", "assertionMethod"],
      "publicKeyJwk": {
        "crv": "P-256",
        "kty": "EC",
        "x": "...",
        "y": "..."
      }
    },
    {
      "id": "#lattice-key",
      "type": "KemJsonKey2025",
      "controller": "did:QLink:chain123:4LxQvAd4QGvQsLWL3iK6LdN9Xzvk",
      "usage": ["keyAgreement"],
      "publicKeyJwk": {
        "crv": "Kyber768",
        "kty": "KYBER",
        "x": "..."
      },
      "publicKeyMultibase": "zBase58OrBase64EncodedKey"
    }
  ],
  "assertionMethod": ["#authentication-key"],
  "keyAgreement": ["#lattice-key"]
}
```
  你提供的 JSON 基本符合 W3C DID Core，只需将 did:easyblock 改为 did:QLink 并补充链 ID。

5. 核心功能模块
   5.1 DID 注册
- 客户端生成 DID Document（含 P‑256 + Kyber768 公钥）
- 计算 Document 哈希
- 调用链上合约 registerDID(did, hash, metadata)
- 将完整 Document 上传至链下存储（IPFS/DB）
  5.2 DID 更新
- 生成新 Document（密钥轮换或端点更新）
- 更新链上哈希与链下存储内容
- 保留版本历史
  5.3 DID 撤销
- 链上标记撤销状态
- 链下保留历史版本供审计
  5.4 DID 解析
- 输入 DID → 链上取哈希 → 链下取 Document → 校验哈希 → 返回 JSON-LD
  5.5 三方质询认证
- 参与方：Client、Server、链上 Verifier 合约
- 流程：
- Client 发起认证请求（含 DID、nonce）
- Server 调用解析器获取公钥
- Server 将挑战提交链上 Verifier 合约
- 合约验证签名/密钥协商结果
- 返回认证结果，建立加密通道
  5.6 通信加密
- 使用 Kyber768 KEM 协商会话密钥
- 会话密钥用于对称加密（AES‑GCM）

6. 安全设计
- 抗量子安全：Kyber768 用于密钥协商，防御量子计算攻击
- 防篡改：链上存储 Document 哈希，链下内容不可单独修改
- 防重放：认证过程使用 nonce + 时间戳
- 密钥轮换：支持多密钥并行与撤销
