<template>
  <el-drawer v-model="visible" title="版本历史与回溯" size="500px">
    <div v-loading="loading" class="history-content">
      <el-timeline>
        <el-timeline-item
          v-for="v in versionHistory"
          :key="v.id"
          :timestamp="formatDate(v.created_at)"
          placement="top"
        >
          <el-card class="history-card">
            <div class="history-info">
              <div class="history-main">
                <span class="history-ver">{{ v.semver || 'v' + v.version_num }}</span>
                <el-tag size="small" :type="v.state === 'ACTIVE' ? 'success' : 'info'">{{ statusMap[v.state] || v.state }}</el-tag>
                <el-tag v-if="v.id === currentVersionId" size="small" effect="dark">当前</el-tag>
              </div>
              <div class="history-actions">
                <el-button link type="primary" @click="$emit('download', v.download_url)">下载</el-button>
                <el-button 
                  v-if="v.id !== currentVersionId" 
                  link 
                  type="warning" 
                  @click="$emit('rollback', v)"
                >
                  设为主版本
                </el-button>
              </div>
            </div>
          </el-card>
        </el-timeline-item>
      </el-timeline>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  modelValue: boolean
  versionHistory: any[]
  loading: boolean
  currentVersionId?: string
  statusMap: Record<string, string>
}>()

const emit = defineEmits(['update:modelValue', 'download', 'rollback'])

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const formatDate = (dateString: string) => {
  if (!dateString) return ''
  const date = new Date(dateString)
  const pad = (num: number) => num.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}
</script>

<style scoped lang="scss">
.history-content {
  padding: 0 20px;
}

.history-card {
  margin-bottom: 4px;
  :deep(.el-card__body) {
    padding: 12px;
  }
}

.history-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.history-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

.history-ver {
  font-weight: 600;
  font-size: 14px;
}

.history-actions {
  display: flex;
  gap: 4px;
}
</style>
