import { ref, Ref } from 'vue'
import request from '../../../core/utils/request'
import { ElMessage } from 'element-plus'
import type { Resource, ResourceDependency } from '../../../core/types/resource'

export function useDependency(currentResource: Ref<Resource | null>) {
  const depDrawerVisible = ref(false)
  const depLoading = ref(false)
  const depTree = ref<ResourceDependency[]>([])
  
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
      const res = await request.get<ResourceDependency[]>(`/api/v1/resources/versions/${row.latest_version.id}/dependency-tree`)
      depTree.value = res || []
    } catch (err: any) {
    } finally {
      depLoading.value = false
    }
  }

  const downloadBundle = async () => {
    if (!currentResource.value?.latest_version?.id) return
    bundleLoading.value = true
    try {
      const data = await request.get<any>(`/api/v1/resources/versions/${currentResource.value.latest_version.id}/bundle`)
      
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `bundle-${currentResource.value.name}-${data.root_version}.json`
      a.click()
      URL.revokeObjectURL(url)
      
      ElMessage.success('依赖包清单已生成并下载')
    } catch (err: any) {
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

  const saveDependencies = async (vid: string, deps: { target_resource_id: string, constraint: string }[]) => {
    try {
      await request.patch(`/api/v1/resources/versions/${vid}/dependencies`, deps)
      ElMessage.success('资源依赖关联已成功更新')
      // Refresh tree
      if (currentResource.value) {
        await viewDependencies(currentResource.value, false)
      }
      return true
    } catch (err: any) {
      return false
    }
  }

  return {
    depDrawerVisible,
    depLoading,
    depTree,
    bundleLoading,
    packLoading,
    viewDependencies,
    downloadBundle,
    downloadSimPack,
    saveDependencies
  }
}
