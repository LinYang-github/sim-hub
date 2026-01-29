<template>
  <div class="score-rater">
     <el-rate 
        v-model="rateValue" 
        :max="10" 
        show-score
        text-color="#ff9900"
     />
     <span class="tip">（自定义组件演示：请打分）</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElRate } from 'element-plus'

const props = defineProps<{
  modelValue?: number
}>()

const emit = defineEmits(['update:modelValue'])

const rateValue = computed({
  get: () => props.modelValue || 0,
  set: (val) => emit('update:modelValue', val || 0)
})
</script>

<style scoped>
.score-rater {
  display: flex;
  align-items: center;
  gap: 10px;
  background: var(--el-fill-color-light);
  padding: 8px 12px;
  border-radius: 8px;
}
.tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
