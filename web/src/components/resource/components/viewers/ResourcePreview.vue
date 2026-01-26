<template>
  <div class="resource-preview">
    <template v-if="isActive && downloadUrl">
      <component 
        :is="getViewerComponent(viewer)" 
        :url="downloadUrl" 
        :force="force"
        :type-key="typeKey"
        :icon="icon"
        :icon-size="48"
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
import { computed } from 'vue'
import GLBPreview from './GLBPreview.vue'
import DefaultIconPreview from './DefaultIconPreview.vue'
import { RESOURCE_STATE } from '../../../../core/constants/resource'

const props = defineProps<{
  typeKey: string
  downloadUrl?: string
  state?: string
  statusText?: string
  force?: boolean
  viewer?: string
  icon?: string
}>()

const viewerMap: Record<string, any> = {
  'GLBPreview': GLBPreview,
  'DefaultIconPreview': DefaultIconPreview
}

const getViewerComponent = (name?: string) => {
  return (name && viewerMap[name]) ? viewerMap[name] : DefaultIconPreview
}

const isActive = computed(() => props.state === RESOURCE_STATE.ACTIVE)

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
