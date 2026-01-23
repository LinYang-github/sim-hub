<template>
  <el-dialog v-model="visible" title="管理标签" width="400px">
    <el-select
      v-model="localTags"
      multiple
      filterable
      allow-create
      default-first-option
      placeholder="输入标签并按回车"
      style="width: 100%"
    >
      <el-option
        v-for="item in existingTags"
        :key="item"
        :label="item"
        :value="item"
      />
    </el-select>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="save" :loading="loading">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'

const props = defineProps<{
  modelValue: boolean
  tags: string[]
  existingTags: string[]
  loading: boolean
}>()

const emit = defineEmits(['update:modelValue', 'save'])

const localTags = ref<string[]>([])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

watch(() => props.tags, (newTags) => {
  localTags.value = [...newTags]
}, { immediate: true })

const save = () => {
  emit('save', localTags.value)
}
</script>
