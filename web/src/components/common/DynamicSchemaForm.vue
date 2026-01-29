<template>
  <el-form 
    ref="formRef" 
    :model="formData" 
    label-position="top"
    class="dynamic-schema-form"
    :rules="rules"
  >
    <el-row :gutter="24">
      <template v-for="(propDef, key) in schema.properties" :key="key">
        <el-col :span="getColumnSpan(propDef)">
          <el-form-item 
            :label="propDef.description || key" 
            :prop="String(key)"
          >
            <!-- Enum Select -->
            <el-select 
              v-if="propDef.enum" 
              v-model="formData[key]" 
              placeholder="请选择"
              style="width: 100%"
              clearable
            >
              <el-option 
                v-for="opt in propDef.enum" 
                :key="opt" 
                :label="opt" 
                :value="opt" 
              />
            </el-select>

            <!-- Number Input -->
            <el-input-number 
              v-else-if="propDef.type === 'number' || propDef.type === 'integer'"
              v-model="formData[key]"
              :placeholder="`请输入 ${propDef.description || key}`"
              style="width: 100%"
              controls-position="right"
            />

            <!-- Boolean Switch -->
            <el-switch 
              v-else-if="propDef.type === 'boolean'"
              v-model="formData[key]"
            />

            <!-- String Input (Default) -->
            <el-input 
              v-else 
              v-model="formData[key]" 
              :placeholder="`请输入 ${propDef.description || key}`"
            />
            
            <!-- Contextual Help (Only if different from label) -->
            <div v-if="propDef.description && propDef.description !== (propDef.description || key)" class="form-help-text">
              {{ propDef.description }}
            </div>
          </el-form-item>
        </el-col>
      </template>
    </el-row>
  </el-form>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import type { FormInstance } from 'element-plus'

const props = defineProps<{
  schema: any
  modelValue: any
}>()

const emit = defineEmits(['update:modelValue'])

const formData = ref<any>({ ...props.modelValue })
const formRef = ref<FormInstance>()

// Watch for internal changes and emit up
watch(formData, (val) => {
  emit('update:modelValue', val)
}, { deep: true })

// Generate Rules from Schema
const rules = computed(() => {
  const r: any = {}
  const required = props.schema.required || []
  
  if (props.schema.properties) {
    Object.keys(props.schema.properties).forEach(key => {
      if (required.includes(key)) {
        r[key] = [{ required: true, message: '此项必填', trigger: 'change' }]
      }
    })
  }
  return r
})

// Helper to determine column span
const getColumnSpan = (prop: any) => {
  // Complex types or explicitly long fields take full width
  if (prop.type === 'object' || prop.type === 'array') return 24
  if (prop.format === 'textarea') return 24
  
  // Default to half width for standard inputs
  return 12
}

// Expose validate method
const validate = async () => {
  if (!formRef.value) return false
  return await formRef.value.validate()
}

defineExpose({ validate })
</script>

<style scoped>
.form-help-text {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 4px;
}
:deep(.el-form-item__label) {
  font-weight: 500;
  color: var(--el-text-color-regular);
}
</style>
