import { RouteRecordRaw } from 'vue-router'

export interface MenuOption {
  label: string
  path: string
  icon?: string
}

export interface SimHubModule {
  key: string
  // Internal Module Props
  routes?: RouteRecordRaw[]
  menu?: MenuOption[]
  
  // External Integration Props
  label?: string       // Menu label for external
  externalUrl?: string // Target URL
  integrationMode?: 'iframe' | 'new-tab'
}

