<template>
  <div class="image-preview-container">
    <el-image 
      :src="url" 
      fit="contain" 
      class="preview-image"
      :preview-src-list="[url]"
      hide-on-click-modal
    >
      <template #placeholder>
        <div class="image-loading">
          <el-icon class="is-loading"><Loading /></el-icon>
        </div>
      </template>
      <template #error>
        <div class="image-error">
          <el-icon><Picture /></el-icon>
          <span>图片读取失败</span>
        </div>
      </template>
    </el-image>
  </div>
</template>

<script setup lang="ts">
import { Picture, Loading } from '@element-plus/icons-vue'

defineProps<{
  url: string
}>()
</script>

<style scoped>
.image-preview-container {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #1a1a1a; /* 通用暗色背景，适合展示各类贴图 */
  background-image: 
    linear-gradient(45deg, #222 25%, transparent 25%),
    linear-gradient(-45deg, #222 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, #222 75%),
    linear-gradient(-45deg, transparent 75%, #222 75%);
  background-size: 20px 20px;
  background-position: 0 0, 0 10px, 10px -10px, -10px 0px;
}

.preview-image {
  width: 100%;
  height: 100%;
}

.image-loading, .image-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--el-text-color-placeholder);
  gap: 8px;
}
</style>
