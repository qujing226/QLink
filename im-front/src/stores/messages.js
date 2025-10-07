import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'
import { generateKyberEncapsulate, generateSharedSecretFromCiphertext, encryptMessage } from '../utils/crypto'

const API_BASE = 'http://localhost:8082/api/v1'

export const useMessagesStore = defineStore('messages', () => {
  const messages = ref([])
  const conversations = ref([])
  const currentChat = ref(null)
  const websocket = ref(null)
  const pendingExchanges = ref([])
  const sessionKeys = ref({}) // { [friendDID]: sessionKey }

  const connectWebSocket = (token) => {
    // WebSocket 路由为 /api/v1/ws，参数要求 did
    const storedUser = localStorage.getItem('qlink_user')
    const did = storedUser ? JSON.parse(storedUser).did : null
    const wsUrl = `ws://localhost:8082/api/v1/ws?did=${encodeURIComponent(did || '')}`
    websocket.value = new WebSocket(wsUrl)
    
    websocket.value.onopen = () => {
      console.log('WebSocket connected')
    }
    
    websocket.value.onmessage = (event) => {
      const message = JSON.parse(event.data)
      if (message.type === 'message') {
        messages.value.push(message.data)
      }
    }
    
    websocket.value.onclose = () => {
      console.log('WebSocket disconnected')
    }
    
    websocket.value.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  const disconnectWebSocket = () => {
    if (websocket.value) {
      websocket.value.close()
      websocket.value = null
    }
  }

  const getMessages = async (friendDID, limit = 50, offset = 0) => {
    try {
      const response = await axios.get(`${API_BASE}/messages`, {
        params: { friend_did: friendDID, limit, offset }
      })
      const raw = response.data?.messages || []
      // 统一前端消息结构以匹配现有Chat.vue模板字段
      messages.value = raw.map(item => ({
        id: item.id,
        sender_did: item.from,
        content: item.content || '[encrypted]',
        created_at: item.timestamp,
        encrypted: !!item.encrypted,
        status: item.status || 'sent'
      }))
      return { success: true }
    } catch (error) {
      console.error('Get messages failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Get messages failed' 
      }
    }
  }

  const sendMessage = async (friendDID, content) => {
    try {
      // 确保存在会话密钥
      let key = sessionKeys.value[friendDID]
      if (!key) {
        const fetched = await getSession(friendDID)
        if (!fetched.success) {
          return { success: false, error: '会话密钥不存在，请先完成密钥交换' }
        }
        key = sessionKeys.value[friendDID]
      }

      // 使用AES-GCM加密消息，得到密文与nonce
      const { ciphertext, nonce } = await encryptMessage(content, key)

      await axios.post(`${API_BASE}/messages/send`, {
        to: friendDID,
        ciphertext,
        nonce
      })

      // 重新拉取最新消息列表，保证展示服务端解密后的明文
      await getMessages(friendDID)
      return { success: true }
    } catch (error) {
      console.error('Send message failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Send message failed' 
      }
    }
  }

  const markAsRead = async (messageId) => {
    try {
      await axios.put(`${API_BASE}/messages/${messageId}/read`)
      return { success: true }
    } catch (error) {
      console.error('Mark as read failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Mark as read failed' 
      }
    }
  }

  const createSession = async (friendDID, sessionKey) => {
    try {
      // 后端路由为 /sessions，参数命名为下划线风格：friend_did, session_key
      await axios.post(`${API_BASE}/sessions`, {
        friend_did: friendDID,
        session_key: sessionKey
      })
      // 缓存到前端，供加密使用
      sessionKeys.value[friendDID] = sessionKey
      return { success: true }
    } catch (error) {
      console.error('Create session failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Create session failed' 
      }
    }
  }

  const getSession = async (friendDID) => {
    try {
      const resp = await axios.get(`${API_BASE}/sessions/${encodeURIComponent(friendDID)}`)
      const key = resp.data?.session?.session_key
      if (key) {
        sessionKeys.value[friendDID] = key
        return { success: true, sessionKey: key }
      }
      return { success: false, error: 'Session not found' }
    } catch (error) {
      return { success: false, error: error.response?.data?.error || 'Get session failed' }
    }
  }

  // 获取目标用户的格加密公钥
  const getLatticePublicKey = async (did) => {
    try {
      const response = await axios.get(`${API_BASE}/auth/lattice-pubkey/${encodeURIComponent(did)}`)
      return { success: true, publicKey: response.data?.lattice_public_key }
    } catch (error) {
      console.error('Get lattice public key failed:', error)
      return {
        success: false,
        error: error.response?.data?.error || 'Get lattice public key failed'
      }
    }
  }

  // 发起密钥交换：Bob 侧
  const enableEncryption = async (targetDID) => {
    try {
      // 获取目标用户的格加密公钥
      const pkResp = await getLatticePublicKey(targetDID)
      if (!pkResp.success || !pkResp.publicKey) {
        throw new Error(pkResp.error || '无法获取对方格加密公钥')
      }

      // Kyber 封装生成密文（同时本地持有共享密钥）
      const { ciphertext, sharedSecret } = await generateKyberEncapsulate(pkResp.publicKey)

      // 发送到后端，创建密钥交换记录
      const response = await axios.post(`${API_BASE}/key-exchange/initiate`, {
        target_did: targetDID,
        ciphertext
      })

      // 为当前用户建立本地会话密钥（便于后续加密发送）
      const sess = await createSession(targetDID, sharedSecret)
      if (!sess.success) {
        console.warn('创建本地会话密钥失败：', sess.error)
      }

      return { success: true, data: response.data }
    } catch (error) {
      console.error('Enable encryption failed:', error)
      return {
        success: false,
        error: error.response?.data?.error || error.message || 'Enable encryption failed'
      }
    }
  }

  // 获取待处理的密钥交换：Alice 侧
  const getPendingKeyExchanges = async () => {
    try {
      const response = await axios.get(`${API_BASE}/key-exchange/pending`)
      pendingExchanges.value = response.data || []
      return { success: true, exchanges: pendingExchanges.value }
    } catch (error) {
      console.error('Get pending key exchanges failed:', error)
      return {
        success: false,
        error: error.response?.data?.error || 'Get pending key exchanges failed'
      }
    }
  }

  // 完成密钥交换（通知后端记录完成）
  const completeKeyExchange = async (id) => {
    try {
      await axios.post(`${API_BASE}/key-exchange/${id}/complete`)
      // 刷新待处理列表
      await getPendingKeyExchanges()
      return { success: true }
    } catch (error) {
      console.error('Complete key exchange failed:', error)
      return {
        success: false,
        error: error.response?.data?.error || 'Complete key exchange failed'
      }
    }
  }

  // 从密文与私钥推导共享密钥（前端本地）
  const deriveSharedSecret = async (ciphertext, privateKey) => {
    try {
      const secret = await generateSharedSecretFromCiphertext(ciphertext, privateKey)
      return { success: true, sharedSecret: secret }
    } catch (error) {
      console.error('Derive shared secret failed:', error)
      return { success: false, error: error.message || 'Derive shared secret failed' }
    }
  }

  const getConversations = async () => {
    try {
      // 后端暂未提供 /conversations 路由，此处使用好友列表构造会话视图
      const response = await axios.get(`${API_BASE}/friends`)
      const friends = response.data?.friends || []
      conversations.value = (friends || [])
        .filter(f => (f.status || '').toLowerCase() === 'accepted')
        .map(f => ({
          participant_did: f.friend_did || f.FriendDID || '',
          updated_at: f.updated_at || new Date().toISOString(),
          last_message: '',
          unread_count: 0,
          online: false
        }))
      return { success: true }
    } catch (error) {
      console.error('Get conversations failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Get conversations failed' 
      }
    }
  }

  const createConversation = async (participantDID) => {
    try {
      // 构造本地会话对象（后端无会话创建路由）
      const existing = conversations.value.find(c => c.participant_did === participantDID)
      if (existing) return { success: true, conversation: existing }
      const conv = {
        participant_did: participantDID,
        updated_at: new Date().toISOString(),
        last_message: '',
        unread_count: 0,
        online: false
      }
      conversations.value.push(conv)
      return { success: true, conversation: conv }
    } catch (error) {
      console.error('Create conversation failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Create conversation failed' 
      }
    }
  }

  const connect = () => {
    const token = localStorage.getItem('qlink_token')
    if (token) {
      connectWebSocket(token)
    }
  }

  const disconnect = () => {
    disconnectWebSocket()
  }

  // 注意：旧版 initiateKeyExchange 已废弃，改用 enableEncryption + 完成交换流程

  return {
    messages,
    conversations,
    currentChat,
    websocket,
    pendingExchanges,
    sessionKeys,
    connectWebSocket,
    disconnectWebSocket,
    getMessages,
    sendMessage,
    markAsRead,
    createSession,
    getSession,
    getConversations,
    createConversation,
    connect,
    disconnect,
    // 加密相关导出
    getLatticePublicKey,
    enableEncryption,
    getPendingKeyExchanges,
    completeKeyExchange,
    deriveSharedSecret
  }
})