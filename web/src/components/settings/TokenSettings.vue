<template>
  <div class="token-settings">
    <div class="header-section">
      <div class="title-group">
        <h2 class="title">个人访问令牌</h2>
        <p class="subtitle">用于 SDK 集成与 Open API 自动化的身份凭证。令牌代表您的操作权限，请妥善保管。</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="showCreateDialog = true">生成新令牌</el-button>
    </div>

    <el-card class="token-card" shadow="never">
      <el-table :data="tokens" v-loading="loading" style="width: 100%">
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column label="最后使用时间" width="200">
          <template #default="{ row }">
            <span v-if="row.last_used_at">{{ formatDate(row.last_used_at) }}</span>
            <span v-else class="text-secondary">从未被使用</span>
          </template>
        </el-table-column>
        <el-table-column label="到期时间" width="200">
          <template #default="{ row }">
            <span v-if="row.expires_at">{{ formatDate(row.expires_at) }}</span>
            <span v-else>永不过期</span>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="200">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-popconfirm title="确定要撤销此令牌吗？撤销后 SDK 将无法使用该令牌访问。" @confirm="handleRevoke(row.id)">
              <template #reference>
                <el-button type="danger" link>撤销</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Create Dialog -->
    <el-dialog v-model="showCreateDialog" title="生成新令牌" width="500px">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="令牌名称" required>
          <el-input v-model="createForm.name" placeholder="例如：SimEngine-Integration" />
        </el-form-item>
        <el-form-item label="有效期（天）">
          <el-select v-model="createForm.expire_days" placeholder="选择有效期">
            <el-option label="7 天" :value="7" />
            <el-option label="30 天" :value="30" />
            <el-option label="90 天" :value="90" />
            <el-option label="180 天" :value="180" />
            <el-option label="永不过期" :value="0" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" @click="handleCreate" :loading="creating">生成</el-button>
      </template>
    </el-dialog>

    <!-- Success Dialog (Show Token Once) -->
    <el-dialog v-model="showSuccessDialog" title="令牌生成成功" width="550px" :close-on-click-modal="false" :show-close="false">
      <el-alert
        title="重要提示：请务必立即复制并保存您的令牌。出于安全考虑，退出此页面后您将无法再次看到完整的令牌内容。"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 20px"
      />
      
      <div class="token-display-box">
        <code class="raw-token">{{ newToken?.token }}</code>
        <el-button type="primary" link :icon="CopyDocument" @click="copyToken">复制</el-button>
      </div>

      <template #footer>
        <el-button type="primary" @click="closeSuccessDialog">我已保存并确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import axios from 'axios'

const tokens = ref([])
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)
const showSuccessDialog = ref(false)
const newToken = ref<{ token: string } | null>(null)

const createForm = ref({
  name: '',
  expire_days: 90
})

const fetchTokens = async () => {
  loading.value = true
  try {
    const res = await axios.get('/api/v1/auth/tokens?user_id=admin')
    tokens.value = res.data || []
  } catch (err) {
    ElMessage.error('无法获取令牌列表')
  } finally {
    loading.value = false
  }
}

const handleCreate = async () => {
  if (!createForm.value.name) {
    ElMessage.warning('请输入令牌名称')
    return
  }
  creating.value = true
  try {
    const res = await axios.post('/api/v1/auth/tokens', {
      user_id: 'admin',
      ...createForm.value
    })
    newToken.value = res.data
    showCreateDialog.value = false
    showSuccessDialog.value = true
    fetchTokens()
  } catch (err) {
    ElMessage.error('令牌生成失败')
  } finally {
    creating.value = false
  }
}

const handleRevoke = async (id: string) => {
  try {
    await axios.delete(`/api/v1/auth/tokens/${id}?user_id=admin`)
    ElMessage.success('令牌已撤销')
    fetchTokens()
  } catch (err) {
    ElMessage.error('撤销失败')
  }
}

const copyToken = () => {
  if (newToken.value) {
    navigator.clipboard.writeText(newToken.value.token)
    ElMessage.success('已复制到剪贴板')
  }
}

const closeSuccessDialog = () => {
  showSuccessDialog.value = false
  newToken.value = null
  createForm.value.name = ''
}

const formatDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(fetchTokens)
</script>

<style scoped lang="scss">
.token-settings {
  max-width: 1000px;
  margin: 0 auto;
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 30px;

  .title {
    margin: 0 0 8px 0;
    font-size: 24px;
    font-weight: 600;
  }
  
  .subtitle {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }
}

.token-card {
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
}

.token-display-box {
  background: var(--el-fill-color-darker);
  padding: 16px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 1px solid var(--el-border-color);

  .raw-token {
    font-family: 'Fira Code', 'Roboto Mono', monospace;
    font-size: 13px;
    color: var(--el-color-primary);
    word-break: break-all;
    user-select: all;
  }
}

.text-secondary {
  color: var(--el-text-color-secondary);
  font-style: italic;
  font-size: 13px;
}
</style>
