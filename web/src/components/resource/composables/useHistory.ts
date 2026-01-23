import { ref } from 'vue'
import request from '../../../core/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Resource, ResourceVersion } from '../../../core/types/resource'

export function useHistory(fetchList: () => void) {
  const historyDrawerVisible = ref(false)
  const historyLoading = ref(false)
  const versionHistory = ref<ResourceVersion[]>([])
  const currentResource = ref<Resource | null>(null)

  const viewHistory = async (row: Resource, openDrawer = true) => {
    currentResource.value = row
    if (openDrawer) historyDrawerVisible.value = true
    historyLoading.value = true
    try {
      const res = await request.get<ResourceVersion[]>(`/api/v1/resources/${row.id}/versions`)
      versionHistory.value = res || []
    } catch (err: any) {
    } finally {
      historyLoading.value = false
    }
  }

  const rollback = async (ver: ResourceVersion) => {
    if (!currentResource.value) return
    ElMessageBox.confirm(`确定要将版本切换回 ${ver.semver || 'v' + ver.version_num} 吗？此操作会影响所有下游依赖。`, '版本回溯确认', {
      type: 'warning'
    }).then(async () => {
      try {
        await request.post(`/api/v1/resources/${currentResource.value?.id}/latest`, {
          version_id: ver.id
        })
        ElMessage.success('版本切换成功')
        fetchList()
        viewHistory(currentResource.value!)
      } catch (err: any) {
      }
    })
  }

  return {
    historyDrawerVisible,
    historyLoading,
    versionHistory,
    currentResource, // exposed for filtering in template 'current line' logic
    viewHistory,
    rollback
  }
}
