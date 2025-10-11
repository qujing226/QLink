<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <div class="logo">
          <div class="logo-icon">🔐</div>
          <h1 class="logo-text">QLink</h1>
        </div>
        <p class="subtitle">安全的去中心化即时通讯</p>
      </div>

      <!-- 登录方式选择器 -->
      <div class="login-method-selector">
        <div class="method-tabs">
          <button 
            :class="['method-tab', { active: loginMethod === 'did' }]"
            @click="loginMethod = 'did'"
          >
            DID登录
          </button>
          <button 
            :class="['method-tab', { active: loginMethod === 'plugin' }]"
            @click="loginMethod = 'plugin'"
          >
            插件登录
          </button>
        </div>
      </div>

      <!-- DID手动登录 -->
      <div v-if="loginMethod === 'did'" class="manual-login">
        <div class="form-header">
          <h3>DID身份登录</h3>
          <p>使用您的DID身份和私钥登录</p>
        </div>
        
        <form @submit.prevent="loginWithDID" class="login-form">
          <div class="form-group">
            <label for="did">DID身份</label>
            <input
              id="did"
              v-model="manualDID"
              type="text"
              class="form-input"
              placeholder="输入您的DID (例如: did:qlink:123...)"
              required
            />
          </div>
          
          <div class="form-group">
            <label for="privateKey">私钥</label>
            <input
              id="privateKey"
              v-model="privateKey"
              type="password"
              class="form-input"
              placeholder="输入您的私钥"
              required
            />
          </div>
          
          <div class="form-actions">
            <button 
              type="submit" 
              class="login-btn"
              :disabled="loggingIn || !manualDID || !privateKey"
            >
              {{ loggingIn ? '登录中...' : '登录' }}
            </button>
            <button 
              type="button" 
              class="register-btn"
              @click="goToRegister"
            >
              注册新DID
            </button>
          </div>
        </form>
        
        <div class="plugin-download-hint">
          <p>推荐使用浏览器插件获得更好的体验</p>
          <button @click="goToPluginDownload" class="download-hint-btn">
            下载插件
          </button>
        </div>
      </div>

      <!-- 插件登录 -->
      <div v-else class="plugin-login">
        <div v-if="!pluginInstalled" class="plugin-notice">
          <div class="notice-icon">🔌</div>
          <h3>需要安装浏览器插件</h3>
          <p>QLink需要浏览器插件来管理您的DID身份和密钥</p>
          
          <div class="install-instructions">
            <h4>安装步骤：</h4>
            <div class="browser-tabs">
              <button 
                :class="['tab-btn', { active: selectedBrowser === 'chrome' }]"
                @click="selectedBrowser = 'chrome'"
              >
                Chrome
              </button>
              <button 
                :class="['tab-btn', { active: selectedBrowser === 'firefox' }]"
                @click="selectedBrowser = 'firefox'"
              >
                Firefox
              </button>
            </div>
            
            <div v-if="selectedBrowser === 'chrome'" class="install-guide">
              <ol>
                <li>下载插件文件夹</li>
                <li>打开Chrome扩展管理页面 (chrome://extensions/)</li>
                <li>开启"开发者模式"</li>
                <li>点击"加载已解压的扩展程序"</li>
                <li>选择下载的插件文件夹</li>
              </ol>
            </div>
            
            <div v-else class="install-guide">
              <ol>
                <li>下载插件ZIP包</li>
                <li>打开Firefox附加组件管理页面 (about:addons)</li>
                <li>点击设置按钮，选择"从文件安装附加组件"</li>
                <li>选择下载的ZIP文件</li>
              </ol>
            </div>
          </div>
          
          <div class="install-actions">
            <button @click="downloadPlugin" class="download-btn">
              {{ selectedBrowser === 'chrome' ? '下载插件文件夹' : '下载ZIP包' }}
            </button>
            <button @click="openInstallGuide" class="guide-btn">
              打开安装指南
            </button>
            <button @click="checkPlugin" class="check-btn">
              重新检测
            </button>
          </div>
        </div>

        <div v-else class="plugin-ready">
          <div class="ready-icon">✅</div>
          <h3>插件已安装</h3>
          <p>检测到 {{ userDID || 'DID身份' }}</p>
          
          <button 
            @click="loginWithPlugin" 
            class="login-btn"
            :disabled="connecting || loggingIn"
          >
            {{ connecting ? '连接中...' : loggingIn ? '登录中...' : '使用插件登录' }}
          </button>
        </div>
      </div>

      <!-- 错误信息 -->
      <div v-if="error" class="error-message">
        {{ error }}
      </div>

      <!-- 功能特性 -->
      <div class="features">
        <div class="feature-item">
          <span class="feature-icon">🔒</span>
          <span>端到端加密</span>
        </div>
        <div class="feature-item">
          <span class="feature-icon">🌐</span>
          <span>去中心化身份</span>
        </div>
        <div class="feature-item">
          <span class="feature-icon">🚀</span>
          <span>高性能通讯</span>
        </div>
      </div>
    </div>

    <!-- 背景装饰 -->
    <div class="background-decoration">
      <div class="decoration-circle circle-1"></div>
      <div class="decoration-circle circle-2"></div>
      <div class="decoration-circle circle-3"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { generateHMACSignature } from '@/utils/crypto'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

// 响应式数据
const pluginInstalled = ref(false)
const userDID = ref('')
const connecting = ref(false)
const loggingIn = ref(false)
const error = ref('')
const selectedBrowser = ref('chrome')
const loginMethod = ref('did') // 'did' 或 'plugin'
const manualDID = ref('')
const privateKey = ref('')

// 检查插件是否安装
const checkPlugin = async (showError = true) => {
  try {
    connecting.value = true
    error.value = ''
    
    // 检查插件是否存在
    if (typeof window.qlink === 'undefined') {
      pluginInstalled.value = false
      if (showError) {
        error.value = '未检测到QLink插件，请先安装插件'
      }
      return
    }
    
    // 获取用户DID
    const did = await window.qlink.getDID()
    if (did) {
      pluginInstalled.value = true
      userDID.value = did
    } else {
      pluginInstalled.value = false
      if (showError) {
        error.value = '插件中未找到DID身份，请先创建身份'
      }
    }
  } catch (err) {
    pluginInstalled.value = false
    if (showError) {
      error.value = '插件连接失败: ' + err.message
    }
  } finally {
    connecting.value = false
  }
}

// 使用插件登录（改为与后端一致的 HMAC-SHA256 签名）
const loginWithPlugin = async () => {
  try {
    loggingIn.value = true
    error.value = ''
    
    // 获取质询（统一使用 auth store）
    const challengeData = await getChallenge(userDID.value)
    
    // 使用与后端一致的 HMAC-SHA256 方案签名质询
    const signature = await generateHMACSignature(challengeData.challenge, userDID.value)
    
    // 验证签名并登录（由 auth store 完成持久化）
    const result = await verifySignature(userDID.value, challengeData, signature)
    
    if (result.success) {
      router.push('/chat')
    } else {
      error.value = '登录失败: ' + (result.message || '验证失败')
    }
  } catch (err) {
    error.value = '登录失败: ' + err.message
  } finally {
    loggingIn.value = false
  }
}

// DID手动登录（改为与后端一致的 HMAC-SHA256 签名）
const loginWithDID = async () => {
  try {
    loggingIn.value = true
    error.value = ''
    
    // 验证DID格式
    if (!isValidDID(manualDID.value)) {
      error.value = 'DID格式不正确'
      return
    }
    
    // 获取质询（统一使用 auth store）
    const challengeData = await getChallenge(manualDID.value)
    
    // 使用与后端一致的 HMAC-SHA256 方案签名质询
    const signature = await signChallenge(challengeData.challenge, manualDID.value)
    
    // 验证签名并登录（由 auth store 完成持久化）
    const result = await verifySignature(manualDID.value, challengeData, signature)
    
    if (result.success) {
      router.push('/chat')
    } else {
      error.value = '登录失败: ' + (result.message || '验证失败')
    }
  } catch (err) {
    error.value = '登录失败: ' + err.message
  } finally {
    loggingIn.value = false
  }
}

// 获取质询：支持完整DID或仅标识段（自动补齐前缀）
const getChallenge = async (did = null) => {
  try {
    let targetDID = (did || manualDID.value || '').trim()
    // 如果只传入最后一段，则自动补齐前缀
    if (targetDID && !targetDID.startsWith('did:')) {
      targetDID = `did:qlink:${targetDID}`
    }
    const resp = await authStore.createChallenge(targetDID)
    if (!resp.success) {
      throw new Error(resp.error || '获取质询失败')
    }
    return {
      challenge_id: resp.challenge_id,
      challenge: resp.challenge
    }
  } catch (error) {
    console.error('获取质询失败:', error)
    throw new Error('获取质询失败: ' + error.message)
  }
}

// 验证签名
const verifySignature = async (did, challengeData, signature) => {
  try {
    const result = await authStore.verifyChallenge(signature, did)
    return { success: result.success, message: result.error }
  } catch (error) {
    console.error('登录验证失败:', error)
    return { success: false, message: error.response?.data?.error || '登录验证失败' }
  }
}

// 与后端一致的 HMAC-SHA256 签名函数
const signChallenge = async (challenge, did) => {
  try {
    // 从challenge对象中提取nonce值
    const nonce = challenge.challenge || challenge
    // 基于 DID 的标识段派生密钥并做 HMAC-SHA256
    const signature = await generateHMACSignature(nonce, did)
    return signature
  } catch (error) {
    console.error('签名生成失败:', error)
    throw new Error('签名生成失败: ' + error.message)
  }
}

// 保留占位：若未来需要切换回真实ECDSA，可在此实现
const generateECDSASignatureLocal = async () => {
  throw new Error('当前登录流程使用HMAC-SHA256，不再生成ECDSA签名')
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

// 验证DID格式：允许完整DID或仅最后一段；最后一段需>8字符
const isValidDID = (input) => {
  if (!input) return false
  const did = input.trim()
  if (did.startsWith('did:')) {
    const parts = did.split(':')
    const last = parts[parts.length - 1]
    return parts.length >= 3 && last && last.length > 8
  }
  // 仅传入标识段
  return did.length > 8
}

// 跳转到注册页面
const goToRegister = () => {
  router.push('/blockchain')
}

// 跳转到插件下载页面
const goToPluginDownload = () => {
  router.push('/install')
}

// 下载插件
const downloadPlugin = () => {
  // 模拟下载插件
  const link = document.createElement('a')
  link.href = selectedBrowser.value === 'chrome' 
    ? '/downloads/qlink-chrome-extension.zip' 
    : '/downloads/qlink-firefox-extension.xpi'
  link.download = selectedBrowser.value === 'chrome' 
    ? 'qlink-chrome-extension.zip' 
    : 'qlink-firefox-extension.xpi'
  link.click()
}

// 打开安装指南
const openInstallGuide = () => {
  window.open('/install-guide', '_blank')
}

// 格式化DID显示
const formatDID = (did) => {
  if (!did) return ''
  if (did.length > 20) {
    return did.substr(0, 15) + '...' + did.substr(-5)
  }
  return did
}

// 获取头像文本
const getAvatarText = (did) => {
  if (!did) return 'U'
  const parts = did.split(':')
  return parts[parts.length - 1].substr(0, 2).toUpperCase()
}

// 组件挂载时检查插件
onMounted(() => {
  if (loginMethod.value === 'plugin') {
    checkPlugin(false)
  }
})
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 20px;
  position: relative;
  overflow: hidden;
}

.login-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 20px;
  padding: 40px;
  width: 100%;
  max-width: 450px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
  position: relative;
  z-index: 1;
  animation: fadeIn 0.6s ease-out;
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  margin-bottom: 10px;
}

.logo-icon {
  font-size: 32px;
  background: linear-gradient(135deg, #667eea, #764ba2);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.logo-text {
  font-size: 28px;
  font-weight: 700;
  background: linear-gradient(135deg, #667eea, #764ba2);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin: 0;
}

.subtitle {
  color: #666;
  font-size: 14px;
  margin: 0;
}

/* 登录方式选择器 */
.login-method-selector {
  margin-bottom: 25px;
}

.method-tabs {
  display: flex;
  background: #f5f5f5;
  border-radius: 12px;
  padding: 4px;
  gap: 4px;
}

.method-tab {
  flex: 1;
  padding: 12px 16px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #666;
  cursor: pointer;
  transition: all 0.3s ease;
}

.method-tab.active {
  background: white;
  color: #667eea;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.method-tab:hover:not(.active) {
  color: #333;
}

/* DID手动登录 */
.manual-login {
  animation: fadeIn 0.3s ease-out;
}

.form-header {
  text-align: center;
  margin-bottom: 25px;
}

.form-header h3 {
  margin: 0 0 8px 0;
  color: #333;
  font-size: 18px;
  font-weight: 600;
}

.form-header p {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  color: #333;
  font-weight: 500;
  font-size: 14px;
}

.form-input {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid #e1e5e9;
  border-radius: 10px;
  font-size: 14px;
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.form-actions {
  display: flex;
  gap: 12px;
  margin-top: 25px;
}

.login-btn {
  flex: 1;
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
  border: none;
  padding: 14px 24px;
  border-radius: 10px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
}

.login-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(102, 126, 234, 0.3);
}

.login-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.register-btn {
  background: transparent;
  color: #667eea;
  border: 2px solid #667eea;
  padding: 12px 20px;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.register-btn:hover {
  background: #667eea;
  color: white;
}

/* 插件下载提示 */
.plugin-download-hint {
  text-align: center;
  margin-top: 20px;
  padding: 15px;
  background: #f8f9ff;
  border-radius: 10px;
  border: 1px solid #e1e8ff;
}

.plugin-download-hint p {
  margin: 0 0 10px 0;
  color: #666;
  font-size: 13px;
}

.download-hint-btn {
  background: #667eea;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.download-hint-btn:hover {
  background: #5a6fd8;
}

/* 插件登录 */
.plugin-login {
  animation: fadeIn 0.3s ease-out;
}

.plugin-notice {
  text-align: center;
}

.notice-icon {
  font-size: 48px;
  margin-bottom: 15px;
}

.plugin-notice h3 {
  margin: 0 0 10px 0;
  color: #333;
  font-size: 18px;
  font-weight: 600;
}

.plugin-notice p {
  margin: 0 0 25px 0;
  color: #666;
  font-size: 14px;
  line-height: 1.5;
}

.install-instructions {
  text-align: left;
  background: #f8f9fa;
  padding: 20px;
  border-radius: 10px;
  margin-bottom: 20px;
}

.install-instructions h4 {
  margin: 0 0 15px 0;
  color: #333;
  font-size: 16px;
  font-weight: 600;
}

.browser-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 15px;
}

.tab-btn {
  padding: 8px 16px;
  border: 1px solid #ddd;
  background: white;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.tab-btn.active {
  background: #667eea;
  color: white;
  border-color: #667eea;
}

.install-guide ol {
  margin: 0;
  padding-left: 20px;
}

.install-guide li {
  margin-bottom: 8px;
  color: #555;
  font-size: 14px;
  line-height: 1.4;
}

.install-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.download-btn, .guide-btn, .check-btn {
  padding: 12px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.download-btn {
  background: #28a745;
  color: white;
  border: none;
}

.download-btn:hover {
  background: #218838;
}

.guide-btn {
  background: #17a2b8;
  color: white;
  border: none;
}

.guide-btn:hover {
  background: #138496;
}

.check-btn {
  background: transparent;
  color: #667eea;
  border: 2px solid #667eea;
}

.check-btn:hover {
  background: #667eea;
  color: white;
}

.plugin-ready {
  text-align: center;
}

.ready-icon {
  font-size: 48px;
  margin-bottom: 15px;
}

.plugin-ready h3 {
  margin: 0 0 10px 0;
  color: #28a745;
  font-size: 18px;
  font-weight: 600;
}

.plugin-ready p {
  margin: 0 0 25px 0;
  color: #666;
  font-size: 14px;
}

.error-message {
  background: #f8d7da;
  color: #721c24;
  padding: 12px 16px;
  border-radius: 8px;
  margin: 20px 0;
  font-size: 14px;
  border: 1px solid #f5c6cb;
}

.features {
  display: flex;
  justify-content: space-around;
  margin-top: 30px;
  padding-top: 25px;
  border-top: 1px solid #eee;
}

.feature-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #666;
}

.feature-icon {
  font-size: 20px;
}

.background-decoration {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
}

.decoration-circle {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
}

.circle-1 {
  width: 200px;
  height: 200px;
  top: -100px;
  right: -100px;
  animation: float 6s ease-in-out infinite;
}

.circle-2 {
  width: 150px;
  height: 150px;
  bottom: -75px;
  left: -75px;
  animation: float 8s ease-in-out infinite reverse;
}

.circle-3 {
  width: 100px;
  height: 100px;
  top: 50%;
  right: 10%;
  animation: float 10s ease-in-out infinite;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes float {
  0%, 100% {
    transform: translateY(0px);
  }
  50% {
    transform: translateY(-20px);
  }
}

@media (max-width: 768px) {
  .login-card {
    padding: 30px 25px;
    margin: 10px;
  }
  
  .form-actions {
    flex-direction: column;
  }
  
  .features {
    flex-direction: column;
    gap: 15px;
  }
  
  .feature-item {
    flex-direction: row;
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .login-card {
    padding: 30px 20px;
    margin: 10px;
  }
  
  .features {
    flex-direction: column;
    gap: 15px;
  }
  
  .feature-item {
    flex-direction: row;
    justify-content: center;
  }
}
</style>