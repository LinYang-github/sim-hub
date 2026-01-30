<template>
  <div class="dashboard-container">

    <!-- Stats Cards -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6" v-for="stat in stats" :key="stat.label">
        <div class="stat-card">
          <div class="stat-icon" :style="{ backgroundColor: stat.color + '22', color: stat.color }">
            <el-icon><component :is="stat.icon" /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ stat.value }}</div>
            <div class="stat-label">{{ stat.label }}</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- Content Sections -->
    <el-row :gutter="20" class="dashboard-body">
      <el-col :span="16">
        <div class="content-panel">
          <div class="panel-header">
            <span class="panel-title">最近上传资源</span>
            <el-button link type="primary" @click="triggerGlobalSearch">查看全部</el-button>
          </div>
          <div class="panel-body recent-list-body">
            <template v-if="recentResources && recentResources.length > 0">
               <div class="recent-list">
                  <div 
                    v-for="res in recentResources" 
                    :key="res.id" 
                    class="recent-res-item"
                    @click="router.push(`/res/${res.type_key}`)"
                  >
                     <div class="res-main">
                        <el-icon class="res-icon"><component :is="res.icon" /></el-icon>
                        <div class="res-info">
                           <span class="res-name">{{ res.name }}</span>
                           <span class="res-type">{{ res.typeName }}</span>
                        </div>
                     </div>
                     <div class="res-side">
                        <el-tag size="small" :type="getStatusType(res.latest_version?.state)" effect="plain">
                           {{ res.latest_version?.state || 'UNKNOWN' }}
                        </el-tag>
                        <span class="res-date">{{ formatDate(res.created_at) }}</span>
                     </div>
                  </div>
               </div>
            </template>
            <el-empty v-else description="暂无近期上传数据" :image-size="80" />
          </div>
        </div>
      </el-col>
      <el-col :span="8">
        <div class="content-panel">
          <div class="panel-header">
            <span class="panel-title">快速入口</span>
          </div>
          <div class="panel-body quick-actions">
            <div v-for="action in quickActions" :key="action.name" class="action-item" @click="router.push(action.path)">
              <el-icon class="action-icon"><component :is="action.icon" /></el-icon>
              <div class="action-name">{{ action.name }}</div>
            </div>
          </div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, markRaw } from 'vue'
import { useRouter } from 'vue-router'
import { Files, Document, Promotion, Connection, Box, Location, Folder, Search, Tickets } from '@element-plus/icons-vue'
import request from '../core/utils/request'
import { moduleManager } from '../core/moduleManager'
import dayjs from 'dayjs'

const router = useRouter()

const stats = ref([
  { label: '想定资源', key: 'scenario', value: '0', icon: markRaw(Folder), color: '#409eff' },
  { label: '3D 模型', key: 'model_glb', value: '0', icon: markRaw(Box), color: '#67c23a' },
  { label: '地图服务', key: 'map_service', value: '0', icon: markRaw(Location), color: '#e6a23c' },
  { label: '测试资源', key: 'test_db', value: '0', icon: markRaw(Tickets), color: '#f56c6c' }
])

const recentResources = ref<any[]>([])

const triggerGlobalSearch = () => {
    window.dispatchEvent(new CustomEvent('open-global-search'))
}

const fetchStats = async () => {
    try {
        const data = await request.get<any>('/api/v1/dashboard/stats')
        if (data && data.total_counts) {
            stats.value.forEach(s => {
                if (data.total_counts[s.key] !== undefined) {
                    s.value = data.total_counts[s.key].toString()
                }
            })
        }
        if (data && data.recent_items) {
            recentResources.value = data.recent_items.map((item: any) => {
                const typeConfig = moduleManager.getActiveModules().value.find(t => t.key === item.type_key)
                const icon = typeConfig?.icon || 'Files'
                return {
                    ...item,
                    typeName: typeConfig?.typeName || item.type_key,
                    icon: typeof icon === 'string' ? icon : markRaw(icon)
                }
            })
        }
    } catch (e) {}
}

onMounted(() => {
    fetchStats()
})

const getStatusType = (state?: string) => {
    if (state === 'ACTIVE') return 'success'
    if (state === 'PROCESSING') return 'primary'
    if (state === 'FAILED') return 'danger'
    return 'info'
}

const formatDate = (date: string) => dayjs(date).format('MM-DD HH:mm')

interface QuickAction {
  name: string
  path: string
  icon: any
}

const quickActions: QuickAction[] = [
  { name: '想定库', path: '/res/scenario', icon: markRaw(Folder) },
  { name: '模型库', path: '/res/model_glb', icon: markRaw(Box) },
  { name: '地图服务', path: '/res/map_service', icon: markRaw(Location) },
  { name: '帮助文档', path: '/', icon: markRaw(Document) }
]
</script>

<style scoped lang="scss">
.dashboard-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  animation: fadeIn 0.4s ease-out;
}


.stats-row {
  margin-bottom: 4px;
}

.stat-card {
  background: var(--sidebar-bg);
  padding: 24px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: var(--el-box-shadow-lighter);
  transition: transform 0.2s;

  &:hover {
    transform: translateY(-4px);
  }

  .stat-icon {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
  }

  .stat-value {
    font-size: 24px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .stat-label {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    margin-top: 2px;
  }
}

.content-panel {
  background: var(--sidebar-bg);
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-lighter);
  display: flex;
  flex-direction: column;
  min-height: 400px;

  .panel-header {
    height: 56px;
    padding: 0 20px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--el-border-color-lighter);

    .panel-title {
      font-size: 15px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .panel-body {
    flex: 1;
    padding: 20px;
    display: flex;
    align-items: center;
    justify-content: center;

    &.recent-list-body {
       align-items: stretch;
       justify-content: flex-start;
       padding: 8px 0;
    }
  }

  .recent-list {
     width: 100%;
     display: flex;
     flex-direction: column;
     
     .recent-res-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 20px;
        cursor: pointer;
        transition: background 0.2s;

        &:hover {
           background: var(--el-fill-color-light);
        }

        .res-main {
           display: flex;
           align-items: center;
           gap: 12px;

           .res-icon {
              font-size: 20px;
              color: var(--el-color-primary);
              width: 36px;
              height: 36px;
              background: var(--el-color-primary-light-9);
              border-radius: 8px;
              display: flex;
              align-items: center;
              justify-content: center;
           }

           .res-info {
              display: flex;
              flex-direction: column;

              .res-name {
                 font-size: 14px;
                 font-weight: 500;
                 color: var(--el-text-color-primary);
              }
              .res-type {
                 font-size: 12px;
                 color: var(--el-text-color-secondary);
              }
           }
        }

        .res-side {
           display: flex;
           flex-direction: column;
           align-items: flex-end;
           gap: 4px;

           .res-date {
              font-size: 11px;
              color: var(--el-text-color-placeholder);
           }
        }
     }
  }

  .quick-actions {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;
    align-content: start;
    justify-items: center;

    .action-item {
      width: 100%;
      height: 100px;
      background: var(--el-fill-color-lighter);
      border-radius: 12px;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 10px;
      cursor: pointer;
      transition: all 0.2s;

      &:hover {
        background: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
      }

      .action-icon {
        font-size: 28px;
      }

      .action-name {
        font-size: 13px;
        font-weight: 500;
      }
    }
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
