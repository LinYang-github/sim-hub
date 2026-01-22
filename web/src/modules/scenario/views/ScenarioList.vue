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
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
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

// 获取资源列表
const fetchList = async () => {
    loading.value = true
    try {
        const res = await axios.get('/api/v1/resources', { params: { type: 'scenario' } })
        scenarios.value = res.data.items || []
    } finally {
        loading.value = false
    }
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
    // 假定根文件夹名称即为想定名称
    // files[0].webkitRelativePath 示例: "MyScenario/data.json"
    const rootFolderName = files[0].webkitRelativePath.split('/')[0]
    
    await uploadFolderAsZip(rootFolderName, files)
    
    // 重置输入框以便再次触发变更事件
    input.value = ''
}

// 将文件夹打包为 ZIP 并上传
const uploadFolderAsZip = async (name: string, files: File[]) => {
    uploading.value = true
    compressing.value = true
    progress.value = 0

    try {
        const zip = new JSZip()
        
        // 将文件添加到 ZIP
        files.forEach(file => {
            // 保留相对路径结构
            zip.file(file.webkitRelativePath, file)
        })

        // 生成 ZIP 二进制对象 (Blob)
        const content = await zip.generateAsync({ 
            type: 'blob',
            compression: 'DEFLATE',
            compressionOptions: { level: 6 }
        }, (meta) => {
            progress.value = Number(meta.percent.toFixed(0))
            currentFile.value = meta.currentFile || ''
        })

        compressing.value = false
        // 开始上传流程
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
    // 1. 获取上传令牌
    const res = await axios.post('/api/v1/integration/upload/token', {
        resource_type: 'scenario',
        checksum: 'skip-for-now', // 暂时跳过校验码
        size: blob.size,
        filename: name + '.zip'
    })
    
    const { ticket_id, presigned_url } = res.data

    // 2. 直接上传到 MinIO
    await axios.put(presigned_url, blob, {
        headers: { 'Content-Type': 'application/zip' },
        onUploadProgress: (p) => {
            if (p.total) {
                uploadPercent.value = Math.round((p.loaded * 100) / p.total)
            }
        }
    })

    // 3. 确认上传完成
    await axios.post('/api/v1/integration/upload/confirm', {
        ticket_id,
        type_key: 'scenario',
        name: name,
        owner_id: 'admin', // 模拟管理员 ID
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
    // 开启轮询，每 3 秒刷新一次列表以获取处理状态
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
