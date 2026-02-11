# QLink 协议：理论框架与核心能力

> **学术发表的抽象模型定义**
> 
> 本文档概述了 QLink 协议的理论贡献。它通过定义适合顶级学术研究 (SCI Q1) 的规范性规则和数学模型，填补了“实现细节”与“协议固有机制”之间的空白。

---

## 1. 理论贡献 A：S-AKE (推测式认证密钥交换)

### 1.1 问题定义：安全-时延悖论 (The Security-Latency Paradox)
在去中心化身份 (DID) 网络中，身份验证依赖于区块链账本。这引入了一个根本性的冲突：
*   **安全性 (Safety)**: 需要等待区块链共识（高延迟，$T_{chain}$）。
*   **活性 (Liveness)**: 需要即时通信（低延迟，$T_{net}$）。

传统协议（如 TLS 1.3）假设存在快速、中心化的 PKI。QLink 针对高延迟信任锚点提出了一种全新的交互模型。

### 1.2 核心机制：乐观可验证缓存 (Optimistic Verifiable Caching)
**定义：** S-AKE 是一种协议机制，允许会话基于本地历史数据进入 `Speculative_State`（推测状态），同时异步地向 `Verified_State`（验证状态）转换。

**协议固有规则 (Protocol-Inherent Rules):**
为了将其从“实现细节”提升为“协议特性”，QLink 强制规定：
1.  **乐观握手信号**: `KEMInit` 数据包必须包含 `parent_hash`（所使用 DID 文档的哈希值）。
2.  **不匹配终止规则**: 如果响应方 (Responder) 或中继 (Relay) 检测到 `parent_hash` 与链上真值不同，必须生成错误代码为 `0xBAD_KEY` 的 `PROTOCOL_VIOLATION` 警报。
3.  **强制回滚指令**: 发起方 (Initiator) 在收到 `0xBAD_KEY` 后，必须立即销毁已派生的会话密钥并通知应用层。

### 1.3 协议推测状态的形式化安全模型 (Formal Security Model of Speculative Protocol State)

为了解决“安全-时延悖论”，本协议引入了乐观执行机制。为了严格论证该机制的安全性，我们定义了**推测状态 (Speculative State)** 及其相关的**一致性概率模型**。

#### 1.3.1 推测状态的形式化定义 (Formal Definition)

我们将 **推测状态** 定义为协议在区块链确认完成之前，基于本地缓存凭证进行乐观执行所产生的瞬时配置。

形式化地，设 $\mathbb{S}$ 为所有 **账本确认状态 (Ledger-Confirmed States)** 的集合（即已在不可篡改的区块链上获得终局性的状态）。一个推测状态 $\hat{s} \in \hat{\mathbb{S}}$ 具有以下三个关键特征：

1. **非终局性 (Non-Finality):** $\hat{s}$ 尚未达到区块链的终局一致性，即 $\hat{s} \notin \mathbb{S}$。
    
2. **缓存依赖性 (Cache Dependency):** $\hat{s}$ 的生成依赖于本地 DID 缓存 $\mathcal{C}$，并基于一个乐观假设：即本地缓存 $\mathcal{C}$ 与当前的账本状态 $\mathcal{L}$ 是一致的（$\mathcal{C} \cong \mathcal{L}$）。
    
3. **回滚语义 (Rollback Semantics):** 若系统检测到状态冲突（即 $\mathcal{C} \neq \mathcal{L}$，例如 DID 私钥已撤销但缓存未更新），推测状态 $\hat{s}$ 将被立即丢弃，会话回滚至最近的确认状态 $s \in \mathbb{S}$ 或直接终止。
    
因此，推测状态集 $\hat{\mathbb{S}}$ 是有效状态集 $\mathbb{S}$ 的超集，满足 $\hat{\mathbb{S}} \supset \mathbb{S}$。

#### 1.3.2 一致性概率函数 (Consistency Probability Function)

为了刻画在推测窗口期内的安全风险，我们引入 **一致性概率函数** $P(\Delta t)$。设 $\Delta t$ 为推测状态 $\hat{s}$ 生成时刻（握手）与区块链最终验证时刻（查链）之间的时间延迟。

**定义：**

$$P(\Delta t) = \Pr[\text{攻击者在 } \Delta t \text{ 时间窗口内利用过期凭证成功入侵} \mid \hat{s} \text{ 处于活跃状态}]$$

**模型解释：**

- 与 CPU 微架构中的侧信道泄露不同，本协议中的风险主要来源于 **过期 DID 文档 (Stale DID Document)**（例如：用户刚刚挂失了私钥，但中继服务器的缓存尚未过期）。
    
- 由于 DID 的密钥撤销与更替是低频事件（通常以月或年为单位），而推测窗口 $\Delta t$ 的上界由区块链查询延迟 $T_{chain}$ 决定（通常为毫秒至秒级）。
    

**边界条件：**

- $P(0) = 0$：在握手的瞬间，攻击者无法利用时间差进行攻击。
    
- $\lim_{\Delta t \to T_{chain}} P(\Delta t) \approx \epsilon$：随着时间推移并接近上链验证完成，攻击窗口关闭，风险收敛于一个极小值 $\epsilon$。
    

**安全结论：**

S-AKE 机制本质上是利用 $P(\Delta t)$ 的极低概率特性，交换了 $O(T_{chain})$ 的性能增益。只要保证在 $\Delta t$ 结束后执行强制性的链上状态校验与回滚（Mandatory Rollback），系统即可在保持最终安全性的前提下实现 0-RTT 的交互体验。

---

## 2. 理论贡献 B：Q-Ratchet (轻量级后量子前向安全)

### 2.1 问题定义：后量子适配的代价
现有的前向安全 (FS) 解决方案（如 Signal 的双棘轮）严重依赖 Diffie-Hellman (DH) 运算。
*   **问题**: 在后量子时代，DH 被 KEM（如 Kyber）取代。KEM 计算量更大且密文尺寸更大。在移动/IoT 设备上对 *每条* 消息都进行 KEM 交换（连续 KEM 棘轮）是不切实际的。

### 2.2 核心机制：混合熵注入 (Hybrid Entropy-Injection)
**定义：** Q-Ratchet 是一种混合状态机，利用轻量级对称哈希实现数据包级的安全性，利用周期性的 KEM 实现 Epoch 级（时代级）的安全性。

**协议固有规则 (Protocol-Inherent Rules):**
1.  **确定性密钥演化**: 所有实现必须使用以下递归函数派生消息密钥：
    $$ ChainKey_{i+1} = HMAC(ChainKey_i, \text{"next"}) $$
    $$ MessageKey_{i} = HMAC(ChainKey_i, \text{"msg"}) $$
2.  **序列强制**: 数据包必须包含严格递增的 `sequence_number`。乱序执行依赖于基于链的密钥预计算。
3.  **Epoch 重置 (未来工作)**: 协议保留特定标志位 `FLAG_REKEY`。置位时，Payload 不是消息，而是新的 `KEMInit` 数据包，用于重置 `ChainKey`（注入新鲜的量子熵）。

### 2.3 数学模型：熵衰减 (Entropy Decay)
我们对系统的“前向安全强度” $S(t)$ 进行建模。
*   在纯哈希链中，$S(t)$ 会衰减，因为被攻破的状态无法自愈。
*   在 Q-Ratchet 中，我们在周期 $N$ 处引入 **熵注入点 (Entropy Injection Points)**。
*   **优化目标**: 在保持安全性 $S > Threshold$ 的前提下，最小化通信成本 $C$。
    $$ Minimize 	ext{ } C = N 
cdot C_{hash} + C_{KEM} $$
    $$ Subject 	ext{ } to 	ext{ } Security\_Constraint $$

---

## 3. 核心能力总结

| 能力                      |                                                  |
|:------------------------|:-------------------------------------------------|
| **0-RTT 恢复**            | **S-AKE**: 针对高延迟信任锚点的一致性概率模型。                    |
| **自愈性 (Self-Healing)**  | **可验证回滚 (Verifiable Rollback)**: 确保最终一致性的强制状态转换。 |
| **PFS (前向安全)**          | **Q-Ratchet**: 面向受限环境的、KEM 种子化的对称密钥演化函数。         |
| **身份绑定**                | **密码学绑定**: 基于路由 Header 上的 Ed25519 签名强制实现的不可否认性。  |

本文档作为 QLink 规范及相关研究论文的基础逻辑支撑。