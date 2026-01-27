<template>
  <div class="metadata-editor">
    <div class="editor-header">
      <div class="header-left">
        <el-icon><Operation /></el-icon>
        <span class="title">元数据 JSON 编辑器</span>
      </div>
      <div class="header-right">
        <el-button 
          type="primary" 
          size="small" 
          :loading="saving" 
          @click="handleSave"
        >
          保存变更
        </el-button>
      </div>
    </div>
    
    <div class="editor-main">
      <el-input
        v-model="jsonText"
        type="textarea"
        :rows="15"
        class="json-code-input"
        spellcheck="false"
        placeholder="{}"
      />
    </div>
    
    <div class="editor-footer">
      <div v-if="error" class="error-msg">
        <el-icon><CircleCloseFilled /></el-icon>
        <span>{{ error }}</span>
      </div>
      <div v-else class="success-tip">
        <el-icon><InfoFilled /></el-icon>
        <span>直接修改 JSON 对象，保存后将触发 Sidecar 刷新。</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Operation, InfoFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '../../../core/utils/request'

const props = defineProps<{
  versionId: string | undefined
  initialData: any
}>()

const emit = defineEmits<{
  (e: 'success'): void
}>()

const jsonText = ref('')
const error = ref('')
const saving = ref(false)

watch(() => props.initialData, (newVal) => {
  jsonText.value = JSON.stringify(newVal || {}, null, 2)
}, { immediate: true })

const handleSave = async () => {
  if (!props.versionId) return
  
  error.value = ''
  let parsed: any
  try {
    parsed = JSON.parse(jsonText.value)
  } catch (e: any) {
    error.value = '无效的 JSON 格式: ' + e.message
    return
  }
  
  saving.value = true
  try {
    await request.patch(`/api/v1/resources/versions/${props.versionId}/meta`, {
      meta_data: parsed
    })
    ElMessage.success('元数据已成功更新')
    emit('success')
  } catch (err: any) {
    error.value = err.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped lang="scss">
.metadata-editor {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-lighter);
  
  .header-left {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }
}

.editor-main {
  :deep(.el-textarea__inner) {
    font-family: 'Fira Code', 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
    font-size: 13px;
    padding: 16px;
    line-height: 1.6;
    background-color: var(--el-fill-color-extra-light);
    border: none;
    box-shadow: none !important;
    color: var(--el-text-color-primary);
    
    &:focus {
      background-color: var(--el-bg-color);
    }
  }
}

.editor-footer {
  padding: 10px 16px;
  background: var(--el-fill-color-lighter);
  border-top: 1px solid var(--el-border-color-lighter);
  
  .error-msg {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--el-color-danger);
    font-size: 12px;
  }
  
  .success-tip {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}
</style>
