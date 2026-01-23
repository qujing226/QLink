# QLink Protocol: Theoretical Framework & Core Capabilities

> **Abstract Model Definition for Scientific Publication**
> 
> This document outlines the theoretical contributions of the QLink protocol. It bridges the gap between "implementation details" and "protocol-inherent mechanisms" by defining normative rules and mathematical models suitable for top-tier academic research (SCI Q1).

---

## 1. Theoretical Contribution A: S-AKE (Speculative Authenticated Key Exchange)

### 1.1 The Problem: The Security-Latency Paradox
In decentralized identity (DID) networks, the verification of identity relies on blockchain ledgers. This introduces a fundamental conflict:
*   **Safety**: Requires waiting for blockchain consensus (High Latency, $T_{chain}$). 
*   **Liveness**: Requires instant communication (Low Latency, $T_{net}$). 

Traditional protocols (like TLS 1.3) assume a fast, centralized PKI. QLink proposes a new model for high-latency trust anchors.

### 1.2 The Mechanism: Optimistic Verifiable Caching
**Definition:** S-AKE is a protocol mechanism that allows a session to enter a `Speculative_State` based on local historical data, while asynchronously transitioning to a `Verified_State`.

**Protocol-Inherent Rules (How to make it standard):**
To elevate this from an "implementation detail" to a "protocol feature," QLink mandates:
1.  **The Optimistic Handshake Signal**: The `KEMInit` packet MUST include a `parent_hash` (hash of the DID Doc used).
2.  **The Mismatch Termination Rule**: If the Responder (or Relay) detects that `parent_hash` differs from the on-chain truth, it MUST generate a `PROTOCOL_VIOLATION` alert with error code `0xBAD_KEY`.
3.  **The Rollback Mandate**: Upon receiving `0xBAD_KEY`, the Initiator MUST immediately destroy derived session keys and notify the application layer.

### 1.3 Mathematical Model: Probabilistic Consistency
We define the security of the handshake not as a binary (Safe/Unsafe), but as a function of time $t$.

Let $P_{compromise}(\Delta t)$ be the probability of a key compromise occurring exactly within the time window $\Delta t$, where $\Delta t = T_{chain}$. 

*   **Proposition**: Since $\Delta t \approx 500ms$, and key compromise is a rare event (years), $P_{compromise}(\Delta t) \to 0$.
*   **Conclusion**: S-AKE exchanges absolute consistency for a latency reduction of $O(T_{chain})$ with negligible security loss.

---

## 2. Theoretical Contribution B: Q-Ratchet (Lightweight Post-Quantum Forward Secrecy)

### 2.1 The Problem: The Cost of Post-Quantum Adaptation
Existing Forward Secrecy (FS) solutions (like Signal's Double Ratchet) rely heavily on Diffie-Hellman (DH) operations.
*   **Issue**: In the Post-Quantum era, DH is replaced by KEM (e.g., Kyber). KEM is computationally heavier and has larger ciphertext sizes. Doing a KEM exchange for *every* message (Continuous KEM-Ratchet) is impractical for mobile/IoT devices.

### 2.2 The Mechanism: Hybrid Entropy-Injection
**Definition:** Q-Ratchet is a hybrid state machine that uses lightweight symmetric hashing for packet-level security and periodic KEM for epoch-level security.

**Protocol-Inherent Rules (How to make it standard):**
1.  **Deterministic Key Evolution**: All implementations MUST derive message keys using the following recursive function:
    $$ ChainKey_{i+1} = HMAC(ChainKey_i, \text{"next"}) $$
    $$ MessageKey_{i} = HMAC(ChainKey_i, \text{"msg"}) $$
2.  **Sequence Enforcement**: Packets MUST contain a strictly increasing `sequence_number`. Out-of-order execution relies on the pre-calculation of keys, strictly defined by the chain.
3.  **The Epoch Reset (Future Work)**: The protocol reserves a specific flag `FLAG_REKEY`. When set, the payload IS NOT a message, but a new `KEMInit` packet to reset the `ChainKey` (injecting fresh quantum entropy).

### 2.3 Mathematical Model: Entropy Decay
We model the "Forward Secrecy Strength" $S(t)$ of the system.
*   In a pure Hash Chain, $S(t)$ decays because compromised state cannot self-heal.
*   In Q-Ratchet, we introduce **Entropy Injection Points** at period $N$.
*   **Optimization**: The protocol seeks to minimize Communication Cost $C$ while maintaining Security $S > Threshold$.
    $$ Minimize \ C = N \cdot C_{hash} + C_{KEM} $$
    $$ Subject 	o \ext{Security\_Constraint} $$

---

## 3. Summary of Core Capabilities

| Capability | Engineering View (Implementation) | Scientific View (Protocol Theory) |
| :--- | :--- | :--- |
| **0-RTT Resumption** | "I check the cache map." | **S-AKE**: A probabilistic consistency model for high-latency trust anchors. |
| **Self-Healing** | "I re-check the chain in background." | **Verifiable Rollback**: A mandatory state transition ensuring eventual consistency. |
| **PFS (Forward Secrecy)**| "I hash the key every time." | **Q-Ratchet**: A KEM-seeded, symmetric key evolution function for constrained environments. |
| **Identity Binding** | "I sign the packet." | **Cryptographic Binding**: Non-repudiation enforced via Ed25519 signatures over routing headers. |

This file serves as the foundational logic for the QLink specification and associated research papers.
