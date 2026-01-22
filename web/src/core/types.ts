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
  
  // External Integration Props
  label?: string       // Menu label for external
  externalUrl?: string // Target URL
  integrationMode?: 'iframe' | 'new-tab' | 'internal'
}

