<template>
  <div class="iframe-container" v-loading="loading">
    <iframe 
      v-if="url" 
      :src="url" 
      frameborder="0" 
      width="100%" 
      height="100%"
      @load="onLoad"
    ></iframe>
    <div v-else class="error">No URL provided</div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  url: string
}>()

const loading = ref(true)

const onLoad = () => {
  loading.value = false
}

// Reset loading when URL changes (if component is reused)
watch(() => props.url, () => {
  loading.value = true
})
</script>

<style scoped>
.iframe-container {
  width: 100%;
  height: calc(100vh - 80px); /* Adjust based on header height */
  overflow: hidden;
}
</style>
