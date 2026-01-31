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
  files?: string[]
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
      let firstImageFile: File | null = null
      const imageExtensions = ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.bmp']
      
      files.forEach(file => {
        zip.file(file.webkitRelativePath, file)
        if (!firstImageFile && imageExtensions.some(ext => file.name.toLowerCase().endsWith(ext))) {
          firstImageFile = file
        }
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
        filename: rootFolderName + '.zip',
        files: files.map(f => f.webkitRelativePath)
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

  const performUpload = async (
    displayName: string, 
    blob: Blob, 
    contentType: string, 
    filename: string, 
    categoryIdVal: string, 
    fileList?: string[]
  ) => {
    const res = await request.post<{ ticket_id: string; presigned_url: string }>('/api/v1/integration/upload/token', {
      resource_type: typeKey.value,
      size: blob.size,
      checksum: 'skip-for-now',
      filename: encodeURIComponent(filename) // 使用 URL 编码确保所有特殊字符（包括中文、括号等）在签名时保持一致
    })
    
    const { ticket_id, presigned_url } = res
    
    // 使用新的 axios 实例上传，避免全局拦截器注入 Authorization Header
    // MinIO/S3 若检测到 Presigned URL + Auth Header 同时存在会报 400 错误
    await axios.create().put(presigned_url, blob, {
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
      extra_meta: {
        files: fileList
      }
    })
  }

  const confirmAndDoUpload = async () => {
    if (!pendingUploadData.value) return
    const { displayName, blob, contentType, filename } = pendingUploadData.value
    const categoryIdVal = selectedCategoryId.value
    
    uploading.value = true
    try {
      const PART_SIZE = 5 * 1024 * 1024
      const MULTIPART_THRESHOLD = 50 * 1024 * 1024
      
      if (blob.size <= MULTIPART_THRESHOLD) {
        await performUpload(
          displayName, 
          blob, 
          contentType, 
          filename, 
          categoryIdVal, 
          pendingUploadData.value.files
        )
      } else {
        await performMultipartUpload(
           displayName,
           blob,
           filename,
           categoryIdVal,
           PART_SIZE,
           pendingUploadData.value.files
        )
      }
      
      ElMessage.success('任务已提交并自动关联依赖')
      uploadConfirmVisible.value = false
      pendingUploadData.value = null
      onSuccess()
    } catch (e: any) {
      console.error(e)
      ElMessage.error(e.message || '上传失败')
    } finally {
      uploading.value = false
    }
  }

  const performMultipartUpload = async (
    displayName: string,
    blob: Blob,
    filename: string,
    categoryIdVal: string,
    partSize: number,
    fileList?: string[]
  ) => {
    // 1. Init
    const partCount = Math.ceil(blob.size / partSize)
    const initRes = await request.post<{ upload_id: string; ticket_id: string; object_key: string }>('/api/v1/integration/upload/multipart/init', {
      resource_type: typeKey.value,
      filename: encodeURIComponent(filename),
      part_count: partCount
    })
    
    const { upload_id, ticket_id, object_key } = initRes

    // 2. Concurrent Upload
    const parts: { part_number: number; etag: string }[] = []
    const concurrency = 4
    let uploadedBytes = 0
    let activeWorkers = 0
    let nextPartIndex = 0
    let hasError = false
    let errorMsg = ''

    return new Promise<void>((resolve, reject) => {
       const startWorker = async () => {
         if (hasError) return
         if (nextPartIndex >= partCount) {
           if (activeWorkers === 0) {
             // All done
             finalize()
           }
           return
         }

         const partNum = nextPartIndex + 1
         const offset = nextPartIndex * partSize
         nextPartIndex++
         activeWorkers++

         try {
           const chunk = blob.slice(offset, Math.min(offset + partSize, blob.size))
           
           // Get URL
           const urlRes = await request.post<{ url: string }>('/api/v1/integration/upload/multipart/part-url', {
             upload_id,
             ticket_id,
             part_number: partNum
           })
           
           // Upload Part
           const uploadRes = await axios.create().put(urlRes.url, chunk, {
             headers: { 'Content-Type': 'application/octet-stream' }
           })
           
           let etag = uploadRes.headers['etag']
           if (etag && etag.startsWith('"') && etag.endsWith('"')) {
             etag = etag.substring(1, etag.length - 1)
           }
           parts.push({ part_number: partNum, etag })
           
           uploadedBytes += chunk.size
           uploadPercent.value = Math.round((uploadedBytes * 100) / blob.size)

           activeWorkers--
           startWorker()
         } catch(e: any) {
           hasError = true
           errorMsg = e.message
           reject(e)
         }
       }
       
       const finalize = async () => {
         try {
           // Sort parts
           parts.sort((a, b) => a.part_number - b.part_number)
           
           await request.post('/api/v1/integration/upload/multipart/complete', {
             upload_id,
             ticket_id,
             object_key,
             parts,
             type_key: typeKey.value,
             category_id: categoryIdVal === ROOT_CATEGORY_ID ? '' : categoryIdVal,
             name: displayName,
             owner_id: DEFAULT_ADMIN_ID,
             scope: 'public',
             semver: uploadForm.value.semver,
             dependencies: uploadForm.value.dependencies.map(d => ({
                target_resource_id: d.id,
                constraint: 'latest'
             })),
             extra_meta: { files: fileList }
           })
           resolve()
         } catch (e) {
           reject(e)
         }
       }

       // Kick off initial workers
       for (let i = 0; i < Math.min(concurrency, partCount); i++) {
         startWorker()
       }
    })
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
