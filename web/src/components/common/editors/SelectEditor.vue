<template>
  <el-select 
    v-model="internalValue" 
    :placeholder="placeholder || '请选择'"
    style="width: 100%"
    clearable
  >
    <template v-if="options && options.length">
      <el-option 
        v-for="opt in options" 
        :key="opt.value" 
        :label="opt.label" 
        :value="opt.value" 
      />
    </template>
    <template v-else-if="enums && enums.length">
      <el-option 
        v-for="opt in enums" 
        :key="opt" 
        :label="opt" 
        :value="opt" 
      />
    </template>
  </el-select>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  modelValue?: any
  enums?: any[]
  options?: { label: string, value: any }[]
  placeholder?: string
}>()

const emit = defineEmits(['update:modelValue'])

const internalValue = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>
