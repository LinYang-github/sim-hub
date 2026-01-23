import { ref, watch, Ref } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

export interface Resource {
  id: string
  name: string
  tags: string[]
  owner_id: string
  scope: 'PRIVATE' | 'PUBLIC'
  created_at: string
  latest_version?: {
    id: string
    version_num: number
    semver?: string
    state: string
    meta_data?: any
    file_size?: number
    download_url?: string
  }
}

export function useResourceList(
  typeKey: Ref<string>,
  enableScope: Ref<boolean>,
  selectedCategoryId: Ref<string>
) {
  const resources = ref<Resource[]>([])
  const loading = ref(false)
  const activeScope = ref<'ALL' | 'PRIVATE' | 'PUBLIC'>(enableScope.value ? 'ALL' : 'PUBLIC')
  const searchQuery = ref('')
  const syncing = ref(false)

  // 用于解决竞态问题：记录最后一次发起的请求标识
  let lastRequestId = 0

  const fetchList = async () => {
    const requestId = ++lastRequestId
    loading.value = true
    
    try {
      const params: any = { 
        type: typeKey.value,
        name: searchQuery.value 
      }
      
      if (selectedCategoryId.value !== 'all') {
        params.category_id = selectedCategoryId.value
      }

      const currentUserId = 'admin'
      if (activeScope.value === 'PRIVATE') {
        params.scope = 'PRIVATE'
        params.owner_id = currentUserId
      } else if (activeScope.value === 'PUBLIC') {
        params.scope = 'PUBLIC'
      } else {
        params.owner_id = currentUserId
      }

      const res = await axios.get('/api/v1/resources', { params })
      
      // 只有最后一次发起的请求才有效，避免旧请求残留数据覆盖新数据
      if (requestId === lastRequestId) {
        resources.value = res.data.items || []
      }
    } catch (err: any) {
      if (requestId === lastRequestId) {
        ElMessage.error('获取列表失败: ' + (err.response?.data?.error || err.message))
      }
    } finally {
      if (requestId === lastRequestId) {
        loading.value = false
      }
    }
  }

  const syncFromStorage = async () => {
    syncing.value = true
    try {
      const res = await axios.post('/api/v1/resources/sync')
      ElMessage.success(`同步完成，共恢复 ${res.data.count} 个资源`)
      fetchList()
    } catch (err: any) {
      ElMessage.error('同步失败: ' + (err.response?.data?.error || err.message))
    } finally {
      syncing.value = false
    }
  }

  // 监听所有可能触发列表刷新的响应式变量，并立即执行一次首屏加载
  watch([typeKey, selectedCategoryId, activeScope], () => {
    fetchList()
  }, { immediate: true })

  // 当 enableScope 配置变化时同步 internal state
  watch(enableScope, (val) => {
    activeScope.value = val ? 'ALL' : 'PUBLIC'
  })

  return {
    resources,
    loading,
    activeScope,
    searchQuery,
    syncing,
    fetchList,
    syncFromStorage
  }
}
