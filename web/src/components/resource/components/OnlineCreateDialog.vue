<template>
  <el-dialog
    v-model="visible"
    title="新建资源"
    width="600px"
    destroy-on-close
    :close-on-click-modal="false"
  >
    <div class="create-dialog-content">
      <!-- 1. General Info -->
      <div class="section-title">基础信息</div>
      <el-form 
        :model="baseForm" 
        label-position="top" 
        :rules="baseRules" 
        ref="baseFormRef"
      >
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="资源名称" prop="name">
              <el-input v-model="baseForm.name" placeholder="例如：红方通信干扰策略V1" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="初始版本号" prop="semver">
              <el-input v-model="baseForm.semver" placeholder="v1.0.0" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="所属分类">
              <el-tree-select
                v-if="categoryNodes.length > 0"
                v-model="baseForm.category_id"
                :data="categoryNodes"
                check-strictly
                placeholder="选择分类（可选）"
                style="width: 100%"
                clearable
              />
              <span v-else class="text-gray">暂无分类</span>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <!-- 2. Dynamic Schema Form -->
      <div class="section-title">业务参数配置</div>
      <div class="schema-form-wrapper" v-if="schema">
        <DynamicSchemaForm
          ref="dynamicFormRef"
          v-model="payloadData"
          :schema="schema"
        />
      </div>
      <el-empty v-else description="无法加载配置模版" :image-size="60" />

    </div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          立即创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import DynamicSchemaForm from '../../common/DynamicSchemaForm.vue'
import request from '../../../core/utils/request'
import { ElMessage } from 'element-plus'

const props = defineProps<{
  typeKey: string
  categoryNodes: any[]
  typeName: string
  schema?: any // Current type resource schema def
}>()

const emit = defineEmits(['success'])
const visible = defineModel<boolean>('modelValue')

const submitting = ref(false)
const baseFormRef = ref()
const dynamicFormRef = ref()

const baseForm = ref({
  name: '',
  semver: 'v1.0.0',
  category_id: ''
})

const payloadData = ref<any>({})

const baseRules = {
  name: [{ required: true, message: '请输入资源名称', trigger: 'blur' }],
  semver: [{ required: true, message: '请输入版本号', trigger: 'blur' }]
}

const handleSubmit = async () => {
  // 1. Validate Base
  if (!baseFormRef.value) return
  await baseFormRef.value.validate()

  // 2. Validate Dynamic
  if (dynamicFormRef.value) {
    try {
      await dynamicFormRef.value.validate()
    } catch (e) {
      return
    }
  }

  submitting.value = true
  try {
    // 3. Submit
    await request.post('/api/v1/resources/create', {
      type_key: props.typeKey,
      name: baseForm.value.name,
      semver: baseForm.value.semver,
      category_id: baseForm.value.category_id || undefined,
      data: payloadData.value, // The JSON Payload
      scope: 'PUBLIC' // Default for online resources (often shared rules)
    })
    
    ElMessage.success('创建成功')
    visible.value = false
    emit('success')
    
    // Reset
    baseForm.value = { name: '', semver: 'v1.0.0', category_id: '' }
    payloadData.value = {}
    
  } catch (e: any) {
    // Error handled by interceptor
  } finally {
    submitting.value = false
  }
}

</script>

<style scoped>
.section-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 20px;
  margin-top: 24px;
  display: flex;
  align-items: center;
}
.section-title::before {
    content: '';
    width: 4px;
    height: 16px;
    background: var(--el-color-primary);
    margin-right: 8px;
    border-radius: 2px;
}
.section-title:first-child {
  margin-top: 0;
}
.schema-form-wrapper {
  /* Removed bg color for seamless integration */
  padding: 0; 
}
.text-gray {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>
