<template>
  <el-dialog v-model="visible" title="确认上传信息" width="550px" class="premium-dialog">
    <el-form :model="form" label-position="top">
      <el-form-item label="资源名称" v-if="data">
         <el-input v-model="data.displayName" disabled />
      </el-form-item>
      
      <el-form-item label="语义化版本 (SemVer)" required>
        <el-input v-model="form.semver" placeholder="例如: v1.0.0" />
        <div class="input-tip">建议遵循语义化版本规范，方便后续依赖追踪。</div>
      </el-form-item>
      
      <el-form-item label="资源依赖">
        <el-select
          v-model="form.dependencies"
          multiple
          filterable
          remote
          reserve-keyword
          placeholder="搜索并选择关联资源"
          :remote-method="(q) => $emit('search-dependency', q)"
          :loading="searchLoading"
          style="width: 100%"
          value-key="id"
        >
          <el-option
            v-for="item in searchResults"
            :key="item.id"
            :label="item.name"
            :value="item"
          >
            <div class="search-option">
              <span class="option-name">{{ item.name }}</span>
              <span class="option-type">{{ item.type }}</span>
            </div>
          </el-option>
        </el-select>
        <div class="input-tip">你可以搜索并关联目前系统中已有的其他资源。</div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" @click="$emit('confirm')" :loading="loading">
          开始上传
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Resource } from '../../../core/types/resource'
import type { PendingUploadData, UploadFormState } from '../composables/useUpload'

const props = defineProps<{
  modelValue: boolean
  data: PendingUploadData | null
  form: UploadFormState
  loading: boolean
  searchResults: Resource[]
  searchLoading: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [val: boolean]
  'confirm': []
  'search-dependency': [query: string]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})
</script>

<style scoped lang="scss">
.input-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
  margin-top: 4px;
}

.search-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  
  .option-name {
    font-weight: 500;
  }
  
  .option-type {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    background: var(--el-fill-color-light);
    padding: 0 4px;
    border-radius: 4px;
  }
}
</style>
