import { ref } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import type { Resource } from './useResourceList'

export function useDependency(currentResource: { value: Resource | null }) {
  const depDrawerVisible = ref(false)
  const depLoading = ref(false)
  const depTree = ref<any[]>([])
  
  const bundleLoading = ref(false)
  const packLoading = ref(false)

  const viewDependencies = async (row: Resource, openDrawer = true) => {
    if (!row.latest_version?.id) {
      ElMessage.warning('未能获取该资源的版本信息')
      return
    }
    // Update the ref passed from useHistory or a shared ref
    currentResource.value = row 
    
    if (openDrawer) depDrawerVisible.value = true
    depLoading.value = true
    try {
      const res = await axios.get(`/api/v1/resources/versions/${row.latest_version.id}/dependency-tree`)
      depTree.value = Array.isArray(res.data) ? res.data : []
    } catch (err: any) {
      ElMessage.error('获取依赖树失败: ' + (err.response?.data?.error || err.message))
    } finally {
      depLoading.value = false
    }
  }

  const downloadBundle = async () => {
    if (!currentResource.value?.latest_version?.id) return
    bundleLoading.value = true
    try {
      const res = await axios.get(`/api/v1/resources/versions/${currentResource.value.latest_version.id}/bundle`)
      const data = res.data
      
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `bundle-${currentResource.value.name}-${data.root_version}.json`
      a.click()
      URL.revokeObjectURL(url)
      
      ElMessage.success('依赖包清单已生成并下载')
    } catch (err: any) {
      ElMessage.error('生成打包清单失败: ' + (err.response?.data?.error || err.message))
    } finally {
      bundleLoading.value = false
    }
  }

  const downloadSimPack = async () => {
    if (!currentResource.value?.latest_version?.id) return
    const vid = currentResource.value.latest_version.id
    const downloadUrl = `/api/v1/resources/versions/${vid}/download-pack`
    window.open(downloadUrl, '_blank')
    ElMessage.success('已开始生成离线包并下载')
  }

  return {
    depDrawerVisible,
    depLoading,
    depTree,
    bundleLoading,
    packLoading,
    viewDependencies,
    downloadBundle,
    downloadSimPack
  }
}
