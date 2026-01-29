<template>
  <div class="json-preview-container">
    <div v-if="loading" class="loading-state">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>解析数据中...</span>
    </div>
    <div v-else-if="error" class="error-state">
      <el-icon><Warning /></el-icon>
      <span>解析失败 (非合法 JSON)</span>
    </div>
    <div v-else class="json-content">
      <div class="json-header">
        <el-icon><Document /></el-icon>
        <span>参数配置详情</span>
      </div>
      <pre class="json-tree">{{ formattedJson }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue'
import { Loading, Warning, Document } from '@element-plus/icons-vue'
import axios from 'axios'

const props = defineProps<{
  url: string
  metaData?: any
}>()

const loading = ref(true)
const error = ref(false)
const jsonData = ref<any>(null)

const formattedJson = computed(() => {
  if (!jsonData.value) return ''
  return JSON.stringify(jsonData.value, null, 2)
})

const fetchData = async () => {
  if (!props.url) {
      if (props.metaData) {
          jsonData.value = props.metaData
          loading.value = false
      }
      return
  }
  
  loading.value = true
  error.value = false
  try {
    const res = await axios.get(props.url)
    jsonData.value = res.data
  } catch (e) {
    if (props.metaData) {
        jsonData.value = props.metaData
    } else {
        error.value = true
    }
  } finally {
    loading.value = false
  }
}

watch(() => props.url, fetchData)
onMounted(fetchData)
</script>

<style scoped lang="scss">
.json-preview-container {
  width: 100%;
  height: 100%;
  padding: 20px;
  background: var(--el-bg-color-page);
  border-radius: 8px;
  overflow: auto;
  font-family: 'Fira Code', 'Roboto Mono', monospace;
}

.loading-state, .error-state {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--el-text-color-secondary);
}

.json-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 12px;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.json-tree {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--el-color-primary);
}

.dark .json-tree {
  color: #a5d6ff;
}
</style>
