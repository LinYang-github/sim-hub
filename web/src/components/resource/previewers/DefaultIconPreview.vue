<template>
  <div class="placeholder-icon">
    <el-icon :size="iconSize">
      <component :is="icon || 'Files'"></component>
    </el-icon>
  </div>
</template>

<script setup lang="ts">

defineProps<{
  typeKey: string
  icon?: string
  iconSize?: number
}>()
</script>

<style scoped>
.placeholder-icon {
  color: var(--el-text-color-placeholder);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  width: 100%;
}
</style>
