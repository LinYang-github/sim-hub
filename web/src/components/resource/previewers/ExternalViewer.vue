<template>
  <div class="external-viewer-container" v-loading="loading">
    <iframe 
      ref="iframeRef"
      :src="iframeUrl" 
      class="external-iframe"
      @load="handleLoad"
    ></iframe>
    
    <div v-if="error" class="error-mask">
      <el-result icon="error" title="加载失败" sub-title="无法连接到外部预览器">
        <template #extra>
          <el-button type="primary" @click="retry">重试</el-button>
        </template>
      </el-result>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'

const props = defineProps<{
  url: string              // Base URL of the external app
  resource?: any           // Optional: Full resource object to send
  params?: Record<string, any> // Additional params
}>()

const loading = ref(true)
const error = ref(false)
const iframeRef = ref<HTMLIFrameElement | null>(null)

// Build URL with mode=preview
const iframeUrl = computed(() => {
    let base = window.location.origin
    
    // In dev mode, if url is relative, it probably points to the consolidated examples hub
    if (import.meta.env.DEV && props.url.startsWith('/')) {
        base = import.meta.env.VITE_EXT_APP_DEV_URL || 'http://localhost:30031'
    }

    const u = new URL(props.url, base)
    u.searchParams.set('mode', 'preview')
    u.searchParams.set('t', Date.now().toString()) // Anti-cache
    return u.toString()
})

// Theme sync
import { useDark } from '@vueuse/core'
const isDark = useDark()

const sendTheme = () => {
    if (iframeRef.value && iframeRef.value.contentWindow) {
        iframeRef.value.contentWindow.postMessage({
            type: 'THEME_UPDATE',
            payload: {
                theme: isDark.value ? 'dark' : 'light'
            }
        }, '*')
    }
}

const handleLoad = () => {
    loading.value = false
    sendTheme()
    syncData()
}

// Global message handler for guest handshake
const handleMessage = (e: MessageEvent) => {
    if (e.data && e.data.type === 'GUEST_READY') {
        sendTheme()
        syncData()
    }
}

const syncData = () => {
    if (iframeRef.value && iframeRef.value.contentWindow && props.resource) {
        try {
            // Use JSON.parse(JSON.stringify()) to ensure it's serializable and not a complex proxy
            const serializableResource = JSON.parse(JSON.stringify(props.resource))
            // Send data to guest app
            iframeRef.value.contentWindow.postMessage({
                type: 'PREVIEW_DATA',
                payload: {
                    resource: serializableResource
                }
            }, '*')
        } catch (e) {
            console.warn('Failed to serialize resource for external viewer:', e)
        }
    }
}

const retry = () => {
    loading.value = true
    error.value = false
}

onMounted(() => {
    window.addEventListener('message', handleMessage)
})

onUnmounted(() => {
    window.removeEventListener('message', handleMessage)
})

// Re-sync if resource changes while open
watch(() => props.resource, () => {
    syncData()
}, { deep: true })

// Sync if theme changes
watch(isDark, () => {
    sendTheme()
})

</script>

<style scoped lang="scss">
.external-viewer-container {
  width: 100%;
  height: 100%;
  position: relative;
  background: #fbfbfb;
}

.external-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: transparent;
}

:deep(.dark) .external-viewer-container {
    background: #1d1e1f;
}

.error-mask {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: var(--el-bg-color);
    display: flex;
    align-items: center;
    justify-content: center;
}
</style>
