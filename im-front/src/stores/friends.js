import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

const API_BASE = 'http://localhost:8082/api/v1'

export const useFriendsStore = defineStore('friends', () => {
  const friends = ref([])
  const friendRequests = ref([])

  const getFriends = async () => {
    try {
      const response = await axios.get(`${API_BASE}/friends`)
      // 后端返回形如 { friends: [...] }
      friends.value = response.data?.friends || []
      return { success: true }
    } catch (error) {
      console.error('Get friends failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Get friends failed' 
      }
    }
  }

  const getFriendRequests = async () => {
    try {
      const response = await axios.get(`${API_BASE}/friends/requests`)
      friendRequests.value = response.data?.requests || []
      return { success: true }
    } catch (error) {
      console.error('Get friend requests failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Get friend requests failed' 
      }
    }
  }

  const addFriend = async (friendDID, message) => {
    try {
      // 后端期望 JSON 为下划线小写：friend_did、message
      await axios.post(`${API_BASE}/friends/add`, {
        friend_did: friendDID,
        message: message
      })
      return { success: true }
    } catch (error) {
      console.error('Add friend failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Add friend failed' 
      }
    }
  }

  const acceptFriend = async (friendDID) => {
    try {
      await axios.post(`${API_BASE}/friends/accept`, { friend_did: friendDID })
      await getFriends()
      await getFriendRequests()
      return { success: true }
    } catch (error) {
      console.error('Accept friend failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Accept friend failed' 
      }
    }
  }

  const rejectFriend = async (friendDID) => {
    try {
      await axios.post(`${API_BASE}/friends/reject`, { friend_did: friendDID })
      await getFriendRequests()
      return { success: true }
    } catch (error) {
      console.error('Reject friend failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Reject friend failed' 
      }
    }
  }

  const blockFriend = async (friendDID) => {
    try {
      await axios.post(`${API_BASE}/friends/block`, { friend_did: friendDID })
      await getFriends()
      return { success: true }
    } catch (error) {
      console.error('Block friend failed:', error)
      return { 
        success: false, 
        error: error.response?.data?.error || 'Block friend failed' 
      }
    }
  }

  return {
    friends,
    friendRequests,
    getFriends,
    getFriendRequests,
    addFriend,
    acceptFriend,
    rejectFriend,
    blockFriend
  }
})