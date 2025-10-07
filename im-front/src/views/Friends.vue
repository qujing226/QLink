<template>
  <div class="friends-container">
    <!-- 侧边栏 -->
    <div class="sidebar">
      <div class="sidebar-header">
        <h2>好友管理</h2>
        <button class="add-friend-btn" @click="showAddFriendModal = true" title="添加好友">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
          </svg>
        </button>
      </div>

      <!-- 搜索栏 -->
      <div class="search-container">
        <input 
          v-model="searchQuery" 
          type="text" 
          placeholder="搜索好友..." 
          class="search-input"
        />
      </div>

      <!-- 好友请求 -->
      <div v-if="filteredRequests.length > 0" class="section">
        <div class="section-header">
          <h3>好友请求</h3>
          <span class="count-badge">{{ filteredRequests.length }}</span>
        </div>
        <div class="requests-list">
          <div 
            v-for="request in filteredRequests" 
            :key="request.id"
            class="request-item"
          >
            <div class="request-avatar">
              {{ getAvatarText(request.user_did) }}
            </div>
            <div class="request-info">
              <div class="request-name">{{ formatDID(request.user_did) }}</div>
              <div class="request-time">{{ formatTime(request.created_at) }}</div>
            </div>
            <div class="request-actions">
              <button 
                @click="acceptFriend(request.user_did)" 
                :disabled="processing"
                class="accept-btn"
                title="接受"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                </svg>
              </button>
              <button 
                @click="rejectFriend(request.user_did)" 
                :disabled="processing"
                class="reject-btn"
                title="拒绝"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 好友列表 -->
      <div class="section">
        <div class="section-header">
          <h3>我的好友</h3>
          <span class="count-badge">{{ filteredFriends.length }}</span>
        </div>
        <div class="friends-list">
          <div 
            v-for="friend in filteredFriends" 
            :key="friend.id"
            class="friend-item"
            @click="startChat(friend.friend_did)"
          >
            <div class="friend-avatar">
              {{ getAvatarText(friend.friend_did) }}
              <div class="online-indicator"></div>
            </div>
            <div class="friend-info">
              <div class="friend-name">{{ formatDID(friend.friend_did) }}</div>
              <div class="friend-status">在线</div>
            </div>
            <div class="friend-actions">
              <button 
                @click.stop="removeFriend(friend.friend_did)" 
                :disabled="processing"
                class="remove-btn"
                title="删除好友"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 主内容区域 -->
    <div class="main-content">
      <div class="welcome-screen">
        <svg width="120" height="120" viewBox="0 0 24 24" fill="currentColor">
          <path d="M16 4c0-1.11.89-2 2-2s2 .89 2 2-.89 2-2 2-2-.89-2-2zm4 18v-6h2.5l-2.54-7.63A1.5 1.5 0 0 0 18.54 8H16c-.8 0-1.54.37-2.01.99L12 11l-1.99-2.01A2.5 2.5 0 0 0 8 8H5.46c-.8 0-1.54.37-2.01.99L1 14.5V22h2v-6h2l2-6 2 6h2v6h4z"/>
        </svg>
        <h3>好友管理</h3>
        <p>在这里管理您的好友关系，添加新好友或处理好友请求</p>
        <div class="quick-actions">
          <button @click="showAddFriendModal = true" class="primary-btn">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M15 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm-9-2V7H4v3H1v2h3v3h2v-3h3v-2H6zm9 4c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
            </svg>
            添加好友
          </button>
          <button @click="$router.push('/chat')" class="secondary-btn">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M20 2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h4v3c0 .6.4 1 1 1 .2 0 .5-.1.7-.3L14.4 18H20c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/>
            </svg>
            开始聊天
          </button>
        </div>
      </div>
    </div>

    <!-- 添加好友弹窗 -->
    <div v-if="showAddFriendModal" class="modal-overlay" @click="showAddFriendModal = false">
      <div class="add-friend-modal" @click.stop>
        <div class="modal-header">
          <h3>添加好友</h3>
          <button class="close-btn" @click="showAddFriendModal = false">×</button>
        </div>
        <div class="modal-content">
          <form @submit.prevent="addFriend" class="add-friend-form">
            <div class="form-group">
              <label for="friendDid">好友DID地址</label>
              <input 
                id="friendDid"
                v-model="newFriendDid" 
                type="text" 
                placeholder="输入好友的DID地址..." 
                :disabled="processing"
                class="form-input"
                required
              />
            </div>
            <div class="form-actions">
              <button type="button" @click="showAddFriendModal = false" class="cancel-btn">
                取消
              </button>
              <button type="submit" :disabled="!newFriendDid.trim() || processing" class="submit-btn">
                {{ processing ? '发送中...' : '发送请求' }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="error-toast">{{ error }}</div>
    
    <!-- 成功提示 -->
    <div v-if="success" class="success-toast">{{ success }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useFriendsStore } from '../stores/friends'

const router = useRouter()
const authStore = useAuthStore()
const friendsStore = useFriendsStore()

const showAddFriendModal = ref(false)
const newFriendDid = ref('')
const processing = ref(false)
const error = ref('')
const success = ref('')
const searchQuery = ref('')

const { friends, friendRequests } = friendsStore

// 过滤好友请求
const filteredRequests = computed(() => {
  if (!searchQuery.value) return friendRequests.value
  return friendRequests.value.filter(request => 
    (request.user_did || '').toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// 过滤好友列表
const filteredFriends = computed(() => {
  if (!searchQuery.value) return friends.value
  return friends.value.filter(friend => 
    friend.friend_did.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// 格式化DID显示
const formatDID = (did) => {
  if (!did) return ''
  if (did.length > 20) {
    return did.substring(0, 10) + '...' + did.substring(did.length - 6)
  }
  return did
}

// 获取头像文字
const getAvatarText = (did) => {
  if (!did) return 'U'
  return did.substring(4, 6).toUpperCase()
}

// 格式化时间
const formatTime = (timestamp) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

const loadData = async () => {
  try {
    await Promise.all([
      friendsStore.getFriends(),
      friendsStore.getFriendRequests()
    ])
  } catch (err) {
    error.value = '加载数据失败'
    console.error('Load data error:', err)
  }
}

const loadFriends = async () => {
  try {
    const response = await fetch('http://localhost:8082/api/v1/friends', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    
    if (response.ok) {
      const data = await response.json()
      friends.value = data.friends || []
    }
  } catch (err) {
    console.error('Load friends error:', err)
  }
}

const loadFriendRequests = async () => {
  try {
    const response = await fetch('http://localhost:8082/api/v1/friends/requests', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    
    if (response.ok) {
      const data = await response.json()
      friendRequests.value = data.requests || []
    }
  } catch (err) {
    console.error('Load friend requests error:', err)
  }
}

const addFriend = async () => {
  if (!newFriendDid.value.trim()) return
  
  processing.value = true
  error.value = ''
  success.value = ''
  
  try {
    const resp = await friendsStore.addFriend(newFriendDid.value.trim(), '你好，一起聊天吧')
    if (resp.success) {
      success.value = '好友请求已发送'
      showAddFriendModal.value = false
      newFriendDid.value = ''
      setTimeout(() => success.value = '', 3000)
    } else {
      error.value = resp.error || '发送好友请求失败'
    }
  } catch (err) {
    error.value = '网络错误，请重试'
    console.error('Add friend error:', err)
  }
  
  processing.value = false
}

const acceptFriend = async (friendDid) => {
  processing.value = true
  error.value = ''
  
  try {
    const resp = await friendsStore.acceptFriend(friendDid)
    if (resp.success) {
      success.value = '已接受好友请求'
      await loadData()
      setTimeout(() => success.value = '', 3000)
    } else {
      error.value = resp.error || '接受好友请求失败'
    }
  } catch (err) {
    error.value = '网络错误，请重试'
    console.error('Accept friend error:', err)
  }
  
  processing.value = false
}

const rejectFriend = async (friendDid) => {
  processing.value = true
  error.value = ''
  
  try {
    const resp = await friendsStore.rejectFriend(friendDid)
    if (resp.success) {
      success.value = '已拒绝好友请求'
      await loadData()
      setTimeout(() => success.value = '', 3000)
    } else {
      error.value = resp.error || '拒绝好友请求失败'
    }
  } catch (err) {
    error.value = '网络错误，请重试'
    console.error('Reject friend error:', err)
  }
  
  processing.value = false
}

const removeFriend = async (friendDid) => {
  if (!confirm('确定要屏蔽该好友吗？')) return
  
  processing.value = true
  error.value = ''
  
  try {
    const resp = await friendsStore.blockFriend(friendDid)
    if (resp.success) {
      success.value = '已屏蔽该好友'
      await loadData()
      setTimeout(() => success.value = '', 3000)
    } else {
      error.value = resp.error || '操作失败'
    }
  } catch (err) {
    error.value = '网络错误，请重试'
    console.error('Block friend error:', err)
  }
  
  processing.value = false
}

const startChat = (friendDid) => {
  router.push(`/chat?friend=${friendDid}`)
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.friends-container {
  display: flex;
  height: 100vh;
  background: #f5f5f5;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* 侧边栏 */
.sidebar {
  width: 320px;
  background: white;
  border-right: 1px solid #e5e5e5;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  padding: 20px;
  border-bottom: 1px solid #e5e5e5;
  background: white;
}

.sidebar-title {
  font-size: 20px;
  font-weight: 600;
  color: #333;
  margin: 0 0 16px 0;
}

.search-box {
  position: relative;
  margin-bottom: 16px;
}

.search-input {
  width: 100%;
  padding: 10px 16px 10px 40px;
  border: 1px solid #e5e5e5;
  border-radius: 20px;
  font-size: 14px;
  background: #f8f9fa;
  outline: none;
  transition: all 0.2s;
}

.search-input:focus {
  border-color: #0088cc;
  background: white;
}

.search-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: #999;
  font-size: 16px;
}

.add-friend-btn {
  width: 100%;
  padding: 10px;
  background: #0088cc;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.add-friend-btn:hover {
  background: #0077b3;
}

/* 好友请求区域 */
.requests-section {
  border-bottom: 1px solid #e5e5e5;
}

.section-header {
  padding: 16px 20px 8px;
  font-size: 14px;
  font-weight: 600;
  color: #666;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.request-item {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.2s;
}

.request-item:hover {
  background: #f8f9fa;
}

.request-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 16px;
  margin-right: 12px;
  flex-shrink: 0;
}

.request-info {
  flex: 1;
  min-width: 0;
}

.request-name {
  font-weight: 500;
  color: #333;
  font-size: 14px;
  margin-bottom: 2px;
}

.request-did {
  font-size: 12px;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.request-actions {
  display: flex;
  gap: 8px;
  margin-left: 8px;
}

.accept-btn, .reject-btn {
  padding: 6px 12px;
  border: none;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.accept-btn {
  background: #4caf50;
  color: white;
}

.accept-btn:hover {
  background: #45a049;
}

.reject-btn {
  background: #f44336;
  color: white;
}

.reject-btn:hover {
  background: #da190b;
}

/* 好友列表 */
.friends-section {
  flex: 1;
  overflow-y: auto;
}

.friend-item {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background 0.2s;
}

.friend-item:hover {
  background: #f8f9fa;
}

.friend-avatar {
  position: relative;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 16px;
  margin-right: 12px;
  flex-shrink: 0;
}

.online-indicator {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 12px;
  height: 12px;
  background: #4caf50;
  border: 2px solid white;
  border-radius: 50%;
}

.friend-info {
  flex: 1;
  min-width: 0;
}

.friend-name {
  font-weight: 500;
  color: #333;
  font-size: 14px;
  margin-bottom: 2px;
}

.friend-did {
  font-size: 12px;
  color: #999;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.friend-actions {
  display: flex;
  gap: 8px;
  opacity: 0;
  transition: opacity 0.2s;
}

.friend-item:hover .friend-actions {
  opacity: 1;
}

.chat-btn, .remove-btn {
  padding: 6px 12px;
  border: none;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.chat-btn {
  background: #0088cc;
  color: white;
}

.chat-btn:hover {
  background: #0077b3;
}

.remove-btn {
  background: #f44336;
  color: white;
}

.remove-btn:hover {
  background: #da190b;
}

/* 主内容区域 */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: white;
  padding: 40px;
}

.welcome-content {
  text-align: center;
  max-width: 400px;
}

.welcome-icon {
  font-size: 64px;
  color: #0088cc;
  margin-bottom: 24px;
}

.welcome-title {
  font-size: 24px;
  font-weight: 600;
  color: #333;
  margin-bottom: 12px;
}

.welcome-subtitle {
  font-size: 16px;
  color: #666;
  margin-bottom: 32px;
  line-height: 1.5;
}

.quick-actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.quick-action-btn {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.primary-btn {
  background: #0088cc;
  color: white;
}

.primary-btn:hover {
  background: #0077b3;
}

.secondary-btn {
  background: #f8f9fa;
  color: #333;
  border: 1px solid #e5e5e5;
}

.secondary-btn:hover {
  background: #e9ecef;
}

/* 弹窗样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: white;
  border-radius: 12px;
  padding: 24px;
  width: 400px;
  max-width: 90vw;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.modal-title {
  font-size: 18px;
  font-weight: 600;
  color: #333;
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: #999;
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: all 0.2s;
}

.close-btn:hover {
  background: #f0f0f0;
  color: #666;
}

.form-group {
  margin-bottom: 16px;
}

.form-label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #333;
  margin-bottom: 8px;
}

.form-input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.form-input:focus {
  border-color: #0088cc;
}

.modal-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  margin-top: 24px;
}

.cancel-btn, .submit-btn {
  padding: 10px 20px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.cancel-btn {
  background: #f8f9fa;
  color: #666;
  border: 1px solid #e5e5e5;
}

.cancel-btn:hover {
  background: #e9ecef;
}

.submit-btn {
  background: #0088cc;
  color: white;
}

.submit-btn:hover {
  background: #0077b3;
}

.submit-btn:disabled {
  background: #ccc;
  cursor: not-allowed;
}

/* 消息提示 */
.message {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 12px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  z-index: 1001;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.error-message {
  background: #f44336;
  color: white;
}

.success-message {
  background: #4caf50;
  color: white;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .friends-container {
    flex-direction: column;
  }
  
  .sidebar {
    width: 100%;
    height: auto;
    border-right: none;
    border-bottom: 1px solid #e5e5e5;
  }
  
  .main-content {
    flex: 1;
    padding: 20px;
  }
  
  .quick-actions {
    flex-direction: column;
  }
  
  .modal {
    width: 90vw;
    margin: 20px;
  }
}

/* 滚动条样式 */
.friends-section::-webkit-scrollbar {
  width: 6px;
}

.friends-section::-webkit-scrollbar-track {
  background: #f1f1f1;
}

.friends-section::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.friends-section::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>