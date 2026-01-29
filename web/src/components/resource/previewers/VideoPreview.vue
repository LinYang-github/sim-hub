<template>
  <div class="video-preview-container">
    <video 
      ref="videoPlayer"
      :src="url" 
      controls 
      class="video-player"
      preload="metadata"
    >
      您的浏览器不支持视频播放
    </video>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  url: string
}>()

const videoPlayer = ref<HTMLVideoElement | null>(null)
</script>

<style scoped>
.video-preview-container {
  width: 100%;
  height: 100%;
  background: #000;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.video-player {
  max-width: 100%;
  max-height: 100%;
  outline: none;
}
</style>
