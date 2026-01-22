<template>
  <div class="scenario-layout">
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
    <div class="scenario-main">
      <div class="premium-header">
        <div class="title-group">
          <h2>{{ currentCategoryName }} <small>想定资源库</small></h2>
        </div>
        
        <div class="action-group">
          <el-button-group>
            <el-button type="primary" class="upload-btn" @click="triggerFolderUpload">
              <el-icon><Upload /></el-icon> 导入想定包
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
        <el-table :data="scenarios" style="width: 100%" v-loading="loading" class="premium-table">
          <el-table-column label="想定详情" min-width="250">
            <template #default="scope">
              <div class="scenario-info-cell">
                <div class="scenario-icon">
                  <el-icon><Files /></el-icon>
                </div>
                <div class="scenario-text">
                  <div class="scenario-name">{{ scope.row.name }}</div>
                  <div class="scenario-meta">
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

          <el-table-column label="版本" width="100">
            <template #default="scope">
              <span class="version-badge">v{{ scope.row.latest_version?.version_num || 1 }}</span>
            </template>
          </el-table-column>

          <el-table-column label="状态" width="140">
            <template #default="scope">
              <div class="status-cell">
                <div :class="['status-dot', scope.row.latest_version?.state.toLowerCase()]"></div>
                <span class="status-text">{{ statusMap[scope.row.latest_version?.state] || scope.row.latest_version?.state }}</span>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="操作" width="120" fixed="right">
            <template #default="scope">
              <el-button type="primary" link :disabled="scope.row.latest_version?.state !== 'ACTIVE'" @click="download(scope.row)">
                <el-icon><Download /></el-icon> 下载
              </el-button>
              <el-button type="danger" link @click="confirmDelete(scope.row)">
                <el-icon><Delete /></el-icon> 删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { 
  Upload, Refresh, Plus, Folder, FolderOpened, Delete, 
  PriceTag, Connection, Grid, Clock, Files, DataLine, 
  Download, Search 
} from '@element-plus/icons-vue'
import axios from 'axios'
import JSZip from 'jszip'
import { ElMessage, ElMessageBox } from 'element-plus'
import { buildTree } from '../../../core/utils/tree'

interface Category {
  id: string
  name: string
  parent_id?: string
}

interface Resource {
  id: string
  name: string
  tags: string[]
  created_at: string
  latest_version?: {
    version_num: number
    state: string
    meta_data?: any
    file_size?: number
  }
}

const scenarios = ref<Resource[]>([])
const categories = ref<Category[]>([])
const loading = ref(false)
const syncing = ref(false)
const uploading = ref(false)
const compressing = ref(false)
const progress = ref(0)
const uploadPercent = ref(0)
const currentFile = ref('')
const selectedCategoryId = ref('all')
const tagDialogVisible = ref(false)
const tagLoading = ref(false)
const editingTags = ref<string[]>([])
const currentResourceId = ref('')
const existingTags = computed(() => {
    const tags = new Set<string>()
    scenarios.value.forEach(s => {
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

// 格式化分类树 (支持多级嵌套)
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
  ACTIVE: '已激活',
  PROCESSING: '处理中',
  PENDING: '待处理',
  FAILED: '失败',
  UNKNOWN: '未知'
}

const formatDate = (dateString: string) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleString()
}

const formatSize = (bytes?: number) => {
  if (bytes === undefined || bytes === null) return 'N/A'
  if (bytes === 0) return '0 Bytes'
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 获取资源列表
const fetchList = async () => {
    loading.value = true
    try {
        const params: any = { type: 'scenario' }
        if (selectedCategoryId.value !== 'all') {
            params.category_id = selectedCategoryId.value
        }
        const res = await axios.get('/api/v1/resources', { params })
        scenarios.value = res.data.items || []
    } catch (err: any) {
        ElMessage.error('获取列表失败: ' + (err.response?.data?.error || err.message))
    } finally {
        loading.value = false
    }
}

// 同步存储
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

// 获取分类列表
const fetchCategories = async () => {
    const res = await axios.get('/api/v1/categories', { params: { type: 'scenario' } })
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
            type_key: 'scenario',
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

// 触发隐藏的文件夹输入框
const triggerFolderUpload = () => {
    document.getElementById('folderInput')?.click()
}

// 处理文件夹选择
const handleFolderSelect = async (event: Event) => {
    const input = event.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return

    const files = Array.from(input.files)
    const rootFolderName = files[0].webkitRelativePath.split('/')[0]
    
    await uploadFolderAsZip(rootFolderName, files)
    input.value = ''
}

// 将文件夹打包为 ZIP 并上传
const uploadFolderAsZip = async (name: string, files: File[]) => {
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
        await uploadZip(name, content)
        
        ElMessage.success('上传成功')
        fetchList()
    } catch (e: any) {
        console.error(e)
        ElMessage.error('处理失败: ' + (e.message || '未知错误'))
    } finally {
        uploading.value = false
    }
}

// 执行 ZIP 文件上传
const uploadZip = async (name: string, blob: Blob) => {
    const res = await axios.post('/api/v1/integration/upload/token', {
        resource_type: 'scenario',
        checksum: 'skip-for-now',
        size: blob.size,
        filename: name + '.zip'
    })
    
    const { ticket_id, presigned_url } = res.data

    await axios.put(presigned_url, blob, {
        headers: { 'Content-Type': 'application/zip' },
        onUploadProgress: (p) => {
            if (p.total) {
                uploadPercent.value = Math.round((p.loaded * 100) / p.total)
            }
        }
    })

    await axios.post('/api/v1/integration/upload/confirm', {
        ticket_id,
        type_key: 'scenario',
        category_id: selectedCategoryId.value === 'all' ? '' : selectedCategoryId.value,
        name: name,
        owner_id: 'admin',
        size: blob.size,
        extra_meta: {}
    })
}

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

// 处理下载请求
const download = async (row: any) => {
    const res = await axios.get(`/api/v1/resources/${row.id}`)
    const url = res.data.latest_version?.download_url
    if (url) {
        window.open(url, '_blank')
    } else {
        ElMessage.warning('下载链接无效')
    }
}

let pollInterval: any = null

onMounted(() => {
    fetchList()
    fetchCategories()
    pollInterval = setInterval(() => {
        const hasProcessing = scenarios.value.some((s: any) => 
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

<style scoped>
.scenario-layout {
  display: flex;
  height: calc(100vh - 84px); /* 减去顶部 Workstation 导航 */
  padding: 16px;
  background-color: #f5f7fa;
  font-family: 'Inter', system-ui, -apple-system, sans-serif;
  gap: 16px;
}

/* 侧边栏：Glassmorphism */
.category-sidebar {
  width: 240px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(10px);
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #1e293b;
  border-bottom: 1px solid #f1f5f9;
}

.sidebar-header span {
  flex: 1;
}

.custom-tree {
  background: transparent;
  padding: 8px;
}

.custom-tree-node {
  display: flex;
  align-items: center;
  width: 100%;
  padding-right: 8px;
}

.node-label {
  margin-left: 8px;
  flex: 1;
  font-size: 13.5px;
}

.node-actions {
  opacity: 0;
  transition: opacity 0.2s;
}

.custom-tree-node:hover .node-actions {
  opacity: 1;
}

.delete-icon {
  color: #94a3b8;
  cursor: pointer;
}

.delete-icon:hover {
  color: #f43f5e;
}

/* 主内容区 */
.scenario-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow: hidden;
}

.premium-header {
  background: #ffffff;
  padding: 12px 24px;
  border-radius: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.premium-header h2 {
  margin: 0;
  color: #0f172a;
  font-size: 20px;
}

.premium-header h2 small {
  font-weight: 400;
  color: #64748b;
  font-size: 14px;
  margin-left: 8px;
}

.action-group {
  display: flex;
  gap: 12px;
}

.content-container {
  flex: 1;
  background: #ffffff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.05);
}

/* 表格定制 */
.premium-table {
  --el-table-header-bg-color: #f8fafc;
}

.scenario-info-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.scenario-icon {
  width: 40px;
  height: 40px;
  background: #eff6ff;
  color: #3b82f6;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
}

.scenario-name {
  font-weight: 600;
  color: #1e293b;
  font-size: 14px;
}

.scenario-meta {
  font-size: 12px;
  color: #94a3b8;
  display: flex;
  gap: 12px;
  margin-top: 4px;
}

.scenario-meta span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.tag-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.premium-tag {
  border: none;
  background: #f1f5f9;
  color: #475569;
}

.add-tag-btn {
  background: #f8fafc;
  border: 1px dashed #cbd5e1;
  color: #64748b;
}

.version-badge {
  background: #f1f5f9;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  color: #475569;
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #cbd5e1;
}

.status-dot.active {
  background: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

.status-dot.processing {
  background: #3b82f6;
  animation: pulse 1.5s infinite;
}

.status-dot.failed {
  background: #f43f5e;
}

@keyframes pulse {
  0% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
  70% { box-shadow: 0 0 0 10px rgba(59, 130, 246, 0); }
  100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0); }
}

.status-text {
  font-size: 13px;
  color: #475569;
}

/* 进度条定制 */
.upload-status {
  padding: 12px;
  background: #f8fafc;
  border-radius: 8px;
  margin-top: 10px;
}
</style>
