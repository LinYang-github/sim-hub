<template>
  <div class="scenario-container">
    <!-- 左侧分类树 -->
    <div class="category-sidebar">
      <div class="sidebar-header">
        <span>分类目录</span>
        <el-button type="primary" link @click="promptAddCategory">
          <el-icon><Plus /></el-icon>
        </el-button>
      </div>
      <el-tree 
        :data="categoryTree" 
        :props="defaultProps" 
        @node-click="handleCategoryClick"
        highlight-current
        default-expand-all
      >
        <template #default="{ node, data }">
          <span class="custom-tree-node">
            <span><el-icon><FolderOpened v-if="node.expanded"/><Folder v-else/></el-icon> {{ node.label }}</span>
            <span class="node-actions" v-if="data.id !== 'all'">
              <el-icon @click.stop="confirmDeleteCategory(data)"><Delete /></el-icon>
            </span>
          </span>
        </template>
      </el-tree>
    </div>

    <!-- 右侧列表 -->
    <div class="scenario-main">
      <div class="toolbar">
        <h3>{{ currentCategoryName }} 想定库</h3>
        <div class="actions">
          <input
            type="file"
            id="folderInput"
            webkitdirectory
            directory
            style="display: none"
            @change="handleFolderSelect"
          />
          <el-button type="primary" @click="triggerFolderUpload">
            <el-icon><Upload /></el-icon> 上传想定包
          </el-button>
          <el-button @click="fetchList"><el-icon><Refresh /></el-icon></el-button>
        </div>
      </div>

      <!-- 上传进度展示 -->
      <div v-if="uploading" class="upload-status">
        <p v-if="compressing">正在打包文件夹: {{ currentFile }} ({{ progress }}%)</p>
        <el-progress v-else :percentage="uploadPercent" />
      </div>

      <el-table :data="scenarios" style="width: 100%" v-loading="loading">
      <el-table-column prop="name" label="想定名称" />
      <el-table-column prop="version" label="版本" width="80">
          <template #default="scope">
              v{{ scope.row.latest_version?.version_num || 1 }}
          </template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="scope">
          <el-tag :type="getStatusType(scope.row.latest_version?.state)">
            {{ scope.row.latest_version?.state || 'UNKNOWN' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" />
      <el-table-column type="expand" label="详情">
        <template #default="scope">
          <div style="padding: 10px">
            <p v-if="scope.row.latest_version?.meta_data?.scenario_type">
              <b>场景类型:</b> {{ scope.row.latest_version.meta_data.scenario_type }}
            </p>
            <p v-if="scope.row.latest_version?.meta_data?.estimated_duration">
              <b>预估时长:</b> {{ scope.row.latest_version.meta_data.estimated_duration }}s
            </p>
            <p v-if="scope.row.latest_version?.meta_data?.files_count">
              <b>文件数量:</b> {{ scope.row.latest_version.meta_data.files_count }}
            </p>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150">
        <template #default="scope">
          <el-button 
            type="primary" 
            link 
            size="small" 
            :disabled="scope.row.latest_version?.state !== 'ACTIVE'"
            @click="download(scope.row)"
          >下载</el-button>
          <el-button type="danger" link size="small">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { Upload, Refresh, Plus, Folder, FolderOpened, Delete } from '@element-plus/icons-vue'
import axios from 'axios'
import JSZip from 'jszip'
import { ElMessage, ElMessageBox } from 'element-plus'

interface Category {
  id: string
  name: string
  parent_id?: string
}

interface Resource {
  id: string
  name: string
  latest_version?: {
    version_num: number
    state: string
    meta_data?: any
  }
}

const scenarios = ref<Resource[]>([])
const categories = ref<Category[]>([])
const loading = ref(false)
const uploading = ref(false)
const compressing = ref(false)
const progress = ref(0)
const uploadPercent = ref(0)
const currentFile = ref('')
const selectedCategoryId = ref('all')

const defaultProps = {
  children: 'children',
  label: 'name',
}

// 格式化分类树
const categoryTree = computed(() => {
  return [
    { id: 'all', name: '全部分类' },
    ...categories.value
  ]
})

const currentCategoryName = computed(() => {
    if (selectedCategoryId.value === 'all') return '全部'
    const cat = categories.value.find(c => c.id === selectedCategoryId.value)
    return cat ? cat.name : ''
})

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
    } finally {
        loading.value = false
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

const confirmDeleteCategory = (data: any) => {
    ElMessageBox.confirm(`确定要删除分类 "${data.name}" 吗？`, '警告', {
        type: 'warning'
    }).then(async () => {
        await axios.delete(`/api/v1/categories/${data.id}`)
        ElMessage.success('删除成功')
        if (selectedCategoryId.value === data.id) {
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

const getStatusType = (state: string) => {
    switch (state) {
        case 'ACTIVE': return 'success'
        case 'PROCESSING': return 'warning'
        case 'PENDING': return 'info'
        case 'FAILED': return 'danger'
        default: return 'info'
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
.scenario-container {
  display: flex;
  height: calc(100vh - 120px);
  gap: 20px;
}

.category-sidebar {
  width: 240px;
  background: #fff;
  border-right: 1px solid #ebeef5;
  padding: 15px;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  font-weight: bold;
  font-size: 14px;
}

.custom-tree-node {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 14px;
  padding-right: 8px;
}

.node-actions {
  display: none;
  font-size: 12px;
  color: #f56c6c;
}

.custom-tree-node:hover .node-actions {
  display: block;
}

.scenario-main {
  flex: 1;
  overflow: auto;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.upload-status {
    margin: 10px 0;
    padding: 10px;
    background: #f0f9eb;
    border: 1px solid #e1f3d8;
    border-radius: 4px;
}
</style>
