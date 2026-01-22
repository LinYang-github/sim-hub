<template>
  <div>
    <h1>资源库</h1>
    <el-table :data="tableData" style="width: 100%" v-loading="loading">
      <el-table-column prop="name" label="资源名称" width="280" />
      <el-table-column prop="type_key" label="类型" width="180" />
      <el-table-column prop="owner_id" label="所有者" width="120" />
      <el-table-column prop="created_at" label="创建时间" />
      <el-table-column fixed="right" label="操作" width="120">
        <template #default="scope">
          <el-button link type="primary" size="small" @click="download(scope.row.id)">下载</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'

const tableData = ref([])
const loading = ref(false)

const fetchResources = async () => {
    loading.value = true
    try {
        const res = await axios.get('/api/v1/resources')
        // Mock data structure adjustment if needed
        tableData.value = res.data.items
    } catch (err) {
        console.error(err)
    } finally {
        loading.value = false
    }
}

const download = async (id: string) => {
    try {
        const res = await axios.get(`/api/v1/resources/${id}`)
        const url = res.data.latest_version?.download_url
        if (url) {
            window.open(url, '_blank')
        } else {
            alert('No download URL found')
        }
    } catch (err) {
        alert('Failed to get download link')
    }
}

onMounted(() => {
    // In dev, we need proxy or CORS. For now assume proxy configured in vite.config.ts
    fetchResources()
})
</script>
