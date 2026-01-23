import { RouteRecordRaw } from 'vue-router'
import { Component } from 'vue'

export interface MenuOption {
  label: string
  path: string
  icon?: string | Component
}

export interface SimHubModule {
  key: string
  // Internal Module Props
  routes?: RouteRecordRaw[]
  menu?: MenuOption[]
  // Generic Resource Props (used if no custom routes provided)
  typeName?: string
  uploadMode?: 'single' | 'folder-zip'
  accept?: string
  enableScope?: boolean // 是否开启作用域（我的/公共）管理

  // External Integration Props
  label?: string       // Menu label for external
  externalUrl?: string // 生产环境相对路径
  devUrl?: string      // 仅开发环境开发端口
  integrationMode?: 'iframe' | 'new-tab' | 'internal'
}

