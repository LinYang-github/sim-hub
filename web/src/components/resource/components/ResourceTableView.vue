<template>
  <el-table :data="resources" style="width: 100%" v-loading="loading" class="premium-table">
    <el-table-column label="资源详情" min-width="250">
      <template #default="scope">
        <div class="resource-info-cell">
          <div class="resource-icon">
            <el-icon v-if="typeKey === 'model_glb'"><Box /></el-icon>
            <el-icon v-else-if="typeKey === 'map_terrain'"><Location /></el-icon>
            <el-icon v-else><Files /></el-icon>
          </div>
          <div class="resource-text">
            <div class="name-row">
              <span class="resource-name">{{ scope.row.name }}</span>
              <el-tag v-if="scope.row.scope === 'PUBLIC'" size="small" type="success" effect="plain" class="scope-tag">公共</el-tag>
            </div>
            <div class="resource-meta">
              <span><el-icon><Clock /></el-icon> {{ formatDate(scope.row.created_at) }}</span>
              <span><el-icon><DataLine /></el-icon> {{ formatSize(scope.row.latest_version?.file_size) }}</span>
            </div>
          </div>
        </div>
      </template>
    </el-table-column>

    <el-table-column label="标签" min-width="150">
      <template #default="scope">
        <div class="tags-wrapper">
          <el-tag 
            v-for="tag in (scope.row.tags || [])" 
            :key="tag" 
            size="small" 
            effect="light"
            round
          >
            {{ tag }}
          </el-tag>
          <el-button link size="small" icon="PriceTag" @click="$emit('edit-tags', scope.row)" />
        </div>
      </template>
    </el-table-column>

    <el-table-column label="版本" width="120">
      <template #default="scope">
        <el-tooltip :content="'点击查看历史版本 - 当前流水: ' + (scope.row.latest_version?.version_num || 1)" placement="top">
          <span class="version-badge clickable" @click="$emit('view-history', scope.row)">
            {{ scope.row.latest_version?.semver || 'v' + (scope.row.latest_version?.version_num || 1) }}
          </span>
        </el-tooltip>
      </template>
    </el-table-column>

    <el-table-column label="状态" width="120">
      <template #default="scope">
        <div class="status-cell">
          <el-tooltip 
            v-if="scope.row.latest_version?.state === 'ACTIVE' && !(scope.row.latest_version?.meta_data?.processed)"
            content="该资源类型无需后端处理，已直接可用"
            placement="top"
          >
            <el-icon class="skip-icon"><CircleCheckFilled /></el-icon>
          </el-tooltip>
          <div v-else :class="['status-dot', scope.row.latest_version?.state.toLowerCase()]"></div>
          <span class="status-text">{{ statusMap[scope.row.latest_version?.state] || scope.row.latest_version?.state }}</span>
        </div>
      </template>
    </el-table-column>

    <el-table-column label="依赖" width="100">
      <template #default="scope">
        <el-button link type="primary" @click="$emit('view-dependencies', scope.row)">
          <el-icon><Connection /></el-icon> 依赖关系
        </el-button>
      </template>
    </el-table-column>

    <el-table-column label="操作" width="220" fixed="right">
      <template #default="scope">
          <div class="op-actions">
            <el-button link type="primary" :disabled="scope.row.latest_version?.state !== 'ACTIVE'" @click="$emit('download', scope.row)">
            <el-icon><Download /></el-icon> 下载
            </el-button>
            <el-button link type="danger" @click="$emit('delete', scope.row)">
            <el-icon><Delete /></el-icon> 删除
            </el-button>
            <el-dropdown v-if="enableScope && scope.row.owner_id === 'admin'" @command="(cmd) => $emit('change-scope', scope.row, cmd)">
            <span class="el-dropdown-link">
              <el-icon><Promotion /></el-icon> 权限
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="PRIVATE" :disabled="scope.row.scope === 'PRIVATE'">设为私有</el-dropdown-item>
                <el-dropdown-item command="PUBLIC" :disabled="scope.row.scope === 'PUBLIC'">设为公开</el-dropdown-item>
              </el-dropdown-menu>
            </template>
            </el-dropdown>
          </div>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { 
  Box, Location, Files, Clock, DataLine, 
  CircleCheckFilled, Connection, Download, Delete, Promotion, PriceTag 
} from '@element-plus/icons-vue'
import { formatDate, formatSize } from '../../../core/utils/format'

defineProps<{
  resources: any[]
  loading: boolean
  typeKey: string
  enableScope: boolean
  statusMap: Record<string, string>
}>()

defineEmits(['edit-tags', 'view-history', 'view-dependencies', 'download', 'delete', 'change-scope'])
</script>

<style scoped lang="scss">
.premium-table {
  --el-table-header-bg-color: var(--el-fill-color-lighter);
  
  :deep(th.el-table__cell) {
    font-weight: 600;
    color: var(--el-text-color-regular);
    font-size: 13px;
    padding: 12px 0;
  }
  
  :deep(td.el-table__cell) {
    padding: 14px 0;
  }
}

.resource-info-cell {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.resource-icon {
  width: 40px;
  height: 40px;
  background: var(--el-fill-color);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-primary);
  font-size: 20px;
  flex-shrink: 0;
}

.resource-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-name {
  font-weight: 500;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.resource-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  
  span {
    display: flex;
    align-items: center;
    gap: 4px;
  }
}

.version-badge {
  background: var(--el-fill-color-light);
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  font-weight: 500;
  
  &.clickable {
    cursor: pointer;
    transition: all 0.2s;
    &:hover {
      background: var(--el-color-primary-light-8);
      color: var(--el-color-primary);
    }
  }
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
  
  &.active {
    background: var(--el-color-success);
    box-shadow: 0 0 8px var(--el-color-success-light-5);
  }
  
  &.processing {
    background: var(--el-color-primary);
    animation: statusPulse 2s infinite;
  }
  
  &.failed {
    background: var(--el-color-danger);
  }
}

.status-text {
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.tags-wrapper {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}

.op-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.el-dropdown-link {
  cursor: pointer;
  color: var(--el-color-primary);
  display: flex;
  align-items: center;
  font-size: 12px;
  gap: 4px;
}

@keyframes statusPulse {
  0% { transform: scale(0.9); opacity: 0.6; }
  50% { transform: scale(1.1); opacity: 1; }
  100% { transform: scale(0.9); opacity: 0.6; }
}
</style>
