<template>
  <div class="resource-layout">
    <!-- Breadcrumb or Header for context -->
    
    <!-- 侧边栏：Glassmorphism 设计 -->
    <div class="category-sidebar">
      <div class="sidebar-header">
        <el-icon><FolderOpened /></el-icon>
        <span>资源分类</span>
        <el-button link type="primary" @click="promptAddCategory">
          <el-icon><Plus /></el-icon>
        </el-button>
      </div>
      
      <el-scrollbar>
        <el-tree
          :data="categoryTree"
          :props="defaultProps"
          node-key="id"
          class="custom-tree"
          @node-click="handleCategoryClick"
          highlight-current
          :default-expanded-keys="['all']"
        >
          <template #default="{ node, data }">
            <span class="custom-tree-node">
              <el-icon v-if="data.id === 'all'"><Grid /></el-icon>
              <el-icon v-else><Folder /></el-icon>
              <span class="node-label">{{ node.label }}</span>
              <span class="node-actions" v-if="data.id !== 'all'">
                <el-icon class="delete-icon" @click.stop="confirmDeleteCategory(data.id)"><Delete /></el-icon>
              </span>
            </span>
          </template>
        </el-tree>
      </el-scrollbar>
    </div>

    <!-- 主内容区 -->
    <div class="resource-main">
      <div class="premium-header">
        <div class="title-group">
          <h2>{{ currentCategoryName }} <small>{{ typeName }}资源库</small></h2>
        </div>
        
        <div v-if="enableScope" class="scope-tabs">
          <el-radio-group v-model="activeScope" size="small">
            <el-radio-button value="ALL">全部</el-radio-button>
            <el-radio-button value="PUBLIC">公共库</el-radio-button>
            <el-radio-button value="PRIVATE">我的</el-radio-button>
          </el-radio-group>
        </div>

        <div class="action-group">
          <el-button-group>
            <!-- Folder Upload -->
            <el-button v-if="uploadMode === 'folder-zip'" type="primary" class="upload-btn" @click="triggerFolderUpload">
              <el-icon><Upload /></el-icon> 导入{{ typeName }}包
            </el-button>
            <!-- Single File Upload -->
            <el-button v-else type="primary" class="upload-btn" @click="triggerFileUpload">
              <el-icon><Upload /></el-icon> 上传{{ typeName }}
            </el-button>

            <el-button class="sync-btn" @click="syncFromStorage" :loading="syncing">
              <el-icon><Connection /></el-icon> 同步存储
            </el-button>
          </el-button-group>
          <el-button circle @click="fetchList" class="refresh-btn">
            <el-icon><Refresh /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 上传进度展示 -->
      <div v-if="uploading" class="upload-status">
        <p v-if="compressing">正在打包文件夹: {{ currentFile }} ({{ progress }}%)</p>
        <el-progress v-else :percentage="uploadPercent" />
      </div>

      <!-- 列表区 -->
      <div class="content-container">
        <el-table :data="resources" style="width: 100%" v-loading="loading" class="premium-table">
          <el-table-column label="资源详情" min-width="250">
            <template #default="scope">
              <div class="resource-info-cell">
                <div class="resource-icon">
                  <el-icon v-if="typeKey === 'model_glb'"><Box /></el-icon>
                  <el-icon v-else-if="typeKey === 'map_terrain'"><Location /></el-icon>
                  <el-icon v-else><Files /></el-icon>
                </div>
                <div class="resource-text">
                  <div class="name-row">
                    <span class="resource-name">{{ scope.row.name }}</span>
                    <el-tag v-if="scope.row.scope === 'PUBLIC'" size="small" type="success" effect="plain" class="scope-tag">公共</el-tag>
                  </div>
                  <div class="resource-meta">
                    <span><el-icon><Clock /></el-icon> {{ formatDate(scope.row.created_at) }}</span>
                    <span><el-icon><DataLine /></el-icon> {{ formatSize(scope.row.latest_version?.file_size) }}</span>
                  </div>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="标签" width="220">
            <template #default="scope">
              <div class="tag-wrap">
                <el-tag 
                  v-for="tag in scope.row.tags" 
                  :key="tag" 
                  round
                  size="small"
                  class="premium-tag"
                >
                  {{ tag }}
                </el-tag>
                <el-button circle size="small" class="add-tag-btn" @click="openTagEditor(scope.row)">
                  <el-icon><PriceTag /></el-icon>
                </el-button>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="版本" width="120">
            <template #default="scope">
              <el-tooltip :content="'点击查看历史版本 - 当前流水: ' + (scope.row.latest_version?.version_num || 1)" placement="top">
                <span class="version-badge clickable" @click="viewHistory(scope.row)">
                  {{ scope.row.latest_version?.semver || 'v' + (scope.row.latest_version?.version_num || 1) }}
                </span>
              </el-tooltip>
            </template>
          </el-table-column>

          <el-table-column label="状态" width="120">
            <template #default="scope">
              <div class="status-cell">
                <el-tooltip 
                  v-if="scope.row.latest_version?.state === 'ACTIVE' && !(scope.row.latest_version?.meta_data?.processed)"
                  content="该资源类型无需后端处理，已直接可用"
                  placement="top"
                >
                  <el-icon class="skip-icon"><CircleCheckFilled /></el-icon>
                </el-tooltip>
                <div v-else :class="['status-dot', scope.row.latest_version?.state.toLowerCase()]"></div>
                <span class="status-text">{{ statusMap[scope.row.latest_version?.state] || scope.row.latest_version?.state }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="依赖" width="100">
            <template #default="scope">
              <el-button link type="primary" @click="viewDependencies(scope.row)">
                <el-icon><Connection /></el-icon> 依赖关系
              </el-button>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="220" fixed="right">
            <template #default="scope">
              <div class="operation-buttons">
                <el-button type="primary" link :disabled="scope.row.latest_version?.state !== 'ACTIVE'" @click="download(scope.row)">
                  <el-icon><Download /></el-icon> 下载
                </el-button>
                <el-button v-if="scope.row.scope === 'PRIVATE'" type="success" link @click="publishResource(scope.row)">
                  <el-icon><Promotion /></el-icon> 发布
                </el-button>
                <el-button type="danger" link @click="confirmDelete(scope.row)">
                  <el-icon><Delete /></el-icon> 删除
                </el-button>
              </div>
            </template>
          </el-table-column>
        </el-table>
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

    <!-- 标签编辑对话框 -->
    <el-dialog v-model="tagDialogVisible" title="管理标签" width="400px">
      <el-select
        v-model="editingTags"
        multiple
        filterable
        allow-create
        default-first-option
        placeholder="输入标签并按回车"
        style="width: 100%"
      >
        <el-option
          v-for="item in existingTags"
          :key="item"
          :label="item"
          :value="item"
        />
      </el-select>
      <template #footer>
        <el-button @click="tagDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveTags" :loading="tagLoading">保存</el-button>
      </template>
    </el-dialog>

    <!-- 依赖树展示对话框 -->
    <el-drawer v-model="depDrawerVisible" title="资源依赖全景图" size="450px" class="dep-drawer">
      <template #header>
        <div class="drawer-header">
          <span>资源依赖全景图</span>
          <div class="header-actions">
            <el-button type="warning" size="small" @click="downloadBundle" :loading="bundleLoading">清单下载</el-button>
            <el-button type="success" size="small" @click="downloadSimPack" :loading="packLoading">离线打包 (.simpack)</el-button>
          </div>
        </div>
      </template>
      <div v-loading="depLoading" class="dep-content">
        <template v-if="depTree.length > 0">
          <el-tree
            :data="depTree"
            :props="{ label: 'resource_name', children: 'dependencies' }"
            default-expand-all
            class="dep-tree"
          >
            <template #default="{ node, data }">
              <div class="dep-node">
                <el-icon class="dep-icon"><Share /></el-icon>
                <div class="dep-info">
                  <span class="dep-name">{{ data.resource_name }}</span>
                  <div class="dep-meta">
                    <el-tag size="small" type="info" class="dep-ver">{{ data.semver || 'latest' }}</el-tag>
                    <span class="dep-constraint" v-if="data.constraint">约束: {{ data.constraint }}</span>
                  </div>
                </div>
              </div>
            </template>
          </el-tree>
        </template>
        <el-empty v-else description="该资源暂无任何依赖" />
      </div>
    </el-drawer>

    <!-- 版本历史对话框 -->
    <el-drawer v-model="historyDrawerVisible" title="版本历史与回溯" size="500px">
      <div v-loading="historyLoading" class="history-content">
        <el-timeline>
          <el-timeline-item
            v-for="v in versionHistory"
            :key="v.id"
            :timestamp="formatDate(v.created_at)"
            placement="top"
          >
            <el-card class="history-card">
              <div class="history-info">
                <div class="history-main">
                  <span class="history-ver">{{ v.semver || 'v' + v.version_num }}</span>
                  <el-tag size="small" :type="v.state === 'ACTIVE' ? 'success' : 'info'">{{ statusMap[v.state] }}</el-tag>
                  <el-tag v-if="v.id === currentResource?.latest_version?.id" size="small" effect="dark">当前</el-tag>
                </div>
                <div class="history-actions">
                  <el-button link type="primary" @click="handleDownload(v.download_url)">下载</el-button>
                  <el-button 
                    v-if="v.id !== currentResource?.latest_version?.id" 
                    link 
                    type="warning" 
                    @click="rollback(v)"
                  >
                    设为主版本
                  </el-button>
                </div>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-drawer>

    <!-- 上传确认与依赖选择对话框 -->
    <el-dialog v-model="uploadConfirmVisible" title="确认上传信息" width="550px" class="premium-dialog">
      <el-form :model="uploadForm" label-position="top">
        <el-form-item label="语义化版本 (SemVer)" required>
          <el-input v-model="uploadForm.semver" placeholder="例如: v1.0.0" />
          <div class="input-tip">建议遵循语义化版本规范，方便后续依赖追踪。</div>
        </el-form-item>
        
        <el-form-item label="资源依赖">
          <el-select
            v-model="uploadForm.dependencies"
            multiple
            filterable
            remote
            reserve-keyword
            placeholder="搜索并选择关联资源"
            :remote-method="searchTargetResources"
            :loading="searchLoading"
            style="width: 100%"
            value-key="id"
          >
            <el-option
              v-for="item in searchResults"
              :key="item.id"
              :label="item.name"
              :value="item"
            >
              <div class="search-option">
                <span class="option-name">{{ item.name }}</span>
                <span class="option-type">{{ item.type_key }}</span>
              </div>
            </el-option>
          </el-select>
          <div class="input-tip">你可以搜索并关联目前系统中已有的其他资源。</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="uploadConfirmVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmAndDoUpload" :loading="uploading">
           开始上传
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { 
  Upload, Refresh, Plus, Folder, FolderOpened, Delete, 
  PriceTag, Connection, Grid, Clock, Files, DataLine, 
  Download, Search, Box, Location, Promotion, Share, CircleCheckFilled
} from '@element-plus/icons-vue'
import axios from 'axios'
import JSZip from 'jszip'
import { ElMessage, ElMessageBox } from 'element-plus'
import { buildTree } from '../../core/utils/tree'

interface Category {
  id: string
  name: string
  parent_id?: string
}

interface Resource {
  id: string
  name: string
  tags: string[]
  owner_id: string
  scope: 'PRIVATE' | 'PUBLIC'
  created_at: string
  latest_version?: {
    id: string
    version_num: number
    semver?: string
    state: string
    meta_data?: any
    file_size?: number
    download_url?: string
  }
}

const props = defineProps<{
  typeKey: string
  typeName: string
  uploadMode?: 'single' | 'folder-zip'
  accept?: string
  enableScope?: boolean
}>()

const resources = ref<Resource[]>([])
const categories = ref<Category[]>([])
const loading = ref(false)
const syncing = ref(false)
const uploading = ref(false)
const compressing = ref(false)
const progress = ref(0)
const uploadPercent = ref(0)
const currentFile = ref('')
const selectedCategoryId = ref('all')
const activeScope = ref<'ALL' | 'PRIVATE' | 'PUBLIC'>(props.enableScope ? 'ALL' : 'PUBLIC')
const tagDialogVisible = ref(false)
const tagLoading = ref(false)
const editingTags = ref<string[]>([])
const currentResourceId = ref('')
const depDrawerVisible = ref(false)
const depLoading = ref(false)
const depTree = ref<any[]>([])
const currentResource = ref<Resource | null>(null)

// 版本历史相关
const historyDrawerVisible = ref(false)
const historyLoading = ref(false)
const versionHistory = ref<any[]>([])

// 打包下载相关
const bundleLoading = ref(false)
const packLoading = ref(false)

// 上传表单相关
const uploadConfirmVisible = ref(false)
const uploadForm = ref({
    semver: 'v1.0.0',
    dependencies: [] as any[]
})
const searchLoading = ref(false)
const searchResults = ref<any[]>([])
const pendingUploadData = ref<{displayName: string, blob: Blob, contentType: string, filename: string} | null>(null)

const existingTags = computed(() => {
    const tags = new Set<string>()
    resources.value.forEach(s => {
        if (s.tags) s.tags.forEach(t => tags.add(t))
    })
    return Array.from(tags)
})

const openTagEditor = (row: Resource) => {
    currentResourceId.value = row.id
    editingTags.value = [...(row.tags || [])]
    tagDialogVisible.value = true
}

const saveTags = async () => {
    tagLoading.value = true
    try {
        await axios.patch(`/api/v1/resources/${currentResourceId.value}/tags`, {
            tags: editingTags.value
        })
        ElMessage.success('标签更新成功')
        tagDialogVisible.value = false
        fetchList()
    } finally {
        tagLoading.value = false
    }
}

const defaultProps = {
  children: 'children',
  label: 'name',
}

const categoryTree = computed(() => {
  const tree = buildTree(categories.value)
  return [
    { id: 'all', name: '全部分类' },
    ...tree
  ]
})

const currentCategoryName = computed(() => {
    if (selectedCategoryId.value === 'all') return '全部'
    const cat = categories.value.find(c => c.id === selectedCategoryId.value)
    return cat ? cat.name : ''
})

const statusMap: Record<string, string> = {
  ACTIVE: '已就绪',
  READY: '就绪',
  PROCESSING: '处理中',
  PENDING: '排队中',
  FAILED: '处理失败',
}

const formatDate = (dateString: string) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  const pad = (num: number) => num.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const formatSize = (bytes?: number) => {
  if (bytes === undefined || bytes === null) return 'N/A'
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const fetchList = async () => {
    loading.value = true
    try {
        const params: any = { type: props.typeKey }
        if (selectedCategoryId.value !== 'all') {
            params.category_id = selectedCategoryId.value
        }
        
        const currentUserId = 'admin'
        if (activeScope.value === 'PRIVATE') {
          params.scope = 'PRIVATE'
          params.owner_id = currentUserId
        } else if (activeScope.value === 'PUBLIC') {
          params.scope = 'PUBLIC'
        } else {
          // 全部：通过接口逻辑（后端已适配：如果不传 scope 且传了 owner_id，则显示公共 + 该用户的私有）
          params.owner_id = currentUserId
        }

        const res = await axios.get('/api/v1/resources', { params })
        resources.value = res.data.items || []
    } catch (err: any) {
        ElMessage.error('获取列表失败: ' + (err.response?.data?.error || err.message))
    } finally {
        loading.value = false
    }
}

watch([selectedCategoryId, activeScope], () => {
    fetchList()
})

const syncFromStorage = async () => {
    syncing.value = true
    try {
        const res = await axios.post('/api/v1/resources/sync')
        ElMessage.success(`同步完成，共恢复 ${res.data.count} 个资源`)
        fetchList()
    } catch (err: any) {
        ElMessage.error('同步失败: ' + (err.response?.data?.error || err.message))
    } finally {
        syncing.value = false
    }
}

const fetchCategories = async () => {
    const res = await axios.get('/api/v1/categories', { params: { type: props.typeKey } })
    categories.value = res.data || []
}

const handleCategoryClick = (data: any) => {
    selectedCategoryId.value = data.id
    fetchList()
}

const promptAddCategory = () => {
    ElMessageBox.prompt('请输入分类名称', '新建分类', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
    }).then(async ({ value }) => {
        if (!value) return
        await axios.post('/api/v1/categories', {
            type_key: props.typeKey,
            name: value,
            parent_id: ''
        })
        ElMessage.success('创建成功')
        fetchCategories()
    })
}

const confirmDeleteCategory = (id: string) => {
    const categoryToDelete = categories.value.find(c => c.id === id);
    const categoryName = categoryToDelete ? categoryToDelete.name : '该分类';

    ElMessageBox.confirm(`确定要删除分类 "${categoryName}" 吗？`, '警告', {
        type: 'warning'
    }).then(async () => {
        await axios.delete(`/api/v1/categories/${id}`)
        ElMessage.success('删除成功')
        if (selectedCategoryId.value === id) {
            selectedCategoryId.value = 'all'
            fetchList()
        }
        fetchCategories()
    })
}

// ---------------- 上传逻辑 ----------------

const triggerFolderUpload = () => {
    document.getElementById('folderInput')?.click()
}

const triggerFileUpload = () => {
    document.getElementById('fileInput')?.click()
}

const handleFolderSelect = async (event: Event) => {
    const input = event.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return

    const files = Array.from(input.files)
    const rootFolderName = files[0].webkitRelativePath.split('/')[0]
    
    uploading.value = true
    compressing.value = true
    progress.value = 0

    try {
        const zip = new JSZip()
        files.forEach(file => {
            zip.file(file.webkitRelativePath, file)
        })

        const content = await zip.generateAsync({ 
            type: 'blob',
            compression: 'DEFLATE',
            compressionOptions: { level: 6 }
        }, (meta) => {
            progress.value = Number(meta.percent.toFixed(0))
            currentFile.value = meta.currentFile || ''
        })

        compressing.value = false
        
        pendingUploadData.value = {
            displayName: rootFolderName,
            blob: content,
            contentType: 'application/zip',
            filename: rootFolderName + '.zip'
        }
        uploadConfirmVisible.value = true
        fetchList()
    } catch (e: any) {
        console.error(e)
        ElMessage.error('处理失败: ' + (e.message || '未知错误'))
    } finally {
        uploading.value = false
        input.value = ''
    }
}

const handleFileSelect = async (event: Event) => {
    const input = event.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return
    const file = input.files[0]
    
    uploading.value = true
    try {
        const nameWithoutExt = file.name.substring(0, file.name.lastIndexOf('.')) || file.name
        pendingUploadData.value = {
            displayName: nameWithoutExt,
            blob: file,
            contentType: file.type || 'application/octet-stream',
            filename: file.name
        }
        uploadConfirmVisible.value = true
        fetchList()
    } catch (e: any) {
        console.error(e)
        ElMessage.error('上传失败: ' + (e.message || '未知错误'))
    } finally {
        uploading.value = false
        input.value = ''
    }
}

const searchTargetResources = async (query: string) => {
    if (query) {
        searchLoading.value = true
        try {
            // 目前简单搜索全部资源，生产环境应根据 typeKey 进行相关性筛选
            const res = await axios.get('/api/v1/resources', { params: { name: query } })
            searchResults.value = res.data.items || []
        } finally {
            searchLoading.value = false
        }
    } else {
        searchResults.value = []
    }
}

const confirmAndDoUpload = async () => {
    if (!pendingUploadData.value) return
    const { displayName, blob, contentType, filename } = pendingUploadData.value
    
    uploading.value = true
    try {
        await performUpload(displayName, blob, contentType, filename)
        ElMessage.success('任务已提交并自动关联依赖')
        uploadConfirmVisible.value = false
        pendingUploadData.value = null
        fetchList()
    } catch (e: any) {
        ElMessage.error('上传失败: ' + e.message)
    } finally {
        uploading.value = false
    }
}

const performUpload = async (displayName: string, blob: Blob, contentType: string, filename: string) => {
    const res = await axios.post('/api/v1/integration/upload/token', {
        resource_type: props.typeKey,
        checksum: 'skip-for-now',
        size: blob.size,
        filename: filename
    })
    
    const { ticket_id, presigned_url } = res.data

    await axios.put(presigned_url, blob, {
        headers: { 'Content-Type': contentType },
        onUploadProgress: (p) => {
            if (p.total) {
                uploadPercent.value = Math.round((p.loaded * 100) / p.total)
            }
        }
    })

    await axios.post('/api/v1/integration/upload/confirm', {
        ticket_id,
        type_key: props.typeKey,
        category_id: selectedCategoryId.value === 'all' ? '' : selectedCategoryId.value,
        name: displayName,
        owner_id: 'admin',
        size: blob.size,
        semver: uploadForm.value.semver,
        dependencies: uploadForm.value.dependencies.map(d => ({
            target_resource_id: d.id,
            constraint: 'latest' // 目前默认锁定到最新版
        })),
        extra_meta: {}
    })
}

const publishResource = (row: Resource) => {
    ElMessageBox.confirm(`确定要将资源 "${row.name}" 发布到公共库吗？发布后所有用户可见。`, '发布确认', {
        type: 'success',
        confirmButtonText: '确定发布',
        cancelButtonText: '取消'
    }).then(async () => {
        try {
            await axios.patch(`/api/v1/resources/${row.id}/scope`, { scope: 'PUBLIC' })
            ElMessage.success('发布成功')
            fetchList()
        } catch (err: any) {
            ElMessage.error('发布失败: ' + (err.response?.data?.error || err.message))
        }
    })
}

// -----------------------------------------

const confirmDelete = (row: any) => {
    ElMessageBox.confirm(`确定要删除资源 "${row.name}" 吗？`, '警告', {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消'
    }).then(async () => {
        try {
            await axios.delete(`/api/v1/resources/${row.id}`)
            ElMessage.success('删除成功')
            fetchList()
        } catch (err: any) {
            ElMessage.error('删除失败: ' + (err.response?.data?.error || err.message))
        }
    })
}

const download = async (row: any) => {
    const res = await axios.get(`/api/v1/resources/${row.id}`)
    const url = res.data.latest_version?.download_url
    if (url) {
        window.open(url, '_blank')
    } else {
        ElMessage.warning('下载链接无效')
    }
}

const handleDownload = (url?: string) => {
  if (url) {
    window.open(url, '_blank')
  } else {
    ElMessage.warning('下载链接无效')
  }
}

const viewDependencies = async (row: Resource) => {
    if (!row.latest_version?.id) {
        ElMessage.warning('未能获取该资源的版本信息')
        return
    }
    currentResource.value = row
    depDrawerVisible.value = true
    depLoading.value = true
    try {
        const res = await axios.get(`/api/v1/resources/versions/${row.latest_version.id}/dependency-tree`)
        depTree.value = Array.isArray(res.data) ? res.data : []
    } catch (err: any) {
        ElMessage.error('获取依赖树失败: ' + (err.response?.data?.error || err.message))
    } finally {
        depLoading.value = false
    }
}

const viewHistory = async (row: Resource) => {
    currentResource.value = row
    historyDrawerVisible.value = true
    historyLoading.value = true
    try {
        const res = await axios.get(`/api/v1/resources/${row.id}/versions`)
        versionHistory.value = res.data || []
    } catch (err: any) {
        ElMessage.error('获取历史失败: ' + (err.response?.data?.error || err.message))
    } finally {
        historyLoading.value = false
    }
}

const rollback = async (ver: any) => {
    if (!currentResource.value) return
    ElMessageBox.confirm(`确定要将版本切换回 ${ver.semver || 'v' + ver.version_num} 吗？此操作会影响所有下游依赖。`, '版本回溯确认', {
        type: 'warning'
    }).then(async () => {
        try {
            await axios.post(`/api/v1/resources/${currentResource.value?.id}/latest`, {
                version_id: ver.id
            })
            ElMessage.success('版本切换成功')
            // 更新当前列表和状态
            fetchList()
            viewHistory(currentResource.value!)
        } catch (err: any) {
            ElMessage.error('切换失败: ' + (err.response?.data?.error || err.message))
        }
    })
}

const downloadBundle = async () => {
    if (!currentResource.value?.latest_version?.id) return
    bundleLoading.value = true
    try {
        const res = await axios.get(`/api/v1/resources/versions/${currentResource.value.latest_version.id}/bundle`)
        const data = res.data
        
        // 将清单作为 JSON 下载
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `bundle-${currentResource.value.name}-${data.root_version}.json`
        a.click()
        URL.revokeObjectURL(url)
        
        ElMessage.success('依赖包清单已生成并下载')
    } catch (err: any) {
        ElMessage.error('生成打包清单失败: ' + (err.response?.data?.error || err.message))
    } finally {
        bundleLoading.value = false
    }
}

const downloadSimPack = async () => {
    if (!currentResource.value?.latest_version?.id) return
    const vid = currentResource.value.latest_version.id
    // 直接调用后端流式下载接口
    const downloadUrl = `/api/v1/resources/versions/${vid}/download-pack`
    window.open(downloadUrl, '_blank')
    ElMessage.success('已开始生成离线包并下载')
}

let pollInterval: any = null

const initData = () => {
    fetchList()
    fetchCategories()
}

watch(() => props.typeKey, () => {
    selectedCategoryId.value = 'all'
    initData()
})

onMounted(() => {
    initData()
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

/* 侧边栏：更精致的分类树 */
.category-sidebar {
  width: 220px;
  background: var(--sidebar-bg);
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
  display: flex;
  flex-direction: column;
  box-shadow: var(--el-box-shadow-lighter);
}

.sidebar-header {
  padding: 16px 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-lighter);

  span { flex: 1; }
}

.custom-tree {
  background: transparent;
  padding: 12px 8px;
  
  :deep(.el-tree-node__content) {
    height: 36px;
    border-radius: 6px;
    margin-bottom: 2px;
    
    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }
  
  :deep(.is-current > .el-tree-node__content) {
    background-color: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
  }
}

.custom-tree-node {
  display: flex;
  align-items: center;
  width: 100%;
  font-size: 13px;
  padding-right: 12px;
  
  .node-label {
    margin-left: 8px;
    flex: 1;
  }
  
  .node-actions {
    opacity: 0;
    transition: opacity 0.2s;
    color: var(--el-text-color-placeholder);
    
    &:hover { color: var(--el-color-danger); }
  }
}

.custom-tree-node:hover .node-actions { opacity: 1; }

/* 主内容区 */
.resource-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.premium-header {
  background: var(--sidebar-bg);
  padding: 16px 24px;
  border-radius: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: var(--el-box-shadow-lighter);

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .header-right-actions {
    display: flex;
    gap: 12px;
  }
  
  h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    
    small {
      font-weight: 400;
      color: var(--el-text-color-secondary);
      font-size: 13px;
      margin-left: 8px;
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

/* 表格样式定制 */
.premium-table {
  --el-table-header-bg-color: var(--el-fill-color-lighter);
  
  :deep(th.el-table__cell) {
    font-weight: 600;
    color: var(--el-text-color-regular);
    font-size: 13px;
    padding: 12px 0;
  }
  
  :deep(td.el-table__cell) {
    padding: 14px 0;
  }
}

.resource-info-cell {
  display: flex;
  align-items: center;
  gap: 14px;
}

.resource-icon {
  width: 40px;
  height: 40px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.resource-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

.name-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.scope-tag {
  font-family: var(--el-font-family);
  height: 20px;
  padding: 0 6px;
  border-radius: 4px;
  font-size: 11px;
}

.resource-meta {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  display: flex;
  gap: 12px;
  margin-top: 4px;
  
  span {
    display: flex;
    align-items: center;
    gap: 4px;
  }
}

.tag-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.premium-tag {
  border: none;
  background: var(--el-fill-color);
  color: var(--el-text-color-regular);
}

.add-tag-btn {
  background: transparent;
  border: 1px dashed var(--el-border-color);
  color: var(--el-text-color-placeholder);
  
  &:hover {
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
  }
}

.version-badge {
  background: var(--el-fill-color-light);
  padding: 2px 8px;
  border-radius: 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
  font-weight: 500;
  
  &.clickable {
    cursor: pointer;
    transition: all 0.2s;
    &:hover {
      background: var(--el-color-primary-light-8);
      color: var(--el-color-primary);
    }
  }
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  padding-right: 20px;

  .header-actions {
    display: flex;
    gap: 8px;
  }
}

.history-content {
  padding: 0 20px;
}

.history-card {
  margin-bottom: 4px;
  :deep(.el-card__body) {
    padding: 12px;
  }
}

.history-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.history-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

.history-ver {
  font-weight: 600;
  font-size: 14px;
}

.history-actions {
  display: flex;
  gap: 4px;
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.skip-icon {
  color: var(--el-color-success);
  font-size: 16px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
  
  &.active {
    background: var(--el-color-success);
    box-shadow: 0 0 8px var(--el-color-success-light-5);
  }
  
  &.processing {
    background: var(--el-color-primary);
    animation: statusPulse 2s infinite;
  }
  
  &.failed {
    background: var(--el-color-danger);
  }
}

@keyframes statusPulse {
  0% { transform: scale(0.9); opacity: 0.6; }
  50% { transform: scale(1.1); opacity: 1; }
  100% { transform: scale(0.9); opacity: 0.6; }
}

.status-text {
  font-size: 13px;
  color: var(--el-text-color-regular);
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

.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: flex-start;
  flex-wrap: nowrap;
}

/* 依赖树样式 */
.dep-drawer {
  :deep(.el-drawer__body) {
    padding: 0;
  }
}

.dep-content {
  padding: 20px;
  height: 100%;
}

.dep-tree {
  background: transparent;
  
  :deep(.el-tree-node__content) {
    height: auto;
    padding: 8px 0;
    
    &:hover {
      background-color: var(--el-fill-color-light);
    }
  }
}

.dep-node {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
}

.dep-icon {
  margin-top: 4px;
  font-size: 18px;
  color: var(--el-color-primary);
}

.dep-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dep-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.dep-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dep-ver {
  height: 20px;
  padding: 0 6px;
  font-size: 11px;
}

.dep-constraint {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

/* 搜索与表单样式 */
.input-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
  margin-top: 4px;
}

.search-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  
  .option-name {
    font-weight: 500;
  }
  
  .option-type {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    background: var(--el-fill-color-light);
    padding: 0 4px;
    border-radius: 4px;
  }
}
</style>
