<template>
  <div class="card-view-container">
    <el-row :gutter="20">
      <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" v-for="item in resources" :key="item.id">
        <el-card class="resource-card" shadow="hover" :body-style="{ padding: '0px' }">
          <!-- 卡片封面区域 -->
          <div class="card-cover" @click="$emit('view-details', item)">
            <div class="card-status-badge">
                <el-tag v-if="item.scope === RESOURCE_SCOPE.PUBLIC" size="small" type="success" effect="dark">公共</el-tag>
            </div>
            <ResourcePreview 
              :type-key="typeKey" 
              :viewer="viewer"
              :icon="icon"
              :download-url="item.latest_version?.download_url"
              :state="item.latest_version?.state"
              :status-text="item.latest_version?.state ? (statusMap[item.latest_version!.state] || item.latest_version!.state) : '-'"
              :meta-data="item.latest_version?.meta_data"
            />
            <div class="card-ver-badge">
                {{ item.latest_version?.semver || 'v' + (item.latest_version?.version_num || 0) }}
            </div>
          </div>
          
          <div class="card-info" @click="$emit('view-details', item)">
              <div class="card-title" :title="item.name">{{ item.name }}</div>
              <div class="card-meta">
                <span>{{ formatSize(item.latest_version?.file_size) }}</span>
                <span :class="['card-status-text', item.latest_version?.state?.toLowerCase()]">
                  {{ item.latest_version?.state ? (statusMap[item.latest_version!.state] || item.latest_version!.state) : '-' }}
                </span>
              </div>
          </div>

          <div class="card-footer">
              <el-button link type="primary" :disabled="item.latest_version?.state !== RESOURCE_STATE.ACTIVE" @click="$emit('download', item)">
                <el-icon><Download /></el-icon> 下载
              </el-button>
              
              <el-dropdown trigger="click" popper-class="resource-popper" @command="(cmd) => handleCommand(cmd, item)">
                <el-button link>
                  <el-icon><MoreFilled /></el-icon> 更多
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="details">
                      <div class="menu-item-content">
                        <el-icon><InfoFilled /></el-icon>
                        <span>查看详情</span>
                      </div>
                    </el-dropdown-item>
                    
                    <template v-if="customActions && customActions.length">
                      <el-dropdown-item 
                        v-for="action in customActions" 
                        :key="action.key" 
                        :command="{ key: action.key, custom: true }"
                      >
                        <div class="menu-item-content">
                          <el-icon>
                              <component :is="action.icon" />
                          </el-icon>
                          <span>{{ action.label }}</span>
                        </div>
                      </el-dropdown-item>
                    </template>
                    
                    <el-dropdown-item command="tags">
                      <div class="menu-item-content">
                        <el-icon><PriceTag /></el-icon>
                        <span>编辑标签</span>
                      </div>
                    </el-dropdown-item>
                    
                    <el-dropdown-item command="rename">
                      <div class="menu-item-content">
                        <el-icon><Edit /></el-icon>
                        <span>重命名</span>
                      </div>
                    </el-dropdown-item>

                    <el-dropdown-item command="move">
                      <div class="menu-item-content">
                        <el-icon><Promotion /></el-icon>
                        <span>移动分类</span>
                      </div>
                    </el-dropdown-item>
                    
                    <el-dropdown-item v-if="enableScope && item.owner_id === DEFAULT_ADMIN_ID" class="nested-menu-parent">
                      <el-dropdown trigger="hover" placement="right" popper-class="resource-popper" @command="(scopeCmd) => $emit('change-scope', item, scopeCmd)">
                         <div class="menu-item-content">
                           <el-icon><Promotion /></el-icon>
                           <span>权限设置</span>
                         </div>
                         <template #dropdown>
                           <el-dropdown-menu>
                             <el-dropdown-item :command="RESOURCE_SCOPE.PRIVATE" :disabled="item.scope === RESOURCE_SCOPE.PRIVATE">
                                <div class="menu-item-content">
                                  <el-icon><Lock /></el-icon>
                                  <span>设为私有</span>
                                </div>
                             </el-dropdown-item>
                             <el-dropdown-item :command="RESOURCE_SCOPE.PUBLIC" :disabled="item.scope === RESOURCE_SCOPE.PUBLIC">
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
                        <span>从库删除</span>
                      </div>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts">
export const viewMeta = {
  key: 'card', 
  label: '卡片视图', 
  icon: 'Grid'
}
</script>

<script setup lang="ts">
import { 
  Download, Delete, PriceTag, MoreFilled, InfoFilled, Promotion, Lock, Edit
} from '@element-plus/icons-vue'
import { formatSize } from '../../../core/utils/format'
import type { Resource, ResourceScope } from '../../../core/types/resource'
import type { CustomAction } from '../../../core/types'
import { RESOURCE_STATE, RESOURCE_SCOPE, DEFAULT_ADMIN_ID } from '../../../core/constants/resource'
import ResourcePreview from '../previewers/ResourcePreview.vue'

defineProps<{
  resources: Resource[]
  typeKey: string
  enableScope?: boolean
  statusMap: Record<string, string>
  viewer?: string
  icon?: string
  customActions?: CustomAction[]
}>()

const emit = defineEmits<{
  (e: 'edit-tags', row: Resource): void
  (e: 'view-details', row: Resource): void
  (e: 'download', row: Resource): void
  (e: 'delete', row: Resource): void
  (e: 'change-scope', row: Resource, scope: ResourceScope): void
  (e: 'rename', row: Resource): void
  (e: 'move', row: Resource): void
  (e: 'custom-action', key: string, row: Resource): void
}>()

const handleCommand = (cmd: string | any, row: Resource) => {
  if (typeof cmd === 'object' && cmd.custom) {
      emit('custom-action', cmd.key, row)
      return
  }
  switch(cmd) {
    case 'details': emit('view-details', row); break;
    case 'tags': emit('edit-tags', row); break;
    case 'rename': emit('rename', row); break;
    case 'move': emit('move', row); break;
    case 'delete': emit('delete', row); break;
  }
}
</script>

<style lang="scss">
/* 全局样式共用 block (确保 Table/Card 视图下拉样式绝对统一) */
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

    &.danger-item .menu-item-content {
      color: var(--el-color-danger) !important;
      .el-icon { color: var(--el-color-danger) !important; }

      &:hover {
        background-color: var(--el-color-danger-light-9);
      }
    }

    &.el-dropdown-menu__item--divided {
      margin-top: 6px;
      &::before { margin: 0; background-color: var(--el-border-color-lighter); }
    }
  }
}
</style>

<style scoped lang="scss">
.card-view-container {
  padding: 10px 0;
}

.resource-card {
  margin-bottom: 20px;
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s;
  border: 1px solid var(--el-border-color-lighter);

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 12px 24px rgba(0, 0, 0, 0.08);
  }
}

.card-cover {
  height: 120px;
  background: linear-gradient(135deg, var(--el-fill-color-light) 0%, var(--el-fill-color) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  cursor: pointer;
  overflow: hidden;
  
  .card-status-badge {
    position: absolute;
    top: 10px;
    right: 10px;
    z-index: 10;
  }
  
  .card-ver-badge {
    position: absolute;
    bottom: 8px;
    right: 8px;
    background: rgba(0,0,0,0.5);
    color: #fff;
    padding: 1px 6px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 500;
    z-index: 10;
  }
}

.card-info {
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  cursor: pointer;
  
  .card-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 8px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  
  .card-meta {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .card-status-text {
    &.active { color: var(--el-color-success); }
    &.processing { color: var(--el-color-primary); }
  }
}

.card-footer {
  padding: 6px 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--el-fill-color-extra-light);
}

:global(.dark) .card-cover {
  background: linear-gradient(135deg, #2c2c2c 0%, #1f1f1f 100%);
}
</style>
