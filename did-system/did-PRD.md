## DID
### 🧩 第一阶段：聚焦 DID 的关键问题

DID 部分需要先回答几个核心问题，这些问题会直接影响架构设计和实现细节：

1. **DID 方法选择**
    - 用现有标准方法（如 did:key、did:web、did:ethr）还是自定义方法（did:yourchain）
    - 自定义方法需要定义解析规则、标识符格式、链上存储结构

2. **DID 文档（DID Document）存储与解析**

    - 存储位置：链上（不可篡改） vs 链下（IPFS/数据库）
    - 解析方式：本地解析器 vs 远程解析服务
    - 更新机制：如何安全地更新公钥、服务端点等信息

3. **密钥管理与轮换**

    - 支持哪些密钥类型（Ed25519、Secp256k1、Kyber768 公钥）
    - 如何实现密钥轮换（Key Rotation）与撤销（Revocation）
    - 抗量子密钥与传统密钥的共存策略

4. **认证与授权流程**

    - DID 如何参与通信系统的身份认证（Challenge-Response、JWT、VC）
    - 授权模型：基于 VC（Verifiable Credential）还是 ACL（Access Control List）

5. **互操作性与标准对齐**

    - 对齐 W3C DID Core 规范
    - 是否支持 DID Resolution v1.0 API
    - 与现有 DID Registry（如 DIF Universal Resolver）的兼容性

6. **安全与隐私**

    - 防止 DID 劫持、重放攻击
    - DID 文档中服务端点的隐私保护（如通过加密或访问控制）

### 🔧 第二阶段：实现事项清单

基于上面的问题，可以列出重构时的主要实现任务：

- **DID 方法实现**

- 定义 `did:yourmethod` 的标识符格式
- 编写链上合约/模块，支持 DID 注册、更新、撤销
- 编写解析器（Resolver），支持链上链下混合解析

- **DID 文档管理**

- 设计 DID Document JSON-LD 模板
- 实现链上存储（哈希）+ 链下存储（IPFS/数据库）
- 提供 DID Document 更新 API

- **密钥与加密支持**

- 集成 Kyber768 公钥到 DID Document
- 支持多密钥类型与用途（authentication、assertionMethod、keyAgreement）
- 实现密钥轮换与撤销流程

- **认证与授权模块**

- 实现基于 DID 的 Challenge-Response 认证
- 支持 VC 签发与验证
- 集成到通信系统的握手流程

- **互操作性**

- 提供符合 W3C DID Resolution API 的接口
- 可选接入 DIF Universal Resolver

- **安全防护**

- 防重放机制（nonce、时间戳）
- DID 文档端点加密或访问控制
- 审计日志与事件追踪

### 📄 第三阶段：PRD 框架（初稿）

我建议 PRD 按以下结构来写，方便后续扩展：

1. **项目背景与目标**

- 为什么要重构 DID 部分
- 目标：提升安全性、标准兼容性、可扩展性

3. **功能需求**

- DID 方法定义
- DID Document 管理
- 密钥管理（含抗量子支持）
- 认证与授权
- 互操作性
- 安全防护

5. **非功能需求**

- 性能（解析延迟、注册吞吐量）
- 可用性（API SLA、容错机制）
- 兼容性（与现有通信系统、区块链平台）

7. **系统架构**

- 模块划分（链上合约、解析器、API 网关、存储层）
- 数据流与调用流程图

9. **接口设计**

- DID 注册/解析 API
- DID Document 更新 API
- 认证/授权 API

11. **安全与合规**

- 威胁模型
- 安全策略
- 标准对齐（W3C DID、NIST PQC）

13. **实施计划**

- 开发阶段划分
- 测试与验收标准
- 上线与运维方案