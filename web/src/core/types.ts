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
  
  // Generic Resource Props
  typeName?: string
  uploadMode?: 'single' | 'folder-zip'
  accept?: string
  enableScope?: boolean

  // Integration & UI Props
  label?: string
  icon?: string | Component
  externalUrl?: string 
  devUrl?: string      
  integrationMode?: 'iframe' | 'new-tab' | 'internal'
}
