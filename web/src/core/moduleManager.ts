
import { App, shallowRef, Component } from 'vue'
import { Router } from 'vue-router'
import request from './utils/request'
import IframeContainer from '../components/resource/previewers/ExternalViewer.vue'
import { SimHubModule, SupportedView, CustomAction } from './types'

class ModuleManager {
  // 已配置（活跃）的模块列表 (使用 shallowRef 确保 UI 响应式同步)
  private activeModules = shallowRef<SimHubModule[]>([])
  
  public getActiveModules() {
    return this.activeModules
  }
  
  // 可用的代码实现（内部模块映射表）
  private implementations: Map<string, SimHubModule> = new Map()

  // 统一存储从后端拉取的原始配置项
  private configItems: SimHubModule[] = []
  
  // 视图注册表 (使用 shallowRef 确保 UI 可以响应注册变化)
  public viewRegistry = shallowRef<Map<string, SupportedView>>(new Map())

  // 动作注册表
  public actionRegistry = shallowRef<Map<string, CustomAction>>(new Map())

  // 预览组件注册表
  public viewerRegistry = shallowRef<Map<string, { key: string, label: string, path: string }>>(new Map())

  /**
   * 注册视图元数据
   */
  registerView(meta: SupportedView) {
      const newMap = new Map(this.viewRegistry.value)
      newMap.set(meta.key, meta)
      this.viewRegistry.value = newMap
      console.log(`[ViewRegistry] Registered: ${meta.key}`, meta)
  }

  /**
   * 注册自定义动作
   */
  registerAction(action: CustomAction) {
      const newMap = new Map(this.actionRegistry.value)
      newMap.set(action.key, action)
      this.actionRegistry.value = newMap
      console.log(`[ActionRegistry] Registered: ${action.key}`, action)
  }

  /**
   * 解析视图元数据（支持 key 查找）
   */
  resolveView(view: string | SupportedView): SupportedView {
      const key = typeof view === 'string' ? view : view.key
      const registered = this.viewRegistry.value.get(key)
      if (registered) {
          return typeof view === 'string' ? registered : { ...registered, ...view }
      }
      return typeof view === 'string' ? { key, label: key, icon: 'Document' } : view
  }

  /**
   * 解析动作元数据与处理器
   */
  resolveAction(action: string | CustomAction): CustomAction {
      const key = typeof action === 'string' ? action : action.key
      const registered = this.actionRegistry.value.get(key)
      if (registered) {
           return typeof action === 'string' ? registered : { ...registered, ...action }
      }
      return typeof action === 'string' ? { 
          key, 
          label: key, 
          icon: 'Promotion', 
          handler: () => console.warn('Action handler not found yet for', key) 
      } : action
  }

  /**
   * 注册预览组件
   */
  registerViewer(meta: { key: string, label: string, path: string }) {
      const newMap = new Map(this.viewerRegistry.value)
      newMap.set(meta.key, meta)
      this.viewerRegistry.value = newMap
      console.log(`[ViewerRegistry] Registered: ${meta.key}`)
  }

  /**
   * 解析预览组件路径
   */
  resolveViewer(viewer: string): string {
      const registered = this.viewerRegistry.value.get(viewer)
      if (registered) return registered.path
      return viewer
  }

  private handleSupportedViews(views: any): (string | SupportedView)[] {
    if (!Array.isArray(views)) return []
    return views.map((v: any) => typeof v === 'object' ? v.key : v)
  }

  private handleCustomActions(actions: any): (string | CustomAction)[] {
    if (!Array.isArray(actions)) return []
    return actions.map((a: any) => typeof a === 'object' ? a.key : a)
  }

  /**
   * 注册代码模块实现（内部）
   * 使该模块可以被 modules.json 中的配置激活
   */
  registerImplementation(module: SimHubModule) {
    this.implementations.set(module.key, module)
  }

  async loadConfig(url: string = '/api/v1/resource-types') {
    try {
      const finalUrl = url + (url.includes('?') ? '&' : '?') + 't=' + new Date().getTime()
      const rawItems = await request.get<any[]>(finalUrl) || []
      
      this.configItems = rawItems.map(item => {
          const meta = item.meta_data || {}
          
          // Auto-register Viewer if defined as object
          if (meta.viewer && typeof meta.viewer === 'object' && meta.viewer.key) {
               this.registerViewer(meta.viewer)
               // Flatten for prop usage
               meta.viewer = meta.viewer.key
          }

          // Auto-register Views if defined as objects
          if (meta.supported_views && Array.isArray(meta.supported_views)) {
              meta.supported_views.forEach((v: any) => {
                  if (typeof v === 'object' && v.key) {
                      this.registerView(v)
                  }
              })
          }

          // Auto-register Actions if defined as objects
          if (meta.custom_actions && Array.isArray(meta.custom_actions)) {
              meta.custom_actions.forEach((a: any) => {
                  console.log('[AutoRegister] Checking action:', a)
                  if (typeof a === 'object' && a.key) {
                      this.registerAction(a)
                  }
              })
          }

          return {
              key: item.type_key,
              label: item.type_name,
              typeName: item.type_name,
              icon: meta.icon,
              integrationMode: item.integration_mode || 'internal',
              uploadMode: item.upload_mode || 'online',
              enableScope: meta.enable_scope,
              categoryMode: item.category_mode || 'flat',
              viewer: typeof meta.viewer === 'string' ? meta.viewer : meta.viewer?.key,
              supportedViews: this.handleSupportedViews(meta.supported_views),
              customActions: this.handleCustomActions(meta.custom_actions),
              externalUrl: meta.external_url,
              devUrl: meta.dev_url,
              shortName: meta.short_name,
              example: meta.example
          }
      })
      
      const newActiveModules: SimHubModule[] = []

      this.configItems.forEach(item => {
        if (item.integrationMode === 'internal') {
            const impl = this.implementations.get(item.key)
            if (impl) {
                const merged = { ...impl, ...item }
                if (item.supportedViews && item.supportedViews.length > 0) {
                    merged.supportedViews = item.supportedViews
                }
                if (merged.routes) {
                    merged.routes.forEach(r => {
                        if (r.props && typeof r.props === 'object') {
                            if (merged.supportedViews) (r.props as any).supportedViews = merged.supportedViews
                            if (merged.customActions) (r.props as any).customActions = merged.customActions
                            if (merged.example) (r.props as any).example = merged.example
                        }
                    })
                }
                newActiveModules.push(merged)
            } else {
                const fallback: SimHubModule = {
                    ...item,
                    menu: [{
                        label: item.label || item.key,
                        path: `/res/${item.key}`,
                        icon: 'Box'
                    }],
                    routes: [{
                        path: `/res/${item.key}`,
                        component: () => import('../components/resource/ResourceList.vue'),
                        props: {
                          typeKey: item.key,
                          typeName: item.typeName || item.label || '资源',
                          shortName: item.shortName,
                          uploadMode: item.uploadMode || 'single',
                          accept: item.accept,
                          enableScope: item.enableScope,
                          categoryMode: item.categoryMode,
                          viewer: item.viewer,
                          icon: item.icon,
                          supportedViews: item.supportedViews,
                          example: item.example,
                          customActions: item.customActions
                        }
                    }]
                }
                newActiveModules.push(fallback)
            }
        } else {
            newActiveModules.push(item)
        }
      })
      this.activeModules.value = newActiveModules
      console.log(`已成功加载 ${this.activeModules.value.length} 个活跃模块`)
    } catch (e) {
      console.error("加载模块配置时发生异常", e)
    }
  }

  install(_app: App, router: Router) {
    this.activeModules.value.forEach(m => {
      if (m.routes) {
        m.routes.forEach(r => router.addRoute(r))
      }
      if (m.integrationMode === 'iframe') {
        const url = (import.meta.env.DEV && m.devUrl) ? m.devUrl : m.externalUrl
        if (url) {
          router.addRoute({
            path: `/ext/${m.key}`,
            component: IframeContainer,
            props: { url: url }
          })
        }
      }
    })
  }

  getMenus() {
    return this.activeModules.value.flatMap(m => {
      if (m.menu) {
          return m.menu.map(item => ({ 
              ...item, 
              label: m.label || item.label,
              icon: m.icon || item.icon 
          }))
      }
      const isExternal = m.integrationMode === 'iframe' || m.integrationMode === 'new-tab'
      if (isExternal && m.label) {
        const url = (import.meta.env.DEV && m.devUrl) ? m.devUrl : m.externalUrl
        const path = m.integrationMode === 'new-tab' ? url : `/ext/${m.key}`
        return [{
          label: m.label,
          path: path || '#',
          icon: m.icon
        }]
      }
      return []
    })
  }
}

export const moduleManager = new ModuleManager()
