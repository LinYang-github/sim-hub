<template>
  <div class="gallery-view">
    <el-row :gutter="24">
      <el-col 
        v-for="res in resources" 
        :key="res.id" 
        :xs="24" :sm="24" :md="12" :lg="12" 
        class="gallery-item-col"
      >
        <div class="gallery-card" @click="$emit('view-details', res)">
          <!-- Top: Interactive or Static Map Surface -->
          <div class="card-visual-header">
            <template v-if="res.type_key === 'map_service'">
              <GeoPreview 
                v-if="isGisService(res)"
                :url="res.latest_version?.download_url || ''"
                :meta-data="res.latest_version?.meta_data" 
                :interactive="false"
                class="mini-map"
              />
              <div v-else class="preview-placeholder">
                <el-icon class="ph-icon"><MapLocation /></el-icon>
                <span>地图服务预览不可用</span>
              </div>
            </template>
            <template v-else>
               <div class="preview-placeholder generic">
                 <el-icon class="ph-icon"><component :is="icon || 'Box'" /></el-icon>
                 <span>{{ res.name }}</span>
               </div>
            </template>

            <!-- Status Tag Overlay -->
            <div class="status-overlay">
              <el-tag 
                :type="statusMap[res.latest_version?.state || 0]?.type" 
                size="small" 
                effect="dark"
              >
                {{ statusMap[res.latest_version?.state || 0]?.text }}
              </el-tag>
            </div>
            
            <!-- Actions Overlay (Visible on Hover) -->
            <div class="actions-overlay" @click.stop>
              <el-tooltip content="下载" placement="top">
                <el-button circle size="small" @click="$emit('download', res)">
                  <el-icon><Download /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip :content="`更多操作`" placement="top">
                <el-dropdown trigger="click" @command="(cmd: string) => handleCommand(cmd, res)">
                  <el-button circle size="small">
                    <el-icon><MoreFilled /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="rename"><el-icon><EditPen /></el-icon>重命名</el-dropdown-item>
                      <el-dropdown-item command="move"><el-icon><Rank /></el-icon>移动分类</el-dropdown-item>
                      <el-dropdown-item command="tags"><el-icon><PriceTag /></el-icon>编辑标签</el-dropdown-item>
                      <el-dropdown-item command="history"><el-icon><Clock /></el-icon>版本历史</el-dropdown-item>
                      
                      <!-- Custom Actions -->
                      <template v-if="customActions && customActions.length">
                        <el-dropdown-item divided disabled>特殊操作</el-dropdown-item>
                        <el-dropdown-item 
                          v-for="action in customActions" 
                          :key="action.key" 
                          :command="`custom:${action.key}`"
                        >
                          <el-icon v-if="action.icon"><component :is="action.icon" /></el-icon>
                          {{ action.label }}
                        </el-dropdown-item>
                      </template>

                      <el-dropdown-item divided command="delete" style="color: var(--el-color-danger)">
                        <el-icon><Delete /></el-icon>彻底删除
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </el-tooltip>
            </div>
          </div>

          <!-- Bottom: Info Info -->
          <div class="card-info">
            <div class="info-main">
              <h3 class="res-name">{{ res.name }}</h3>
              <p class="res-meta">
                <span class="version">{{ res.latest_version?.semver }}</span>
                <span class="dot">·</span>
                <span class="time">{{ formatDate(res.updated_at) }}</span>
              </p>
            </div>
            <div class="info-tags" v-if="res.tags?.length">
              <el-tag 
                v-for="tag in res.tags.slice(0, 2)" 
                :key="tag" 
                size="small" 
                round 
                class="tag-pill"
              >
                {{ tag }}
              </el-tag>
              <span v-if="res.tags.length > 2" class="more-tags">+{{ res.tags.length - 2 }}</span>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { MapLocation, Download, PriceTag, MoreFilled, EditPen, Rank, Clock, Delete } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import GeoPreview from '../previewers/GeoPreview.vue'

const props = defineProps<{
  resources: any[]
  loading: boolean
  statusMap: any
  icon?: string
  customActions?: any[]
}>()

const emit = defineEmits(['view-details', 'download', 'edit-tags', 'delete', 'rename', 'move', 'view-history', 'custom-action'])

const formatDate = (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm')

const isGisService = (res: any) => {
  const meta = res.latest_version?.meta_data || {}
  return !!(meta.url || meta.center || meta.bounds || meta.geometry)
}

const handleCommand = (command: string, res: any) => {
  if (command === 'delete') emit('delete', res)
  else if (command === 'rename') emit('rename', res)
  else if (command === 'move') emit('move', res)
  else if (command === 'tags') emit('edit-tags', res)
  else if (command === 'history') emit('view-history', res)
  else if (command.startsWith('custom:')) {
    emit('custom-action', command.replace('custom:', ''), res)
  }
}
</script>

<script lang="ts">
export const viewMeta = {
  key: 'gallery',
  label: '空间画廊',
  icon: 'Collection'
}
</script>

<style scoped lang="scss">
.gallery-view {
  padding: 8px;
}

.gallery-item-col {
  margin-bottom: 24px;
}

.gallery-card {
  background: var(--sidebar-bg);
  border-radius: 16px;
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 4px 20px -8px rgba(0, 0, 0, 0.08);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  position: relative;

  &:hover {
    transform: translateY(-8px);
    box-shadow: 0 12px 32px -12px rgba(0, 0, 0, 0.15);
    border-color: var(--el-color-primary-light-5);

    .actions-overlay {
      opacity: 1;
      transform: translateY(0);
    }
    
    .card-visual-header::after {
        opacity: 0.2;
    }
  }
}

.card-visual-header {
  height: 280px;
  background: var(--el-fill-color-light);
  position: relative;
  overflow: hidden;

  // Subtle vignette gradient
  &::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(to top, rgba(0,0,0,0.4), transparent 60%);
    opacity: 0;
    transition: opacity 0.3s;
    pointer-events: none;
  }

  .mini-map {
    width: 100%;
    height: 100%;
    pointer-events: none; // Keep card clickable
    filter: saturate(0.8) contrast(1.1);
  }

  .preview-placeholder {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    color: var(--el-text-color-placeholder);
    background: linear-gradient(135deg, var(--el-fill-color-light) 0%, var(--el-fill-color) 100%);

    .ph-icon {
      font-size: 48px;
      opacity: 0.6;
    }
    
    span {
        font-size: 14px;
        font-weight: 500;
    }
  }
}

.status-overlay {
  position: absolute;
  top: 16px;
  left: 16px;
  z-index: 2;
}

.actions-overlay {
  position: absolute;
  top: 16px;
  right: 16px;
  display: flex;
  gap: 8px;
  z-index: 10;
  opacity: 0;
  transform: translateY(-10px);
  transition: all 0.2s ease;
  
  :deep(.el-button) {
    background: rgba(255, 255, 255, 0.9);
    backdrop-filter: blur(4px);
    border: none;
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    
    &:hover {
      background: #fff;
      color: var(--el-color-primary);
    }
  }
}

.card-info {
  padding: 20px;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;

  .info-main {
    flex: 1;
    min-width: 0;
    
    .res-name {
      margin: 0 0 6px 0;
      font-size: 17px;
      font-weight: 700;
      color: var(--el-text-color-primary);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .res-meta {
      margin: 0;
      font-size: 13px;
      color: var(--el-text-color-secondary);
      display: flex;
      align-items: center;
      gap: 6px;

      .version {
        font-family: var(--el-font-family-mono);
        color: var(--el-color-primary);
        font-weight: 600;
      }
      
      .dot {
          opacity: 0.5;
      }
    }
  }
  
  .info-tags {
      display: flex;
      gap: 4px;
      align-items: center;
      
      .tag-pill {
          background: var(--el-fill-color-lighter);
          border: none;
          color: var(--el-text-color-regular);
          font-weight: 500;
      }
      
      .more-tags {
          font-size: 11px;
          color: var(--el-text-color-placeholder);
          font-weight: 600;
      }
  }
}

.dark {
  .gallery-card {
    background: #1d1d1f;
    
    .card-visual-header {
        background: #2c2c2e;
    }
    
    .actions-overlay :deep(.el-button) {
        background: rgba(0, 0, 0, 0.6);
        color: #fff;
    }
  }
}
</style>
