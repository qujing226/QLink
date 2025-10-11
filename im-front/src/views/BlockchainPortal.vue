<template>
  <div class="blockchain-portal">
    <!-- 头部导航 -->
    <header class="portal-header">
      <div class="header-content">
        <div class="logo">
          <h1>QLink</h1>
        </div>
        <!-- 顶部按钮组：与标题同排，分屏切换 -->
        <div class="header-actions nav-menu">
          <button class="nav-tab" :class="{ active: activeSection === 'home' }" @click="goSection('home')">概览</button>
          <button class="nav-tab" :class="{ active: activeSection === 'register' }" @click="goSection('register')">注册</button>
          <button class="nav-tab" :class="{ active: activeSection === 'query' }" @click="goSection('query')">查询</button>
          <button class="nav-tab" :class="{ active: activeSection === 'manage' }" @click="goSection('manage')">管理</button>
        </div>
      </div>
    </header>

    <!-- 主要内容区域 -->
    <main class="portal-main">
      <div class="container">
        <!-- 页面级简要介绍（置于标题栏下、首页横幅之上） -->
        <div class="page-lead">QLink：以自我主权身份为核心的可信网络。</div>
        <!-- 首页 -->
        <section id="home" class="tab-content home-section full-screen full-bleed">
          <div class="hero-banner" :style="{ opacity: heroOpacity, transform: 'translateY(' + heroTranslateY + 'px)' }">
            <div class="hero-overlay">
              <div class="hero-grid">
                <div class="hero-copy">
                  <h2 class="hero-title">欢迎来到 QLink</h2>
                  <p class="hero-subtitle">以自我主权身份为核心的可信网络</p>
                  <div class="hero-description">
                    <p>
                      采用可验证凭证与隐私保护证明，最小披露而可证明可信，助力在零信任环境中完成授权、协作与合规。
                    </p>
                    <p>
                      量子抗性与现代密码学并行的安全架构，让身份在不同系统间优雅迁移，同时保留对数据的最终控制权。
                    </p>
                  </div>
                </div>
                <div class="hero-actions-grid">
                  <button class="btn btn-primary" @click="goSection('register')">开始注册</button>
                  <button class="btn btn-secondary" @click="goSection('query')">查询DID</button>
                  <button class="btn btn-secondary" @click="goSection('manage')">管理身份</button>
                  <button class="btn btn-secondary" @click="contactUs">联系我们</button>
                </div>
              </div>
            </div>
          </div>
        </section>
        <!-- DID注册 -->
        <section id="register" class="tab-content full-screen gradient-section">
          <div class="two-col">
            <div class="col-left form-card">
              <div class="section-header">
                <h2>DID身份注册</h2>
                <p>创建您的去中心化身份标识符</p>
              </div>
              <div class="register-form">
                <div class="form-group">
                  <label>类型</label>
                  <select v-model="registerForm.didType" class="form-select">
                    <option value="did:qlink">did:qlink</option>
                    <option value="did:ethr">did:ethr</option>
                    <option value="did:key">did:key</option>
                  </select>
                </div>
                <div class="form-group">
                  <label>标识</label>
                  <input 
                    v-model="registerForm.identifier" 
                    type="text" 
                    class="form-input"
                    placeholder="留空将自动生成"
                  />
                </div>
                <div class="form-group">
                  <label>描述</label>
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
                    {{ registering ? '注册中...' : '生成DID身份' }}
                  </button>
                </div>
              </div>
            </div>
            <div class="col-right">
              <div v-if="registerResult" class="form-card register-result">
                <h3>✅ 注册成功！</h3>
                <div class="result-item">
                  <label>DID标识符:</label>
                  <div class="result-value">
                    <code>{{ registerResult.did }}</code>
                    <button @click="copyToClipboard(registerResult.did)" class="copy-btn">📋</button>
                  </div>
                </div>
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
              </div>
              <div v-else class="result-placeholder"></div>
            </div>
          </div>
        </section>

        

        <!-- DID查询 -->
        <section id="query" class="tab-content full-screen">
          <div class="two-col">
            <div class="col-left form-card">
              <div class="section-header">
                <h2>DID身份查询</h2>
                <p>查询已注册的DID身份信息</p>
              </div>
              <div class="query-form">
                <div class="form-group">
                  <label>DID标识符</label>
                  <input 
                    v-model="queryForm.did" 
                    type="text" 
                    class="form-input"
                    placeholder="输入要查询的DID"
                    @keydown.enter="queryDID"
                  />
                </div>
                <div class="form-actions">
                  <button 
                    class="btn btn-primary" 
                    @click="queryDID"
                    :disabled="querying"
                  >
                    {{ querying ? '查询中...' : '查询' }}
                  </button>
                </div>
              </div>
            </div>
            <div class="col-right">
              <div v-if="queryResult" class="form-card query-result">
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
              <div v-else class="result-placeholder"></div>
            </div>
          </div>
        </section>

        <!-- DID管理 -->
        <section id="manage" class="tab-content full-screen">
          <div class="two-col">
            <div class="col-left form-card">
              <div class="section-header">
                <h2>DID身份管理</h2>
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
                    {{ verifying ? '验证中...' : '验证' }}
                  </button>
                </div>
              </div>
            </div>
            <div class="col-right">
              <div v-if="ownershipVerified" class="form-card management-actions">
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
              <div v-else class="result-placeholder"></div>
            </div>
          </div>
        </section>

        
      </div>
    </main>

    <!-- 错误提示 -->
    <div v-if="error" class="error-toast" @click="error = ''">
      {{ error }}
    </div>

    <!-- 成功提示 -->
    <div v-if="success" class="success-toast" @click="success = ''">
      {{ success }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { generateDualKeyPair, generateDID, signData } from '../utils/crypto.js'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

// 响应式数据
const error = ref('')
const success = ref('')

// 首页英雄区滚动特效
const heroOpacity = ref(1)
const heroTranslateY = ref(0)
let heroScrollHandler = null

// 分屏滚动与顶部按钮状态
const sections = ['home', 'register', 'query', 'manage']
const activeSection = ref('home')
let wheelLock = false
let sectionObserver = null

const wheelHandler = (e) => {
  // 更平滑的分屏滚动：仅在切换分屏时阻止默认滚动
  if (wheelLock) return
  const idx = sections.indexOf(activeSection.value)
  let target = null
  const threshold = 25
  if (e.deltaY > threshold) {
    // 下滚：切换到下一屏
    if (idx < sections.length - 1) target = sections[idx + 1]
  } else if (e.deltaY < -threshold) {
    // 上滚：首页允许默认滚动以查看标题，其余切换上一屏
    if (idx > 0) target = sections[idx - 1]
    else return
  }

  if (target) {
    e.preventDefault()
    wheelLock = true
    goSection(target)
    setTimeout(() => { wheelLock = false }, 500)
  }
}

const goSection = (id) => {
  activeSection.value = id
  const el = document.getElementById(id)
  if (!el) return
  // 使用平滑滚动，提升体验
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// 联系我们：打开默认邮件客户端，且提供复制邮箱的兜底
const contactUs = async () => {
  const email = 'contact@qlink.local'
  const subject = encodeURIComponent('QLink咨询')
  const body = encodeURIComponent('请简要描述您的需求或问题')
  const mailto = `mailto:${email}?subject=${subject}&body=${body}`
  try {
    window.location.href = mailto
    success.value = '已尝试打开邮件客户端'
  } catch (err) {
    // 兜底：复制邮箱地址
    try {
      await navigator.clipboard.writeText(email)
      success.value = '已复制邮箱地址：' + email
    } catch (copyErr) {
      error.value = '请手动联系邮箱：' + email
    }
  }
}

// 采用单页滚动分区，不再使用选项卡

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

// 登录相关（已移除）

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
    
    // 构造最终DID：优先使用用户输入的标识符
    const didType = (registerForm.value.didType || 'did:qlink').trim()
    const identifier = (registerForm.value.identifier || '').trim()
    if (identifier && identifier.length <= 8) {
      throw new Error('标识需大于8个字符')
    }
    const finalDID = identifier ? `${didType}:${identifier}` : generateDID(dualKeyPair.ecdsaKeyPair.publicKey)
    console.log('最终DID:', finalDID)

    // 构造DID文档（包含双公钥，ECDSA采用JsonWebKey2020/P-256）
    const didDocument = {
      '@context': 'https://www.w3.org/ns/did/v1',
      id: finalDID,
      verificationMethod: [
        {
          id: `${finalDID}#ecdsa-key-1`,
          type: 'JsonWebKey2020',
          controller: finalDID,
          publicKeyJwk: {
            kty: dualKeyPair.ecdsaKeyPair.jwk.kty,
            crv: dualKeyPair.ecdsaKeyPair.jwk.crv,
            x: dualKeyPair.ecdsaKeyPair.jwk.x,
            y: dualKeyPair.ecdsaKeyPair.jwk.y
          }
        },
        {
          id: `${finalDID}#lattice-key-1`,
          type: 'Kyber768VerificationKey2023',
          controller: finalDID,
          publicKeyLattice: {
            algorithm: 'Kyber768',
            publicKey: dualKeyPair.latticeKeyPair.publicKey
          }
        }
      ],
      authentication: [`${finalDID}#ecdsa-key-1`],
      keyAgreement: [`${finalDID}#lattice-key-1`],
      service: [{
        id: `${finalDID}#service-1`,
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
          did: finalDID,
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
      did: finalDID,
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

// 登录相关方法已删除

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

// 滚动到指定分区
const scrollTo = (id) => {
  const el = document.getElementById(id)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
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

// 已去除返回登录入口

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
  heroScrollHandler = () => {
    const y = window.scrollY || 0
    const max = 300
    const ratio = Math.min(y / max, 1)
    heroOpacity.value = 1 - ratio * 0.6
    heroTranslateY.value = ratio * 40
  }
  window.addEventListener('scroll', heroScrollHandler, { passive: true })
  heroScrollHandler()

  // 观察分屏分区，动态同步头部按钮状态
  sectionObserver = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting && entry.intersectionRatio > 0.6) {
        activeSection.value = entry.target.id
      }
    })
  }, { threshold: [0.6] })

  sections.forEach(id => {
    const el = document.getElementById(id)
    if (el) sectionObserver.observe(el)
  })

  // 分屏滚轮
  window.addEventListener('wheel', wheelHandler, { passive: false })
})

onUnmounted(() => {
  if (heroScrollHandler) {
    window.removeEventListener('scroll', heroScrollHandler)
    heroScrollHandler = null
  }
  window.removeEventListener('wheel', wheelHandler)
  if (sectionObserver) {
    sectionObserver.disconnect()
    sectionObserver = null
  }
})
</script>

<style scoped>
.blockchain-portal {
  min-height: 100vh;
  background: #f6f7fb;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* 头部导航 */
.portal-header {
  background: #ffffff;
  border-bottom: 1px solid #e5e7eb;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  width: 100%;
  z-index: 100;
  --header-h: 80px;
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
  color: #111827;
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
  background: #e5e7eb;
  color: #111827;
}

.nav-tab:hover:not(.active) {
  background: #f3f4f6;
  color: #111827;
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
  padding: calc(var(--header-h, 80px) + 40px) 20px 40px 20px;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
}

.page-lead {
  width: 100%;
  margin: 0 0 16px 0;
  padding: 10px 16px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  color: #64748b;
  font-size: 14px;
}

/* 让首页分区支持全幅显示（不受.container限制） */
.full-bleed {
  margin-left: calc((100vw - 1200px) / -2);
  margin-right: calc((100vw - 1200px) / -2);
}

.full-bleed .hero-banner {
  width: 100vw;
  border-radius: 0;
}

/* 顶部按钮组与标题同排 */
.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-tab.active {
  background: #e5e7eb;
  color: #111827;
  font-weight: 600;
}

/* 全屏分区样式 */
.full-screen {
  min-height: calc(100vh - 120px);
  display: flex;
  align-items: center;
}

.full-screen.tab-content {
  background: transparent;
  border: none;
  box-shadow: none;
  padding: 0;
}

/* 紫色渐变英雄横幅 */
.hero-banner {
  width: 100%;
  height: calc(100vh - var(--header-h, 80px));
  border-radius: 12px;
  background: linear-gradient(135deg, #6a11cb 0%, #2575fc 100%);
  position: relative;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  /* 与页面容器左缘对齐，使文案更靠左 */
  padding-left: calc((100vw - 1200px) / 2 + 20px);
  color: #fff;
}

.hero-overlay {
  max-width: 1080px;
  padding: 32px;
}

.hero-grid {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  align-items: center;
  gap: 24px;
}

.hero-copy {
  text-align: left;
}

.hero-title {
  margin: 0 0 12px 0;
  font-size: 40px;
  font-weight: 800;
}

.hero-subtitle {
  margin: 0 0 16px 0;
  font-size: 18px;
  opacity: 0.92;
}

.hero-description p {
  margin: 0 0 8px 0;
  opacity: 0.92;
}

.hero-actions-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(140px, 1fr));
  grid-auto-rows: 48px;
  gap: 12px;
}

.hero-actions-grid .btn {
  justify-content: center;
}

.tab-content {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 40px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.06);
}

/* 注册分区渐变背景 */
.gradient-section {
  background: linear-gradient(135deg, rgba(106,17,203,0.12) 0%, rgba(37,117,252,0.12) 100%);
  border: none;
  box-shadow: none;
}

.gradient-section .section-header h2,
.gradient-section .section-header p {
  color: #0b0d0e;
}

.section-header {
  text-align: left;
  margin-bottom: 24px;
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
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 14px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  box-sizing: border-box;
  font-family: inherit;
  background: #ffffff;
  color: #111827;
}

.form-input:focus, .form-select:focus, .form-textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59,130,246,0.15);
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

/* 查询与管理页的动作按钮左对齐，更贴近表单语义 */
.query-form .form-actions,
.manage-form .form-actions {
  justify-content: flex-start;
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
  background: #3b82f6;
  color: #ffffff;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  background: #2563eb;
}

.btn-secondary {
  background: #f9fafb;
  color: #111827;
  border: 1px solid #e5e7eb;
}

.btn-secondary:hover {
  background: #f3f4f6;
  color: #111827;
}

.btn-success {
  background: #10b981;
  color: #ffffff;
}

.btn-success:hover {
  background: #0ea76a;
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
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
}

.register-result h3 {
  margin: 0 0 20px 0;
  color: #e5e7eb;
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
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  word-break: break-all;
  color: #111827;
}

.private-key {
  background: #ffffff !important;
  border: 1px solid #e5e7eb;
}

.copy-btn, .toggle-btn {
  padding: 6px 8px;
  border: 1px solid #e5e7eb;
  background: #f9fafb;
  color: #111827;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.3s ease;
}

.copy-btn:hover, .toggle-btn:hover {
  background: #f3f4f6;
}

.warning {
  margin: 20px 0;
  padding: 12px 16px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  color: #4b5563;
  font-size: 14px;
  font-weight: 500;
}

/* 查询结果 */
.query-result {
  margin-top: 32px;
}

.query-result h3 {
  margin: 0 0 16px 0;
  color: #e5e7eb;
  font-size: 20px;
}

.result-card {
  background: #ffffff;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e5e7eb;
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
  color: #9ca3af;
  margin: 0;
}

.status.active {
  color: #4caf50;
}

/* 管理操作 */
.management-actions {
  margin-top: 32px;
  padding: 24px;
  background: #ffffff;
  border-radius: 12px;
  border: 1px solid #e5e7eb;
}

.management-actions h3 {
  margin: 0 0 20px 0;
  color: #e5e7eb;
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
  border: 1px solid #1f2937;
  background: #0b0d0e;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 12px;
  text-align: left;
}

.action-btn:hover {
  border-color: #374151;
  transform: translateY(-2px);
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
  top: calc(var(--header-h, 80px) + 10px);
  left: 50%;
  transform: translateX(-50%);
  right: auto;
  width: calc(100% - 40px);
  max-width: 1200px;
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
<style scoped>
.two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.col-left, .col-right { width: 100%; }

.form-card {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 24px;
}

.result-placeholder {
  background: #ffffff;
  border: 1px dashed #cbd5e1;
  border-radius: 12px;
  padding: 24px;
  color: #64748b;
}

/* 顶部菜单激活态下划线强调 */
.nav-menu .nav-tab.active {
  border-bottom: 2px solid #667eea;
}

/* 移动端提示条适配容器宽度 */
@media (max-width: 768px) {
  .error-toast, .success-toast {
    width: calc(100% - 24px);
  }
}
</style>