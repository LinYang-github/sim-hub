/**
 * 资源状态常量
 */
export const RESOURCE_STATE = {
  ACTIVE: 'ACTIVE',
  READY: 'READY',
  PROCESSING: 'PROCESSING',
  PENDING: 'PENDING',
  FAILED: 'FAILED',
} as const

/**
 * 资源可见性范围
 */
export const RESOURCE_SCOPE = {
  ALL: 'ALL',
  PUBLIC: 'PUBLIC',
  PRIVATE: 'PRIVATE',
} as const

/**
 * 资源状态文本映射
 */
export const RESOURCE_STATUS_TEXT: Record<string, string> = {
  [RESOURCE_STATE.ACTIVE]: '已就绪',
  [RESOURCE_STATE.READY]: '就绪',
  [RESOURCE_STATE.PROCESSING]: '处理中',
  [RESOURCE_STATE.PENDING]: '排队中',
  [RESOURCE_STATE.FAILED]: '处理失败',
}

/**
 * 权限分段选择器配置
 */
export const SCOPE_OPTIONS = [
  { label: '全部', val: RESOURCE_SCOPE.ALL },
  { label: '公共', val: RESOURCE_SCOPE.PUBLIC },
  { label: '我的', val: RESOURCE_SCOPE.PRIVATE },
] as const

/**
 * 根分类 ID
 */
export const ROOT_CATEGORY_ID = 'all'

/**
 * 默认管理员 ID
 */
export const DEFAULT_ADMIN_ID = 'admin'

