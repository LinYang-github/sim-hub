<template>
  <div class="dependency-editor">
    <div class="editor-header">
      <div class="header-left">
        <el-icon><Share /></el-icon>
        <span class="title">依赖关联管理</span>
      </div>
      <div class="header-right">
        <el-button 
          type="primary" 
          size="small" 
          :loading="saving" 
          @click="handleSave"
        >
          保存关联
        </el-button>
      </div>
    </div>
    
    <div class="editor-main">
      <el-form label-position="top">
        <el-form-item label="直接依赖资源">
          <el-select
            v-model="selectedDeps"
            multiple
            filterable
            remote
            reserve-keyword
            placeholder="搜索并选择要关联的资源"
            :remote-method="searchResources"
            :loading="searching"
            style="width: 100%"
            value-key="id"
          >
            <el-option
              v-for="item in searchResults"
              :key="item.id"
              :label="item.name"
              :value="{ id: item.id, name: item.name, type: item.type_key }"
            >
              <div class="search-option">
                <span class="option-name">{{ item.name }}</span>
                <span class="option-type">{{ item.type_key }}</span>
              </div>
            </el-option>
          </el-select>
          <div class="input-tip">搜索并添加该资源运行所需的其他依赖。修改后将自动刷新拓扑图并影响打包下载。</div>
        </el-form-item>
      </el-form>
      
      <div class="current-deps" v-if="selectedDeps.length > 0">
        <div class="section-subtitle">已选择的直接依赖：</div>
        <div class="dep-tags">
          <el-tag
            v-for="dep in selectedDeps"
            :key="dep.id"
            closable
            @close="removeDep(dep.id)"
            class="dep-tag"
            type="info"
          >
            {{ dep.name }} ({{ dep.type }})
          </el-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { Share, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '../../../core/utils/request'
import type { Resource, ResourceDependency } from '../../../core/types/resource'

const props = defineProps<{
  versionId: string | undefined
  initialDeps: ResourceDependency[]
}>()

const emit = defineEmits<{
  (e: 'success'): void
}>()

const selectedDeps = ref<{ id: string, name: string, type: string }[]>([])
const searching = ref(false)
const searchResults = ref<Resource[]>([])
const saving = ref(false)

// Initialize from props
watch(() => props.initialDeps, (newDeps) => {
  // initialDeps usually comes as items from GetDependencyTree which might be nested.
  // We need to map them to a flat list of direct dependencies.
  // Actually, useDependency should fetch the DIRECT dependencies for editing.
  // Wait, I should make sure I have the direct dependencies list.
  selectedDeps.value = (newDeps || []).map(d => ({
    id: d.resource_id,
    name: d.resource_name || 'Unknown',
    type: d.type_key || ''
  }))
}, { immediate: true })

const searchResources = async (query: string) => {
  if (!query) return;
  searching.value = true
  try {
    const res = await request.get<any>('/api/v1/resources', {
      params: { page: 1, size: 20, query: query }
    })
    searchResults.value = (res.items || []).filter((r: Resource) => r.latest_version?.id !== props.versionId)
  } catch (err) {
  } finally {
    searching.value = false
  }
}

const removeDep = (id: string) => {
  selectedDeps.value = selectedDeps.value.filter(d => d.id !== id)
}

const handleSave = async () => {
  if (!props.versionId) return
  
  saving.value = true
  try {
    const payload = selectedDeps.value.map(d => ({
      target_resource_id: d.id,
      constraint: 'latest'
    }))
    
    await request.patch(`/api/v1/resources/versions/${props.versionId}/dependencies`, payload)
    ElMessage.success('资源依赖已更新')
    emit('success')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped lang="scss">
.dependency-editor {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  overflow: hidden;
  background: var(--el-bg-color);
  margin-top: 16px;
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
  padding: 16px;
}

.input-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
  margin-top: 8px;
}

.search-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  
  .option-name {
    font-weight: 500;
  }
  
  .option-type {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    background: var(--el-fill-color-light);
    padding: 0 4px;
    border-radius: 4px;
  }
}

.current-deps {
  margin-top: 16px;
  .section-subtitle {
     font-size: 12px;
     font-weight: 600;
     color: var(--el-text-color-regular);
     margin-bottom: 8px;
  }
}

.dep-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.dep-tag {
  border-radius: 4px;
}
</style>
