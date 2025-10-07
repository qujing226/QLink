import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'
import { generateDID, validateDID, generateKeyPair, signData, generateChallenge, storeKey, getStoredKey, removeStoredKey } from '../utils/crypto'

const API_BASE = 'http://localhost:8082/api/v1'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const token = ref(localStorage.getItem('qlink_token'))
  const isAuthenticated = ref(!!token.value)
  const challenge = ref('')
  const challengeId = ref('')
  const keyPair = ref(null)

  // 初始化时恢复用户信息
  if (token.value) {
    const storedUser = localStorage.getItem('qlink_user')
    if (storedUser) {
      user.value = JSON.parse(storedUser)
    }
    
    // 恢复密钥对
    const storedPrivateKey = getStoredKey('private_key')
    const storedPublicKey = getStoredKey('public_key')
    if (storedPrivateKey && storedPublicKey) {
      keyPair.value = {
        privateKey: storedPrivateKey,
        publicKey: storedPublicKey
      }
    }
  }

  // 生成新的密钥对
  const generateNewKeyPair = async () => {
    try {
      keyPair.value = await generateKeyPair()
      storeKey('private_key', keyPair.value.privateKey)
      storeKey('public_key', keyPair.value.publicKey)
      return keyPair.value
    } catch (error) {
      console.error('生成密钥对失败:', error)
      throw error
    }
  }

  // 获取或生成DID
  const getOrGenerateDID = () => {
    if (!keyPair.value) {
      throw new Error('密钥对未生成')
    }
    return generateDID(keyPair.value.publicKey)
  }

  // 创建挑战（支持传入DID，否则使用本地密钥派生）
  const createChallenge = async (providedDID = null) => {
    try {
      if (!keyPair.value && !providedDID) {
        await generateNewKeyPair()
      }
      
      const myDID = providedDID || getOrGenerateDID()
      
      const response = await axios.post(`${API_BASE}/auth/challenge`, {
        did: myDID
      })
      
      // 后端返回 challenge_id 与 challenge
      challengeId.value = response.data?.challenge_id || ''
      challenge.value = response.data?.challenge || ''
      return {
        success: true,
        challenge_id: challengeId.value,
        challenge: challenge.value
      }
    } catch (error) {
      console.error('创建挑战失败:', error)
      return {
        success: false,
        error: error.response?.data?.error || '创建挑战失败'
      }
    }
  }

  // 验证挑战并登录（支持传入DID，否则使用本地密钥派生）
  const verifyChallenge = async (signature, providedDID = null) => {
    try {
      if (!challengeId.value) {
        throw new Error('挑战ID不存在')
      }
      if (!keyPair.value && !providedDID) {
        throw new Error('密钥对未生成且未提供DID')
      }

      const myDID = providedDID || getOrGenerateDID()
      
      const response = await axios.post(`${API_BASE}/auth/verify`, {
        did: myDID,
        challenge_id: challengeId.value,
        signature: signature
      })

      const { token: newToken, user: userData } = response.data
      
      // 保存认证信息
      token.value = newToken
      user.value = { ...userData, did: myDID }
      isAuthenticated.value = true
      
      localStorage.setItem('qlink_token', newToken)
      localStorage.setItem('qlink_user', JSON.stringify(user.value))
      
      // 设置axios默认headers
      axios.defaults.headers.common['Authorization'] = `Bearer ${newToken}`
      
      return {
        success: true,
        user: user.value
      }
    } catch (error) {
      console.error('验证挑战失败:', error)
      return {
        success: false,
        error: error.response?.data?.error || '验证挑战失败'
      }
    }
  }

  // 使用DID和签名登录
  const login = async (did, publicKey, signature) => {
    try {
      if (!validateDID(did)) {
        throw new Error('无效的DID格式')
      }

      const response = await axios.post(`${API_BASE}/auth/login`, {
        did,
        public_key: publicKey,
        signature
      })

      const { token: newToken, user: userData } = response.data
      
      // 保存认证信息
      token.value = newToken
      user.value = userData
      isAuthenticated.value = true
      
      localStorage.setItem('qlink_token', newToken)
      localStorage.setItem('qlink_user', JSON.stringify(userData))
      
      // 设置axios默认headers
      axios.defaults.headers.common['Authorization'] = `Bearer ${newToken}`
      
      return {
        success: true,
        user: userData
      }
    } catch (error) {
      console.error('登录失败:', error)
      return {
        success: false,
        error: error.response?.data?.error || '登录失败'
      }
    }
  }

  // 设置认证信息（兼容旧版本调用）
  const setAuth = (authData) => {
    if (authData.token) {
      token.value = authData.token
      localStorage.setItem('qlink_token', authData.token)
      axios.defaults.headers.common['Authorization'] = `Bearer ${authData.token}`
    }
    
    if (authData.did) {
      user.value = { 
        ...user.value, 
        did: authData.did 
      }
      localStorage.setItem('qlink_user', JSON.stringify(user.value))
    }
    
    if (authData.isAuthenticated !== undefined) {
      isAuthenticated.value = authData.isAuthenticated
    }
  }

  const logout = () => {
    token.value = null
    user.value = null
    isAuthenticated.value = false
    
    localStorage.removeItem('qlink_token')
    localStorage.removeItem('qlink_user')
    removeStoredKey('private_key')
    removeStoredKey('public_key')
    keyPair.value = null
    challenge.value = ''
    delete axios.defaults.headers.common['Authorization']
  }

  return {
    user,
    token,
    isAuthenticated,
    keyPair,
    challenge,
    generateNewKeyPair,
    getOrGenerateDID,
    login,
    logout,
    createChallenge,
    verifyChallenge,
    setAuth
  }
})