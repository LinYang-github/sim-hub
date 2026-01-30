<template>
  <div class="resource-preview">
    <template v-if="isActive && (downloadUrl || viewer?.startsWith('External:') || isOnlineService)">
      <component 
        :is="getViewerComponent(viewer)" 
        :url="finalUrl" 
        :force="force"
        :type-key="typeKey"
        :icon="icon"
        :icon-size="48"
        :meta-data="metaData"
        :resource="fullResource"
      />
    </template>
    <div v-else class="preview-fallback">
      <DefaultIconPreview :type-key="typeKey" :icon="icon" :icon-size="48" />
      <div v-if="!isActive" class="status-overlay">
        <el-tag size="small" :type="statusType">{{ statusText }}</el-tag>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, markRaw } from 'vue'
import DefaultIconPreview from './DefaultIconPreview.vue'
import { RESOURCE_STATE } from '../../../core/constants/resource'
import { moduleManager } from '../../../core/moduleManager'

const props = defineProps<{
  typeKey: string
  downloadUrl?: string
  state?: string
  statusText?: string
  force?: boolean
  viewer?: string
  icon?: string
  metaData?: Record<string, any>
  fullResource?: any
}>()

// 1. 定义异步组件 - 只有在使用时才会请求网络下载对应的 JS 包
const AsyncGLBPreview = defineAsyncComponent(() => import('./GLBPreview.vue'))
const AsyncImagePreview = defineAsyncComponent(() => import('./ImagePreview.vue'))
const AsyncVideoPreview = defineAsyncComponent(() => import('./VideoPreview.vue'))
const AsyncDocPreview = defineAsyncComponent(() => import('./DocPreview.vue'))
const AsyncGeoPreview = defineAsyncComponent(() => import('./GeoPreview.vue'))
const AsyncFolderPreview = defineAsyncComponent(() => import('./FolderPreview.vue'))
const AsyncExternalViewer = defineAsyncComponent(() => import('./ExternalViewer.vue'))
const AsyncJsonPreview = defineAsyncComponent(() => import('./JsonPreview.vue'))

// 2. 映射表，支持按需返回组件
const getViewerComponent = (name?: string) => {
  const resolvedName = name ? moduleManager.resolveViewer(name) : name
  
  // If explicitly requested External or starts with External:
  if (resolvedName?.startsWith('External:')) return AsyncExternalViewer

  const viewerMap: Record<string, any> = {
    'GLBPreview': AsyncGLBPreview,
    'ImagePreview': AsyncImagePreview,
    'VideoPreview': AsyncVideoPreview,
    'DocPreview': AsyncDocPreview,
    'GeoPreview': AsyncGeoPreview,
    'CesiumViewer': AsyncGeoPreview,
    'FolderPreview': AsyncFolderPreview,
    'ExternalViewer': AsyncExternalViewer,
    'JsonPreview': AsyncJsonPreview,
    'JsonTreeViewer': AsyncJsonPreview,
    'DefaultIconPreview': DefaultIconPreview
  }
  
  return (name && viewerMap[name]) ? markRaw(viewerMap[name]) : DefaultIconPreview
}

// Special logic for ExternalViewer URL
const finalUrl = computed(() => {
    const resolvedViewer = props.viewer ? moduleManager.resolveViewer(props.viewer) : props.viewer
    if (resolvedViewer?.startsWith('External:')) {
        return resolvedViewer.replace('External:', '')
    }
    return props.downloadUrl
})

const isActive = computed(() => props.state === RESOURCE_STATE.ACTIVE)

// For online services (like map_service), they might not have a downloadUrl but are still previewable via metaData
const isOnlineService = computed(() => props.viewer === 'CesiumViewer' || props.viewer === 'JsonTreeViewer')

const statusType = computed(() => {
  if (props.state === RESOURCE_STATE.PROCESSING) return 'primary'
  if (props.state === RESOURCE_STATE.FAILED) return 'danger'
  return 'info'
})
</script>

<style scoped>
.resource-preview {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.preview-fallback {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.status-overlay {
  margin-top: 8px;
}
</style>
