<template>
  <div class="blockchain-portal">
    <!-- 头部导航 -->
    <header class="portal-header">
      <div class="header-content">
        <div class="logo">
          <span class="logo-icon">🔗</span>
          <h1>QLink 区块链身份门户</h1>
        </div>
        <nav class="nav-menu">
          <button 
            v-for="tab in tabs" 
            :key="tab.id"
            :class="['nav-tab', { active: activeTab === tab.id }]"
            @click="activeTab = tab.id"
          >
            <span class="tab-icon">{{ tab.icon }}</span>
            {{ tab.name }}
          </button>
        </nav>
        <button class="back-btn" @click="goBack">
          <span>←</span> 返回登录
        </button>
      </div>
    </header>

    <!-- 主要内容区域 -->
    <main class="portal-main">
      <div class="container">
        <!-- DID注册 -->
        <div v-if="activeTab === 'register'" class="tab-content">
          <div class="section-header">
            <h2>🆔 DID身份注册</h2>
            <p>创建您的去中心化身份标识符</p>
          </div>
          
          <div class="register-form">
            <div class="form-group">
              <label>选择DID类型</label>
              <select v-model="registerForm.didType" class="form-select">
                <option value="did:qlink">did:qlink (推荐)</option>
                <option value="did:ethr">did:ethr (以太坊)</option>
                <option value="did:key">did:key (密钥)</option>
              </select>
            </div>

            <div class="form-group">
              <label>身份标识符 (可选)</label>
              <input 
                v-model="registerForm.identifier" 
                type="text" 
                class="form-input"
                placeholder="留空将自动生成"
              />
            </div>

            <div class="form-group">
              <label>描述信息 (可选)</label>
              <textarea 
                v-model="registerForm.description" 
                class="form-textarea"
                placeholder="为您的DID添加描述信息"
                rows="3"
              ></textarea>
            </div>

            <div class="form-actions">
              <button 
                class="btn btn-primary" 
                @click="registerDID"
                :disabled="registering"
              >
                <span v-if="registering">⏳</span>
                <span v-else>🔐</span>
                {{ registering ? '注册中...' : '生成DID身份' }}
              </button>
            </div>

            <!-- 注册结果 -->
            <div v-if="registerResult" class="register-result">
              <h3>✅ 注册成功！</h3>
              <div class="result-item">
                <label>DID标识符:</label>
                <div class="result-value">
                  <code>{{ registerResult.did }}</code>
                  <button @click="copyToClipboard(registerResult.did)" class="copy-btn">📋</button>
                </div>
              </div>
              
              <!-- ECDSA密钥信息 -->
              <div class="key-section">
                <h4>🔐 ECDSA密钥 (身份验证)</h4>
                <div class="result-item">
                  <label>ECDSA公钥:</label>
                  <div class="result-value">
                    <code>{{ registerResult.ecdsaKeyPair.publicKey }}</code>
                    <button @click="copyToClipboard(registerResult.ecdsaKeyPair.publicKey)" class="copy-btn">📋</button>
                  </div>
                </div>
                <div class="result-item">
                  <label>ECDSA私钥 (请妥善保管):</label>
                  <div class="result-value">
                    <code class="private-key">{{ showECDSAPrivateKey ? registerResult.ecdsaKeyPair.privateKey : '••••••••••••••••••••••••••••••••' }}</code>
                    <button @click="toggleECDSAPrivateKey" class="toggle-btn">{{ showECDSAPrivateKey ? '👁️' : '👁️‍🗨️' }}</button>
                    <button @click="copyToClipboard(registerResult.ecdsaKeyPair.privateKey)" class="copy-btn">📋</button>
                  </div>
                </div>
              </div>

              <!-- 格加密密钥信息 -->
              <div class="key-section">
                <h4>🛡️ Kyber768密钥 (通信加密)</h4>
                <div class="result-item">
                  <label>Kyber768公钥:</label>
                  <div class="result-value">
                    <code>{{ registerResult.kyberKeyPair.publicKey.substring(0, 64) }}...</code>
                    <button @click="copyToClipboard(registerResult.kyberKeyPair.publicKey)" class="copy-btn">📋</button>
                  </div>
                </div>
                <div class="result-item">
                  <label>Kyber768私钥 (请妥善保管):</label>
                  <div class="result-value">
                    <code class="private-key">{{ showKyberPrivateKey ? registerResult.kyberKeyPair.privateKey.substring(0, 64) + '...' : '••••••••••••••••••••••••••••••••' }}</code>
                    <button @click="toggleKyberPrivateKey" class="toggle-btn">{{ showKyberPrivateKey ? '👁️' : '👁️‍🗨️' }}</button>
                    <button @click="copyToClipboard(registerResult.kyberKeyPair.privateKey)" class="copy-btn">📋</button>
                  </div>
                </div>
              </div>

              <!-- 兼容性显示 (保留原有字段) -->
              <div class="key-section legacy-section">
                <h4>📋 兼容性信息</h4>
                <div class="result-item">
                  <label>主公钥 (ECDSA):</label>
                  <div class="result-value">
                    <code>{{ registerResult.publicKey }}</code>
                    <button @click="copyToClipboard(registerResult.publicKey)" class="copy-btn">📋</button>
                  </div>
                </div>
                <div class="result-item">
                  <label>主私钥 (ECDSA):</label>
                  <div class="result-value">
                    <code class="private-key">{{ showPrivateKey ? registerResult.privateKey : '••••••••••••••••••••••••••••••••' }}</code>
                    <button @click="togglePrivateKey" class="toggle-btn">{{ showPrivateKey ? '👁️' : '👁️‍🗨️' }}</button>
                    <button @click="copyToClipboard(registerResult.privateKey)" class="copy-btn">📋</button>
                  </div>
                </div>
              </div>

              <div class="warning">
                ⚠️ 请务必安全保存您的所有私钥，丢失后无法恢复！<br>
                💡 ECDSA私钥用于身份验证，Kyber768私钥用于通信加密
              </div>
              <button class="btn btn-success" @click="goToLogin">
                前往登录 →
              </button>
            </div>
          </div>
        </div>

        <!-- DID登录 -->
        <div v-if="activeTab === 'login'" class="tab-content">
          <div class="section-header">
            <h2>🔐 DID身份登录</h2>
            <p>使用您的DID进行安全身份验证</p>
          </div>
          
          <div class="login-form">
            <div class="form-group">
              <label>DID标识符</label>
              <input 
                v-model="loginForm.did" 
                type="text" 
                class="form-input"
                placeholder="输入您的DID，如: did:qlink:123456"
              />
            </div>

            <div class="form-group">
              <label>私钥</label>
              <input 
                v-model="loginForm.privateKey" 
                type="password" 
                class="form-input"
                placeholder="输入您的私钥进行身份验证"
              />
            </div>

            <div class="form-actions">
              <button 
                class="btn btn-primary" 
                @click="loginWithDID"
                :disabled="loggingIn"
              >
                <span v-if="loggingIn">⏳</span>
                <span v-else>🔐</span>
                {{ loggingIn ? '验证中...' : '开始登录' }}
              </button>
            </div>

            <!-- 质询-响应流程 -->
            <div v-if="challengeData" class="challenge-section">
              <h3>🎯 身份质询</h3>
              <div class="challenge-info">
                <p>系统已生成质询信息，请确认以下信息并完成签名验证：</p>
                <div class="challenge-details">
                  <div class="detail-item">
                    <label>质询ID:</label>
                    <code>{{ challengeData.id }}</code>
                  </div>
                  <div class="detail-item">
                    <label>质询内容:</label>
                    <code>{{ challengeData.content }}</code>
                  </div>
                  <div class="detail-item">
                    <label>时间戳:</label>
                    <span>{{ formatDate(challengeData.timestamp) }}</span>
                  </div>
                </div>
                
                <div class="form-actions">
                  <button 
                    class="btn btn-secondary" 
                    @click="cancelChallenge"
                  >
                    取消
                  </button>
                  <button 
                    class="btn btn-primary" 
                    @click="signChallenge"
                    :disabled="responding"
                  >
                    <span v-if="responding">⏳</span>
                    <span v-else>✍️</span>
                    {{ responding ? '签名中...' : '签名确认' }}
                  </button>
                </div>
              </div>
            </div>

            <!-- 登录结果 -->
            <div v-if="loginResult" class="login-result">
              <h3>✅ 登录成功！</h3>
              <div class="result-card">
                <div class="result-item">
                  <label>用户DID:</label>
                  <span>{{ loginResult.did }}</span>
                </div>
                <div class="result-item">
                  <label>会话令牌:</label>
                  <code>{{ loginResult.token }}</code>
                </div>
                <div class="result-item">
                  <label>登录时间:</label>
                  <span>{{ loginResult.loginTime }}</span>
                </div>
                <div class="result-item">
                  <label>有效期至:</label>
                  <span>{{ loginResult.expiresAt }}</span>
                </div>
              </div>
              <div class="form-actions">
                <button class="btn btn-success" @click="goToChat">
                  进入聊天室 →
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- DID查询 -->
        <div v-if="activeTab === 'query'" class="tab-content">
          <div class="section-header">
            <h2>🔍 DID身份查询</h2>
            <p>查询已注册的DID身份信息</p>
          </div>

          <div class="query-form">
            <div class="form-group">
              <label>DID标识符</label>
              <input 
                v-model="queryForm.did" 
                type="text" 
                class="form-input"
                placeholder="输入要查询的DID，如: did:qlink:123456"
              />
            </div>
            <div class="form-actions">
              <button 
                class="btn btn-primary" 
                @click="queryDID"
                :disabled="querying"
              >
                <span v-if="querying">⏳</span>
                <span v-else>🔍</span>
                {{ querying ? '查询中...' : '查询DID' }}
              </button>
            </div>

            <!-- 查询结果 -->
            <div v-if="queryResult" class="query-result">
              <h3>📋 DID信息</h3>
              <div class="result-card">
                <div class="result-item">
                  <label>DID:</label>
                  <span>{{ queryResult.did }}</span>
                </div>
                <div class="result-item">
                  <label>状态:</label>
                  <span :class="['status', queryResult.status]">{{ queryResult.status === 'active' ? '✅ 活跃' : '❌ 已停用' }}</span>
                </div>
                <div class="result-item">
                  <label>创建时间:</label>
                  <span>{{ formatDate(queryResult.created) }}</span>
                </div>
                <div class="result-item">
                  <label>公钥:</label>
                  <code>{{ queryResult.publicKey }}</code>
                </div>
                <div v-if="queryResult.description" class="result-item">
                  <label>描述:</label>
                  <span>{{ queryResult.description }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- DID管理 -->
        <div v-if="activeTab === 'manage'" class="tab-content">
          <div class="section-header">
            <h2>⚙️ DID身份管理</h2>
            <p>管理您的DID身份信息</p>
          </div>

          <div class="manage-form">
            <div class="form-group">
              <label>您的DID标识符</label>
              <input 
                v-model="manageForm.did" 
                type="text" 
                class="form-input"
                placeholder="输入您的DID"
              />
            </div>
            <div class="form-group">
              <label>私钥验证</label>
              <input 
                v-model="manageForm.privateKey" 
                type="password" 
                class="form-input"
                placeholder="输入私钥以验证身份"
              />
            </div>
            <div class="form-actions">
              <button 
                class="btn btn-primary" 
                @click="verifyOwnership"
                :disabled="verifying"
              >
                <span v-if="verifying">⏳</span>
                <span v-else">🔐</span>
                {{ verifying ? '验证中...' : '验证身份' }}
              </button>
            </div>

            <!-- 管理操作 -->
            <div v-if="ownershipVerified" class="management-actions">
              <h3>🛠️ 可用操作</h3>
              <div class="action-grid">
                <button class="action-btn update" @click="showUpdateForm = true">
                  <span>📝</span>
                  <div>
                    <strong>更新信息</strong>
                    <small>修改DID描述信息</small>
                  </div>
                </button>
                <button class="action-btn rotate" @click="rotateKeys">
                  <span>🔄</span>
                  <div>
                    <strong>轮换密钥</strong>
                    <small>生成新的密钥对</small>
                  </div>
                </button>
                <button class="action-btn deactivate" @click="deactivateDID">
                  <span>🚫</span>
                  <div>
                    <strong>停用DID</strong>
                    <small>暂时停用此身份</small>
                  </div>
                </button>
                <button class="action-btn delete" @click="deleteDID">
                  <span>🗑️</span>
                  <div>
                    <strong>删除DID</strong>
                    <small>永久删除此身份</small>
                  </div>
                </button>
              </div>

              <!-- 更新表单 -->
              <div v-if="showUpdateForm" class="update-form">
                <h4>📝 更新DID信息</h4>
                <div class="form-group">
                  <label>新的描述信息</label>
                  <textarea 
                    v-model="updateForm.description" 
                    class="form-textarea"
                    rows="3"
                  ></textarea>
                </div>
                <div class="form-actions">
                  <button class="btn btn-secondary" @click="showUpdateForm = false">取消</button>
                  <button class="btn btn-primary" @click="updateDID">更新</button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 帮助文档 -->
        <div v-if="activeTab === 'help'" class="tab-content">
          <div class="section-header">
            <h2>📚 帮助文档</h2>
            <p>了解DID身份系统的使用方法</p>
          </div>

          <div class="help-content">
            <div class="help-section">
              <h3>🤔 什么是DID？</h3>
              <p>DID（Decentralized Identifier，去中心化标识符）是一种新型的身份标识符，它允许用户完全控制自己的数字身份，无需依赖中心化的身份提供商。</p>
            </div>

            <div class="help-section">
              <h3>🔐 密钥管理</h3>
              <ul>
                <li><strong>私钥</strong>：用于签名和证明身份所有权，请务必安全保管</li>
                <li><strong>公钥</strong>：用于验证签名，可以公开分享</li>
                <li><strong>密钥轮换</strong>：定期更换密钥以提高安全性</li>
              </ul>
            </div>

            <div class="help-section">
              <h3>🛡️ 安全建议</h3>
              <ul>
                <li>将私钥保存在安全的地方，建议使用硬件钱包</li>
                <li>不要在不安全的网络环境中输入私钥</li>
                <li>定期备份您的密钥信息</li>
                <li>如果怀疑私钥泄露，立即进行密钥轮换</li>
              </ul>
            </div>

            <div class="help-section">
              <h3>🔄 操作流程</h3>
              <ol>
                <li><strong>注册</strong>：创建新的DID身份</li>
                <li><strong>查询</strong>：验证DID的有效性和状态</li>
                <li><strong>管理</strong>：更新、轮换或删除DID</li>
                <li><strong>登录</strong>：使用DID进行身份验证</li>
              </ol>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- 错误提示 -->
    <div v-if="error" class="error-toast" @click="error = ''">
      <span>❌</span>
      {{ error }}
    </div>

    <!-- 成功提示 -->
    <div v-if="success" class="success-toast" @click="success = ''">
      <span>✅</span>
      {{ success }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { generateDualKeyPair, generateDID, generateECDSASignature, signData } from '../utils/crypto.js'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

// 响应式数据
const activeTab = ref('register')
const error = ref('')
const success = ref('')

// 标签页配置
const tabs = [
  { id: 'register', name: '注册DID', icon: '🆔' },
  { id: 'login', name: 'DID登录', icon: '🔐' },
  { id: 'query', name: '查询DID', icon: '🔍' },
  { id: 'manage', name: '管理DID', icon: '⚙️' },
  { id: 'help', name: '帮助', icon: '📚' }
]

// 注册相关
const registering = ref(false)
const registerForm = ref({
  didType: 'did:qlink',
  identifier: '',
  description: ''
})
const registerResult = ref(null)
const showPrivateKey = ref(false)
const showECDSAPrivateKey = ref(false)
const showKyberPrivateKey = ref(false)

// 登录相关
const loggingIn = ref(false)
const loginForm = ref({
  did: '',
  privateKey: ''
})
const challengeData = ref(null)
const responding = ref(false)
const loginResult = ref(null)

// 查询相关
const querying = ref(false)
const queryForm = ref({
  did: ''
})
const queryResult = ref(null)

// 管理相关
const verifying = ref(false)
const manageForm = ref({
  did: '',
  privateKey: ''
})
const ownershipVerified = ref(false)
const showUpdateForm = ref(false)
const updateForm = ref({
  description: ''
})

// 方法
const goBack = () => {
  router.push('/login')
}

const registerDID = async () => {
  registering.value = true
  error.value = ''
  
  try {
    // 生成双密钥对（ECDSA + 格加密）
    console.log('开始生成双密钥对...')
    const dualKeyPair = await generateDualKeyPair()
    console.log('双密钥对生成成功:', { 
      ecdsaPublicKeyLength: dualKeyPair.ecdsaKeyPair.publicKey.length,
      ecdsaPrivateKeyLength: dualKeyPair.ecdsaKeyPair.privateKey.length,
      latticePublicKeyLength: dualKeyPair.latticeKeyPair.publicKey.length,
      latticePrivateKeyLength: dualKeyPair.latticeKeyPair.privateKey.length
    })
    
    // 使用ECDSA公钥生成DID
    const generatedDID = generateDID(dualKeyPair.ecdsaKeyPair.publicKey)
    console.log('生成的DID:', generatedDID)

    // 构造DID文档（包含双公钥，ECDSA采用JsonWebKey2020/P-256）
    const didDocument = {
      '@context': 'https://www.w3.org/ns/did/v1',
      id: generatedDID,
      verificationMethod: [
        {
          id: `${generatedDID}#ecdsa-key-1`,
          type: 'JsonWebKey2020',
          controller: generatedDID,
          publicKeyJwk: {
            kty: dualKeyPair.ecdsaKeyPair.jwk.kty,
            crv: dualKeyPair.ecdsaKeyPair.jwk.crv,
            x: dualKeyPair.ecdsaKeyPair.jwk.x,
            y: dualKeyPair.ecdsaKeyPair.jwk.y
          }
        },
        {
          id: `${generatedDID}#lattice-key-1`,
          type: 'Kyber768VerificationKey2023',
          controller: generatedDID,
          publicKeyLattice: {
            algorithm: 'Kyber768',
            publicKey: dualKeyPair.latticeKeyPair.publicKey
          }
        }
      ],
      authentication: [`${generatedDID}#ecdsa-key-1`],
      keyAgreement: [`${generatedDID}#lattice-key-1`],
      service: [{
        id: `${generatedDID}#service-1`,
        type: 'DIDCommMessaging',
        serviceEndpoint: 'https://example.com/messaging'
      }]
    }

    // 尝试注册到后端DID系统
    try {
      // 序列化DID文档用于签名
      const documentString = JSON.stringify(didDocument)
      
      // 使用ECDSA私钥对文档进行签名
      const signature = await signData(documentString, dualKeyPair.ecdsaKeyPair.privateKey)
      
      const registerResponse = await fetch('http://localhost:8080/api/v1/did/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          did: generatedDID,
          document: didDocument,
          signature: signature
        })
      })

      if (!registerResponse.ok) {
        const errorText = await registerResponse.text()
        console.warn(`后端注册失败: ${registerResponse.status} - ${errorText}，但密钥已生成`)
      } else {
        console.log('后端注册成功')
      }
    } catch (backendError) {
      console.warn('后端注册失败，但密钥已生成:', backendError.message)
    }
    
    // 无论后端是否成功，都显示生成的双密钥
    registerResult.value = {
      did: generatedDID,
      ecdsaKeyPair: {
        publicKey: dualKeyPair.ecdsaKeyPair.publicKey,
        privateKey: dualKeyPair.ecdsaKeyPair.privateKey
      },
      kyberKeyPair: {
        publicKey: dualKeyPair.latticeKeyPair.publicKey,
        privateKey: dualKeyPair.latticeKeyPair.privateKey
      },
      // 为了向后兼容，保留原有字段（使用ECDSA密钥）
      publicKey: dualKeyPair.ecdsaKeyPair.publicKey,
      privateKey: dualKeyPair.ecdsaKeyPair.privateKey,
      description: registerForm.value.description,
      created: new Date().toISOString()
    }
    
    success.value = 'DID和密钥生成成功！请妥善保存您的私钥。'
    console.log('注册结果已设置:', registerResult.value)
    
  } catch (err) {
    console.error('DID注册失败:', err)
    error.value = `注册失败: ${err.message}`
  } finally {
    registering.value = false
  }
}

const queryDID = async () => {
  if (!queryForm.value.did) {
    error.value = '请输入要查询的DID'
    return
  }

  querying.value = true
  error.value = ''
  success.value = ''

  try {
    // 调用真实的后端API
    const response = await fetch(`http://localhost:8080/api/v1/did/resolve/${encodeURIComponent(queryForm.value.did)}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      }
    })

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error('DID不存在')
      }
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const result = await response.json()
    
    queryResult.value = {
      did: result.did || queryForm.value.did,
      document: result.document || result,
      status: result.status || 'active',
      created: result.created || new Date().toISOString(),
      updated: result.updated || new Date().toISOString()
    }

    success.value = 'DID查询成功！'
    
  } catch (err) {
    console.error('DID查询失败:', err)
    error.value = `查询失败: ${err.message}`
  } finally {
    querying.value = false
  }
}

const verifyOwnership = async () => {
  verifying.value = true
  error.value = ''
  
  try {
    // 模拟身份验证
    await new Promise(resolve => setTimeout(resolve, 1500))
    
    if (!manageForm.value.did || !manageForm.value.privateKey) {
      throw new Error('请输入DID和私钥')
    }
    
    // 模拟验证成功
    ownershipVerified.value = true
    success.value = '身份验证成功！'
  } catch (err) {
    error.value = '验证失败：' + err.message
  } finally {
    verifying.value = false
  }
}

const updateDID = async () => {
  try {
    // 模拟更新操作
    await new Promise(resolve => setTimeout(resolve, 1000))
    success.value = 'DID信息更新成功！'
    showUpdateForm.value = false
  } catch (err) {
    error.value = '更新失败：' + err.message
  }
}

const rotateKeys = async () => {
  if (confirm('确定要轮换密钥吗？这将生成新的密钥对。')) {
    try {
      // 模拟密钥轮换
      await new Promise(resolve => setTimeout(resolve, 2000))
      success.value = '密钥轮换成功！请保存新的私钥。'
    } catch (err) {
      error.value = '密钥轮换失败：' + err.message
    }
  }
}

const deactivateDID = async () => {
  if (confirm('确定要停用此DID吗？')) {
    try {
      // 模拟停用操作
      await new Promise(resolve => setTimeout(resolve, 1000))
      success.value = 'DID已停用'
    } catch (err) {
      error.value = '停用失败：' + err.message
    }
  }
}

const deleteDID = async () => {
  if (confirm('确定要永久删除此DID吗？此操作不可恢复！')) {
    try {
      // 模拟删除操作
      await new Promise(resolve => setTimeout(resolve, 1000))
      success.value = 'DID已删除'
      ownershipVerified.value = false
    } catch (err) {
      error.value = '删除失败：' + err.message
    }
  }
}

// DID登录相关方法
const loginWithDID = async () => {
  if (!loginForm.value.did || !loginForm.value.privateKey) {
    error.value = '请填写DID标识符和私钥'
    return
  }

  loggingIn.value = true
  error.value = ''
  
  try {
    // 第一步：创建质询（改用 auth store）
    const resp = await authStore.createChallenge(loginForm.value.did)
    if (!resp.success) {
      throw new Error(resp.error || '创建质询失败')
    }
    // 显示质询信息
    challengeData.value = {
      id: resp.challenge_id,
      content: resp.challenge,
      timestamp: new Date().toLocaleString(),
      expiresAt: undefined
    }

  } catch (err) {
    console.error('登录失败:', err)
    error.value = '登录失败: ' + err.message
  } finally {
    loggingIn.value = false
  }
}

const signChallenge = async () => {
  responding.value = true
  error.value = ''
  
  try {
    // 使用ECDSA私钥对质询进行签名
    const signature = await generateECDSASignature(challengeData.value.content, loginForm.value.privateKey)

    // 第二步：使用签名验证登录（改用 auth store）
    const result = await authStore.verifyChallenge(signature, loginForm.value.did)
    if (!result.success) {
      throw new Error(result.error || '登录验证失败')
    }

    // 显示登录成功结果（从 store 读取）
    loginResult.value = {
      did: authStore.user?.did || loginForm.value.did,
      token: authStore.token,
      loginTime: new Date().toLocaleString(),
      expiresAt: '24小时后'
    }

    // 跳转到聊天
    router.push('/chat')

    // 清除质询数据
    challengeData.value = null
    success.value = '登录成功！'

  } catch (err) {
    console.error('签名验证失败:', err)
    error.value = '签名验证失败: ' + err.message
  } finally {
    responding.value = false
  }
}

const cancelChallenge = () => {
  challengeData.value = null
}

const enterChatRoom = () => {
  // 这里可以跳转到聊天室或其他页面
  alert('即将进入聊天室...')
}

// 跳转到聊天页面（修复模板中的 goToChat 按钮）
const goToChat = () => {
  router.push('/chat')
}

const togglePrivateKey = () => {
  showPrivateKey.value = !showPrivateKey.value
}

const toggleECDSAPrivateKey = () => {
  showECDSAPrivateKey.value = !showECDSAPrivateKey.value
}

const toggleKyberPrivateKey = () => {
  showKyberPrivateKey.value = !showKyberPrivateKey.value
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    success.value = '已复制到剪贴板'
  } catch (err) {
    error.value = '复制失败'
  }
}

// 生成混合签名
const generateHybridSignature = async (challenge) => {
  try {
    // 检查是否有注册结果中的私钥
    if (!registerResult.value || !registerResult.value.privateKey) {
      throw new Error('未找到私钥，请先注册DID')
    }
    
    // 使用ECDSA签名质询
    const signature = await generateECDSASignatureLocal(challenge, registerResult.value.privateKey)
    return signature
    
  } catch (error) {
    console.error('混合签名生成失败:', error)
    throw new Error('混合签名生成失败: ' + error.message)
  }
}

// 生成ECDSA签名
const generateECDSASignatureLocal = async (message, privateKeyBase64) => {
  try {
    // 使用crypto.js中的generateECDSASignature函数
    return await generateECDSASignature(message, privateKeyBase64)
  } catch (error) {
    console.error('ECDSA签名失败:', error)
    throw new Error('ECDSA签名失败: ' + error.message)
  }
}

// 辅助函数：base64转ArrayBuffer
const base64ToArrayBuffer = (base64) => {
  const binaryString = atob(base64)
  const bytes = new Uint8Array(binaryString.length)
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }
  return bytes.buffer
}

// 辅助函数：ArrayBuffer转base64
const arrayBufferToBase64 = (buffer) => {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
}

const goToLogin = () => {
  activeTab.value = 'login'
  // 如果有注册结果，自动填充DID和私钥
  if (registerResult.value) {
    loginForm.value.did = registerResult.value.did
    loginForm.value.privateKey = registerResult.value.privateKey
  }
}

const formatDate = (dateString) => {
  return new Date(dateString).toLocaleString('zh-CN')
}

// 自动隐藏提示
const hideToast = (type) => {
  setTimeout(() => {
    if (type === 'error') error.value = ''
    if (type === 'success') success.value = ''
  }, 3000)
}

// 监听提示变化
const watchToasts = () => {
  if (error.value) hideToast('error')
  if (success.value) hideToast('success')
}

onMounted(() => {
  // 页面加载完成
})
</script>

<style scoped>
.blockchain-portal {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* 头部导航 */
.portal-header {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
  position: sticky;
  top: 0;
  z-index: 100;
}

.header-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 80px;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-icon {
  font-size: 32px;
}

.logo h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.nav-menu {
  display: flex;
  gap: 8px;
}

.nav-tab {
  padding: 12px 20px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #666;
}

.nav-tab.active {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.nav-tab:hover:not(.active) {
  background: rgba(102, 126, 234, 0.1);
  color: #667eea;
}

.tab-icon {
  font-size: 16px;
}

.back-btn {
  padding: 10px 16px;
  border: 2px solid #667eea;
  background: white;
  color: #667eea;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 14px;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}

.back-btn:hover {
  background: #667eea;
  color: white;
}

/* 主要内容 */
.portal-main {
  padding: 40px 20px;
}

.container {
  max-width: 800px;
  margin: 0 auto;
}

.tab-content {
  background: white;
  border-radius: 20px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
}

.section-header {
  text-align: center;
  margin-bottom: 40px;
}

.section-header h2 {
  margin: 0 0 12px 0;
  font-size: 28px;
  font-weight: 700;
  color: #333;
}

.section-header p {
  margin: 0;
  color: #666;
  font-size: 16px;
}

/* 表单样式 */
.form-group {
  margin-bottom: 24px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #333;
  font-weight: 600;
  font-size: 14px;
}

.form-input, .form-select, .form-textarea {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
  transition: border-color 0.3s ease;
  box-sizing: border-box;
  font-family: inherit;
}

.form-input:focus, .form-select:focus, .form-textarea:focus {
  outline: none;
  border-color: #667eea;
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
}

.form-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

/* 按钮样式 */
.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  text-decoration: none;
}

.btn-primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-secondary {
  background: white;
  color: #667eea;
  border: 2px solid #667eea;
}

.btn-secondary:hover {
  background: #667eea;
  color: white;
}

.btn-success {
  background: #4caf50;
  color: white;
}

.btn-success:hover {
  background: #45a049;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none !important;
}

/* 注册结果 */
.register-result {
  margin-top: 32px;
  padding: 24px;
  background: #f8f9ff;
  border-radius: 12px;
  border: 1px solid #e8eaff;
}

.register-result h3 {
  margin: 0 0 20px 0;
  color: #4caf50;
  font-size: 20px;
}

.result-item {
  margin-bottom: 16px;
}

.result-item label {
  display: block;
  margin-bottom: 4px;
  color: #333;
  font-weight: 600;
  font-size: 13px;
}

.result-value {
  display: flex;
  align-items: center;
  gap: 8px;
}

.result-value code {
  flex: 1;
  padding: 8px 12px;
  background: #f5f5f5;
  border-radius: 6px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  word-break: break-all;
}

.private-key {
  background: #fff3cd !important;
  border: 1px solid #ffeaa7;
}

.copy-btn, .toggle-btn {
  padding: 6px 8px;
  border: none;
  background: #667eea;
  color: white;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.3s ease;
}

.copy-btn:hover, .toggle-btn:hover {
  background: #5a6fd8;
}

.warning {
  margin: 20px 0;
  padding: 12px 16px;
  background: #fff3cd;
  border: 1px solid #ffeaa7;
  border-radius: 8px;
  color: #856404;
  font-size: 14px;
  font-weight: 500;
}

/* 查询结果 */
.query-result {
  margin-top: 32px;
}

.query-result h3 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 20px;
}

.result-card {
  background: #f8f9ff;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e8eaff;
}

.result-card .result-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #eee;
}

.result-card .result-item:last-child {
  border-bottom: none;
}

.result-card .result-item label {
  font-weight: 600;
  color: #666;
  margin: 0;
}

.status.active {
  color: #4caf50;
}

/* 管理操作 */
.management-actions {
  margin-top: 32px;
  padding: 24px;
  background: #f8f9ff;
  border-radius: 12px;
  border: 1px solid #e8eaff;
}

.management-actions h3 {
  margin: 0 0 20px 0;
  color: #333;
  font-size: 20px;
}

.action-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.action-btn {
  padding: 16px;
  border: 2px solid #e0e0e0;
  background: white;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 12px;
  text-align: left;
}

.action-btn:hover {
  border-color: #667eea;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.action-btn span {
  font-size: 24px;
}

.action-btn strong {
  display: block;
  margin-bottom: 4px;
  color: #333;
  font-size: 14px;
}

.action-btn small {
  color: #666;
  font-size: 12px;
}

.action-btn.delete:hover {
  border-color: #f44336;
  color: #f44336;
}

.update-form {
  margin-top: 24px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e0e0e0;
}

.update-form h4 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 16px;
}

/* 帮助文档 */
.help-content {
  max-width: 600px;
  margin: 0 auto;
}

.help-section {
  margin-bottom: 32px;
  padding: 24px;
  background: #f8f9ff;
  border-radius: 12px;
  border: 1px solid #e8eaff;
}

.help-section h3 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 18px;
}

.help-section p {
  margin: 0 0 12px 0;
  color: #666;
  line-height: 1.6;
}

.help-section ul, .help-section ol {
  margin: 0;
  padding-left: 20px;
  color: #666;
  line-height: 1.6;
}

.help-section li {
  margin-bottom: 8px;
}

/* 提示框 */
.error-toast, .success-toast {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 12px 20px;
  border-radius: 8px;
  color: white;
  font-weight: 500;
  cursor: pointer;
  z-index: 1000;
  display: flex;
  align-items: center;
  gap: 8px;
  animation: slideIn 0.3s ease;
}

.error-toast {
  background: #f44336;
}

.success-toast {
  background: #4caf50;
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .header-content {
    flex-direction: column;
    height: auto;
    padding: 20px;
    gap: 20px;
  }

  .nav-menu {
    flex-wrap: wrap;
    justify-content: center;
  }

  .tab-content {
    padding: 24px;
  }

  .action-grid {
    grid-template-columns: 1fr;
  }

  .form-actions {
    flex-direction: column;
  }
}
</style>