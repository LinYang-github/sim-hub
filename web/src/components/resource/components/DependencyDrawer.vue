<template>
  <el-drawer v-model="visible" title="资源依赖全景图" size="450px" class="dep-drawer">
    <template #header>
      <div class="drawer-header">
        <span>资源依赖全景图</span>
        <div class="header-actions">
          <el-button type="warning" size="small" @click="$emit('download-bundle')" :loading="bundleLoading">清单下载</el-button>
          <el-button type="success" size="small" @click="$emit('download-pack')" :loading="packLoading">离线打包 (.simpack)</el-button>
        </div>
      </div>
    </template>
    <div v-loading="loading" class="dep-content">
      <template v-if="depTree.length > 0">
        <el-tree
          :data="depTree"
          :props="{ label: 'resource_name', children: 'dependencies' }"
          default-expand-all
          class="dep-tree"
        >
          <template #default="{ node, data }">
            <div class="dep-node">
              <el-icon class="dep-icon"><Share /></el-icon>
              <div class="dep-info">
                <span class="dep-name">{{ data.resource_name }}</span>
                <div class="dep-meta">
                  <el-tag size="small" type="info" class="dep-ver">{{ data.semver || 'latest' }}</el-tag>
                  <span class="dep-constraint" v-if="data.constraint">约束: {{ data.constraint }}</span>
                </div>
              </div>
            </div>
          </template>
        </el-tree>
      </template>
      <el-empty v-else description="该资源暂无任何依赖" />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Share } from '@element-plus/icons-vue'

const props = defineProps<{
  modelValue: boolean
  depTree: any[]
  loading: boolean
  bundleLoading: boolean
  packLoading: boolean
}>()

const emit = defineEmits(['update:modelValue', 'download-bundle', 'download-pack'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>

<style scoped lang="scss">
.dep-drawer {
  :deep(.el-drawer__body) {
    padding: 0;
  }
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding-right: 20px;

  .header-actions {
    display: flex;
    gap: 8px;
  }
}

.dep-content {
  padding: 20px;
  height: 100%;
}

.dep-tree {
  background: transparent;
  
  :deep(.el-tree-node__content) {
    height: auto;
    padding: 8px 0;
    
    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }
}

.dep-node {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
}

.dep-icon {
  margin-top: 4px;
  font-size: 18px;
  color: var(--el-color-primary);
}

.dep-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dep-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.dep-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dep-ver {
  height: 20px;
  padding: 0 6px;
  font-size: 11px;
}

.dep-constraint {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
</style>
