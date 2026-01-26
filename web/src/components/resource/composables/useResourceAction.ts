import request from '../../../core/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Resource } from '../../../core/types/resource'

export function useResourceAction(fetchList: () => void) {
  
  const confirmDelete = (row: Resource) => {
    ElMessageBox.confirm(`确定要删除资源 "${row.name}" 吗？`, '警告', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    }).then(async () => {
      try {
        await request.delete(`/api/v1/resources/${row.id}`)
        ElMessage.success('删除成功')
        fetchList()
      } catch (err: any) {
      }
    })
  }

  const download = async (row: Resource) => {
    // 优先取主表 ID，兜底取嵌套版本 ID
    const versionId = row.latest_version_id || row.latest_version?.id
    
    if (!versionId) {
      ElMessage.warning('暂无可用版本')
      return
    }

    try {
      ElMessage.info('正在请求打包下载...')
      const response = await request.get(`/api/v1/resources/versions/${versionId}/download-pack`, {
        responseType: 'blob'
      })
      
      const url = window.URL.createObjectURL(new Blob([response as any]))
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', `bundle-${row.name}-${versionId}.zip`)
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch (e: any) {
      // Error handled by interceptor or here
      console.error('Download failed', e)
    }
  }

  const handleDownloadUrl = (url?: string) => {
    if (url) {
      window.open(url, '_blank')
    } else {
      ElMessage.warning('下载链接无效')
    }
  }

  const publishResource = (row: Resource) => {
    ElMessageBox.confirm(`确定要将资源 "${row.name}" 发布到公共库吗？发布后所有用户可见。`, '发布确认', {
      type: 'success',
      confirmButtonText: '确定发布',
      cancelButtonText: '取消'
    }).then(async () => {
      try {
        await request.patch(`/api/v1/resources/${row.id}/scope`, { scope: 'PUBLIC' })
        ElMessage.success('发布成功')
        fetchList()
      } catch (err: any) {
      }
    })
  }

  return {
    confirmDelete,
    download,
    handleDownloadUrl,
    publishResource
  }
}
