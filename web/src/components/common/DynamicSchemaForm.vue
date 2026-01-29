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
            <!-- 1. Priority: Named Slot (key) -->
            <slot :name="key" :model="formData" :prop="propDef">
                
                <!-- 2. Priority: Custom Component via x-component -->
                <component 
                  v-if="propDef['x-component'] && getCustomComponent(propDef['x-component'])"
                  :is="getCustomComponent(propDef['x-component'])"
                  v-model="formData[key]"
                  v-bind="propDef['x-props'] || {}"
                  :prop-def="propDef"
                />

                <!-- 3. Fallback: Standard Types -->
                
                <!-- Enum Select -->
                <SelectEditor 
                  v-else-if="propDef.enum || propDef['x-enum-options']" 
                  v-model="formData[key]" 
                  :enums="propDef.enum"
                  :options="propDef['x-enum-options']"
                  v-bind="propDef['x-props']"
                />

                <!-- Number Input -->
                <NumberEditor 
                  v-else-if="propDef.type === 'number' || propDef.type === 'integer'"
                  v-model="formData[key]"
                  :placeholder="`请输入 ${propDef.description || key}`"
                  :min="propDef.minimum"
                  :max="propDef.maximum"
                  :step="propDef.step"
                  :precision="propDef.precision !== undefined ? propDef.precision : (propDef.type === 'integer' ? 0 : undefined)"
                  v-bind="propDef['x-props']"
                />

                <!-- Boolean Switch -->
                <SwitchEditor 
                  v-else-if="propDef.type === 'boolean'"
                  v-model="formData[key]"
                  v-bind="propDef['x-props']"
                />

                <!-- String Input (Default) -->
                <TextEditor 
                  v-else 
                  v-model="formData[key]" 
                  :type="propDef.format === 'textarea' ? 'textarea' : 'text'"
                  :placeholder="`请输入 ${propDef.description || key}`"
                  v-bind="propDef['x-props']"
                />
            </slot>

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
import SelectEditor from './editors/SelectEditor.vue'
import NumberEditor from './editors/NumberEditor.vue'
import SwitchEditor from './editors/SwitchEditor.vue'
import TextEditor from './editors/TextEditor.vue'

const props = defineProps<{
  schema: any
  modelValue: any
  customComponentsMap?: Record<string, any>
}>()

// Internal map of supported custom components
const builtinCustomComponents: Record<string, any> = {
    // Add built-in custom editors here if needed, e.g. ColorPicker, MarkdownEditor
}

const getCustomComponent = (name: string) => {
    if (props.customComponentsMap && props.customComponentsMap[name]) {
        return props.customComponentsMap[name]
    }
    return builtinCustomComponents[name]
}

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
