<template>
  <div class="iframe-container" v-loading="false">
    <iframe 
      v-if="url" 
      :src="url" 
      ref="iframeRef"
      frameborder="0" 
      width="100%" 
      height="100%"
      @load="onLoad"
    ></iframe>
    <div v-else class="error">No URL provided</div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import { useDark } from '@vueuse/core'
import { hostBridge } from '../bridge/host'

const props = defineProps<{
  url: string
}>()

const isDark = useDark()
const loading = ref(true)
const iframeRef = ref<HTMLIFrameElement | null>(null)

// 同步主题给子页面
const syncTheme = () => {
  hostBridge.broadcast('THEME_UPDATE', {
    theme: isDark.value ? 'dark' : 'light'
  })
}

const onLoad = () => {
  loading.value = false
  syncTheme() // 加载完成后立刻同步一次
}

onMounted(() => {
  if (iframeRef.value) {
    hostBridge.register(iframeRef.value)
  }
})

onUnmounted(() => {
  if (iframeRef.value) {
    hostBridge.unregister(iframeRef.value)
  }
})

// 监听主题变化并实时同步
watch(isDark, () => {
  syncTheme()
})

// Reset loading when URL changes (if component is reused)
watch(() => props.url, () => {
  loading.value = true
})
</script>

<style scoped>
.iframe-container {
  width: 100%;
  height: calc(100vh - var(--header-height) - 40px); 
  overflow: hidden;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--sidebar-bg);
}
</style>
