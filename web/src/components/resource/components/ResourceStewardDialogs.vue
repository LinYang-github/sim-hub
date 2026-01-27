<template>
  <!-- 1. Rename Dialog -->
  <el-dialog
    v-model="renameVisible"
    title="重命名资源"
    width="400px"
    class="steward-dialog"
    destroy-on-close
  >
    <el-form label-position="top">
      <el-form-item label="资源新名称">
        <el-input 
          v-model="renameForm.name" 
          placeholder="请输入新名称" 
          @keyup.enter="handleRename"
        />
      </el-form-item>
      <div class="steward-tip">
        <el-icon><InfoFilled /></el-icon>
        <span>更名将同步更新存储侧的元数据副本。</span>
      </div>
    </el-form>
    <template #footer>
      <el-button @click="renameVisible = false">取消</el-button>
      <el-button type="primary" :loading="renameLoading" @click="handleRename">确认更名</el-button>
    </template>
  </el-dialog>

  <!-- 2. Move Dialog -->
  <el-dialog
    v-model="moveVisible"
    title="移动资源至分类"
    width="450px"
    class="steward-dialog"
    destroy-on-close
  >
    <div class="move-container">
      <div class="move-tip">当前资源: <strong>{{ resource?.name }}</strong></div>
      <div class="tree-wrapper">
        <el-scrollbar max-height="300px">
          <el-tree
            :data="categoryTree"
            :props="{ label: 'name', children: 'children' }"
            node-key="id"
            highlight-current
            default-expand-all
            :current-node-key="targetCategoryId"
            @node-click="(data) => targetCategoryId = data.id"
          >
            <template #default="{ node, data }">
              <div class="tree-node">
                <el-icon class="folder-icon"><Folder /></el-icon>
                <span>{{ node.label }}</span>
                <el-tag v-if="data.id === resource?.category_id" size="small" type="info" class="current-tag">当前</el-tag>
              </div>
            </template>
          </el-tree>
        </el-scrollbar>
      </div>
    </div>
    <template #footer>
      <el-button @click="moveVisible = false">取消</el-button>
      <el-button 
        type="primary" 
        :loading="moveLoading" 
        :disabled="!targetCategoryId || targetCategoryId === resource?.category_id"
        @click="handleMove"
      >
        确认移动
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, reactive } from 'vue'
import { InfoFilled, Folder } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import request from '../../../core/utils/request'
import type { Resource, CategoryNode } from '../../../core/types/resource'

const props = defineProps<{
  resource: Resource | null
  categoryTree: CategoryNode[]
}>()

const emit = defineEmits<{
  (e: 'success'): void
}>()

// 1. Rename
const renameVisible = ref(false)
const renameLoading = ref(false)
const renameForm = reactive({ name: '' })

const openRename = (res: Resource) => {
  renameForm.name = res.name
  renameVisible.value = true
}

const handleRename = async () => {
  if (!props.resource || !renameForm.name) return
  if (renameForm.name === props.resource.name) {
    renameVisible.value = false
    return
  }
  
  renameLoading.value = true
  try {
    await request.patch(`/api/v1/resources/${props.resource.id}`, { name: renameForm.name })
    ElMessage.success('重命名成功')
    renameVisible.value = false
    emit('success')
  } catch (err) {
    console.error(err)
  } finally {
    renameLoading.value = false
  }
}

// 2. Move
const moveVisible = ref(false)
const moveLoading = ref(false)
const targetCategoryId = ref('')

const openMove = (res: Resource) => {
  targetCategoryId.value = res.category_id || ''
  moveVisible.value = true
}

const handleMove = async () => {
  if (!props.resource || !targetCategoryId.value) return
  
  moveLoading.value = true
  try {
    await request.patch(`/api/v1/resources/${props.resource.id}`, { category_id: targetCategoryId.value })
    ElMessage.success('移动成功')
    moveVisible.value = false
    emit('success')
  } catch (err) {
    console.error(err)
  } finally {
    moveLoading.value = false
  }
}

defineExpose({
  openRename,
  openMove
})
</script>

<style scoped lang="scss">
.steward-dialog {
  :deep(.el-dialog__body) {
    padding-top: 10px;
  }
}

.steward-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  padding: 8px 12px;
  border-radius: 4px;
  margin-top: 12px;
  
  .el-icon {
    color: var(--el-color-primary);
  }
}

.move-container {
  .move-tip {
    font-size: 13px;
    margin-bottom: 16px;
    color: var(--el-text-color-regular);
    strong { color: var(--el-color-primary); }
  }
  
  .tree-wrapper {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    padding: 8px;
    background: var(--el-bg-color-page);
  }
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  
  .folder-icon {
    color: var(--el-text-color-placeholder);
  }
  
  .current-tag {
    margin-left: 8px;
    font-size: 10px;
  }
}
</style>
