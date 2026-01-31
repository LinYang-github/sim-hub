<template>
  <div class="category-sidebar">
    <div class="sidebar-header">
      <div class="header-main">
        <el-icon class="title-icon"><FolderOpened /></el-icon>
        <span class="title-text">资源分类</span>
        <el-tooltip v-if="hasPermission('resource:update')" content="新建分类" placement="top">
          <el-button class="add-btn" circle @click="$emit('add-category')">
            <el-icon><Plus /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
      
      <!-- Category Filter -->
      <div class="filter-wrapper">
        <el-input
          v-model="filterText"
          placeholder="过滤分类..."
          size="small"
          clearable
          :prefix-icon="Search"
          class="filter-input"
        />
      </div>
    </div>
    
    <div class="tree-container">
      <el-scrollbar>
        <el-tree
          ref="treeRef"
          :data="categoryTree"
          :props="defaultProps"
          :node-key="'id'"
          :current-node-key="modelValue"
          class="custom-tree"
          @node-click="handleNodeClick"
          highlight-current
          :default-expanded-keys="[ROOT_CATEGORY_ID]"
          :filter-node-method="filterNode"
          draggable
          :allow-drag="allowDrag"
          :allow-drop="allowDrop"
          @node-drop="handleNodeDrop"
        >
          <template #default="{ node, data }">
            <div class="custom-tree-node">
              <el-icon v-if="data.id === ROOT_CATEGORY_ID" class="node-icon root-icon"><Grid /></el-icon>
              <el-icon v-else class="node-icon folder-icon"><Folder /></el-icon>
              <span class="node-label">{{ node.label }}</span>
              <div class="node-actions" v-if="data.id !== ROOT_CATEGORY_ID">
                <el-icon 
                  v-if="hasPermission('resource:update') && categoryMode === 'tree'" 
                  class="action-icon add-icon" 
                  @click.stop="$emit('add-subcategory', data.id)"
                  title="新建子分类"
                ><Plus /></el-icon>
                <el-icon v-if="hasPermission('resource:delete')" class="action-icon delete-icon" @click.stop="$emit('delete-category', data.id)"><Delete /></el-icon>
              </div>
            </div>
          </template>
        </el-tree>
      </el-scrollbar>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { FolderOpened, Plus, Grid, Folder, Delete, Search } from '@element-plus/icons-vue'
import type { ElTree } from 'element-plus'
import type { CategoryNode } from '../../../core/types/resource'
import { ROOT_CATEGORY_ID } from '../../../core/constants/resource'
import { useAuth } from '../../../core/auth'

const { hasPermission } = useAuth()

const props = defineProps<{
  categoryTree: CategoryNode[]
  defaultProps: { label: string; children: string }
  modelValue?: string
  categoryMode?: 'flat' | 'tree'
}>()

const emit = defineEmits<{
  (e: 'add-category'): void
  (e: 'add-subcategory', parentId: string): void
  (e: 'select-category', data: CategoryNode): void
  (e: 'delete-category', id: string): void
  (e: 'update:modelValue', id: string): void
  (e: 'move-category', id: string, parentId: string): void
}>()

const filterText = ref('')
const treeRef = ref<InstanceType<typeof ElTree>>()

watch(filterText, (val) => {
  treeRef.value?.filter(val)
})

const filterNode = (value: string, data: any) => {
  if (!value) return true
  return data.name.toLowerCase().includes(value.toLowerCase())
}

const handleNodeClick = (data: CategoryNode) => {
  emit('select-category', data)
  emit('update:modelValue', data.id)
}

const allowDrag = (node: any) => {
  if (!hasPermission('resource:update')) return false
  // Prevent dragging the root "All Categories" node
  return node.data.id !== ROOT_CATEGORY_ID
}

// type can be 'prev', 'next', 'inner'
const allowDrop = (_draggingNode: any, dropNode: any, type: string) => {
  // 1. Cannot drop 'prev' or 'next' relative to Root node (it must stay at top)
  if (dropNode.data.id === ROOT_CATEGORY_ID && type !== 'inner') {
    return false
  }
  return true
}

const handleNodeDrop = (draggingNode: any, dropNode: any, type: string, _ev: DragEvent) => {
  // Calculate new parent ID
  let newParentId = ''
  if (type === 'inner') {
    newParentId = dropNode.data.id
  } else {
    // 'prev' or 'next' -> same parent as dropNode
    newParentId = dropNode.parent.data.id || ROOT_CATEGORY_ID // default to root if parent is missing logic
    // Note: el-tree internal structure. If dropNode is top level (under fake root), its parent might be root.
    // Our fake root has id ROOT_CATEGORY_ID.
    // If dropNode is child of "Full Categories", dropNode.parent.data.id should be ROOT_CATEGORY_ID.
  }
  
  const targetId = draggingNode.data.id
  // If moving to root (ID='all'), send empty string to backend
  if (newParentId === ROOT_CATEGORY_ID) {
    newParentId = ''
  }
  emit('move-category', targetId, newParentId)
}
</script>

<style scoped lang="scss">
.category-sidebar {
  width: 260px;
  background: var(--sidebar-bg);
  border-radius: 8px; /* Matching the header's refined radius */
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 4px 12px -4px rgba(0, 0, 0, 0.04);
  overflow: hidden;
  transition: all 0.3s ease;
}

.sidebar-header {
  padding: 16px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  border-bottom: 1px solid var(--el-border-color-extra-light);

  .header-main {
    display: flex;
    align-items: center;
    gap: 8px;

    .title-icon {
      font-size: 16px;
      color: var(--el-color-primary);
    }

    .title-text {
      flex: 1;
      font-size: 14px;
      font-weight: 700;
      color: var(--el-text-color-primary);
      letter-spacing: 0.2px;
    }

    .add-btn {
      width: 24px;
      height: 24px;
      border: none;
      background: var(--el-fill-color-light);
      &:hover {
        background: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
      }
    }
  }
}

.filter-wrapper {
  :deep(.el-input__wrapper) {
    background-color: var(--el-fill-color-lighter);
    box-shadow: none !important;
    border: 1px solid transparent;
    transition: all 0.2s;

    &.is-focus {
      background-color: var(--el-bg-color);
      border-color: var(--el-color-primary-light-5);
    }
  }
}

.tree-container {
  flex: 1;
  overflow: hidden;
  padding: 8px;
}

.custom-tree {
  background: transparent;

  :deep(.el-tree-node__content) {
    height: 40px;
    border-radius: 6px;
    margin-bottom: 2px;
    padding-left: 8px !important;
    transition: all 0.2s;

    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }

  :deep(.el-tree-node.is-current > .el-tree-node__content) {
    background-color: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    position: relative;

    &::after {
      content: '';
      position: absolute;
      left: 0;
      top: 10px;
      bottom: 10px;
      width: 3px;
      background: var(--el-color-primary);
      border-radius: 0 4px 4px 0;
    }
    
    .node-label {
      font-weight: 600;
    }

    .node-icon {
      color: var(--el-color-primary);
    }
  }
}

.custom-tree-node {
  display: flex;
  align-items: center;
  width: 100%;
  font-size: 13px;
  
  .node-icon {
    font-size: 16px;
    color: var(--el-text-color-placeholder);
    &.root-icon { color: var(--el-color-warning); }
  }

  .node-label {
    margin-left: 10px;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--el-text-color-regular);
  }

  .node-actions {
    opacity: 0;
    padding: 0 8px;
    transition: opacity 0.2s;
    align-items: center;
    gap: 6px;

    .action-icon {
      font-size: 14px;
      color: var(--el-text-color-placeholder);
      padding: 2px;
      border-radius: 4px;
      
      &:hover { 
        background-color: var(--el-fill-color-darker);
      }
      
      &.delete-icon:hover { color: var(--el-color-danger); }
      &.add-icon:hover { color: var(--el-color-primary); }
    }
  }

  &:hover .node-actions {
    opacity: 1;
  }
}
</style>
