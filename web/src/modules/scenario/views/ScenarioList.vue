<template>
  <div class="scenario-list">
    <div class="toolbar">
      <h3>想定库 (Scenario Repository)</h3>
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
          <el-icon><Upload /></el-icon> 上传想定包 (文件夹)
        </el-button>
        <el-button @click="fetchList"><el-icon><Refresh /></el-icon></el-button>
      </div>
    </div>

    <!-- Upload Progress -->
    <div v-if="uploading" class="upload-status">
      <p v-if="compressing">正在打包文件夹: {{ currentFile }} ({{ progress }}%)</p>
      <el-progress v-else :percentage="uploadPercent" />
    </div>

    <el-table :data="scenarios" style="width: 100%" v-loading="loading">
      <el-table-column prop="name" label="想定名称" />
      <el-table-column prop="version" label="Ver">
          <template #default="scope">
              v{{ scope.row.latest_version?.version_num || 1 }}
          </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" />
      <el-table-column label="操作" width="200">
        <template #default="scope">
          <el-button type="primary" link size="small" @click="download(scope.row)">下载 ZIP</el-button>
          <el-button type="danger" link size="small">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Upload, Refresh } from '@element-plus/icons-vue'
import axios from 'axios'
import JSZip from 'jszip'
import { ElMessage } from 'element-plus'

const scenarios = ref([])
const loading = ref(false)
const uploading = ref(false)
const compressing = ref(false)
const progress = ref(0)
const uploadPercent = ref(0)
const currentFile = ref('')

const fetchList = async () => {
    loading.value = true
    try {
        const res = await axios.get('/api/v1/resources', { params: { type: 'scenario' } })
        scenarios.value = res.data.items || []
    } finally {
        loading.value = false
    }
}

const triggerFolderUpload = () => {
    document.getElementById('folderInput')?.click()
}

const handleFolderSelect = async (event: Event) => {
    const input = event.target as HTMLInputElement
    if (!input.files || input.files.length === 0) return

    const files = Array.from(input.files)
    // Assume root folder name is the scenario name
    // files[0].webkitRelativePath e.g. "MyScenario/data.json"
    const rootFolderName = files[0].webkitRelativePath.split('/')[0]
    
    await uploadFolderAsZip(rootFolderName, files)
    
    // Reset input
    input.value = ''
}

const uploadFolderAsZip = async (name: string, files: File[]) => {
    uploading.value = true
    compressing.value = true
    progress.value = 0

    try {
        const zip = new JSZip()
        let processed = 0
        
        // Add files to ZIP
        files.forEach(file => {
            // Remove root folder from path to keep zip structure clean inside
            // Or keep it? Let's keep relative path structure.
            // But usually users expect zip content to be inside the folder?
            // Let's store the relative path as is.
            zip.file(file.webkitRelativePath, file)
        })

        // Generate ZIP
        const content = await zip.generateAsync({ 
            type: 'blob',
            compression: 'DEFLATE',
            compressionOptions: { level: 6 }
        }, (meta) => {
            progress.value = Number(meta.percent.toFixed(0))
            currentFile.value = meta.currentFile || ''
        })

        compressing.value = false
        // Start Upload Process
        await uploadZip(name, content)
        
        ElMessage.success('上传成功')
        fetchList()
    } catch (e: any) {
        console.error(e)
        ElMessage.error('处理失败: ' + e.message)
    } finally {
        uploading.value = false
    }
}

const uploadZip = async (name: string, blob: Blob) => {
    // 1. Get Token
    const res = await axios.post('/api/v1/integration/upload/token', {
        resource_type: 'scenario',
        checksum: 'skip-for-now',
        size: blob.size,
        filename: name + '.zip'
    })
    
    const { ticket_id, presigned_url } = res.data

    // 2. Upload to MinIO
    await axios.put(presigned_url, blob, {
        headers: { 'Content-Type': 'application/zip' },
        onUploadProgress: (p) => {
            if (p.total) {
                uploadPercent.value = Math.round((p.loaded * 100) / p.total)
            }
        }
    })

    // 3. Confirm
    await axios.post('/api/v1/integration/upload/confirm', {
        ticket_id,
        type_key: 'scenario',
        name: name,
        owner_id: 'admin', // mocked
        size: blob.size,
        extra_meta: {}
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

onMounted(fetchList)
</script>

<style scoped>
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
