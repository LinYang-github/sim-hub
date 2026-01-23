import { ref, Ref } from 'vue'
import axios from 'axios'
import request from '../../../core/utils/request'
import JSZip from 'jszip'
import { ElMessage } from 'element-plus'

import type { Resource } from '../../../core/types/resource'
import { ROOT_CATEGORY_ID, DEFAULT_ADMIN_ID } from '../../../core/constants/resource'

export interface UploadFormState {
  semver: string
  dependencies: Resource[]
}

export interface PendingUploadData {
  displayName: string
  blob: Blob
  contentType: string
  filename: string
}

export function useUpload(
  typeKey: Ref<string>, 
  selectedCategoryId: Ref<string>,
  onSuccess: () => void
) {
  const uploading = ref(false)
  const compressing = ref(false)
  const progress = ref(0)
  const uploadPercent = ref(0)
  const currentFile = ref('')
  
  const pendingUploadData = ref<PendingUploadData | null>(null)
  const uploadConfirmVisible = ref(false)
  
  const uploadForm = ref<UploadFormState>({
    semver: 'v1.0.0',
    dependencies: []
  })

  // Search dependencies
  const searchLoading = ref(false)
  const searchResults = ref<Resource[]>([])

  const triggerFolderUpload = () => {
    document.getElementById('folderInput')?.click()
  }

  const triggerFileUpload = () => {
    document.getElementById('fileInput')?.click()
  }

  const handleFolderSelect = async (event: Event) => {
    const input = event.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return

    const files = Array.from(input.files)
    const rootFolderName = files[0].webkitRelativePath.split('/')[0]
    
    uploading.value = true
    compressing.value = true
    progress.value = 0

    try {
      const zip = new JSZip()
      files.forEach(file => {
        zip.file(file.webkitRelativePath, file)
      })

      const content = await zip.generateAsync({ 
        type: 'blob',
        compression: 'DEFLATE',
        compressionOptions: { level: 6 }
      }, (meta) => {
        progress.value = Number(meta.percent.toFixed(0))
        currentFile.value = meta.currentFile || ''
      })

      compressing.value = false
      
      pendingUploadData.value = {
        displayName: rootFolderName,
        blob: content,
        contentType: 'application/zip',
        filename: rootFolderName + '.zip'
      }
      uploadConfirmVisible.value = true
    } catch (e: any) {
      console.error(e)
      // 处理逻辑错误，不属于 API 错误，保留 ElMessage
      ElMessage.error('处理失败: ' + (e.message || '未知错误'))
    } finally {
      uploading.value = false
      input.value = ''
    }
  }

  const handleFileSelect = async (event: Event) => {
    const input = event.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return
    const file = input.files[0]
    
    uploading.value = true
    try {
      const nameWithoutExt = file.name.substring(0, file.name.lastIndexOf('.')) || file.name
      pendingUploadData.value = {
        displayName: nameWithoutExt,
        blob: file,
        contentType: file.type || 'application/octet-stream',
        filename: file.name
      }
      uploadConfirmVisible.value = true
    } catch (e: any) {
      console.error(e)
      ElMessage.error('上传失败: ' + (e.message || '未知错误'))
    } finally {
      uploading.value = false
      input.value = ''
    }
  }

  const searchTargetResources = async (query: string) => {
    if (query) {
      searchLoading.value = true
      try {
        const res = await request.get<{ items: Resource[] }>('/api/v1/resources', { params: { name: query } })
        searchResults.value = res.items || []
      } finally {
        searchLoading.value = false
      }
    } else {
      searchResults.value = []
    }
  }

  const performUpload = async (displayName: string, blob: Blob, contentType: string, filename: string, categoryIdVal: string) => {
    const res = await request.post<{ ticket_id: string; presigned_url: string }>('/api/v1/integration/upload/token', {
      resource_type: typeKey.value,
      checksum: 'skip-for-now',
      size: blob.size,
      filename: filename
    })
    
    const { ticket_id, presigned_url } = res
    
    await axios.put(presigned_url, blob, {
      headers: { 'Content-Type': contentType },
      onUploadProgress: (p) => {
        if (p.total) {
          uploadPercent.value = Math.round((p.loaded * 100) / p.total)
        }
      }
    })

    await request.post('/api/v1/integration/upload/confirm', {
      ticket_id,
      type_key: typeKey.value,
      category_id: categoryIdVal === ROOT_CATEGORY_ID ? '' : categoryIdVal,
      name: displayName,
      owner_id: DEFAULT_ADMIN_ID,
      size: blob.size,
      semver: uploadForm.value.semver,
      dependencies: uploadForm.value.dependencies.map(d => ({
        target_resource_id: d.id,
        constraint: 'latest'
      })),
      extra_meta: {}
    })
  }

  const confirmAndDoUpload = async () => {
    if (!pendingUploadData.value) return
    const { displayName, blob, contentType, filename } = pendingUploadData.value
    const categoryIdVal = selectedCategoryId.value
    
    uploading.value = true
    try {
      await performUpload(displayName, blob, contentType, filename, categoryIdVal)
      ElMessage.success('任务已提交并自动关联依赖')
      uploadConfirmVisible.value = false
      pendingUploadData.value = null
      onSuccess()
    } catch (e: any) {
      // 这里的错误会由拦截器处理，手动 catch 仅用于重置状态
    } finally {
      uploading.value = false
    }
  }

  return {
    uploading,
    compressing,
    progress,
    uploadPercent,
    currentFile,
    pendingUploadData,
    uploadConfirmVisible,
    uploadForm,
    searchLoading,
    searchResults,
    triggerFolderUpload,
    triggerFileUpload,
    handleFolderSelect,
    handleFileSelect,
    searchTargetResources,
    confirmAndDoUpload
  }
}
