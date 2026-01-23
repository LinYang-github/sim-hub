<template>
  <div class="card-view-container">
    <el-row :gutter="20">
      <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" v-for="item in resources" :key="item.id">
        <el-card class="resource-card" shadow="hover" :body-style="{ padding: '0px' }">
          <!-- 卡片封面区域 -->
          <div class="card-cover" @click="$emit('view-details', item)">
            <div class="card-status-badge">
                <el-tag v-if="item.scope === 'PUBLIC'" size="small" type="success" effect="dark">公共</el-tag>
            </div>
            <div class="card-icon-placeholder">
                <el-icon v-if="typeKey === 'model_glb'" :size="48"><Box /></el-icon>
                <el-icon v-else-if="typeKey === 'map_terrain'" :size="48"><Location /></el-icon>
                <el-icon v-else :size="48"><Files /></el-icon>
            </div>
            <div class="card-ver-badge">
                {{ item.latest_version?.semver || 'v' + (item.latest_version?.version_num || 0) }}
            </div>
          </div>
          
          <div class="card-info" @click="$emit('view-details', item)">
              <div class="card-title" :title="item.name">{{ item.name }}</div>
              <div class="card-meta">
                <span>{{ formatSize(item.latest_version?.file_size) }}</span>
                <span :class="['card-status-text', item.latest_version?.state?.toLowerCase()]">
                  {{ item.latest_version?.state === 'ACTIVE' ? '已就绪' : '处理中' }}
                </span>
              </div>
          </div>

          <div class="card-footer">
              <el-button link type="primary" :disabled="item.latest_version?.state !== 'ACTIVE'" @click="$emit('download', item)">
                <el-icon><Download /></el-icon> 下载
              </el-button>
              
              <el-dropdown trigger="click" @command="(cmd) => handleCommand(cmd, item)">
                <el-button link>
                  <el-icon><MoreFilled /></el-icon> 更多
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu class="premium-dropdown">
                    <el-dropdown-item command="details">
                      <div class="menu-item-content">
                        <el-icon class="menu-icon"><InfoFilled /></el-icon>
                        <span>详情资料</span>
                      </div>
                    </el-dropdown-item>
                    
                    <el-dropdown-item command="tags">
                      <div class="menu-item-content">
                        <el-icon class="menu-icon"><PriceTag /></el-icon>
                        <span>编辑标签</span>
                      </div>
                    </el-dropdown-item>
                    
                    <el-dropdown-item v-if="enableScope && item.owner_id === 'admin'" class="nested-menu-item">
                      <el-dropdown trigger="hover" placement="right" @command="(scopeCmd) => $emit('change-scope', item, scopeCmd)">
                         <div class="menu-item-content">
                           <el-icon class="menu-icon"><Promotion /></el-icon>
                           <span>权限设置</span>
                         </div>
                         <template #dropdown>
                           <el-dropdown-menu>
                             <el-dropdown-item command="PRIVATE" :disabled="item.scope === 'PRIVATE'">设为私有</el-dropdown-item>
                             <el-dropdown-item command="PUBLIC" :disabled="item.scope === 'PUBLIC'">设为公开</el-dropdown-item>
                           </el-dropdown-menu>
                         </template>
                      </el-dropdown>
                    </el-dropdown-item>

                    <el-dropdown-item divided command="delete" class="delete-action">
                      <div class="menu-item-content">
                        <el-icon class="menu-icon"><Delete /></el-icon>
                        <span>删除资源</span>
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

<script setup lang="ts">
import { 
  Box, Location, Files, Download, Delete, PriceTag, MoreFilled, InfoFilled, Promotion
} from '@element-plus/icons-vue'
import { formatSize } from '../../../core/utils/format'

defineProps<{
  resources: any[]
  typeKey: string
  enableScope?: boolean
}>()

const emit = defineEmits(['edit-tags', 'view-details', 'download', 'delete', 'change-scope'])

const handleCommand = (cmd: string, row: any) => {
  switch(cmd) {
    case 'details': emit('view-details', row); break;
    case 'tags': emit('edit-tags', row); break;
    case 'delete': emit('delete', row); break;
  }
}
</script>

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

    .card-icon-placeholder {
      transform: scale(1.05);
      color: var(--el-color-primary);
    }
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
  
  .card-status-badge {
    position: absolute;
    top: 10px;
    right: 10px;
  }
  
  .card-icon-placeholder {
    color: var(--el-text-color-placeholder);
    transition: all 0.3s;
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

:deep(.premium-dropdown) {
  padding: 4px 0;

  .el-dropdown-menu__item {
    padding: 0 !important;
    background-color: transparent !important;
    
    &:hover { background-color: transparent !important; }

    .menu-item-content {
      display: flex;
      align-items: center;
      width: 100%;
      padding: 9px 16px;
      gap: 12px;
      font-size: 14px;
      color: var(--el-text-color-regular);
      transition: all 0.2s ease;
      
      &:hover {
        background-color: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
        .menu-icon { color: var(--el-color-primary); }
      }

      .menu-icon {
        width: 16px;
        font-size: 16px;
        display: flex;
        justify-content: center;
        color: var(--el-text-color-secondary);
      }
    }
  }

  .delete-action .menu-item-content {
    color: var(--el-color-danger);
    .menu-icon { color: var(--el-color-danger); opacity: 0.8; }

    &:hover {
      background-color: var(--el-color-danger-light-9);
      color: var(--el-color-danger);
      .menu-icon { color: var(--el-color-danger); opacity: 1; }
    }
  }
}

:global(.dark) .card-cover {
  background: linear-gradient(135deg, #2c2c2c 0%, #1f1f1f 100%);
}
</style>
