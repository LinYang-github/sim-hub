<template>
  <el-table :data="resources" style="width: 100%" v-loading="loading" class="premium-table">
    <!-- 1. Name -->
    <el-table-column label="资源名称" min-width="200">
      <template #default="scope">
        <div class="resource-info-cell clickable" @click="$emit('view-details', scope.row)">
          <div class="resource-icon">
            <el-icon v-if="typeKey === 'model_glb'"><Box /></el-icon>
            <el-icon v-else-if="typeKey === 'map_terrain'"><Location /></el-icon>
            <el-icon v-else><Files /></el-icon>
          </div>
          <span class="resource-name" :title="scope.row.name">{{ scope.row.name }}</span>
        </div>
      </template>
    </el-table-column>

    <!-- 2. Version -->
    <el-table-column label="当前版本" width="120">
      <template #default="scope">
        <span class="version-text">{{ scope.row.latest_version?.semver || 'v' + (scope.row.latest_version?.version_num || 1) }}</span>
      </template>
    </el-table-column>

    <!-- 3. Status -->
    <el-table-column label="状态" width="120">
      <template #default="scope">
        <div class="status-indicator">
          <div :class="['status-dot', scope.row.latest_version?.state?.toLowerCase()]"></div>
          <span class="status-label">{{ scope.row.latest_version?.state ? (statusMap[scope.row.latest_version!.state] || scope.row.latest_version!.state) : '-' }}</span>
        </div>
      </template>
    </el-table-column>

    <!-- 4. Size -->
    <el-table-column label="大小" width="100">
      <template #default="scope">
        <span class="meta-item">{{ formatSize(scope.row.latest_version?.file_size) }}</span>
      </template>
    </el-table-column>

    <!-- 5. Date -->
    <el-table-column label="更新时间" width="140">
      <template #default="scope">
        <span class="meta-item">{{ formatDate(scope.row.created_at).split(' ')[0] }}</span>
      </template>
    </el-table-column>

    <!-- 6. Actions -->
    <el-table-column label="操作" width="120" fixed="right" align="center" header-align="center">
      <template #default="scope">
          <div class="op-actions">
            <!-- Primary Action: Download -->
            <el-tooltip content="下载资源" placement="top">
              <el-button 
                circle 
                type="primary" 
                plain 
                size="small"
                :disabled="scope.row.latest_version?.state !== RESOURCE_STATE.ACTIVE" 
                @click="$emit('download', scope.row)"
              >
                <el-icon><Download /></el-icon>
              </el-button>
            </el-tooltip>

            <!-- Secondary Actions: Dropdown -->
            <el-dropdown trigger="click" popper-class="resource-popper" @command="(cmd) => handleCommand(cmd, scope.row)">
              <el-button circle size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="details">
                    <div class="menu-item-content">
                      <el-icon><InfoFilled /></el-icon>
                      <span>详情资料</span>
                    </div>
                  </el-dropdown-item>
                  
                  <el-dropdown-item command="tags">
                    <div class="menu-item-content">
                      <el-icon><PriceTag /></el-icon>
                      <span>编辑标签</span>
                    </div>
                  </el-dropdown-item>
                  
                  <el-dropdown-item v-if="enableScope && scope.row.owner_id === DEFAULT_ADMIN_ID" class="nested-menu-parent">
                    <el-dropdown trigger="hover" placement="left" popper-class="resource-popper" @command="(scopeCmd) => $emit('change-scope', scope.row, scopeCmd)">
                       <div class="menu-item-content">
                         <el-icon><Promotion /></el-icon>
                         <span>权限设置</span>
                       </div>
                       <template #dropdown>
                         <el-dropdown-menu>
                           <el-dropdown-item :command="RESOURCE_SCOPE.PRIVATE" :disabled="scope.row.scope === RESOURCE_SCOPE.PRIVATE">
                              <div class="menu-item-content">
                                <el-icon><Lock /></el-icon>
                                <span>设为私有</span>
                              </div>
                           </el-dropdown-item>
                           <el-dropdown-item :command="RESOURCE_SCOPE.PUBLIC" :disabled="scope.row.scope === RESOURCE_SCOPE.PUBLIC">
                              <div class="menu-item-content">
                                <el-icon><Promotion /></el-icon>
                                <span>设为公开</span>
                              </div>
                           </el-dropdown-item>
                         </el-dropdown-menu>
                       </template>
                    </el-dropdown>
                  </el-dropdown-item>

                  <el-dropdown-item divided command="delete" class="danger-item">
                    <div class="menu-item-content">
                      <el-icon><Delete /></el-icon>
                      <span>删除资源</span>
                    </div>
                  </el-dropdown-item>
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
  Download, Delete, Promotion, PriceTag, MoreFilled, InfoFilled, Lock
} from '@element-plus/icons-vue'
import { formatDate, formatSize } from '../../../core/utils/format'
import type { Resource, ResourceScope } from '../../../core/types/resource'
import { RESOURCE_STATE, RESOURCE_SCOPE, DEFAULT_ADMIN_ID } from '../../../core/constants/resource'

defineProps<{
  resources: Resource[]
  loading: boolean
  typeKey: string
  enableScope: boolean
  statusMap: Record<string, string>
}>()

const emit = defineEmits<{
  (e: 'edit-tags', row: Resource): void
  (e: 'view-details', row: Resource): void
  (e: 'download', row: Resource): void
  (e: 'delete', row: Resource): void
  (e: 'change-scope', row: Resource, scope: ResourceScope): void
}>()

const handleCommand = (cmd: string, row: Resource) => {
  if (cmd === 'details') emit('view-details', row)
  if (cmd === 'tags') emit('edit-tags', row)
  if (cmd === 'delete') emit('delete', row)
}
</script>

<style lang="scss">
/* 全局样式 block (非 scoped)，确保传送到 body 的 popper 能生效 */
.resource-popper.el-popper {
  background: var(--el-bg-color-overlay) !important;
  border: 1px solid var(--el-border-color-light) !important;
  border-radius: 8px !important;
  box-shadow: var(--el-box-shadow-light) !important;
  padding: 0 !important;

  .el-dropdown-menu {
    padding: 6px 0 !important;
    min-width: 160px !important;
  }

  .el-dropdown-menu__item {
    padding: 0 !important;
    
    &:hover {
      background-color: transparent !important;
    }

    .menu-item-content {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 10px 16px;
      width: 100%;
      box-sizing: border-box;
      font-size: 14px;
      color: var(--el-text-color-regular);
      transition: all 0.2s;

      .el-icon {
        font-size: 16px;
        width: 16px;
        color: var(--el-text-color-secondary);
      }

      &:hover {
        background-color: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
        .el-icon { color: var(--el-color-primary); }
      }
    }

    // 严谨的红色危险项处理
    &.danger-item .menu-item-content {
      color: var(--el-color-danger) !important;
      .el-icon { color: var(--el-color-danger) !important; }

      &:hover {
        background-color: var(--el-color-danger-light-9);
      }
    }

    // 处理分界线背景
    &.el-dropdown-menu__item--divided {
      margin-top: 6px;
      &::before { margin: 0; background-color: var(--el-border-color-lighter); }
    }
  }
}
</style>

<style scoped lang="scss">
.premium-table {
  --el-table-header-bg-color: var(--el-fill-color-lighter);
  
  :deep(th.el-table__cell) {
    font-weight: 600;
    color: var(--el-text-color-primary);
    font-size: 13px;
    padding: 12px 0;
  }
  
  :deep(td.el-table__cell) {
    padding: 12px 0;
  }
}

.clickable {
  cursor: pointer;
  transition: opacity 0.2s;
  &:hover { opacity: 0.8; }
}

.resource-info-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.resource-icon {
  width: 32px;
  height: 32px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-primary);
  font-size: 16px;
  flex-shrink: 0;
}

.resource-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.version-text {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  padding: 1px 6px;
  border-radius: 4px;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  .status-label { font-size: 12px; color: var(--el-text-color-secondary); }
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
  
  &.active {
    background: var(--el-color-success);
    box-shadow: 0 0 4px var(--el-color-success-light-5);
  }
  
  &.processing {
    background: var(--el-color-primary);
    animation: statusPulse 2s infinite;
  }
  
  &.failed {
    background: var(--el-color-danger);
  }
}

.meta-item {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.op-actions {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
}

@keyframes statusPulse {
  0% { transform: scale(0.9); opacity: 0.6; }
  50% { transform: scale(1.1); opacity: 1; }
  100% { transform: scale(0.9); opacity: 0.6; }
}
</style>
