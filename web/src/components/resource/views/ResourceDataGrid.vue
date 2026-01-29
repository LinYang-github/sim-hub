<template>
  <el-table 
    :data="resources" 
    style="width: 100%" 
    v-loading="loading" 
    class="premium-table grid-mode"
  >
    <!-- 1. Name -->
    <el-table-column key="name" label="资源名称" min-width="200">
      <template #default="scope">
        <div class="resource-info-cell clickable" @click="$emit('view-details', scope.row)">
          <div class="resource-icon">
            <el-icon>
              <component :is="icon || 'Connection'" />
            </el-icon>
          </div>
          <span class="resource-name" :title="scope.row.name">{{ scope.row.name }}</span>
        </div>
      </template>
    </el-table-column>

    <!-- 2. Dynamic Schema Columns -->
    <el-table-column 
        v-for="(propDef, key) in (schema?.properties || {})" 
        :key="'dyn_' + key"
        :label="propDef.description || key"
        :min-width="120"
        show-overflow-tooltip
    >
        <template #default="scope">
          <span>{{ formatSchemaValue(scope.row.latest_version?.meta_data?.[key], propDef) }}</span>
        </template>
    </el-table-column>

    <!-- 3. Version -->
    <el-table-column key="version" label="当前版本" width="100">
      <template #default="scope">
        <span class="version-text">{{ scope.row.latest_version?.semver || 'v' + (scope.row.latest_version?.version_num || 1) }}</span>
      </template>
    </el-table-column>

    <!-- 4. Date -->
    <el-table-column key="date" label="更新时间" width="140">
      <template #default="scope">
        <span class="meta-item">{{ formatDate(scope.row.created_at).split(' ')[0] }}</span>
      </template>
    </el-table-column>

    <!-- 5. Actions -->
    <el-table-column label="操作" width="120" fixed="right" align="center" header-align="center">
      <template #default="scope">
          <div class="op-actions">
            <!-- Primary Action: Details/View -->
            <el-tooltip content="详情/预览" placement="top">
              <el-button 
                circle 
                type="primary" 
                plain 
                size="small"
                @click="$emit('view-details', scope.row)"
              >
                <el-icon><ViewIcon /></el-icon>
              </el-button>
            </el-tooltip>

            <!-- Secondary Actions: Dropdown -->
            <el-dropdown trigger="click" popper-class="resource-popper" @command="(cmd) => handleCommand(cmd, scope.row)">
              <el-button circle size="small">
                <el-icon><MoreFilled /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="download">
                    <div class="menu-item-content">
                      <el-icon><Download /></el-icon>
                      <span>下载 JSON</span>
                    </div>
                  </el-dropdown-item>
                  
                  <el-dropdown-item command="tags">
                    <div class="menu-item-content">
                      <el-icon><PriceTag /></el-icon>
                      <span>编辑标签</span>
                    </div>
                  </el-dropdown-item>

                  <el-dropdown-item command="rename">
                    <div class="menu-item-content">
                      <el-icon><Edit /></el-icon>
                      <span>重命名</span>
                    </div>
                  </el-dropdown-item>

                  <el-dropdown-item command="move">
                    <div class="menu-item-content">
                      <el-icon><Promotion /></el-icon>
                      <span>移动目录</span>
                    </div>
                  </el-dropdown-item>
                  
                  <el-dropdown-item divided command="delete" class="danger-item">
                    <div class="menu-item-content">
                      <el-icon><Delete /></el-icon>
                      <span>删除资源</span>
                    </div>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import { 
  Download, MoreFilled, View as ViewIcon, 
  PriceTag, Edit, Promotion, Delete
} from '@element-plus/icons-vue'
import { formatDate } from '../../../core/utils/format'

const props = defineProps<{
  resources: any[]
  loading: boolean
  icon?: string
  schema?: any
}>()

const emit = defineEmits(['view-details', 'download', 'delete', 'rename', 'move', 'edit-tags'])

const handleCommand = (command: string, row: any) => {
  switch(command) {
    case 'download': emit('download', row); break;
    case 'tags': emit('edit-tags', row); break;
    case 'rename': emit('rename', row); break;
    case 'move': emit('move', row); break;
    case 'delete': emit('delete', row); break;
  }
}

const formatSchemaValue = (val: any, propDef: any) => {
  if (val === undefined || val === null) return '-'
  if (propDef.type === 'boolean') return val ? '是' : '否'
  if (Array.isArray(val)) return val.join(', ')
  if (typeof val === 'object') return JSON.stringify(val)
  return val
}
</script>

<style scoped lang="scss">
.premium-table {
  --el-table-header-bg-color: var(--el-fill-color-lighter);
  
  :deep(th.el-table__cell) {
    font-weight: 600;
    color: var(--el-text-color-primary);
    font-size: 13px;
    padding: 12px 0;
  }
}

.clickable {
  cursor: pointer;
  transition: opacity 0.2s;
  &:hover { opacity: 0.8; }
}

.resource-info-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.resource-icon {
  width: 32px;
  height: 32px;
  background: var(--el-color-primary-light-9);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-primary);
  font-size: 16px;
  flex-shrink: 0;
}

.resource-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
  font-size: 13px;
}

.version-text {
  font-size: 11px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  padding: 1px 6px;
  border-radius: 4px;
}

.meta-item {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.op-actions {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
}
</style>
