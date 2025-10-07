<template>
  <div class="install-container">
    <div class="install-card">
      <!-- 头部 -->
      <div class="header">
        <div class="logo">
          <div class="logo-icon">🔗</div>
          <h1 class="logo-text">QLink</h1>
        </div>
        <p class="subtitle">浏览器扩展安装指南</p>
      </div>

      <!-- 检测到插件时的提示 -->
      <div v-if="pluginInstalled" class="plugin-detected">
        <div class="success-icon">✅</div>
        <h3>插件已安装成功！</h3>
        <p>检测到QLink浏览器扩展已安装，正在跳转到登录页面...</p>
        <div class="loading-spinner"></div>
      </div>

      <!-- 插件未安装时的安装指南 -->
      <div v-else class="install-guide">
        <!-- 浏览器选择 -->
        <div class="browser-selector">
          <h2>选择您的浏览器</h2>
          <div class="browser-tabs">
            <button 
              class="tab-btn" 
              :class="{ active: selectedBrowser === 'chrome' }"
              @click="selectedBrowser = 'chrome'"
            >
              🌐 Chrome/Edge
            </button>
            <button 
              class="tab-btn" 
              :class="{ active: selectedBrowser === 'firefox' }"
              @click="selectedBrowser = 'firefox'"
            >
              🦊 Firefox
            </button>
          </div>
        </div>

        <!-- Chrome/Edge 安装说明 -->
        <div v-if="selectedBrowser === 'chrome'" class="install-section">
          <h3>Chrome/Edge 安装步骤</h3>
          <div class="steps">
            <div class="step">
              <h4>下载扩展文件</h4>
              <p>点击下方按钮下载QLink浏览器扩展文件。</p>
            </div>
            <div class="step">
              <h4>解压文件</h4>
              <p>将下载的ZIP文件解压到一个文件夹中。</p>
            </div>
            <div class="step">
              <h4>打开扩展管理页面</h4>
              <p>在浏览器地址栏输入 <code>chrome://extensions/</code> 或 <code>edge://extensions/</code></p>
            </div>
            <div class="step">
              <h4>启用开发者模式</h4>
              <p>在扩展管理页面右上角，打开"开发者模式"开关。</p>
            </div>
            <div class="step">
              <h4>加载扩展</h4>
              <p>点击"加载已解压的扩展程序"，选择解压后的文件夹。</p>
            </div>
          </div>
        </div>

        <!-- Firefox 安装说明 -->
        <div v-if="selectedBrowser === 'firefox'" class="install-section">
          <h3>Firefox 安装步骤</h3>
          <div class="steps">
            <div class="step">
              <h4>下载扩展文件</h4>
              <p>点击下方按钮下载QLink浏览器扩展文件。</p>
            </div>
            <div class="step">
              <h4>解压文件</h4>
              <p>将下载的ZIP文件解压到一个文件夹中。</p>
            </div>
            <div class="step">
              <h4>打开调试页面</h4>
              <p>在Firefox浏览器中输入 <code>about:debugging</code></p>
            </div>
            <div class="step">
              <h4>选择此Firefox</h4>
              <p>点击左侧的"此Firefox"选项。</p>
            </div>
            <div class="step">
              <h4>加载临时附加组件</h4>
              <p>点击"加载临时附加组件"按钮，选择解压文件夹中的 <code>manifest.json</code> 文件。</p>
            </div>
          </div>
        </div>

        <!-- 下载按钮 -->
        <div class="download-section">
          <h3>下载QLink扩展</h3>
          <div class="download-buttons">
            <a href="/qlink-extension-FINAL-1319.zip" class="download-btn primary" download>
              📦 下载扩展包 (推荐)
            </a>
            <button class="download-btn secondary" @click="downloadFolder">
              📁 查看文件列表
            </button>
          </div>
          <div class="help-note">
            <p>💡 建议下载扩展包，解压后按照上述步骤安装。如遇问题，可查看文件列表单独下载。</p>
          </div>
        </div>

        <!-- 检测按钮 -->
        <div class="check-section">
          <button class="check-btn" @click="checkPlugin" :disabled="checking">
            <span v-if="checking">🔄 检测中...</span>
            <span v-else>🔍 重新检测插件</span>
          </button>
        </div>

        <!-- 返回登录按钮 -->
        <div class="back-section">
          <button class="back-btn" @click="goToLogin">
            ← 返回登录页面
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()

// 响应式数据
const pluginInstalled = ref(false)
const checking = ref(false)
const selectedBrowser = ref('chrome')

// 检查插件是否安装
const checkPlugin = async () => {
  checking.value = true
  try {
    await new Promise(resolve => setTimeout(resolve, 1000)) // 模拟检测延迟
    
    if (window.qlink && window.qlink.isInstalled) {
      pluginInstalled.value = true
      // 延迟跳转到登录页面
      setTimeout(() => {
        router.push('/login')
      }, 2000)
    } else {
      pluginInstalled.value = false
    }
  } catch (err) {
    console.error('检查插件失败:', err)
    pluginInstalled.value = false
  } finally {
    checking.value = false
  }
}

// 下载文件夹
const downloadFolder = () => {
  window.open('/install.html', '_blank')
}

// 返回登录页面
const goToLogin = () => {
  router.push('/login')
}

// 页面加载时自动检测插件
onMounted(() => {
  checkPlugin()
  
  // 定期检测插件安装状态
  const interval = setInterval(() => {
    if (!pluginInstalled.value) {
      checkPlugin()
    } else {
      clearInterval(interval)
    }
  }, 3000) // 每3秒检测一次
})
</script>

<style scoped>
.install-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.install-card {
  background: white;
  border-radius: 20px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
  padding: 40px;
  max-width: 800px;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
}

/* 头部样式 */
.header {
  text-align: center;
  margin-bottom: 40px;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  margin-bottom: 16px;
}

.logo-icon {
  font-size: 48px;
}

.logo-text {
  font-size: 36px;
  font-weight: 700;
  color: #333;
  margin: 0;
}

.subtitle {
  color: #666;
  font-size: 18px;
  margin: 0;
}

/* 插件检测成功样式 */
.plugin-detected {
  text-align: center;
  padding: 40px;
  background: #e8f5e8;
  border-radius: 16px;
  border: 2px solid #4caf50;
}

.success-icon {
  font-size: 64px;
  margin-bottom: 20px;
}

.plugin-detected h3 {
  color: #2e7d32;
  margin: 0 0 16px 0;
  font-size: 24px;
}

.plugin-detected p {
  color: #388e3c;
  margin: 0 0 20px 0;
  font-size: 16px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #e0e0e0;
  border-top: 4px solid #4caf50;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 浏览器选择器 */
.browser-selector {
  margin-bottom: 30px;
  text-align: center;
}

.browser-selector h2 {
  margin-bottom: 20px;
  color: #333;
  font-size: 20px;
}

.browser-tabs {
  display: flex;
  gap: 10px;
  justify-content: center;
  flex-wrap: wrap;
}

.tab-btn {
  padding: 12px 24px;
  border: 2px solid #e0e0e0;
  background: white;
  border-radius: 12px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 600;
  transition: all 0.3s;
}

.tab-btn.active {
  background: #667eea;
  color: white;
  border-color: #667eea;
}

.tab-btn:hover:not(.active) {
  border-color: #667eea;
  color: #667eea;
}

/* 安装步骤 */
.install-section {
  background: #f8f9fa;
  padding: 30px;
  border-radius: 16px;
  margin: 20px 0;
}

.install-section h3 {
  color: #667eea;
  margin-bottom: 20px;
  font-size: 20px;
  text-align: center;
}

.steps {
  counter-reset: step-counter;
}

.step {
  counter-increment: step-counter;
  margin-bottom: 20px;
  padding: 20px;
  background: white;
  border-radius: 12px;
  border-left: 4px solid #667eea;
  position: relative;
}

.step::before {
  content: counter(step-counter);
  position: absolute;
  left: -15px;
  top: 20px;
  background: #667eea;
  color: white;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  font-size: 14px;
}

.step h4 {
  margin-bottom: 10px;
  color: #333;
  font-size: 16px;
}

.step p {
  margin: 0;
  color: #666;
  line-height: 1.5;
}

.step code {
  background: #e9ecef;
  padding: 4px 8px;
  border-radius: 6px;
  font-family: 'Courier New', monospace;
  font-size: 14px;
  color: #495057;
}

/* 下载区域 */
.download-section {
  text-align: center;
  margin: 30px 0;
  padding: 30px;
  background: #e3f2fd;
  border-radius: 16px;
}

.download-section h3 {
  margin-bottom: 20px;
  color: #1976d2;
  font-size: 20px;
}

.download-buttons {
  display: flex;
  gap: 15px;
  justify-content: center;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.download-btn {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 15px 30px;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.3s;
}

.download-btn.primary {
  background: #4caf50;
  color: white;
}

.download-btn.primary:hover {
  background: #45a049;
  transform: translateY(-2px);
}

.download-btn.secondary {
  background: #2196f3;
  color: white;
}

.download-btn.secondary:hover {
  background: #1976d2;
  transform: translateY(-2px);
}

.help-note {
  background: #fff3e0;
  padding: 15px;
  border-radius: 8px;
  border-left: 4px solid #ff9800;
}

.help-note p {
  margin: 0;
  color: #e65100;
  font-size: 14px;
  line-height: 1.4;
}

/* 检测区域 */
.check-section {
  text-align: center;
  margin: 30px 0;
}

.check-btn {
  padding: 15px 30px;
  background: #2196f3;
  color: white;
  border: none;
  border-radius: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s;
}

.check-btn:hover:not(:disabled) {
  background: #1976d2;
  transform: translateY(-2px);
}

.check-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

/* 返回区域 */
.back-section {
  text-align: center;
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px solid #e0e0e0;
}

.back-btn {
  padding: 12px 24px;
  background: #f5f5f5;
  color: #666;
  border: 1px solid #ddd;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
}

.back-btn:hover {
  background: #e0e0e0;
  color: #333;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .install-card {
    padding: 20px;
    margin: 10px;
  }
  
  .logo-text {
    font-size: 28px;
  }
  
  .logo-icon {
    font-size: 40px;
  }
  
  .download-buttons {
    flex-direction: column;
  }
  
  .download-btn {
    width: 100%;
  }
}
</style>