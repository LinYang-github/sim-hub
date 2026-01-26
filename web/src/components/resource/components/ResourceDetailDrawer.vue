<template>
  <el-drawer
    v-model="visible"
    :size="540"
    class="serious-detail-drawer"
    destroy-on-close
  >
    <template #header>
      <div class="serious-header">
        <div class="title-row">
          <el-icon class="title-icon"><InfoFilled /></el-icon>
          <span class="title-text">资源元数据详情</span>
        </div>
        <div class="subtitle-row">
          <span class="resource-id">ID: {{ resource?.id || '-' }}</span>
          <el-tag size="small" effect="plain" type="info">{{ typeName }}</el-tag>
        </div>
      </div>
    </template>

    <div v-if="resource" class="drawer-body-wrapper" v-loading="loadingDetails">
      <!-- 0. 预览区域 (动态渲染) -->
      <div class="detailed-preview-section">
        <ResourcePreview 
          :type-key="resource.type_key" 
          :viewer="viewer"
          :icon="icon"
          :download-url="resource.latest_version?.download_url"
          :state="resource.latest_version?.state"
          :status-text="resource.latest_version?.state ? (statusMap[resource.latest_version!.state] || resource.latest_version!.state) : '-'"
          force
        />
      </div>

      <!-- 1. 核心属性表 -->
      <div class="details-section">
        <div class="section-label">基本属性</div>
        <el-descriptions :column="2" border class="property-grid">
          <el-descriptions-item label="资源名称" :span="2">
            <span class="text-bold">{{ resource.name }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="当前版本">
            <el-tag size="small" type="success" effect="light">
              {{ resource?.latest_version?.semver || 'v' + (resource?.latest_version?.version_num || 1) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="生命周期">
            <div class="status-cell">
              <span :class="['status-dot', resource?.latest_version?.state?.toLowerCase()]"></span>
              {{ resource?.latest_version?.state ? (statusMap[resource.latest_version!.state] || resource.latest_version!.state) : '-' }}
            </div>
          </el-descriptions-item>
          <el-descriptions-item label="文件大小">
            {{ formatSize(resource?.latest_version?.file_size) }}
          </el-descriptions-item>
          <el-descriptions-item label="可见范围">
            {{ resource?.scope === 'PUBLIC' ? '公共 (Public)' : '私有 (Private)' }}
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">
            {{ formatDate(resource?.created_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="所有者">
            {{ resource?.owner_id || 'System' }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <!-- 2. 标签系统 -->
      <div class="details-section">
        <div class="section-label">
          管理标签
          <el-button link type="primary" size="small" @click="$emit('edit-tags', resource)">
            <el-icon><Edit /></el-icon> 编辑
          </el-button>
        </div>
        <div class="tags-row">
          <el-tag 
            v-for="tag in resource.tags" 
            :key="tag" 
            size="small" 
            type="info" 
            effect="plain"
            class="util-tag"
          >
            {{ tag }}
          </el-tag>
          <div v-if="!resource.tags?.length" class="empty-data">未打标</div>
        </div>
      </div>

      <!-- 3. 数据选项卡 -->
      <el-tabs v-model="activeTab" class="serious-tabs">
        <el-tab-pane name="versions" label="版本更迭历史">
          <div class="tab-pane-content">
            <el-table :data="versions" size="small" border stripe class="version-table">
              <el-table-column prop="version" label="版本" width="90">
                <template #default="{ row }">
                  <span class="mono-text">{{ row.semver || 'v' + row.version_num }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="state" label="状态" width="100">
                <template #default="{ row }">
                  <el-tag size="small" :type="getStatusType(row.state)" effect="light">
                    {{ statusMap[row.state] || row.state }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="更新日期" min-width="140">
                <template #default="{ row }">
                  {{ formatDate(row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" width="120" align="center">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="$emit('download-version', row.download_url)">下载</el-button>
                  <el-button 
                    v-if="row.id !== resource?.latest_version?.id" 
                    link 
                    type="warning" 
                    size="small" 
                    @click="$emit('rollback', row)"
                  >
                    回滚
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-tab-pane>

        <el-tab-pane name="dependencies" label="拓扑依赖">
          <div class="tab-pane-content">
            <DependencyGraph 
              v-if="dependencies?.length" 
              :dependencies="dependencies" 
              :root-name="resource.name"
            />
            <el-empty v-else :image-size="40" description="无外部依赖关联" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <div class="serious-drawer-footer">
        <el-button @click="visible = false">关闭窗口</el-button>
        <el-button 
          type="primary" 
          @click="resource && $emit('download', resource)" 
          :disabled="resource?.latest_version?.state !== RESOURCE_STATE.ACTIVE"
        >
          <el-icon><Download /></el-icon> 下载部署包
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { InfoFilled, Edit, Download, Share } from '@element-plus/icons-vue'
import { formatDate, formatSize } from '../../../core/utils/format'
import type { Resource, ResourceVersion, ResourceDependency } from '../../../core/types/resource'
import { RESOURCE_STATE } from '../../../core/constants/resource'
import ResourcePreview from './viewers/ResourcePreview.vue'
import DependencyGraph from './viewers/DependencyGraph.vue'

const props = defineProps<{
  modelValue: boolean
  resource: Resource | null
  typeName: string
  statusMap: Record<string, string>
  versions: ResourceVersion[]
  dependencies: ResourceDependency[]
  loadingDetails: boolean
  viewer?: string
  icon?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'edit-tags', row: Resource): void
  (e: 'download', row: Resource): void
  (e: 'download-version', url: string): void
  (e: 'rollback', ver: ResourceVersion): void
}>()

const activeTab = ref('versions')

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const getStatusType = (state: string) => {
  const map: any = { ACTIVE: 'success', PROCESSING: 'primary', FAILED: 'danger' }
  return map[state] || 'info'
}
</script>

<style scoped lang="scss">
.serious-detail-drawer {
  :deep(.el-drawer__header) {
    margin-bottom: 0;
    padding: 16px 24px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
}

.serious-header {
  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;
    
    .title-icon {
      color: var(--el-color-primary);
      font-size: 18px;
    }
    .title-text {
      font-size: 16px;
      font-weight: 700;
      color: var(--el-text-color-primary);
    }
  }
  .subtitle-row {
    display: flex;
    align-items: center;
    gap: 12px;
    
    .resource-id {
      font-size: 12px;
      color: var(--el-text-color-secondary);
      font-family: monospace;
    }
  }
}

.drawer-body-wrapper {
  padding: 24px;
}

.detailed-preview-section {
  width: 100%;
  height: 240px;
  background: var(--el-fill-color-lighter);
  border-radius: 12px;
  margin-bottom: 24px;
  border: 1px solid var(--el-border-color-lighter);
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: inset 0 2px 8px rgba(0,0,0,0.02);
}

.details-section {
  margin-bottom: 24px;

  .section-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 12px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-left: 3px solid var(--el-color-primary);
    padding-left: 10px;
  }
}

.property-grid {
  :deep(.el-descriptions__label) {
    width: 100px;
    background-color: var(--el-fill-color-light);
    color: var(--el-text-color-regular);
    font-weight: 500;
  }
  .text-bold {
    font-weight: 700;
    color: var(--el-text-color-primary);
  }
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--el-text-color-placeholder);
    
    &.active { background: var(--el-color-success); border: 2px solid var(--el-color-success-light-8); }
    &.processing { background: var(--el-color-primary); }
    &.failed { background: var(--el-color-danger); }
  }
}

.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  .util-tag { border-radius: 2px; }
  .empty-data { font-size: 12px; color: var(--el-text-color-placeholder); }
}

.serious-tabs {
  margin-top: 32px;
  :deep(.el-tabs__item) {
    font-size: 13px;
    font-weight: 600;
  }
}

.tab-pane-content {
  padding: 12px 0;
}

.version-table {
  .mono-text { font-family: monospace; font-weight: 600; }
}

.dep-grid {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  
  .dep-row {
    display: flex;
    align-items: center;
    padding: 10px 12px;
    gap: 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    &:last-child { border-bottom: none; }
    
    .el-icon { color: var(--el-text-color-secondary); font-size: 14px; }
    .name { flex: 1; font-size: 13px; font-weight: 500; }
    .version { font-size: 12px; color: var(--el-text-color-secondary); font-family: monospace; }
  }
}

.serious-drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>
