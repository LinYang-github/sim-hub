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
    
    // In dev mode, if url is relative (no protocol), point to ext-apps dev server
    if (import.meta.env.DEV && !props.url.startsWith('http')) {
        base = import.meta.env.VITE_EXT_APP_DEV_URL || 'http://localhost:30031'
        // Ensure url has leading slash if mixing with host
        if (!props.url.startsWith('/')) {
             // We can't modify props directly, but the new URL constructor handles 'path' vs '/path' slightly differently relative to base.
             // Actually, new URL('foo', 'http://base.com') -> 'http://base.com/foo'
             // new URL('/foo', 'http://base.com') -> 'http://base.com/foo'
             // So it's fine.
        }
    }

    const u = new URL(props.url, base)
    u.searchParams.set('mode', 'preview')
    if (props.resource?.id) {
        u.searchParams.set('resId', props.resource.id)
    }
    u.searchParams.set('t', Date.now().toString()) // Anti-cache
    return u.toString()
})

// Theme sync
import { useDark } from '@vueuse/core'
const isDark = useDark()

const sendTheme = () => {
    if (iframeRef.value && iframeRef.value.contentWindow) {
        const style = getComputedStyle(document.documentElement)
        const tokens = {
            primary: style.getPropertyValue('--el-color-primary').trim(),
            success: style.getPropertyValue('--el-color-success').trim(),
            warning: style.getPropertyValue('--el-color-warning').trim(),
            danger: style.getPropertyValue('--el-color-danger').trim(),
            radius: style.getPropertyValue('--el-border-radius-base').trim()
        }

        iframeRef.value.contentWindow.postMessage({
            type: 'THEME_UPDATE',
            payload: {
                theme: isDark.value ? 'dark' : 'light',
                tokens
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
