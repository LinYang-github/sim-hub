import { App, shallowRef } from 'vue'
import { Router } from 'vue-router'
import { SimHubModule } from './types'
import IframeContainer from './views/IframeContainer.vue'
import request from './utils/request'

class ModuleManager {
  // 已配置（活跃）的模块列表 (使用 shallowRef 确保 UI 响应式同步)
  private activeModules = shallowRef<SimHubModule[]>([])
  
  // 可用的代码实现（内部模块映射表）
  private implementations: Map<string, SimHubModule> = new Map()

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

  async loadConfig(url: string) {
    try {
      // Add timestamp to bypass cache during dev
      const finalUrl = url + (url.includes('?') ? '&' : '?') + 't=' + new Date().getTime()
      const configItems = await request.get<SimHubModule[]>(finalUrl) as unknown as SimHubModule[]
      if (!configItems) return
      
      const newActiveModules: SimHubModule[] = []

      configItems.forEach(item => {
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
                          uploadMode: item.uploadMode || 'single',
                          accept: item.accept,
                          enableScope: item.enableScope,
                          viewer: item.viewer,
                          icon: item.icon,
                          supportedViews: item.supportedViews,
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
