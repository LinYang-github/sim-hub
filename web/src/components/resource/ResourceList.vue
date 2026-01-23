<template>
  <div class="resource-layout">
    <!-- Breadcrumb or Header for context can be added here -->
    
    <!-- 侧边栏 -->
    <CategorySidebar
      :category-tree="categoryTree"
      :default-props="{ children: 'children', label: 'name' }"
      v-model="selectedCategoryId"
      @add-category="promptAddCategory"
      @select-category="handleCategorySelect"
      @delete-category="(id) => confirmDeleteCategory(id, fetchList)"
    />

    <!-- 主内容区 -->
    <div class="resource-main">
      <div class="premium-header">
        <!-- 1. Left: Context & Filter -->
        <div class="header-left">
          <div class="title-block">
            <h2>{{ currentCategoryName }}</h2>
            <span class="subtitle">{{ typeName }}库</span>
          </div>
          
          <div v-if="enableScope" class="scope-segment">
            <div 
              v-for="opt in [{label:'全部', val:'ALL'}, {label:'公共', val:'PUBLIC'}, {label:'我的', val:'PRIVATE'}]"
              :key="opt.val"
              class="segment-item"
              :class="{ active: activeScope === opt.val }"
              @click="activeScope = opt.val as any"
            >
              {{ opt.label }}
            </div>
          </div>
        </div>



        <!-- 3. Right: Actions Cluster -->
        <div class="header-right">
          <!-- Primary Primary Action -->
          <div class="primary-actions">
            <!-- Folder Upload -->
            <el-button v-if="uploadMode === 'folder-zip'" type="primary" class="upload-btn" @click="triggerFolderUpload">
              <el-icon><UploadIcon /></el-icon> 导入{{ typeName }}包
            </el-button>
            <!-- Single File Upload -->
            <el-button v-else type="primary" class="upload-btn" @click="triggerFileUpload">
              <el-icon><UploadIcon /></el-icon> 上传{{ typeName }}
            </el-button>
          </div>

          <div class="divider-vertical"></div>

          <!-- Secondary Actions -->
          <div class="secondary-actions">
            <el-tooltip content="同步存储" placement="bottom">
              <el-button class="icon-btn" @click="syncFromStorage" :loading="syncing" circle>
                <el-icon><Connection /></el-icon>
              </el-button>
            </el-tooltip>

            <el-tooltip content="刷新列表" placement="bottom">
              <el-button class="icon-btn" @click="fetchList()" circle>
                <el-icon><Refresh /></el-icon>
              </el-button>
            </el-tooltip>
            
            <div class="view-toggle-group">
               <div class="toggle-item" :class="{ active: viewMode === 'list' }" @click="viewMode = 'list'">
                 <el-icon><DataLine /></el-icon>
               </div>
               <div class="toggle-item" :class="{ active: viewMode === 'card' }" @click="viewMode = 'card'">
                 <el-icon><Grid /></el-icon>
               </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 上传进度展示 -->
      <div v-if="uploading" class="upload-status">
        <p v-if="compressing">正在打包文件夹: {{ currentFile }} ({{ progress }}%)</p>
        <el-progress v-else :percentage="uploadPercent" />
      </div>

      <!-- 列表视图 -->
      <div v-if="viewMode === 'list'" class="content-container">
        <ResourceTableView
          :resources="resources"
          :loading="loading"
          :type-key="typeKey"
          :enable-scope="enableScope"
          :status-map="statusMap"
          @edit-tags="openTagEditor"
          @view-history="viewHistory"
          @view-dependencies="viewDependencies"
          @download="download"
          @delete="confirmDelete"
          @change-scope="handleScopeChange"
        />
      </div>

      <!-- 卡片视图 -->
      <div v-else-if="viewMode === 'card'" class="content-container">
        <ResourceCardView 
          :resources="resources"
          :type-key="typeKey"
          @edit-tags="openTagEditor"
          @view-history="viewHistory"
          @view-dependencies="viewDependencies"
          @download="download"
          @delete="confirmDelete"
        />
      </div>
    </div>

    <!-- 隐藏的输入框 -->
    <input 
      id="folderInput" 
      type="file" 
      webkitdirectory 
      directory 
      style="display: none" 
      @change="handleFolderSelect"
    />
    <input 
      id="fileInput" 
      type="file" 
      :accept="accept"
      style="display: none" 
      @change="handleFileSelect"
    />

    <!-- 对话框与抽屉 -->
    <TagEditDialog
      v-model="tagDialogVisible"
      :loading="tagLoading"
      :existing-tags="existingTags"
      v-model:tags="editingTags"
      @save="(tags) => { editingTags = tags; saveTags(); }"
    />

    <UploadDialog
      v-model="uploadConfirmVisible"
      :data="pendingUploadData"
      :form="uploadForm"
      :loading="uploading"
      :search-results="searchResults"
      :search-loading="searchLoading"
      @search-dependency="searchTargetResources"
      @confirm="() => confirmAndDoUpload(selectedCategoryId, props.typeKey)"
    />

    <DependencyDrawer
      v-model="depDrawerVisible"
      :dep-tree="depTree"
      :loading="depLoading"
      :bundle-loading="bundleLoading"
      :pack-loading="packLoading"
      @download-bundle="downloadBundle"
      @download-pack="downloadSimPack"
    />

    <HistoryDrawer
      v-model="historyDrawerVisible"
      :version-history="versionHistory"
      :loading="historyLoading"
      :current-version-id="currentResource?.latest_version?.id"
      :status-map="statusMap"
      @download="(url) => handleDownloadUrl(url)"
      @rollback="rollback"
    />

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { 
  Upload as UploadIcon, Connection, DataLine, Grid, Refresh,
  Search, CircleCloseFilled
} from '@element-plus/icons-vue'
import axios from 'axios' // Needed for scope change in this file or move to action
import { ElMessage } from 'element-plus'

// Components
import CategorySidebar from './components/CategorySidebar.vue'
import ResourceTableView from './components/ResourceTableView.vue'
import ResourceCardView from './components/ResourceCardView.vue'
import TagEditDialog from './components/TagEditDialog.vue'
import UploadDialog from './components/UploadDialog.vue'
import DependencyDrawer from './components/DependencyDrawer.vue'
import HistoryDrawer from './components/HistoryDrawer.vue'

// Composables
import { useCategory } from './composables/useCategory'
import { useResourceList } from './composables/useResourceList'
import { useUpload } from './composables/useUpload'
import { useTags } from './composables/useTags'
import { useHistory } from './composables/useHistory'
import { useDependency } from './composables/useDependency'
import { useResourceAction } from './composables/useResourceAction'

const props = defineProps<{
  typeKey: string
  typeName: string
  uploadMode?: 'single' | 'folder-zip'
  accept?: string
  enableScope?: boolean
}>()

const statusMap: Record<string, string> = {
  ACTIVE: '已就绪',
  READY: '就绪',
  PROCESSING: '处理中',
  PENDING: '排队中',
  FAILED: '处理失败',
}

const viewMode = ref('list')
const searchFocused = ref(false)

// 1. Categories
const { 
  categories, selectedCategoryId, categoryTree, currentCategoryName, 
  fetchCategories, promptAddCategory, confirmDeleteCategory 
} = useCategory(props.typeKey)

// 2. Resource List
const { 
  resources, loading, activeScope, searchQuery, syncing, 
  fetchList, syncFromStorage 
} = useResourceList(props.typeKey, !!props.enableScope, selectedCategoryId)

// 3. Upload
const {
  uploading, compressing, progress, uploadPercent, currentFile,
  pendingUploadData, uploadConfirmVisible, uploadForm, searchLoading, searchResults,
  triggerFolderUpload, triggerFileUpload, handleFolderSelect, handleFileSelect,
  searchTargetResources, confirmAndDoUpload
} = useUpload(props.typeKey, selectedCategoryId, fetchList)

// 4. Tags
const {
  tagDialogVisible, tagLoading, editingTags, existingTags,
  openTagEditor, saveTags
} = useTags(resources, fetchList)

// 5. History & Rollback
const {
  historyDrawerVisible, historyLoading, versionHistory, currentResource,
  viewHistory, rollback
} = useHistory(fetchList)

// 6. Dependencies
const {
  depDrawerVisible, depLoading, depTree, bundleLoading, packLoading,
  viewDependencies, downloadBundle, downloadSimPack
} = useDependency(currentResource)

// 7. Actions
const { confirmDelete, download, handleDownloadUrl, publishResource: doPublish } = useResourceAction(fetchList)

// Scope Change (kept here as it's simple or move to useResourceAction)
const handleScopeChange = async (row: any, scope: string) => {
    try {
        await axios.patch(`/api/v1/resources/${row.id}/scope`, { scope })
        ElMessage.success('权限更新成功')
        fetchList()
    } catch (err: any) {
        ElMessage.error('权限更新失败: ' + (err.response?.data?.error || err.message))
    }
}

// Sidebar handling
const handleCategorySelect = (data: any) => {
    selectedCategoryId.value = data.id
    fetchList()
}

// Lifecycle
let pollInterval: any = null

const initData = () => {
    fetchList(props.typeKey)
    fetchCategories(props.typeKey)
}

watch(() => props.typeKey, () => {
    selectedCategoryId.value = 'all'
    initData()
})

onMounted(() => {
    initData()
    // Polling for processing status
    pollInterval = setInterval(() => {
        const hasProcessing = resources.value.some((s: any) => 
            s.latest_version?.state === 'PROCESSING' || s.latest_version?.state === 'PENDING'
        )
        if (hasProcessing) {
            fetchList()
        }
    }, 3000)
})

onUnmounted(() => {
    if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped lang="scss">
.resource-layout {
  display: flex;
  height: calc(100vh - var(--header-height) - 40px);
  gap: 12px;
  animation: fadeIn 0.4s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.resource-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.premium-header {
  background: var(--sidebar-bg);
  height: 60px;
  padding: 0 20px;
  border-radius: 8px; /* Tighter radius */
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 4px 12px -4px rgba(0, 0, 0, 0.04);
  backdrop-filter: blur(10px);
  margin-bottom: 2px;
  transition: all 0.3s ease;
}

/* 1. Header Left */
.header-left {
  display: flex;
  align-items: center;
  gap: 24px;
}

.title-block {
  display: flex;
  align-items: baseline;
  gap: 8px;

  h2 {
    margin: 0;
    font-size: 18px; /* Slightly smaller */
    font-weight: 700;
    color: var(--el-text-color-primary);
    letter-spacing: -0.5px;
  }

  .subtitle {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    font-weight: 500;
  }
}

.scope-segment {
  display: flex;
  background: var(--el-fill-color);
  padding: 3px;
  border-radius: 6px; /* Sharper */
  gap: 2px;

  .segment-item {
    padding: 4px 12px;
    font-size: 12px;
    border-radius: 4px;
    cursor: pointer;
    color: var(--el-text-color-regular);
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    font-weight: 500;

    &:hover {
      color: var(--el-text-color-primary);
      background: rgba(255, 255, 255, 0.5);
    }

    &.active {
      background: var(--el-bg-color);
      color: var(--el-color-primary);
      box-shadow: 0 1px 4px -1px rgba(0, 0, 0, 0.1);
      font-weight: 600;
    }
  }
}

/* 2. Header Right */
.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.divider-vertical {
  width: 1px;
  height: 20px;
  background: var(--el-border-color-lighter);
}

.primary-actions {
  display: flex;
  gap: 12px;

  .upload-btn {
    height: 32px; /* More compact */
    border-radius: 4px; /* Standard sharp corners */
    padding: 0 16px;
    font-weight: 600;
    font-size: 13px;
    box-shadow: none; /* Removed heavy shadow */
    transition: all 0.1s;

    &:hover {
      opacity: 0.9;
    }
    
    &:active {
      transform: translateY(1px);
    }
  }
}

.secondary-actions {
  display: flex;
  align-items: center;
  gap: 8px;

  .icon-btn {
    border: none;
    background: transparent;
    font-size: 16px;
    color: var(--el-text-color-regular);
    transition: all 0.2s;
    width: 32px;
    height: 32px;
    border-radius: 4px;

    &:hover {
      background: var(--el-fill-color-light);
      color: var(--el-color-primary);
    }
  }
}

.view-toggle-group {
  display: flex;
  background: var(--el-fill-color);
  border-radius: 4px;
  padding: 2px;
  gap: 2px;
  margin-left: 8px;

  .toggle-item {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 3px;
    cursor: pointer;
    color: var(--el-text-color-placeholder);
    transition: all 0.2s;

    &:hover {
      color: var(--el-text-color-regular);
    }

    &.active {
      background: var(--el-bg-color);
      color: var(--el-color-primary);
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    }
  }
}

.action-group {
  display: flex;
  gap: 12px;
}

.content-container {
  flex: 1;
  background: var(--sidebar-bg);
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  overflow: hidden;
  box-shadow: var(--el-box-shadow-lighter);
}

.upload-status {
  padding: 16px 24px;
  background: var(--sidebar-bg);
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  margin-bottom: 12px;
  
  p {
    margin: 0 0 10px 0;
    font-size: 13px;
    color: var(--el-text-color-regular);
  }
}
</style>
