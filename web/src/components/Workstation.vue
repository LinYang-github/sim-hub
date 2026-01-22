<template>
  <div>
    <h1>工作台</h1>
    <p>欢迎使用 SimHub 仿真资源管理平台。您可以在此上传和管理各类仿真资源。</p>
    <el-button type="primary" @click="router.push('/resources')">查看资源库</el-button>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>
