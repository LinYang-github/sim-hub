import { ref, watch, Ref } from 'vue'
import request from '../../../core/utils/request'
import type { Resource } from '../../../core/types/resource'
import { RESOURCE_SCOPE, ROOT_CATEGORY_ID, DEFAULT_ADMIN_ID } from '../../../core/constants/resource'

export function useResourceList(
  typeKey: Ref<string>,
  enableScope: Ref<boolean>,
  selectedCategoryId: Ref<string>
) {
  const resources = ref<Resource[]>([])
  const loading = ref(false)
  const activeScope = ref<keyof typeof RESOURCE_SCOPE>(RESOURCE_SCOPE.ALL)
  const searchQuery = ref('')
  const syncing = ref(false)

  // 用于解决竞态问题：记录最后一次发起的请求标识
  let lastRequestId = 0

  const fetchList = async () => {
    const requestId = ++lastRequestId
    loading.value = true
    
    try {
      const params: Record<string, string> = { 
        type: typeKey.value,
        name: searchQuery.value 
      }
      
      if (selectedCategoryId.value !== ROOT_CATEGORY_ID) {
        params.category_id = selectedCategoryId.value
      }

      const currentUserId = DEFAULT_ADMIN_ID
      if (activeScope.value === RESOURCE_SCOPE.PRIVATE) {
        params.scope = RESOURCE_SCOPE.PRIVATE
        params.owner_id = currentUserId
      } else if (activeScope.value === RESOURCE_SCOPE.PUBLIC) {
        params.scope = RESOURCE_SCOPE.PUBLIC
      } else {
        params.owner_id = currentUserId
      }

      const res = await request.get<{ items: Resource[] }>('/api/v1/resources', { params })
      
      // 只有最后一次发起的请求才有效，避免旧请求残留数据覆盖新数据
      if (requestId === lastRequestId) {
        resources.value = res.items || []
      }
    } catch (err: any) {
      // 错误由拦截器统一处理
    } finally {
      if (requestId === lastRequestId) {
        loading.value = false
      }
    }
  }

  const syncFromStorage = async () => {
    syncing.value = true
    try {
      const res = await request.post<{ count: number }>('/api/v1/resources/sync')
      // 这里可以保留特定的业务成功提示
      // ElMessage.success(`同步完成，共恢复 ${res.count} 个资源`)
      fetchList()
    } catch (err: any) {
    } finally {
      syncing.value = false
    }
  }

  // 监听所有可能触发列表刷新的响应式变量，并立即执行一次首屏加载
  watch([typeKey, selectedCategoryId, activeScope, searchQuery], () => {
    fetchList()
  }, { immediate: true })

  // 当 enableScope 配置变化时同步 internal state
  watch(enableScope, () => {
    activeScope.value = RESOURCE_SCOPE.ALL
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
