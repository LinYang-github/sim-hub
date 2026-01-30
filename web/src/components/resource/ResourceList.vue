<template>
  <div class="resource-layout">
    <!-- Breadcrumb or Header for context can be added here -->
    
    <!-- 侧边栏 -->
    <CategorySidebar
      v-if="categoryMode !== 'none'"
      :category-tree="categoryTree"
      :default-props="{ children: 'children', label: 'name' }"
      :category-mode="categoryMode"
      v-model="selectedCategoryId"
      @add-category="() => promptAddCategory()"
      @add-subcategory="(pid) => promptAddCategory(pid)"
      @select-category="handleCategorySelect"
      @delete-category="(id) => confirmDeleteCategory(id, fetchList)"
      @move-category="(id, newParentId) => updateCategory(id, { parent_id: newParentId })"
    />

    <!-- 主内容区 -->
    <div class="resource-main">
      <div class="premium-header">
        <!-- 1. Left: Context & Filter -->
        <div class="header-left">
          <div v-if="enableScope" class="scope-segment">
            <div 
              v-for="opt in scopeOptions"
              :key="opt.val"
              class="segment-item"
              :class="{ active: activeScope === opt.val }"
              @click="activeScope = opt.val"
            >
              {{ opt.label }}
            </div>
          </div>

          <div v-if="enableScope" class="divider-vertical"></div>

          <div class="search-box">
            <el-input
              v-model="searchQuery"
              placeholder="搜索资源..."
              :prefix-icon="Search"
              clearable
              class="search-input"
              @clear="fetchList"
              @keyup.enter="fetchList"
            />
          </div>
        </div>



        <!-- 3. Right: Actions Cluster -->
        <div class="header-right">
          <!-- Primary Primary Action -->
          <div class="primary-actions">
            <!-- Folder Upload -->
            <el-button v-if="uploadMode === 'folder-zip'" type="primary" class="upload-btn" @click="triggerFolderUpload">
              <el-icon><UploadIcon /></el-icon> 导入{{ actionLabel }}包
            </el-button>
            <!-- Online Create -->
            <el-button v-else-if="uploadMode === 'online'" type="primary" class="upload-btn" @click="openOnlineCreate">
              <el-icon><Plus /></el-icon> 新建{{ actionLabel }}
            </el-button>
            <!-- Single File Upload -->
            <el-button v-else type="primary" class="upload-btn" @click="triggerFileUpload">
              <el-icon><UploadIcon /></el-icon> 上传{{ actionLabel }}
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

            <el-tooltip content="清空当前库" placement="bottom">
              <el-button class="icon-btn delete-all-btn" @click="handleClear" circle>
                <el-icon><Delete /></el-icon>
              </el-button>
            </el-tooltip>
            
            <!-- View Toggles (Only show if multiple views are supported) -->
            <div class="view-toggle-group" v-if="resolvedViews.length > 1">
               <div 
                 v-for="v in resolvedViews" 
                 :key="v.key"
                 class="toggle-item" 
                 :class="{ active: viewMode === v.key }" 
                 @click="viewMode = v.key"
                 :title="v.label"
               >
                 <el-icon><component :is="v.icon" /></el-icon>
               </div>
            </div>
            <div class="view-toggle-group" v-else-if="resolvedViews.length === 0">
               <div class="toggle-item" :class="{ active: viewMode === 'table' }" @click="viewMode = 'table'">
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

      <!-- 内容显示区域 -->
      <div class="content-container" :class="{ 'is-loading': loading && !resources.length }">
        <!-- 1. 加载中且无数据 -> 骨架屏 -->
        <ResourceSkeleton v-if="loading && !resources.length" :view-mode="viewMode" />

        <!-- 2. 有数据 -> 正常列表/卡片/外部视图 -->
        <template v-else-if="resources.length > 0">
          <!-- External View Mode -->
          <div v-if="activeViewConfig?.path?.startsWith('External:')" class="external-view-wrapper">
             <ExternalViewer 
               :url="activeViewConfig.path.replace('External:', '')" 
               :resource="{ typeKey, resources, searchQuery, activeScope }" 
             />
          </div>

          <!-- Card View Mode -->
          <ResourceCardView 
            v-else-if="viewMode === 'card'"
            :resources="resources"
            :type-key="typeKey"
            :enable-scope="!!enableScope"
            :status-map="statusMap"
            :viewer="viewer"
            :icon="icon"
            @view-details="handleViewDetails"
            @download="download"
            @delete="confirmDelete"
            :custom-actions="resolvedActions"
            @custom-action="(key, row) => handleCustomAction(key, row)"
          />

          <!-- Gallery View Mode -->
          <ResourceGalleryView
            v-else-if="viewMode === 'gallery'"
            :resources="resources"
            :loading="loading"
            :icon="icon"
            :status-map="galleryStatusMap"
            :custom-actions="resolvedActions"
            @view-details="handleViewDetails"
            @download="download"
            @edit-tags="openTagEditor"
            @delete="confirmDelete"
            @rename="(res) => stewardRef?.openRename(res)"
            @move="(res) => stewardRef?.openMove(res)"
            @view-history="viewHistory"
            @custom-action="(key, row) => handleCustomAction(key, row)"
          />

          <!-- DataGrid View Mode -->
          <ResourceDataGrid
            v-else-if="viewMode === 'data-grid'"
            :resources="resources"
            :loading="loading"
            :icon="icon"
            :schema="currentSchema"
            @edit-tags="openTagEditor"
            @view-details="handleViewDetails"
            @download="download"
            @delete="confirmDelete"
            @rename="(res) => stewardRef?.openRename(res)"
            @move="(res) => stewardRef?.openMove(res)"
            :custom-actions="resolvedActions"
            @custom-action="(key, row) => handleCustomAction(key, row)"
          />

          <!-- Default/Table View Mode -->
          <ResourceTableView
            v-else
              :resources="resources"
              :loading="loading"
              :enable-scope="!!enableScope"
              :icon="icon"
              @edit-tags="openTagEditor"
              @view-details="handleViewDetails"
              @download="download"
              @delete="confirmDelete"
              @change-scope="handleScopeChange"
              @rename="(res) => stewardRef?.openRename(res)"
              @move="(res) => stewardRef?.openMove(res)"
              :custom-actions="resolvedActions"
              @custom-action="(key, row) => handleCustomAction(key, row)"
            />
        </template>

        <!-- 3. 加载结束且无数据 -> 优质空状态 -->
        <div v-else class="premium-empty">
          <el-empty :image-size="160">
            <template #image>
              <div class="empty-icon-wrap">
                <el-icon><FolderDelete /></el-icon>
              </div>
            </template>
            <template #description>
              <div class="empty-desc">
                <p class="main-text">暂无资源数据</p>
                <p class="sub-text">您可以尝试同步存储或上传新资源到该分类</p>
              </div>
            </template>
            <div class="empty-actions">
                <el-button type="primary" plain @click="fetchList()">刷新列表</el-button>
               <el-button @click="syncFromStorage">同步存储</el-button>
            </div>
          </el-empty>
        </div>
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
    <ResourceDetailDrawer
      v-model="detailDrawerVisible"
      :resource="currentResource"
      :type-name="typeName"
      :status-map="statusMap"
      :versions="versionHistory"
      :dependencies="depTree"
      :loading-details="historyLoading || depLoading"
      :viewer="viewer"
      :icon="icon"
      @edit-tags="openTagEditor"
      @download="download"
      @download-version="handleDownloadUrl"
      @rollback="rollback"
      @rename="(res) => stewardRef?.openRename(res)"
      @move="(res) => stewardRef?.openMove(res)"
      @refresh="refreshDetails"
      :current-category-name="currentCategoryName"
    />

    <ResourceStewardDialogs
      ref="stewardRef"
      :resource="currentResource"
      :category-tree="categoryTree"
      @success="fetchList"
    />

    <TagEditDialog
      v-model="tagDialogVisible"
      :loading="tagLoading"
      :existing-tags="existingTags"
      :tags="editingTags"
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
      @confirm="confirmAndDoUpload"
    />

    <OnlineCreateDialog 
      v-if="onlineCreateVisible"
      v-model="onlineCreateVisible" 
      :type-key="typeKey" 
      :type-name="typeName || '资源'"
      :category-nodes="categoryTree || []"
      :schema="currentSchema"
      :example="example"
      @success="fetchList"
    />

    <!-- Custom Action Handlers Renderer -->
    <template v-if="resolvedActions && resolvedActions.length">
        <div v-for="action in resolvedActions" :key="action.key" style="display:none">
            <component 
                v-if="action.handler && typeof action.handler !== 'function' && typeof action.handler !== 'string'"
                :is="action.handler"
                :ref="(el: any) => setActionRef(el, action.key)"
            />
        </div>
    </template>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, toRef, computed, watch, defineAsyncComponent } from 'vue'
// Custom Action Handlers (Loaded dynamically)


const handleCustomAction = (key: string, row: any) => {
    const actionDef = resolvedActions.value.find(a => a.key === key)
    if (!actionDef || !actionDef.handler) {
      console.warn('Action definition or handler missing', key)
      return
    }

    // 情况 1: 处理器是纯函数 (直接执行)
    if (typeof actionDef.handler === 'function') {
        try {
           (actionDef.handler as Function)(row)
        } catch (e) { console.error('Error executing action function', e) }
        return
    }

    // 情况 2: 处理器是组件 (通过 Ref 调用 execute 方法)
    // 组件已被模板中的隐藏区域挂载，我们尝试获取其实例并调用 execute
    setTimeout(() => {
        const componentRef = actionComponentsRef.value[key]
        if (componentRef && componentRef.execute) {
            componentRef.execute(row)
        } else {
             console.warn('Action handler component not ready or invalid', key)
        }
    }, 0)
}

const actionComponentsRef = ref<any>({})
const setActionRef = (el: any, key: string) => {
    if (el) actionComponentsRef.value[key] = el
}

import { 
  Upload as UploadIcon, Connection, DataLine, Grid, Refresh,
  Search, FolderDelete, Delete, Plus
} from '@element-plus/icons-vue'
import request from '../../core/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { moduleManager } from '../../core/moduleManager'
import CategorySidebar from './components/CategorySidebar.vue'
import type { Resource, ResourceScope, CategoryNode } from '../../core/types/resource'
import ResourceTableView from './views/ResourceTableView.vue'
import ResourceCardView from './views/ResourceCardView.vue'
import ResourceSkeleton from './components/ResourceSkeleton.vue'
import ResourceDetailDrawer from './components/ResourceDetailDrawer.vue'
import TagEditDialog from './components/TagEditDialog.vue'
import UploadDialog from './components/UploadDialog.vue'
import ResourceStewardDialogs from './components/ResourceStewardDialogs.vue'
import OnlineCreateDialog from './components/OnlineCreateDialog.vue'
import ResourceDataGrid from './views/ResourceDataGrid.vue'
import ResourceGalleryView from './views/ResourceGalleryView.vue'
import ExternalViewer from './previewers/ExternalViewer.vue'
import { SupportedView, CustomAction } from '../../core/types'

// Composables
import { useCategory } from './composables/useCategory'
import { useResourceList } from './composables/useResourceList'
import { useUpload } from './composables/useUpload'
import { useTags } from './composables/useTags'
import { useHistory } from './composables/useHistory'
import { useDependency } from './composables/useDependency'
import { useResourceAction } from './composables/useResourceAction'

import { RESOURCE_STATUS_TEXT, SCOPE_OPTIONS, RESOURCE_STATE } from '../../core/constants/resource'

const props = defineProps<{
  typeKey: string
  typeName: string
  shortName?: string
  uploadMode?: 'single' | 'folder-zip' | 'online'
  accept?: string
  enableScope?: boolean
  categoryMode?: 'flat' | 'tree' | 'none'
  viewer?: string
  icon?: string
  example?: string
  supportedViews?: (string | SupportedView)[]
  customActions?: (string | CustomAction)[]
}>()

// Computed for button text (use shortName if available, fallback to typeName)
const actionLabel = computed(() => props.shortName || props.typeName)

const viewMode = ref('')
const resolvedViews = computed(() => {
  return (props.supportedViews || []).map(v => moduleManager.resolveView(v))
})

const activeViewConfig = computed(() => {
  return resolvedViews.value.find(v => v.key === viewMode.value)
})

const resolvedActions = computed(() => {
  return (props.customActions || []).map(a => moduleManager.resolveAction(a))
})

watch(() => props.supportedViews, () => {
  if (resolvedViews.value.length > 0) {
    // If current viewMode is not supported, switch to the first available view
    if (!resolvedViews.value.some(v => v.key === viewMode.value)) {
      viewMode.value = resolvedViews.value[0].key
    }
  } else {
    // Fallback if no supported views are provided
    viewMode.value = 'table'
  }
}, { immediate: true })

const statusMap = RESOURCE_STATUS_TEXT

const galleryStatusMap = {
  [RESOURCE_STATE.ACTIVE]: { text: '已生效', type: 'success' },
  [RESOURCE_STATE.READY]: { text: '已就绪', type: 'info' },
  [RESOURCE_STATE.PROCESSING]: { text: '处理中', type: 'primary' },
  [RESOURCE_STATE.PENDING]: { text: '排队中', type: 'warning' },
  [RESOURCE_STATE.FAILED]: { text: '失败', type: 'danger' },
}

// Debug Log
console.log('ResourceList Mounted. TypeKey:', props.typeKey, 'CustomActions:', props.customActions)

watch([resolvedViews, activeViewConfig], () => {
  console.log('[ResourceList] Views updated:', {
    count: resolvedViews.value.length,
    activeKey: viewMode.value,
    hasPath: !!activeViewConfig.value?.path
  })
}, { immediate: true })

// Sync viewMode with supportedViews
watch(() => props.typeKey, () => {
  if (props.supportedViews && props.supportedViews.length > 0) {
    const firstView = props.supportedViews[0]
    viewMode.value = typeof firstView === 'string' ? firstView : firstView.key
  } else {
    // Legacy fallback
    viewMode.value = props.uploadMode === 'online' ? 'table' : 'table'
  }
}, { immediate: true })
const searchFocused = ref(false)
const detailDrawerVisible = ref(false)
const stewardRef = ref<InstanceType<typeof ResourceStewardDialogs>>()

const scopeOptions = SCOPE_OPTIONS

// 1. Categories
const { 
  categories, selectedCategoryId, categoryTree, currentCategoryName, 
  fetchCategories, promptAddCategory, confirmDeleteCategory, updateCategory 
} = useCategory(toRef(props, 'typeKey'))

// 2. Resource List
const { 
  resources, loading, activeScope, searchQuery, syncing, 
  fetchList, syncFromStorage 
} = useResourceList(
  toRef(props, 'typeKey'), 
  computed(() => !!props.enableScope), 
  selectedCategoryId
)

// 3. Upload
const {
  uploading, compressing, progress, uploadPercent, currentFile,
  pendingUploadData, uploadConfirmVisible, uploadForm, searchLoading, searchResults,
  triggerFolderUpload, triggerFileUpload, handleFolderSelect, handleFileSelect,
  searchTargetResources, confirmAndDoUpload
} = useUpload(toRef(props, 'typeKey'), selectedCategoryId, fetchList)

// 4. History & Rollback
const {
  historyDrawerVisible, historyLoading, versionHistory, currentResource,
  viewHistory, rollback
} = useHistory(fetchList)

// 5. Tags
const {
  tagDialogVisible, tagLoading, editingTags, existingTags,
  openTagEditor, saveTags
} = useTags(resources, fetchList, currentResource)

// 6. Dependencies
const {
  depDrawerVisible, depLoading, depTree, bundleLoading, packLoading,
  viewDependencies, downloadBundle, downloadSimPack
} = useDependency(currentResource)

// 7. Actions
const { confirmDelete, download, handleDownloadUrl, publishResource: doPublish } = useResourceAction(fetchList)

const handleViewDetails = (row: Resource) => {
  currentResource.value = row
  detailDrawerVisible.value = true
  viewHistory(row, false) // Fetch versions ONLY, don't open extra drawer
  viewDependencies(row, false) // Fetch deps ONLY, don't open extra drawer
}

const refreshDetails = async () => {
  await fetchList()
  if (currentResource.value) {
    // Re-fetch current resource details to update drawer
    const updated = resources.value.find(r => r.id === currentResource.value?.id)
    if (updated) {
        currentResource.value = updated
        await viewHistory(updated, false)
        await viewDependencies(updated, false)
    }
  }
}

// 8. Online Create
const onlineCreateVisible = ref(false)
const allSchemas = ref<Record<string, any>>({})
const currentSchema = computed(() => {
    const t = allSchemas.value[props.typeKey]
    return t ? t.schema_def : null
})

const fetchSchemas = async () => {
    try {
        const types = await request.get<any[]>('/api/v1/resource-types')
        if (types && Array.isArray(types)) {
            // Map types by key
            const map: any = {}
            types.forEach(t => map[t.type_key] = t)
            allSchemas.value = map
        }
    } catch (e) {}
}

const openOnlineCreate = () => {
    if (!currentSchema.value) {
        // Try fetch if missing
        fetchSchemas().then(() => {
             if (currentSchema.value) onlineCreateVisible.value = true
             else ElMessage.warning('未能获取该资源类型的配置模版')
        })
    } else {
        onlineCreateVisible.value = true
    }
}

// Init Schema
const initSchema = () => {
    if (!allSchemas.value[props.typeKey]) {
        fetchSchemas()
    }
}

watch(() => props.typeKey, () => {
    initSchema()
}, { immediate: true })

onMounted(() => {
    initSchema()
})

// Clear Repository
const handleClear = async () => {
    try {
        await ElMessageBox.confirm(
            `此操作将彻底清空当前 [${props.typeName}] 库下的所有资源，是否继续？`,
            '危险操作提示',
            {
                confirmButtonText: '确定并清空',
                cancelButtonText: '取消',
                type: 'warning',
                confirmButtonClass: 'el-button--danger'
            }
        )
        
        loading.value = true
        await request.post(`/api/v1/resources/clear?type=${props.typeKey}`)
        ElMessage.success('资源库已清空')
        fetchList()
    } catch (err: any) {
        if (err !== 'cancel') {
            console.error(err)
        }
    } finally {
        loading.value = false
    }
}

// Scope Change (kept here as it's simple or move to useResourceAction)
const handleScopeChange = async (row: Resource, scope: ResourceScope) => {
    try {
        await request.patch(`/api/v1/resources/${row.id}/scope`, { scope })
        ElMessage.success('权限更新成功')
        
        // 如果详情面板开着且是同一个资源，同步更新引用
        if (currentResource.value && currentResource.value.id === row.id) {
            currentResource.value.scope = scope
        }
        
        fetchList()
    } catch (err: any) {
    }
}

// Sidebar handling
const handleCategorySelect = (data: CategoryNode) => {
    selectedCategoryId.value = data.id
    fetchList()
}

// Lifecycle
let pollInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
    // Polling for processing status
    pollInterval = setInterval(() => {
        const hasProcessing = resources.value.some((s: Resource) => 
            s.latest_version?.state === RESOURCE_STATE.PROCESSING || s.latest_version?.state === RESOURCE_STATE.PENDING
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

.search-box {
  width: 240px;
  .search-input {
    :deep(.el-input__wrapper) {
      background-color: var(--el-fill-color-lighter);
      box-shadow: none !important;
      border: 1px solid transparent;
      transition: all 0.2s;
      border-radius: 6px;

      &.is-focus {
        background-color: var(--el-bg-color);
        border-color: var(--el-color-primary-light-5);
      }
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

    &.delete-all-btn {
      &:hover {
        background: var(--el-color-danger-light-9);
        color: var(--el-color-danger);
      }
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
  border-radius: 8px; /* Sharper to match header */
  border: 1px solid var(--el-border-color-lighter);
  overflow: auto;
  box-shadow: 0 4px 12px -4px rgba(0, 0, 0, 0.04);
  padding: 12px;
  
  &.is-loading {
    border-color: transparent;
    box-shadow: none;
  }

  .external-view-wrapper {
    width: 100%;
    height: 100%;
    min-height: 500px;
    background: transparent;
  }
}

.premium-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  
  .empty-icon-wrap {
    font-size: 80px;
    color: var(--el-text-color-placeholder);
    opacity: 0.5;
  }
  
  .empty-desc {
    .main-text {
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      margin-bottom: 8px;
    }
    .sub-text {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }
  }
  
  .empty-actions {
    margin-top: 24px;
    display: flex;
    gap: 12px;
    justify-content: center;
  }
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
