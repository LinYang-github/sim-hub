import { ref } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Resource } from './useResourceList'

export function useHistory(fetchList: () => void) {
  const historyDrawerVisible = ref(false)
  const historyLoading = ref(false)
  const versionHistory = ref<any[]>([])
  const currentResource = ref<Resource | null>(null)

  const viewHistory = async (row: Resource, openDrawer = true) => {
    currentResource.value = row
    if (openDrawer) historyDrawerVisible.value = true
    historyLoading.value = true
    try {
      const res = await axios.get(`/api/v1/resources/${row.id}/versions`)
      versionHistory.value = res.data || []
    } catch (err: any) {
      ElMessage.error('获取历史失败: ' + (err.response?.data?.error || err.message))
    } finally {
      historyLoading.value = false
    }
  }

  const rollback = async (ver: any) => {
    if (!currentResource.value) return
    ElMessageBox.confirm(`确定要将版本切换回 ${ver.semver || 'v' + ver.version_num} 吗？此操作会影响所有下游依赖。`, '版本回溯确认', {
      type: 'warning'
    }).then(async () => {
      try {
        await axios.post(`/api/v1/resources/${currentResource.value?.id}/latest`, {
          version_id: ver.id
        })
        ElMessage.success('版本切换成功')
        fetchList()
        viewHistory(currentResource.value!)
      } catch (err: any) {
        ElMessage.error('切换失败: ' + (err.response?.data?.error || err.message))
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
