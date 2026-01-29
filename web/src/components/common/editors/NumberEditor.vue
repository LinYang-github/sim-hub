<template>
  <el-input-number 
    v-model="internalValue"
    :placeholder="placeholder"
    style="width: 100%"
    controls-position="right"
    :min="min"
    :max="max"
    :step="step"
    :precision="precision"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  modelValue?: number
  placeholder?: string
  min?: number
  max?: number
  step?: number
  precision?: number
}>(), {
  modelValue: undefined
})

const emit = defineEmits(['update:modelValue'])

const internalValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>
