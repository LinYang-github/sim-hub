<template>
  <div class="folder-preview-container">
    <div v-if="loading" class="folder-status">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>解析目录中...</span>
    </div>
    
    <div v-else-if="previewUrl" class="image-wrapper">
      <el-image 
        :src="previewUrl" 
        fit="cover" 
        class="preview-img"
        :preview-src-list="[previewUrl]"
      >
        <template #error>
           <div class="folder-fallback">
            <el-icon :size="48"><FolderOpened /></el-icon>
            <span class="folder-name">预览加载失败</span>
          </div>
        </template>
      </el-image>
    </div>

    <div v-else class="folder-fallback">
      <el-icon :size="48"><FolderOpened /></el-icon>
      <span class="folder-name">{{ typeName || '文件夹资源' }}</span>
      <div v-if="corsError" class="cors-hint">
        <el-tag type="danger" size="small">跨域策略限制</el-tag>
        <p>请在 MinIO 设置中配置 CORS 规则</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { FolderOpened, Loading } from '@element-plus/icons-vue'
import JSZip from 'jszip'

const props = defineProps<{
  metaData?: Record<string, any>
  url: string
  typeName?: string
}>()

const loading = ref(false)
const previewUrl = ref<string | null>(null)
const corsError = ref(false)

const cleanup = () => {
  if (previewUrl.value && previewUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(previewUrl.value)
  }
}

const findCoverInZip = async () => {
  if (!props.url) return
  
  cleanup()
  previewUrl.value = null
  corsError.value = false
  
  if (props.metaData?.cover_url) {
    previewUrl.value = props.metaData.cover_url
    return
  }

  loading.value = true
  try {
    // 强制使用 cors 模式并捕获可能的跨域错误
    const response = await fetch(props.url, { mode: 'cors' })
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    
    const arrayBuffer = await response.arrayBuffer()
    const zip = await JSZip.loadAsync(arrayBuffer)
    
    const imageExtensions = ['.png', '.jpg', '.jpeg', '.gif', '.webp', '.bmp']
    let coverFile: JSZip.JSZipObject | null = null
    
    // 按优先级寻找：包含 preview 的图片 -> 根目录下的第一张图 -> 任意目录下的第一张图
    const files = Object.entries(zip.files).filter(([path, f]) => 
      !f.dir && imageExtensions.some(ext => path.toLowerCase().endsWith(ext))
    )

    const priorityFile = files.find(([path]) => path.toLowerCase().includes('preview'))
    const rootFile = files.find(([path]) => !path.includes('/'))
    
    const targetFile = priorityFile?.[1] || rootFile?.[1] || files[0]?.[1]

    if (targetFile) {
      const blob = await targetFile.async('blob')
      previewUrl.value = URL.createObjectURL(blob)
    }
  } catch (err: any) {
    console.error('ZIP Preview Error:', err)
    // 如果是类型为 TypeError 的 fetch 失败，通常是 CORS 问题
    if (err.name === 'TypeError' || err.message.includes('fetch')) {
      corsError.value = true
    }
  } finally {
    loading.value = false
  }
}

watch(() => props.url, findCoverInZip, { immediate: true })
onUnmounted(cleanup)
</script>

<style scoped>
.folder-preview-container {
  width: 100%;
  height: 100%;
  background: var(--el-fill-color-light);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.folder-status {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.image-wrapper {
  width: 100%;
  height: 100%;
}

.preview-img {
  width: 100%;
  height: 100%;
}

.folder-fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--el-text-color-placeholder);
  text-align: center;
  padding: 0 20px;
}

.folder-name {
  font-size: 14px;
  font-weight: 500;
}

.cors-hint {
  margin-top: 8px;
  p {
    font-size: 11px;
    margin: 4px 0 0;
    color: var(--el-color-danger);
  }
}

.img-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}
</style>
