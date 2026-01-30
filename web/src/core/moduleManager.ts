import { App, shallowRef } from 'vue'
import { Router } from 'vue-router'
import { SimHubModule } from './types'
import IframeContainer from './views/IframeContainer.vue'
import request from './utils/request'
// 移除硬编码的视图注册，改为动态注册
// 移除硬编码的视图注册，改为动态注册
import { SupportedView, CustomAction } from './types'


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

  /**
   * 注册视图元数据
   */
  registerView(meta: SupportedView) {
      const newMap = new Map(this.viewRegistry.value)
      newMap.set(meta.key, meta)
      this.viewRegistry.value = newMap
      console.log(`[ViewRegistry] Registered: ${meta.key}`)
  }

  /**
   * 注册自定义动作
   */
  registerAction(action: CustomAction) {
      const newMap = new Map(this.actionRegistry.value)
      newMap.set(action.key, action)
      this.actionRegistry.value = newMap
  }

  /**
   * 解析视图元数据（支持 key 查找）
   */
  resolveView(view: string | SupportedView): SupportedView {
      if (typeof view === 'string') {
          return this.viewRegistry.value.get(view) || { key: view, label: view, icon: 'Document' }
      }
      return view
  }

  /**
   * 解析动作元数据与处理器
   */
  resolveAction(action: string | CustomAction): CustomAction {
      if (typeof action === 'string') {
          return this.actionRegistry.value.get(action) || { 
              key: action, 
              label: action, 
              icon: 'Promotion', 
              handler: () => console.warn('Action handler not found yet for', action) 
          }
      }
      return action
  }

  private handleSupportedViews(views: any) {
      if (!views || !Array.isArray(views)) return views
      return views.map(v => {
          if (typeof v === 'string') {
              const meta = this.viewRegistry.value.get(v)
              if (meta) return meta
              return { label: v, icon: 'Document', key: v }
          }
          return v
      })
  }

  private handleCustomActions(actions: any) {
      if (!actions || !Array.isArray(actions)) return actions
      return actions.map(a => {
          if (typeof a === 'string') {
               const action = this.actionRegistry.value.get(a)
               if (action) return action
               return { key: a, label: a, icon: 'Promotion', handler: () => console.warn('Action handler not found for', a) }
          }
           // If object, try to enrich from registry if handler is missing or string
          const registered = this.actionRegistry.value.get(a.key)
          if (registered) {
              return { ...registered, ...a } // Allow override
          }
          return a
      })
  }

  /**
   * 注册代码模块实现（内部）
   * 使该模块可以被 modules.json 中的配置激活
   */
  registerImplementation(module: SimHubModule) {
    this.implementations.set(module.key, module)
  }

  install(_app: App, router: Router) {
    this.activeModules.value.forEach(m => {
      // 1. 内部路由注册
      if (m.routes) {
        m.routes.forEach(r => router.addRoute(r))
      }

      // 2. 外部路由注册 (若为 iframe 模式则自动生成容器路由)
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
      // 情况 1: 显式菜单（内部模块）- 若配置中有 label 或 icon 则进行覆盖
      if (m.menu) {
          return m.menu.map(item => ({ 
              ...item, 
              label: m.label || item.label,
              icon: m.icon || item.icon 
          }))
      }

      // 情况 2: 生成的菜单（外部模块）
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

  async loadConfig(url: string = '/api/v1/resource-types') {
    try {
      // Add timestamp to bypass cache during dev
      const finalUrl = url + (url.includes('?') ? '&' : '?') + 't=' + new Date().getTime()
      const rawItems = await request.get<any[]>(finalUrl) as any[]
      
      // Map backend snake_case to frontend camelCase
      this.configItems = rawItems.map(item => {
          const meta = item.meta_data || {}
          return {
              key: item.type_key,
              label: item.type_name,
              typeName: item.type_name,
              icon: meta.icon,
              integrationMode: item.integration_mode || 'internal',
              uploadMode: item.upload_mode || 'online',
              enableScope: meta.enable_scope,
              categoryMode: item.category_mode || 'flat',
              viewer: meta.viewer,
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
            // 激活内部模块实现
            const impl = this.implementations.get(item.key)
            if (impl) {
                // 将配置项（如 label）合并到代码实现中
                const merged = { ...impl, ...item }
                // Use supportedViews from config if available, otherwise from impl
                if (item.supportedViews) {
                    merged.supportedViews = item.supportedViews
                }
                
                // If the module uses the generic ResourceList component internally (e.g. via routes),
                // we need to make sure the props in that route definition are updated.
                if (merged.routes) {
                    merged.routes.forEach(r => {
                        if (r.props && typeof r.props === 'object') {
                            // Inject supportedViews into the route props
                            if (merged.supportedViews) {
                                (r.props as any).supportedViews = merged.supportedViews
                            }
                            // Inject customActions into the route props
                             if (merged.customActions) {
                                (r.props as any).customActions = merged.customActions
                             }
                             // Inject example into the route props
                             if (merged.example) {
                                (r.props as any).example = merged.example
                             }
                        }
                    })
                }
                newActiveModules.push(merged)
            } else {
                // 兜底逻辑：如果没有特定实现，自动使用通用的 ResourceList
                console.log(`未找到内容模块 '${item.key}' 的特定实现，使用通用 ResourceList 兜底。`)
                const fallback: SimHubModule = {
                    ...item,
                    menu: [
                      {
                        label: item.label || item.key,
                        path: `/res/${item.key}`,
                        icon: 'Box' // 默认图标
                      }
                    ],
                    routes: [
                      {
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
                          // Pass customActions from config
                          customActions: item.customActions
                        }
                      }
                    ]
                }
                newActiveModules.push(fallback)
            }
        } else {
            // 激活外部模块
            newActiveModules.push(item)
        }
      })
      this.activeModules.value = newActiveModules
      console.log(`已成功加载 ${this.activeModules.value.length} 个活跃模块`)
    } catch (e) {
      console.error("加载模块配置时发生异常", e)
    }
  }
}

export const moduleManager = new ModuleManager()
