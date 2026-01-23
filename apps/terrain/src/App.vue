<template>
  <div class="terrain-container">
    <div class="terrain-dashboard">
      <div class="dashboard-header">
        <h2>高海拔地形数据资产管理</h2>
        <div class="view-toggle">
          <el-radio-group v-model="viewMode" size="small">
            <el-radio-button value="list">资源列表</el-radio-button>
            <el-radio-button value="geo">地理视图</el-radio-button>
          </el-radio-group>
        </div>
      </div>

      <!-- 统计卡片 -->
      <el-row :gutter="20" class="stat-cards">
        <el-col :span="6" v-for="stat in stats" :key="stat.title">
          <el-card shadow="hover" class="stat-card">
            <div class="stat-content">
              <el-icon :size="24" :class="stat.color"><component :is="stat.icon" /></el-icon>
              <div class="stat-info">
                <span class="stat-label">{{ stat.title }}</span>
                <span class="stat-value">{{ stat.value }}</span>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 主要内容区 -->
      <div class="content-wrapper">
        <template v-if="viewMode === 'list'">
          <!-- 这里我们将直接实现地形专用的列表，因为我们现在是独立的项目，可以高度定制 -->
          <div class="resource-header">
            <div class="left">
              <el-button type="primary" :icon="Upload" @click="handleUpload">
                上传地形切片
              </el-button>
            </div>
            <div class="right">
              <el-input
                v-model="searchQuery"
                placeholder="搜索地形资源..."
                prefix-icon="Search"
                style="width: 240px"
              />
            </div>
          </div>

          <el-table :data="filteredResources" border style="width: 100%; margin-top: 12px">
            <el-table-column prop="name" label="名称" min-width="180">
              <template #default="{ row }">
                <div class="name-cell">
                  <el-icon><MapLocation /></el-icon>
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="tileCount" label="瓦片总数" width="120" align="center" />
            <el-table-column prop="size" label="存储占用" width="120" align="center" />
            <el-table-column prop="coverage" label="覆盖范围 (km²)" width="150" align="center" />
            <el-table-column prop="updateTime" label="更新时间" width="180" />
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <div class="operation-buttons">
                  <el-button type="primary" link @click="handlePreview(row)">预览</el-button>
                  <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </template>
        <div v-else class="geo-placeholder">
          <el-empty description="地理可视化组件集成中...">
            <template #extra>
              <p>这里将集成 Cesium.js 或 OpenLayers 进行全球地形覆盖预览</p>
            </template>
          </el-empty>
        </div>
      </div>
    </div>

    <!-- 上传对话框 (地形专用) -->
    <el-dialog v-model="uploadVisible" title="上传高清地形切片" width="500px">
      <el-upload
        class="terrain-uploader"
        drag
        action="/api/v1/resources/upload"
        multiple
        :headers="uploadHeaders"
      >
        <el-icon class="el-icon--upload"><upload-filled /></el-icon>
        <div class="el-upload__text">
          将地形文件夹拖到此处，或 <em>点击上传</em>
        </div>
        <template #tip>
          <div class="el-upload__tip">
            支持 TMS/XYZ 瓦片标准，文件夹建议压缩为 .zip 后上传
          </div>
        </template>
      </el-upload>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Upload, MapLocation, Search, DataAnalysis, Files, Connection } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useDark } from '@vueuse/core'
import { bridge } from './bridge/guest'

const isDark = useDark()
const viewMode = ref('list')

onMounted(() => {
  // 仅保留主题同步
  bridge.on('THEME_UPDATE', (payload) => {
    isDark.value = payload.theme === 'dark'
  })
})

const searchQuery = ref('')
const uploadVisible = ref(false)

const stats = [
  { title: '地形库容量', value: '1.2 TB', icon: 'Files', color: 'blue' },
  { title: '覆盖区域', value: '45,200 km²', icon: 'MapLocation', color: 'green' },
  { title: '瓦片总数', value: '45.2 M', icon: 'DataAnalysis', color: 'orange' },
  { title: 'API 调用/日', value: '2.5k', icon: 'Connection', color: 'purple' }
]

const resources = ref([
  { id: 1, name: '珠穆朗玛峰高程地形 (L1-L15)', tileCount: '850k', size: '12 GB', coverage: '5,000', updateTime: '2024-03-20 12:45:00' },
  { id: 2, name: '川藏公路沿线精细地形', tileCount: '2.1M', size: '45 GB', coverage: '12,000', updateTime: '2024-03-21 09:30:12' },
  { id: 3, name: '标准全球 90m 基础地形', tileCount: '32M', size: '840 GB', coverage: 'Global', updateTime: '2024-03-15 22:10:05' }
])

const filteredResources = computed(() => {
  if (!searchQuery.value) return resources.value
  return resources.value.filter(r => r.name.toLowerCase().includes(searchQuery.value.toLowerCase()))
})

const uploadHeaders = {
  'X-Module-Type': 'terrain'
}

const handleUpload = () => {
  uploadVisible.value = true
}

const handlePreview = (row: any) => {
  ElMessage.info(`预览地形: ${row.name}`)
}

const handleDelete = (row: any) => {
  ElMessageBox.confirm(`确定删除地形资源 ${row.name} 吗？此操作不可撤销。`, '警告', {
    type: 'warning'
  }).then(() => {
    ElMessage.success('已发起删除请求')
  })
}
</script>

<style scoped>
.terrain-container {
  padding: 24px;
  background-color: var(--el-bg-color-page);
  min-height: 100vh;
  transition: background-color 0.3s;
}

.terrain-dashboard {
  max-width: 1200px;
  margin: 0 auto;
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.header-actions {
  display: flex;
  align-items: center;
}

.dashboard-header h2 {
  margin: 0;
  font-size: 24px;
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.stat-cards {
  margin-bottom: 24px;
}

.stat-card {
  border: none;
  border-radius: 12px;
  background-color: var(--el-bg-color-overlay);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-info {
  display: flex;
  vertical-align: middle;
  flex-direction: column;
}

.stat-label {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.stat-value {
  font-size: 20px;
  font-weight: bold;
  color: var(--el-text-color-primary);
}

.blue { color: #409eff; }
.green { color: #67c23a; }
.orange { color: #e6a23c; }
.purple { color: #909399; }

.content-wrapper {
  background: var(--el-bg-color-overlay);
  padding: 20px;
  border-radius: 12px;
  box-shadow: var(--el-box-shadow-light);
  transition: all 0.3s;
}

.resource-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.geo-placeholder {
  height: 400px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: #fcfcfc;
  border: 1px dashed #dcdfe6;
  border-radius: 8px;
}

.operation-buttons {
  display: flex;
  gap: 8px;
}
</style>

<style>
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
}
</style>
