<template>
  <div class="terrain-custom-container">
    <div class="terrain-header">
      <div class="title">
        <el-icon><Location /></el-icon>
        <h2>高程地形管理系统</h2>
      </div>
      <el-button type="primary" @click="triggerUpload">
        <el-icon><Upload /></el-icon> 导入地形 TIF
      </el-button>
    </div>

    <el-row :gutter="20" class="terrain-stats">
      <el-col :span="8">
        <el-card shadow="never" class="stat-card">
          <template #header>数据总量</template>
          <div class="value">2.4 TB</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never" class="stat-card">
          <template #header>服务状态</template>
          <div class="value status-ok">运行中</div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never" class="stat-card">
          <template #header>本月新增</template>
          <div class="value">+12 区域</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 这里暂时复用 ResourceList 但传递特殊 Props，未来可以完全重写 -->
    <div class="terrain-content">
      <ResourceList 
        type-key="map_terrain" 
        type-name="地形" 
        :enable-scope="false" 
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { Location, Upload } from '@element-plus/icons-vue'
import ResourceList from '../../components/resource/ResourceList.vue'

const triggerUpload = () => {
    // 逻辑可以定制化，比如跳转到特殊上传页面
    console.log('Terrain upload triggered')
}
</script>

<style scoped lang="scss">
.terrain-custom-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
  animation: fadeIn 0.4s ease-out;
}

.terrain-header {
  background: var(--sidebar-bg);
  padding: 16px 24px;
  border-radius: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid var(--el-border-color-lighter);
  
  .title {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 20px;
    color: var(--el-text-color-primary);
    
    h2 { margin: 0; font-size: 18px; }
    .el-icon { color: var(--el-color-primary); }
  }
}

.terrain-stats {
  .stat-card {
    border-radius: 12px;
    background: var(--sidebar-bg);
    border: 1px solid var(--el-border-color-lighter);
    
    :deep(.el-card__header) {
      padding: 12px 20px;
      font-size: 13px;
      color: var(--el-text-color-secondary);
      border-bottom: 1px solid var(--el-border-color-lighter);
    }
    
    .value {
      padding: 20px;
      font-size: 24px;
      font-weight: 700;
      color: var(--el-text-color-primary);
      
      &.status-ok { color: var(--el-color-success); }
    }
  }
}

.terrain-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  overflow: hidden;
  
  /* 我们可以通过内联样式覆盖子组件的标题栏，如果需要的话 */
  :deep(.premium-header) {
    display: none; /* 隐藏通用的 Header，因为我们上面有自定义的了 */
  }
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
