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
  typeKey: string,
  enableScope: boolean,
  selectedCategoryId: Ref<string>
) {
  const resources = ref<Resource[]>([])
  const loading = ref(false)
  const activeScope = ref<'ALL' | 'PRIVATE' | 'PUBLIC'>(enableScope ? 'ALL' : 'PUBLIC')
  const searchQuery = ref('')
  const syncing = ref(false)

  const fetchList = async (overrideTypeKey?: string) => {
    loading.value = true
    try {
      const currentKey = overrideTypeKey || typeKey
      const params: any = { 
        type: currentKey,
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
        // Backend logic: if scope missing and owner_id present -> public + private for that user
        params.owner_id = currentUserId
      }

      const res = await axios.get('/api/v1/resources', { params })
      resources.value = res.data.items || []
    } catch (err: any) {
      ElMessage.error('获取列表失败: ' + (err.response?.data?.error || err.message))
    } finally {
      loading.value = false
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

  watch([selectedCategoryId, activeScope], () => {
    fetchList()
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
