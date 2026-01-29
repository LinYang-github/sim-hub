<template>
  <el-input 
    v-model="internalValue" 
    :type="type"
    :rows="rows"
    :placeholder="placeholder"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  type?: 'text' | 'textarea'
  rows?: number
}>(), {
  modelValue: '',
  type: 'text',
  rows: 3
})

const emit = defineEmits(['update:modelValue'])

const internalValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>
