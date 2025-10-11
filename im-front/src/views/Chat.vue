<template>
  <div class="chat-container">
    <!-- 会话列表 -->
    <div class="conversations-panel">
      <div class="panel-header">
        <div class="search-container">
          <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
          </svg>
          <input 
            v-model="searchQuery" 
            type="text" 
            placeholder="搜索" 
            class="search-input"
          />
        </div>
        <button @click="showAddFriendModal = true" class="add-friend-btn" title="添加好友">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
            <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
          </svg>
        </button>
      </div>
      
      <div class="conversations-list">
        <div 
          v-for="conv in filteredSidebarEntries" 
          :key="conv.participant_did"
          @click="selectConversationByDid(conv.participant_did)"
          :class="['conversation-item', { active: selectedConversation && selectedConversation.participant_did === conv.participant_did }]"
        >
          <div class="conversation-avatar">
            <div class="avatar-circle">
              {{ getAvatarText(conv.participant_did) }}
            </div>
            <div v-if="conv.online" class="online-indicator"></div>
          </div>
          <div class="conversation-content">
            <div class="conversation-header">
              <div class="conversation-name">{{ formatDID(conv.participant_did) }}</div>
              <div class="conversation-time">{{ formatTime(conv.updated_at) }}</div>
            </div>
            <div class="conversation-preview">
              <div class="last-message">
                {{ conv.last_message || '开始对话...' }}
              </div>
              <div v-if="conv.unread_count > 0" class="unread-badge">{{ conv.unread_count }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 聊天区域 -->
    <div class="chat-area">
      <!-- 空状态：没有任何会话或好友时，给出提示与入口 -->
      <div v-if="!selectedConversation && !hasSidebarEntries" class="welcome-screen">
        <div class="welcome-content">
          <p>开始聊天：在左侧添加或选择好友</p>
          <button class="welcome-cta" @click="showAddFriendModal = true">添加好友</button>
        </div>
      </div>
      <!-- 无选择时的纯背景 -->
      <div v-else-if="!selectedConversation" class="empty-chat"></div>
      
      <div v-else class="conversation-view">
        <!-- 聊天头部 -->
        <div class="chat-header">
          <div class="chat-user-info">
            <div class="chat-avatar">
              <div class="avatar-circle">
                {{ getAvatarText(selectedConversation.participant_did) }}
              </div>
              <div v-if="selectedConversation.online" class="online-indicator"></div>
            </div>
            <div class="chat-user-details">
              <div class="chat-user-name">{{ formatDID(selectedConversation.participant_did) }}</div>
              <div class="chat-user-status">
                {{ selectedConversation.online ? '在线' : '离线' }}
              </div>
            </div>
          </div>
          <div class="chat-actions">
            <button @click="initiateKeyExchange" :disabled="keyExchanging" class="action-btn" title="密钥协商">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12.65 10C11.83 7.67 9.61 6 7 6c-3.31 0-6 2.69-6 6s2.69 6 6 6c2.61 0 4.83-1.67 5.65-4H17v4h4v-4h2v-4H12.65zM7 14c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2z"/>
              </svg>
            </button>
            <button @click="openKeyExchangeCenter" class="action-btn" title="处理待交换密钥">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 17a2 2 0 0 0 2-2v-3a4 4 0 1 0-8 0v3a2 2 0 0 0 2 2h4zm6-6v3a6 6 0 0 1-12 0v-3a6 6 0 0 1 12 0zm-6-7a3 3 0 0 1 3 3v1H9V7a3 3 0 0 1 3-3z"/>
              </svg>
              <span v-if="(messagesStore.pendingExchanges?.value || []).length" class="pending-badge">{{ messagesStore.pendingExchanges.value.length }}</span>
            </button>
          </div>
        </div>

        <!-- 消息列表 -->
        <div class="messages-container" ref="messagesContainer">
          <div 
            v-for="message in messages" 
            :key="message.id"
            :class="['message-wrapper', { 'own': message.sender_did === (authStore.user ? authStore.user.did : '') }]"
          >
            <div class="message-bubble">
              <div class="message-content">{{ message.content }}</div>
              <div class="message-meta">
                <span class="message-time">{{ formatMessageTime(message.created_at) }}</span>
                <span v-if="message.sender_did === authStore.user?.did" class="message-status">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" opacity="0.6">
                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                  </svg>
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 输入区域 -->
        <div class="message-input-container">
          <form @submit.prevent="sendMessage" class="message-input-form">
            <div class="input-wrapper">
              <button type="button" class="attachment-btn" title="附件">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M16.5 6v11.5c0 2.21-1.79 4-4 4s-4-1.79-4-4V5c0-1.38 1.12-2.5 2.5-2.5s2.5 1.12 2.5 2.5v10.5c0 .55-.45 1-1 1s-1-.45-1-1V6H10v9.5c0 1.38 1.12 2.5 2.5 2.5s2.5-1.12 2.5-2.5V5c0-2.21-1.79-4-4-4S7 2.79 7 5v12.5c0 3.04 2.46 5.5 5.5 5.5s5.5-2.46 5.5-5.5V6h-1.5z"/>
                </svg>
              </button>
              <input 
                v-model="newMessage" 
                type="text" 
                placeholder="输入消息" 
                :disabled="sending"
                class="message-input"
                @keydown.enter.exact.prevent="sendMessage"
              />
              <button type="button" class="emoji-btn" title="表情">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zM15.5 11c.83 0 1.5-.67 1.5-1.5S16.33 8 15.5 8 14 8.67 14 9.5s.67 1.5 1.5 1.5zm-7 0c.83 0 1.5-.67 1.5-1.5S9.33 8 8.5 8 7 8.67 7 9.5 7.67 11 8.5 11zm3.5 6.5c2.33 0 4.31-1.46 5.11-3.5H6.89c.8 2.04 2.78 3.5 5.11 3.5z"/>
                </svg>
              </button>
            </div>
            <button 
              type="submit" 
              :disabled="!newMessage.trim() || sending" 
              class="send-btn"
              title="发送"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
                <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/>
              </svg>
            </button>
          </form>
        </div>
      </div>
    </div>

    <!-- 密钥交换中心弹窗 -->
    <div v-if="showKeyExchangeModal" class="modal-overlay" @click="showKeyExchangeModal = false">
      <div class="key-exchange-modal" @click.stop>
        <div class="modal-header">
          <h3>密钥交换中心</h3>
          <button class="close-btn" @click="showKeyExchangeModal = false">×</button>
        </div>
        <div class="modal-content">
          <div class="input-group">
            <label for="aliceKey">输入您的Kyber私钥</label>
            <input id="aliceKey" v-model="alicePrivateKey" type="text" placeholder="Kyber768私钥（base64）" class="did-input" />
          </div>

          <div v-if="messagesStore.pendingExchanges && messagesStore.pendingExchanges.value && messagesStore.pendingExchanges.value.length" class="pending-list">
            <div v-for="ex in (messagesStore.pendingExchanges ? messagesStore.pendingExchanges.value : [])" :key="ex.id" class="exchange-item">
              <div class="exchange-info">
                <div class="exchange-from">来自：{{ formatDID(ex.from) }}</div>
                <div class="exchange-ct">密文：<code>{{ ex.ciphertext.substring(0, 24) }}...</code></div>
                <div class="exchange-exp">有效期：{{ formatTime(ex.expires_at) }}</div>
              </div>
              <button class="complete-btn" @click="handleCompleteExchange(ex)">完成交换</button>
            </div>
          </div>
          <div v-else class="empty-state">
            <p>暂无待处理的密钥交换</p>
          </div>
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
          <div class="search-friend-section">
            <div class="input-group">
              <label for="friendDid">输入好友DID</label>
              <input 
                id="friendDid"
                v-model="friendDid" 
                type="text" 
                placeholder="did:qlink:..." 
                class="did-input"
                @keydown.enter="searchFriend"
              />
            </div>
            <button 
              @click="searchFriend" 
              :disabled="!friendDid.trim() || searching"
              class="search-btn"
            >
              <svg v-if="!searching" width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
              </svg>
              <div v-else class="loading-spinner"></div>
              {{ searching ? '搜索中...' : '搜索' }}
            </button>
          </div>
          
          <!-- 搜索结果 -->
          <div v-if="searchResult" class="search-result">
            <div class="friend-card">
              <div class="friend-avatar">
                <div class="avatar-circle">
                  {{ getAvatarText(searchResult.did) }}
                </div>
              </div>
              <div class="friend-info">
                <div class="friend-name">{{ formatDID(searchResult.did) }}</div>
                <div class="friend-status">{{ searchResult.online ? '在线' : '离线' }}</div>
              </div>
              <button 
                @click="addFriend" 
                :disabled="adding"
                class="confirm-add-btn"
              >
                <svg v-if="!adding" width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19 13h-6v6h-2v-6H5v-2h6V5h2v6h6v2z"/>
                </svg>
                <div v-else class="loading-spinner"></div>
                {{ adding ? '添加中...' : '添加好友' }}
              </button>
            </div>
          </div>
          
          <!-- 空状态 -->
          <div v-else-if="!searchResult && friendDid.trim() && !searching" class="empty-state">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.4">
              <circle cx="11" cy="11" r="8"/>
              <path d="M21 21l-4.35-4.35"/>
              <path d="M11 8v6"/>
              <path d="M8 11h6"/>
            </svg>
            <p>未找到该用户</p>
            <small>请检查DID是否正确</small>
          </div>
          
          <!-- 默认状态 -->
          <div v-else-if="!friendDid.trim()" class="default-state">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.6">
              <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/>
              <circle cx="9" cy="7" r="4"/>
              <path d="M22 21v-2a4 4 0 0 0-3-3.87"/>
              <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
            </svg>
            <p>输入好友的DID开始搜索</p>
            <small>DID格式：did:qlink:...</small>
          </div>
        </div>
      </div>
    </div>

    <div v-if="error" class="error-toast">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useMessagesStore } from '../stores/messages'
import { useFriendsStore } from '../stores/friends'
import { storeToRefs } from 'pinia'
import axios from 'axios'
const API_BASE = 'http://localhost:8082/api/v1'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const messagesStore = useMessagesStore()
const friendsStore = useFriendsStore()

const selectedConversation = ref(null)
const newMessage = ref('')
const sending = ref(false)
const keyExchanging = ref(false)
const error = ref('')
const messagesContainer = ref(null)
const searchQuery = ref('')
const showAddFriendModal = ref(false)
// 密钥交换（Alice侧）弹窗与输入
const showKeyExchangeModal = ref(false)
const alicePrivateKey = ref('')
let keyExchangePollTimer = null

// 添加好友相关变量
const friendDid = ref('')
const searching = ref(false)
const adding = ref(false)
const searchResult = ref(null)

// 使用 storeToRefs 保持响应式引用，避免解构丢失 .value
const { conversations, messages } = storeToRefs(messagesStore)

// 左侧列表：融合好友与会话，参考 Telegram 左侧栏
const sidebarEntries = computed(() => {
  const map = new Map()
  // 先放入会话信息（防御性处理，确保为数组）
  const convList = Array.isArray(conversations.value) ? conversations.value : []
  for (const c of convList) {
    map.set(c.participant_did, {
      participant_did: c.participant_did,
      online: !!c.online,
      last_message: c.last_message || '',
      updated_at: c.updated_at || 0
    })
  }
  // 再合并好友（如果没有对应会话也展示出来）
  const friendsList = (friendsStore.friends && Array.isArray(friendsStore.friends.value)) ? friendsStore.friends.value : []
  for (const f of friendsList) {
    const did = f.friend_did || f.did || f.participant_did || (typeof f === 'string' ? f : '')
    if (!did) continue
    if (!map.has(did)) {
      map.set(did, {
        participant_did: did,
        online: !!f.online,
        last_message: '',
        updated_at: 0
      })
    } else {
      // 合并在线状态
      const prev = map.get(did)
      map.set(did, { ...prev, online: prev.online || !!f.online })
    }
  }
  // 排序：按更新时间倒序
  return Array.from(map.values()).sort((a, b) => (b.updated_at || 0) - (a.updated_at || 0))
})

// 过滤左侧列表
const filteredSidebarEntries = computed(() => {
  if (!searchQuery.value) return sidebarEntries.value
  return sidebarEntries.value.filter(conv => 
    (conv.participant_did || '').toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

// 是否存在侧边列表条目（好友或会话）
const hasSidebarEntries = computed(() => filteredSidebarEntries.value.length > 0)

// 格式化DID显示
const formatDID = (did) => {
  if (!did) return ''
  if (did.length > 20) {
    return did.substring(0, 10) + '...' + did.substring(did.length - 6)
  }
  return did
}

// 获取头像文字（防御性处理，避免短字符串异常）
const getAvatarText = (did) => {
  if (!did || typeof did !== 'string') return 'U'
  // 优先使用标识符部分（did:method:identifier）
  const parts = did.split(':')
  const ident = parts.length > 2 ? parts[parts.length - 1] : did
  const trimmed = ident.trim()
  if (!trimmed) return 'U'
  return trimmed.slice(0, 2).toUpperCase()
}

// 格式化消息时间
const formatMessageTime = (timestamp) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now - date
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

// 初始化加载好友与会话列表，并根据参数或第一项进行选择
const loadConversations = async () => {
  try {
    error.value = ''

    // 临时放开鉴权以便预览，即使未登录也继续加载空列表

    // 拉取好友与会话数据
    await friendsStore.getFriends()
    await messagesStore.getConversations()

    // 如果URL中有 friend 参数，自动选择或创建会话
    const friendParam = route.query?.friend
    if (friendParam) {
      let conv = conversations.value.find(c => c.participant_did === friendParam)
      if (!conv) {
        const created = await messagesStore.createConversation(friendParam)
        if (created?.success) {
          await messagesStore.getConversations()
          conv = conversations.value.find(c => c.participant_did === friendParam)
        }
      }
      if (conv) {
        await selectConversation(conv)
        return
      }
    }

    // 如果URL带有 did 查询参数，优先选中该会话
    const didParam = route.query?.did
    if (didParam) {
      await selectConversationByDid(didParam)
      return
    }

    // 默认选中第一个会话，提升首屏可见性（防御性处理）
    const convArr = Array.isArray(messagesStore.conversations.value)
      ? messagesStore.conversations.value
      : []
    const first = convArr.length > 0 ? convArr[0] : null
    if (first) {
      await selectConversation(first)
    }
  } catch (e) {
    console.error('加载会话失败:', e)
    error.value = e.message || '加载会话失败'
  }
}

const isEncrypted = computed(() => {
  return selectedConversation.value?.encrypted || false
})

// 通过 DID 选择或创建会话
const selectConversationByDid = async (did) => {
  let conversation = conversations.value.find(c => c.participant_did === did)
  if (!conversation) {
    const result = await messagesStore.createConversation(did)
    if (result.success) {
      await messagesStore.getConversations()
      conversation = conversations.value.find(c => c.participant_did === did)
    }
  }
  if (conversation) {
    await selectConversation(conversation)
  }
}
const selectConversation = async (conversation) => {
  selectedConversation.value = conversation
  // 按后端规范使用 friend_did 拉取消息
  await messagesStore.getMessages(conversation.participant_did)

  // 标记对方发来的消息为已读
  try {
    const myDid = authStore.user?.did
    const msgList = Array.isArray(messages.value) ? messages.value : []
    for (const msg of msgList) {
      if (msg.sender_did !== myDid && msg.status !== 'read') {
        await messagesStore.markAsRead(msg.id)
      }
    }
  } catch (e) {
    console.warn('标记已读失败:', e)
  }
  
  // 滚动到底部
  nextTick(() => {
    scrollToBottom()
  })
}

const sendMessage = async () => {
  if (!newMessage.value.trim() || !selectedConversation.value) return
  
  sending.value = true
  error.value = ''
  
  const result = await messagesStore.sendMessage(
    selectedConversation.value.participant_did,
    newMessage.value.trim()
  )
  
  if (result.success) {
    newMessage.value = ''
    nextTick(() => {
      scrollToBottom()
    })
  } else {
    error.value = result.error
  }
  
  sending.value = false
}

const initiateKeyExchange = async () => {
  if (!selectedConversation.value) return
  
  keyExchanging.value = true
  error.value = ''
  
  const result = await messagesStore.enableEncryption(
    selectedConversation.value.participant_did
  )
  
  if (result.success) {
    selectedConversation.value.encrypted = true
  } else {
    error.value = result.error
  }
  
  keyExchanging.value = false
}

// 打开密钥交换中心（Alice侧查看并处理待交换密文）
const openKeyExchangeCenter = async () => {
  showKeyExchangeModal.value = true
  await messagesStore.getPendingKeyExchanges()
}

// 完成单条密钥交换
const handleCompleteExchange = async (exchange) => {
  try {
    if (!alicePrivateKey.value.trim()) {
      throw new Error('请输入Kyber私钥以完成交换')
    }
    const derived = await messagesStore.deriveSharedSecret(exchange.ciphertext, alicePrivateKey.value.trim())
    if (!derived.success) throw new Error(derived.error || '无法派生共享密钥')

    const sess = await messagesStore.createSession(exchange.from, derived.sharedSecret)
    if (!sess.success) throw new Error(sess.error || '创建会话失败')

    const done = await messagesStore.completeKeyExchange(exchange.id)
    if (!done.success) throw new Error(done.error || '通知完成失败')

    const conv = conversations.value.find(c => c.participant_did === exchange.from)
    if (conv) conv.encrypted = true

    await messagesStore.getPendingKeyExchanges()
    if ((messagesStore.pendingExchanges?.value || []).length === 0) {
      showKeyExchangeModal.value = false
      alicePrivateKey.value = ''
    }
  } catch (err) {
    error.value = err.message
  }
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const formatTime = (timestamp) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else {
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
  }
}

const logout = () => {
  messagesStore.disconnect()
  authStore.logout()
  router.push('/login')
}

// 搜索好友函数
const searchFriend = async () => {
  if (!friendDid.value.trim()) return
  
  searching.value = true
  searchResult.value = null
  error.value = ''
  
  try {
    // 调用后端搜索接口
    const resp = await axios.get(`${API_BASE}/users/search`, {
      params: { q: friendDid.value.trim() }
    })
    const users = resp.data?.users || []
    if (!users.length) {
      throw new Error('未找到该用户')
    }

    // 选择第一个匹配项或与输入相同的DID
    const match = users.find(u => u.did === friendDid.value.trim()) || users[0]
    searchResult.value = {
      did: match.did || friendDid.value.trim(),
      online: !!match.online,
      exists: true
    }
  } catch (err) {
    error.value = err.message
    searchResult.value = null
  } finally {
    searching.value = false
  }
}

// 添加好友函数
const addFriend = async () => {
  if (!searchResult.value) return
  
  adding.value = true
  error.value = ''
  
  try {
    // 调用后端添加好友接口
    const addResp = await friendsStore.addFriend(searchResult.value.did, '你好，一起聊天吧')
    if (!addResp.success) {
      throw new Error(addResp.error || '添加好友失败')
    }
    
    // 检查是否已存在会话
    const existingConversation = messagesStore.conversations.value.find(
      conv => conv.participant_did === searchResult.value.did
    )
    
    if (existingConversation) {
      // 如果已存在会话，直接选择
      selectConversation(existingConversation)
    } else {
      // 创建新会话（调用后端）
      const convResp = await messagesStore.createConversation(searchResult.value.did)
      if (!convResp.success) {
        throw new Error(convResp.error || '创建会话失败')
      }
      await messagesStore.getConversations()
      const created = messagesStore.conversations.value.find(
        conv => conv.participant_did === searchResult.value.did
      )
      if (created) selectConversation(created)
    }
    
    // 关闭弹窗
    showAddFriendModal.value = false
    friendDid.value = ''
    searchResult.value = null
    
  } catch (err) {
    error.value = err.message
  } finally {
    adding.value = false
  }
}

onMounted(() => {
  loadConversations()
  messagesStore.connect()
  // 轮询待处理密钥交换
  messagesStore.getPendingKeyExchanges()
  keyExchangePollTimer = setInterval(() => {
    messagesStore.getPendingKeyExchanges()
  }, 5000)
})

onUnmounted(() => {
  messagesStore.disconnect()
  if (keyExchangePollTimer) {
    clearInterval(keyExchangePollTimer)
    keyExchangePollTimer = null
  }
})
</script>

<style scoped>
:root { --brand-1: #667eea; --brand-2: #764ba2; }
.chat-container {
  display: flex;
  min-height: 100vh;
  background: #f6f7fb;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.conversations-panel {
  width: 300px;
  background: #fff;
  border-right: 1px solid #e1e5e9;
  display: flex;
  flex-direction: column;
  box-shadow: none;
  border-radius: 0;
}

.panel-header {
  padding: 10px;
  background: #fff;
  color: #333;
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #e1e8ed;
}

.sidebar-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
}

.add-friend-btn {
  background: #0088cc;
  border: none;
  border-radius: 8px;
  width: 32px;
  height: 32px;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s ease;
}

.add-friend-btn:hover {
  background: #0077b3;
}

.search-container {
  position: relative;
  flex: 1;
}

.search-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  color: #8b8b8b;
}

.search-input {
  width: 100%;
  height: 32px;
  padding: 0 12px 0 36px;
  border: 1px solid #e1e5e9;
  border-radius: 16px;
  background: #f8f9fa;
  font-size: 14px;
  outline: none;
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.search-input:focus {
  background: #fff;
  border-color: #0088cc;
}

.conversations-list {
  flex: 1;
  overflow-y: auto;
}

.conversation-item {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  cursor: pointer;
  transition: background-color 0.2s;
  position: relative;
}

.conversation-item:hover {
  background: #f8f9fa;
}

.conversation-item.active {
  background: #eaf5ff;
}

.conversation-item.active::after {
  content: '';
  position: absolute;
  right: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background: linear-gradient(135deg, var(--brand-1), var(--brand-2));
}

.conversation-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--brand-1) 0%, var(--brand-2) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 18px;
  margin-right: 12px;
  position: relative;
}

.online-indicator {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 12px;
  height: 12px;
  background: #4caf50;
  border: 2px solid white;
  border-radius: 50%;
}

.conversation-info {
  flex: 1;
  min-width: 0;
}

.conversation-name {
  font-weight: 500;
  color: #333;
  margin-bottom: 4px;
  font-size: 15px;
}

.last-message {
  color: #8b8b8b;
  font-size: 14px;
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conversation-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  margin-left: 8px;
}

.message-time {
  font-size: 12px;
  color: #8b8b8b;
}

.unread-badge {
  background: linear-gradient(135deg, var(--brand-1), var(--brand-2));
  color: white;
  border-radius: 12px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 500;
  min-width: 20px;
  text-align: center;
  box-shadow: 0 2px 6px rgba(102, 126, 234, 0.3);
}

.chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: white;
}

/* Flex scroll fixes: allow children to shrink for scrollable areas */
.conversations-panel,
.chat-area {
  min-height: 0;
}

.conversations-list,
.messages-container {
  min-height: 0;
}

.welcome-screen {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
.welcome-content {
  text-align: center;
  color: #666;
}
.welcome-cta {
  margin-top: 12px;
  background: #0088cc;
  color: white;
  border: none;
  border-radius: 20px;
  padding: 8px 16px;
  cursor: pointer;
}
.welcome-cta:hover { background: #0077b3; }
/* 无选择时的干净背景 */
.empty-chat {
  flex: 1;
  background: linear-gradient(135deg, #f5f7fa, #e9eff5);
}

.welcome-screen svg {
  margin-bottom: 20px;
  opacity: 0.5;
}

.welcome-screen h3 {
  margin: 0 0 8px 0;
  color: #333;
  font-weight: 500;
}

.welcome-screen p {
  margin: 0;
  font-size: 14px;
}

.chat-header {
  padding: 16px 20px;
  background: white;
  border-bottom: 1px solid #e1e5e9;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chat-header-info {
  display: flex;
  align-items: center;
}

.chat-header-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--brand-1) 0%, var(--brand-2) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  margin-right: 12px;
}

.chat-header-details h3 {
  margin: 0 0 2px 0;
  font-size: 16px;
  font-weight: 500;
  color: #333;
}

.chat-header-status {
  font-size: 13px;
  color: #8b8b8b;
  display: flex;
  align-items: center;
  gap: 4px;
}

.encryption-status {
  color: #4caf50;
  font-weight: 500;
}

.chat-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn {
  background: none;
  border: none;
  padding: 8px;
  border-radius: 50%;
  cursor: pointer;
  color: #8b8b8b;
  transition: all 0.2s;
  position: relative;
}

.action-btn:hover {
  background: #f0f0f0;
  color: #333;
}

.key-exchange-btn {
  background: #4caf50;
  color: white;
  padding: 8px 16px;
  border: none;
  border-radius: 20px;
  font-size: 13px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.key-exchange-btn:hover {
  background: #45a049;
}

.key-exchange-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: #f8f9fa;
}

.message-wrapper {
  display: flex;
  margin-bottom: 8px;
}

.message-wrapper.own {
  justify-content: flex-end;
}

.message-bubble {
  max-width: 70%;
  position: relative;
}

.message-content {
  padding: 12px 16px;
  border-radius: 18px;
  background: white;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  word-wrap: break-word;
  font-size: 15px;
  line-height: 1.4;
}

.message-wrapper.own .message-content {
  background: linear-gradient(135deg, var(--brand-1), var(--brand-2));
  color: white;
}

.message-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  margin-top: 4px;
  font-size: 12px;
  color: #8b8b8b;
}

.message-wrapper.own .message-meta {
  color: rgba(255, 255, 255, 0.8);
}

.encrypted-icon {
  font-size: 12px;
}

.message-status {
  display: flex;
  align-items: center;
}

.message-status svg {
  width: 14px;
  height: 14px;
}

.message-input-container {
  padding: 16px 20px;
  background: white;
  border-top: 1px solid #e1e5e9;
}

.message-input-form {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.input-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  background: #f8f9fa;
  border-radius: 24px;
  padding: 8px 16px;
  border: 1px solid #e1e5e9;
  transition: all 0.2s;
}

.input-wrapper:focus-within {
  border-color: #667eea;
  background: white;
}

.attachment-btn,
.emoji-btn {
  background: none;
  border: none;
  padding: 4px;
  color: #8b8b8b;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.2s;
}

.attachment-btn:hover,
.emoji-btn:hover {
  background: #e1e5e9;
  color: #333;
}

.message-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  padding: 8px 12px;
  font-size: 15px;
  resize: none;
  max-height: 120px;
}

.send-btn {
  background: linear-gradient(135deg, #667eea, #764ba2);
  border: none;
  border-radius: 10px;
  width: 40px;
  height: 40px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.send-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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

.add-friend-modal {
  background: white;
  border-radius: 12px;
  padding: 0;
  width: 400px;
  max-width: 90vw;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.modal-header {
  padding: 20px 24px;
  border-bottom: 1px solid #e1e5e9;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: #8b8b8b;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  padding: 24px;
  text-align: center;
}

.search-friend-section {
  margin-bottom: 24px;
}

.input-group {
  margin-bottom: 16px;
  text-align: left;
}

.input-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.did-input {
  width: 100%;
  padding: 12px 16px;
  border: 2px solid #e1e5e9;
  border-radius: 8px;
  font-size: 14px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.did-input:focus {
  outline: none;
  border-color: #54a3ff;
}

.search-btn {
  width: 100%;
  padding: 12px 20px;
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
  border: none;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.search-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.search-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid transparent;
  border-top: 2px solid currentColor;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.search-result {
  margin-bottom: 24px;
}

.friend-card {
  display: flex;
  align-items: center;
  padding: 16px;
  border: 1px solid #e1e5e9;
  border-radius: 12px;
  background: #f8f9fa;
  gap: 12px;
}

.friend-avatar .avatar-circle {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 16px;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
}

.friend-info {
  flex: 1;
  text-align: left;
}

.friend-name {
  font-weight: 500;
  font-size: 14px;
  color: #333;
  margin-bottom: 4px;
}

.friend-status {
  font-size: 12px;
  color: #8b8b8b;
}

.confirm-add-btn {
  padding: 8px 16px;
  background: #0088cc;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: background-color 0.2s;
}

.confirm-add-btn:hover:not(:disabled) {
  background: #0077b3;
}

.confirm-add-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.empty-state,
.default-state {
  text-align: center;
  padding: 32px 16px;
  color: #8b8b8b;
}

.empty-state p,
.default-state p {
  margin: 16px 0 8px 0;
  font-size: 16px;
  font-weight: 500;
}

.empty-state small,
.default-state small {
  font-size: 12px;
  color: #aaa;
}

.friend-suggestion {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
}

.friend-suggestion svg {
  margin-bottom: 16px;
  color: #8b8b8b;
}

.friend-suggestion p {
  margin: 0;
  color: #666;
  font-size: 15px;
}

.primary-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
}

.primary-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

/* 移除欢迎文案相关样式，保持主区域简洁 */

/* 密钥交换中心样式 */
.pending-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  background: #ff5252;
  color: white;
  border-radius: 10px;
  padding: 0 6px;
  font-size: 11px;
  line-height: 16px;
  min-width: 16px;
  text-align: center;
}

.key-exchange-modal {
  width: 600px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0,0,0,0.12);
  overflow: hidden;
}

.key-exchange-modal .modal-header {
  padding: 16px 20px;
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: white;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.key-exchange-modal .modal-content {
  padding: 20px;
}

.pending-list {
  margin-top: 12px;
}

.exchange-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border: 1px solid #e1e5e9;
  border-radius: 10px;
  margin-bottom: 12px;
}

.exchange-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #333;
}

.complete-btn {
  background: #4caf50;
  color: white;
  border: none;
  border-radius: 8px;
  padding: 8px 12px;
  cursor: pointer;
}

.chat-header {
  padding: 16px 20px;
  background: white;
  border-bottom: 1px solid #e1e5e9;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.chat-header-info {
  display: flex;
  align-items: center;
}

.chat-header-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  margin-right: 12px;
}

.chat-header-details h3 {
  margin: 0 0 2px 0;
  font-size: 16px;
  font-weight: 500;
  color: #333;
}

.chat-header-status {
  font-size: 13px;
  color: #8b8b8b;
  display: flex;
  align-items: center;
  gap: 4px;
}

.encryption-status {
  color: #4caf50;
  font-weight: 500;
}

.chat-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn {
  background: none;
  border: none;
  padding: 8px;
  border-radius: 50%;
  cursor: pointer;
  color: #8b8b8b;
  transition: all 0.2s;
}

.action-btn:hover {
  background: #f0f0f0;
  color: #333;
}

.key-exchange-btn {
  background: #4caf50;
  color: white;
  padding: 8px 16px;
  border: none;
  border-radius: 20px;
  font-size: 13px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.key-exchange-btn:hover {
  background: #45a049;
}

.key-exchange-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: #f8f9fa;
}

.message-wrapper {
  display: flex;
  margin-bottom: 8px;
}

.message-wrapper.own {
  justify-content: flex-end;
}

.message-bubble {
  max-width: 70%;
  position: relative;
}

.message-content {
  padding: 12px 16px;
  border-radius: 18px;
  background: white;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  word-wrap: break-word;
  font-size: 15px;
  line-height: 1.4;
}

.message-wrapper.own .message-content {
  background: linear-gradient(135deg, var(--brand-1), var(--brand-2));
  color: white;
}

.message-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  margin-top: 4px;
  font-size: 12px;
  color: #8b8b8b;
}

.message-wrapper.own .message-meta {
  color: rgba(255, 255, 255, 0.8);
}

.encrypted-icon {
  font-size: 12px;
}

.message-status {
  display: flex;
  align-items: center;
}

.message-status svg {
  width: 14px;
  height: 14px;
}

.message-input-container {
  padding: 16px 20px;
  background: white;
  border-top: 1px solid #e1e5e9;
}

.message-input-form {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.input-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  background: #f8f9fa;
  border-radius: 24px;
  padding: 8px 16px;
  border: 1px solid #e1e5e9;
  transition: all 0.2s;
}

.input-wrapper:focus-within {
  border-color: #667eea;
  background: white;
}

.attachment-btn,
.emoji-btn {
  background: none;
  border: none;
  padding: 4px;
  color: #8b8b8b;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.2s;
}

.attachment-btn:hover,
.emoji-btn:hover {
  background: #e1e5e9;
  color: #333;
}

.message-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  padding: 8px 12px;
  font-size: 15px;
  resize: none;
  max-height: 120px;
}

.send-btn {
  background: linear-gradient(135deg, var(--brand-1), var(--brand-2));
  border: none;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.3);
  transition: all 0.2s;
}

.send-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* 错误提示 */
.error-toast {
  position: fixed;
  top: 20px;
  right: 20px;
  background: #f44336;
  color: white;
  padding: 12px 20px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  z-index: 1001;
}

/* 滚动条样式 */
.conversations-list::-webkit-scrollbar,
.messages-container::-webkit-scrollbar {
  width: 6px;
}

.conversations-list::-webkit-scrollbar-track,
.messages-container::-webkit-scrollbar-track {
  background: transparent;
}

.conversations-list::-webkit-scrollbar-thumb,
.messages-container::-webkit-scrollbar-thumb {
  background: #ddd;
  border-radius: 3px;
}

.conversations-list::-webkit-scrollbar-thumb:hover,
.messages-container::-webkit-scrollbar-thumb:hover {
  background: #bbb;
}
</style>