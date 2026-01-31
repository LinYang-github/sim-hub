import { ref, computed } from 'vue'
import axios from 'axios'

interface Role {
  id: string
  name: string
  key: string
  permissions: string[]
}

interface User {
  id: string
  username: string
  role: Role
}

const currentUser = ref<User | null>(null)
const loading = ref(false)

export function useAuth() {
  const permissions = computed(() => currentUser.value?.role?.permissions || [])

  const hasPermission = (permission: string) => {
    if (permissions.value.includes('*')) return true
    return permissions.value.includes(permission)
  }

  const fetchCurrentUser = async () => {
    const token = localStorage.getItem('simhub_token')
    const username = localStorage.getItem('simhub_user')
    if (!token || !username) {
      currentUser.value = null
      return
    }

    loading.value = true
    try {
      // 这里的接口通过 Authorization Header 自动识别身份
      const res = await axios.get('/api/v1/auth/me')
      currentUser.value = res.data
    } catch (err) {
      console.error('Failed to fetch user profile', err)
      currentUser.value = null
    } finally {
      loading.value = false
    }
  }

  return {
    currentUser,
    permissions,
    hasPermission,
    fetchCurrentUser,
    loading
  }
}
