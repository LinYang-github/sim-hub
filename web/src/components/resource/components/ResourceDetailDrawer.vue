<template>
  <el-drawer
    v-model="visible"
    title="资源详情"
    size="500px"
    class="detail-drawer"
    destroy-on-close
  >
    <template #header="{ titleId, titleClass }">
      <div class="drawer-header">
        <h4 :id="titleId" :class="titleClass">
          <el-icon><InfoFilled /></el-icon> 资源详情
        </h4>
        <div class="header-actions">
           <el-button type="primary" size="small" @click="$emit('download', resource)" :disabled="resource?.latest_version?.state !== 'ACTIVE'">
             下载资源
           </el-button>
        </div>
      </div>
    </template>

    <div v-if="resource" class="detail-container">
      <!-- 1. Basic Info Section -->
      <section class="detail-section">
        <div class="section-title">基本信息</div>
        <div class="info-grid">
          <div class="info-item">
            <label>资源名称</label>
            <div class="value">{{ resource.name }}</div>
          </div>
          <div class="info-item">
            <label>资源类型</label>
            <div class="value"><el-tag size="small">{{ typeName }}</el-tag></div>
          </div>
          <div class="info-item">
            <label>创建时间</label>
            <div class="value">{{ formatDate(resource.created_at) }}</div>
          </div>
          <div class="info-item">
            <label>文件大小</label>
            <div class="value">{{ formatSize(resource.latest_version?.file_size) }}</div>
          </div>
          <div class="info-item">
            <label>可见性</label>
            <div class="value">
              <el-tag :type="resource.scope === 'PUBLIC' ? 'success' : 'info'" size="small">
                {{ resource.scope === 'PUBLIC' ? '公开' : '私有' }}
              </el-tag>
            </div>
          </div>
        </div>
      </section>

      <!-- 2. Tags Section -->
      <section class="detail-section">
        <div class="section-title">
          标签管理
          <el-button link type="primary" size="small" @click="$emit('edit-tags', resource)">编辑</el-button>
        </div>
        <div class="tags-container">
          <el-tag v-for="tag in resource.tags" :key="tag" round size="small" class="detail-tag">
            {{ tag }}
          </el-tag>
          <span v-if="!resource.tags?.length" class="empty-text">暂无标签</span>
        </div>
      </section>

      <!-- 3. Tabs for Complex Info -->
      <el-tabs v-model="activeTab" class="detail-tabs">
        <el-tab-pane label="版本记录" name="versions">
          <div class="tab-content" v-loading="loadingDetails">
             <el-timeline v-if="versions.length > 0">
               <el-timeline-item
                 v-for="ver in versions"
                 :key="ver.id"
                 :timestamp="formatDate(ver.created_at)"
                 :type="ver.id === resource.latest_version?.id ? 'primary' : undefined"
               >
                 <div class="version-item">
                   <div class="ver-head">
                     <span class="ver-name">{{ ver.semver || 'v' + ver.version_num }}</span>
                     <el-tag size="small" :type="getStatusType(ver.state)">{{ statusMap[ver.state] || ver.state }}</el-tag>
                   </div>
                   <div class="ver-actions">
                     <el-button link size="small" @click="$emit('download-version', ver.download_url)" :disabled="ver.state !== 'ACTIVE'">下载</el-button>
                     <el-button v-if="ver.id !== resource.latest_version?.id" link type="warning" size="small" @click="$emit('rollback', ver.id)">回滚</el-button>
                   </div>
                 </div>
               </el-timeline-item>
             </el-timeline>
             <el-empty v-else :image-size="60" description="加载中..." />
          </div>
        </el-tab-pane>

        <el-tab-pane label="依赖关系" name="dependencies">
          <div class="tab-content" v-loading="loadingDetails">
             <div v-if="dependencies && dependencies.length > 0" class="dep-list">
                <div v-for="dep in dependencies" :key="dep.id" class="dep-node">
                  <el-icon><Connection /></el-icon>
                  <span class="dep-name">{{ dep.name }}</span>
                  <span class="dep-ver">{{ dep.version }}</span>
                </div>
             </div>
             <el-empty v-else :image-size="60" description="无任何依赖项" />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { InfoFilled, Connection } from '@element-plus/icons-vue'
import { formatDate, formatSize } from '../../../core/utils/format'

const props = defineProps<{
  modelValue: boolean
  resource: any
  typeName: string
  statusMap: Record<string, string>
  versions: any[]
  dependencies: any[]
  loadingDetails: boolean
}>()

const emit = defineEmits([
  'update:modelValue', 
  'edit-tags', 
  'download', 
  'download-version', 
  'rollback'
])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const activeTab = ref('versions')

const getStatusType = (state: string) => {
  const map: any = { ACTIVE: 'success', PROCESSING: 'primary', FAILED: 'danger' }
  return map[state] || 'info'
}
</script>

<style scoped lang="scss">
.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  h4 { margin: 0; display: flex; align-items: center; gap: 8px; }
}

.detail-container {
  padding: 0 4px;
}

.detail-section {
  margin-bottom: 24px;

  .section-title {
    font-size: 14px;
    font-weight: 700;
    color: var(--el-text-color-primary);
    margin-bottom: 12px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  background: var(--el-fill-color-lighter);
  padding: 16px;
  border-radius: 8px;
}

.info-item {
  label { font-size: 12px; color: var(--el-text-color-secondary); display: block; margin-bottom: 4px; }
  .value { font-size: 13px; color: var(--el-text-color-primary); font-weight: 500; }
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.detail-tag { margin: 0; }

.detail-tabs {
  margin-top: 24px;
}

.tab-content {
  min-height: 200px;
  padding: 12px 4px;
}

.version-item {
  .ver-head { display: flex; align-items: center; gap: 12px; margin-bottom: 8px; }
  .ver-name { font-weight: 600; font-size: 14px; }
  .ver-actions { display: flex; gap: 8px; }
}

.dep-node {
  display: flex; align-items: center; gap: 12px; padding: 10px;
  background: var(--el-fill-color-lighter); border-radius: 6px; margin-bottom: 8px;
  .dep-name { flex: 1; font-size: 13px; font-weight: 500; }
  .dep-ver { font-size: 12px; color: var(--el-text-color-secondary); }
}

.empty-text { font-size: 12px; color: var(--el-text-color-placeholder); }
</style>
