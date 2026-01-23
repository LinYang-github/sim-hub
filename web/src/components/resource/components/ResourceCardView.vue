<template>
  <div class="card-view-container">
    <el-row :gutter="20">
      <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" v-for="item in resources" :key="item.id">
        <el-card class="resource-card" shadow="hover" :body-style="{ padding: '0px' }">
          <!-- 卡片封面区域 -->
          <div class="card-cover" @click="$emit('view-history', item)">
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
          
          <div class="card-info">
              <div class="card-title" :title="item.name">{{ item.name }}</div>
              <div class="card-meta">
                <span>{{ formatSize(item.latest_version?.file_size) }}</span>
                <span class="card-date">{{ formatDate(item.created_at).split(' ')[0] }}</span>
              </div>
              
              <div class="card-tags">
                <el-tag v-for="tag in (item.tags || []).slice(0,2)" :key="tag" size="small" effect="plain" round>{{ tag }}</el-tag>
                <el-tag v-if="(item.tags || []).length > 2" size="small" effect="plain" round>+{{ item.tags.length - 2 }}</el-tag>
                <el-button link size="small" icon="PriceTag" @click="$emit('edit-tags', item)" v-if="!(item.tags?.length)" style="padding:0">添加标签</el-button>
              </div>
          </div>

          <div class="card-footer">
              <el-tooltip content="依赖关系" placement="top">
                <el-button link @click="$emit('view-dependencies', item)"><el-icon><Connection /></el-icon></el-button>
              </el-tooltip>
              <div class="footer-right">
                <el-button link type="primary" :disabled="item.latest_version?.state !== 'ACTIVE'" @click="$emit('download', item)"><el-icon><Download /></el-icon></el-button>
                <el-button link type="danger" @click="$emit('delete', item)"><el-icon><Delete /></el-icon></el-button>
              </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <el-empty v-if="resources.length === 0" description="暂无资源数据" />
  </div>
</template>

<script setup lang="ts">
import { 
  Box, Location, Files, Connection, Download, Delete, PriceTag 
} from '@element-plus/icons-vue'
import { formatDate, formatSize } from '../../../core/utils/format'

defineProps<{
  resources: any[]
  typeKey: string
}>()

defineEmits(['edit-tags', 'view-history', 'view-dependencies', 'download', 'delete'])
</script>

<style scoped lang="scss">
.card-view-container {
  padding: 10px 0;
}

.resource-card {
  margin-bottom: 20px;
  border-radius: 12px;
  overflow: hidden;
  transition: all 0.3s;
  border: 1px solid var(--el-border-color-lighter);

  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 12px 24px rgba(0, 0, 0, 0.08);

    .card-icon-placeholder {
      transform: scale(1.1);
      color: var(--el-color-primary);
    }
  }
}

.card-cover {
  height: 140px;
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
    bottom: 10px;
    right: 10px;
    background: rgba(0,0,0,0.6);
    color: #fff;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
  }
}

.card-info {
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  
  .card-title {
    font-size: 16px;
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
    margin-bottom: 8px;
  }
  
  .card-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    height: 24px; 
    overflow: hidden;
  }
}

.card-footer {
  padding: 8px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--el-fill-color-light);
  
  .footer-right {
    display: flex;
    gap: 4px;
  }
}

:global(.dark) .card-cover {
  background: linear-gradient(135deg, #2c2c2c 0%, #1f1f1f 100%);
}
</style>
