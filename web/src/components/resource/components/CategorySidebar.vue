<template>
  <div class="category-sidebar">
    <div class="sidebar-header">
      <el-icon><FolderOpened /></el-icon>
      <span>资源分类</span>
      <el-button link type="primary" @click="$emit('add-category')">
        <el-icon><Plus /></el-icon>
      </el-button>
    </div>
    
    <el-scrollbar>
      <el-tree
        :data="categoryTree"
        :props="defaultProps"
        node-key="id"
        class="custom-tree"
        @node-click="(data) => $emit('select-category', data)"
        highlight-current
        :default-expanded-keys="['all']"
      >
        <template #default="{ node, data }">
          <span class="custom-tree-node">
            <el-icon v-if="data.id === 'all'"><Grid /></el-icon>
            <el-icon v-else><Folder /></el-icon>
            <span class="node-label">{{ node.label }}</span>
            <span class="node-actions" v-if="data.id !== 'all'">
              <el-icon class="delete-icon" @click.stop="$emit('delete-category', data.id)"><Delete /></el-icon>
            </span>
          </span>
        </template>
      </el-tree>
    </el-scrollbar>
  </div>
</template>

<script setup lang="ts">
import { FolderOpened, Plus, Grid, Folder, Delete } from '@element-plus/icons-vue'

defineProps<{
  categoryTree: any[]
  defaultProps: any
}>()

defineEmits(['add-category', 'select-category', 'delete-category'])
</script>

<style scoped lang="scss">
.category-sidebar {
  width: 240px;
  background: var(--sidebar-bg);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-lighter);
  overflow: hidden;
}

.sidebar-header {
  height: 50px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-weight: 600;
  color: var(--el-text-color-primary);
  
  .el-icon { font-size: 18px; }
  span { flex: 1; }
}

.custom-tree {
  padding: 10px;
  background: transparent;
  
  :deep(.el-tree-node__content) {
    height: 36px;
    border-radius: 6px;
    margin-bottom: 2px;
    
    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }
  
  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    background-color: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
  }
}

.custom-tree-node {
  display: flex;
  align-items: center;
  width: 100%;
  font-size: 13px;
  padding-right: 12px;
  
  .node-label {
    margin-left: 8px;
    flex: 1;
  }
  
  .node-actions {
    opacity: 0;
    transition: opacity 0.2s;
    color: var(--el-text-color-placeholder);
    
    &:hover { color: var(--el-color-danger); }
  }
}

.custom-tree-node:hover .node-actions { opacity: 1; }
</style>
