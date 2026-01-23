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
import { ref, watch } from 'vue'
import { useDark } from '@vueuse/core'

const props = defineProps<{
  url: string
}>()

const isDark = useDark()
const loading = ref(true)
const iframeRef = ref<HTMLIFrameElement | null>(null)

// 同步主题给子页面
const syncTheme = () => {
  if (iframeRef.value && iframeRef.value.contentWindow) {
    iframeRef.value.contentWindow.postMessage({
      type: 'SIMHUB_THEME_CHANGE',
      theme: isDark.value ? 'dark' : 'light'
    }, '*')
  }
}

const onLoad = () => {
  loading.value = false
  syncTheme() // 加载完成后立刻同步一次
}

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
