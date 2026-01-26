<template>
  <div class="doc-preview-container">
    <div v-if="loading" class="doc-status">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>正在加载文档...</span>
    </div>
    
    <div v-else-if="error" class="doc-status error">
      <el-icon><Warning /></el-icon>
      <span>{{ errorMsg }}</span>
    </div>

    <template v-else>
      <!-- PDF Preview -->
      <iframe 
        v-if="isPdf" 
        :src="url" 
        class="pdf-viewer"
        frameborder="0"
      ></iframe>

      <!-- Markdown / Text Preview -->
      <div v-else class="text-viewer" ref="textScroll">
        <div v-if="isMarkdown" class="markdown-body" v-html="renderedMarkdown"></div>
        <pre v-else class="plain-text">{{ textContent }}</pre>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Loading, Warning } from '@element-plus/icons-vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{
  url: string
}>()

const loading = ref(false)
const error = ref(false)
const errorMsg = ref('加载失败')
const textContent = ref('')
const renderedMarkdown = ref('')

const isPdf = computed(() => {
  const lowercaseUrl = props.url.toLowerCase();
  // 不仅检查结尾，还要检查是否包含 .pdf (处理带查询参数的情况)
  return lowercaseUrl.includes('.pdf') || lowercaseUrl.endsWith('.pdf');
})
const isMarkdown = computed(() => props.url.toLowerCase().endsWith('.md'))

const md = new MarkdownIt({
  html: true,
  linkify: true,
  typographer: true
})

const fetchContent = async () => {
  if (isPdf.value) return
  
  loading.value = true
  error.value = false
  
  try {
    const response = await fetch(props.url)
    if (!response.ok) throw new Error('网络请求异常')
    const text = await response.text()
    textContent.value = text
    
    if (isMarkdown.value) {
      renderedMarkdown.value = md.render(text)
    }
  } catch (err: any) {
    error.value = true
    errorMsg.value = err.message || '文档加载失败'
  } finally {
    loading.value = false
  }
}

watch(() => props.url, fetchContent, { immediate: true })
</script>

<style scoped>
.doc-preview-container {
  width: 100%;
  height: 100%;
  background: var(--el-bg-color);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.doc-status {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--el-text-color-secondary);
}

.doc-status.error {
  color: var(--el-color-danger);
}

.pdf-viewer {
  width: 100%;
  height: 100%;
}

.text-viewer {
  width: 100%;
  height: 100%;
  padding: 20px;
  overflow-y: auto;
  box-sizing: border-box;
}

.plain-text {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: var(--el-font-family-mono);
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
}

/* 简单的 Markdown 样式覆盖 */
.markdown-body {
  font-size: 14px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
}

.markdown-body :deep(h1), .markdown-body :deep(h2) {
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding-bottom: 8px;
  margin-top: 24px;
}

.markdown-body :deep(code) {
  background: var(--el-fill-color-light);
  padding: 2px 4px;
  border-radius: 4px;
  font-family: var(--el-font-family-mono);
}

.markdown-body :deep(pre) {
  background: var(--el-fill-color-darker);
  color: #fff;
  padding: 16px;
  border-radius: 8px;
  overflow-x: auto;
}
</style>
