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
            <el-button link type="primary">查看全部</el-button>
          </div>
          <div class="panel-body">
            <el-empty description="暂无近期上传数据" :image-size="80" />
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
import { useRouter } from 'vue-router'
import { Files, Document, Promotion, Connection, Box, Location, Folder } from '@element-plus/icons-vue'

const router = useRouter()

const stats = [
  { label: '想定资源', value: '12', icon: Folder, color: '#409eff' },
  { label: '3D 模型', value: '45', icon: Box, color: '#67c23a' },
  { label: '地形图', value: '8', icon: Location, color: '#e6a23c' },
  { label: '系统通知', value: '23', icon: Promotion, color: '#f56c6c' }
]

const quickActions = [
  { name: '想定库', path: '/scenarios', icon: Folder },
  { name: '模型库', path: '/res/model_glb', icon: Box },
  { name: '地形库', path: '/res/map_terrain', icon: Location },
  { name: '帮助文档', path: '/', icon: Document }
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
