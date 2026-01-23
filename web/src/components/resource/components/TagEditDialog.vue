<template>
  <el-dialog 
    v-model="visible" 
    title="管理资源标签" 
    width="460px"
    class="serious-tag-dialog"
    destroy-on-close
  >
    <div class="tag-manager-wrapper">
      <!-- 顶部搜索添加区 -->
      <div class="operation-bar">
        <el-input
          v-model="tagInput"
          placeholder="输入标签名称 (回车创建或搜索)"
          @keyup.enter="addNewTag"
          clearable
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>

      <!-- 当前状态汇总 -->
      <div class="tags-section">
        <div class="section-header">已分配标签 <span class="count">({{ localTags.length }})</span></div>
        <div class="tag-viewport assigned-list">
          <transition-group name="fade">
            <el-tag 
              v-for="tag in localTags" 
              :key="tag" 
              closable 
              class="serious-tag"
              @close="removeTag(tag)"
            >
              {{ tag }}
            </el-tag>
          </transition-group>
          <div v-if="localTags.length === 0" class="empty-hint">尚未分配任何标签</div>
        </div>
      </div>

      <el-divider />

      <!-- 快捷选择区 -->
      <div class="tags-section">
        <div class="section-header">常用标签库</div>
        <div class="tag-viewport registry-list">
          <el-tag
            v-for="tag in filteredExistingTags"
            :key="tag"
            class="clickable-tag"
            :class="{ 'is-selected': isSelected(tag) }"
            @click="toggleTag(tag)"
          >
            <el-icon v-if="isSelected(tag)"><Check /></el-icon>
            <el-icon v-else><Plus /></el-icon>
            {{ tag }}
          </el-tag>
          <div v-if="filteredExistingTags.length === 0" class="empty-hint">无匹配的标签项</div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="save" :loading="loading">保存更改</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Search, Plus, Check } from '@element-plus/icons-vue'

const props = defineProps<{
  modelValue: boolean
  tags: string[]
  existingTags: string[]
  loading: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  'save': [tags: string[]]
}>()

const localTags = ref<string[]>([])
const tagInput = ref('')

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

watch(() => props.tags, (newTags) => {
  localTags.value = Array.isArray(newTags) ? [...newTags] : []
}, { immediate: true })

const filteredExistingTags = computed(() => {
  if (!tagInput.value) return props.existingTags
  const query = tagInput.value.toLowerCase()
  return props.existingTags.filter(t => t.toLowerCase().includes(query))
})

const isSelected = (tag: string) => localTags.value.includes(tag)

const toggleTag = (tag: string) => {
  if (isSelected(tag)) {
    removeTag(tag)
  } else {
    localTags.value.push(tag)
  }
}

const addNewTag = () => {
  const val = tagInput.value.trim()
  if (val && !localTags.value.includes(val)) {
    localTags.value.push(val)
    tagInput.value = ''
  }
}

const removeTag = (tag: string) => {
  localTags.value = localTags.value.filter(t => t !== tag)
}

const save = () => {
  emit('save', localTags.value)
}
</script>

<style scoped lang="scss">
.serious-tag-dialog {
  :deep(.el-dialog__body) {
    padding: 20px 24px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}

.tag-manager-wrapper {
  .operation-bar {
    margin-bottom: 24px;
  }
}

.tags-section {
  .section-header {
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 8px;

    .count {
      color: var(--el-text-color-secondary);
      font-weight: normal;
    }
  }

  .tag-viewport {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    min-height: 32px;
  }

  .empty-hint {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
    padding: 8px 0;
  }
}

.serious-tag {
  height: 28px;
  padding: 0 10px;
  border-radius: 4px;
}

.clickable-tag {
  cursor: pointer;
  height: 28px;
  padding: 0 12px;
  border-radius: 4px;
  user-select: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  border: 1px solid var(--el-border-color-lighter);
  background-color: var(--el-fill-color-blank);
  color: var(--el-text-color-regular);
  font-size: 13px;

  /* 悬停状态：微妙的蓝调背景 */
  &:hover {
    border-color: var(--el-color-primary-light-5);
    background-color: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
  }

  /* 选中后的状态增强 */
  &.is-selected {
    border-color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
    font-weight: 600;
  }
  
  .el-icon {
    font-size: 12px;
    opacity: 0.8;
  }
}

.assigned-list {
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.registry-list {
  max-height: 160px;
  overflow-y: auto;
  padding: 4px 0;
}

.dialog-footer {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}

.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

.el-divider--horizontal {
  margin: 20px 0;
}
</style>
