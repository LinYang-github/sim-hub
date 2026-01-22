import { App } from 'vue'
import { Router } from 'vue-router'
import { SimHubModule } from './types'
import IframeContainer from './views/IframeContainer.vue'

class ModuleManager {
  // 已配置（活跃）的模块列表
  private activeModules: SimHubModule[] = []
  
  // 可用的代码实现（内部模块映射表）
  private implementations: Map<string, SimHubModule> = new Map()

  /**
   * 注册代码模块实现（内部）
   * 使该模块可以被 modules.json 中的配置激活
   */
  registerImplementation(module: SimHubModule) {
    this.implementations.set(module.key, module)
  }

  install(app: App, router: Router) {
    this.activeModules.forEach(m => {
      // 1. 内部路由注册
      if (m.routes) {
        m.routes.forEach(r => router.addRoute(r))
      }

      // 2. 外部路由注册 (若为 iframe 模式则自动生成容器路由)
      if (m.externalUrl && m.integrationMode === 'iframe') {
        router.addRoute({
          path: `/ext/${m.key}`,
          component: IframeContainer,
          props: { url: m.externalUrl }
        })
      }
    })
  }

  getMenus() {
    return this.activeModules.flatMap(m => {
      // 情况 1: 显式菜单（内部模块）- 若配置中有 label 则进行覆盖
      if (m.menu) {
          // 若配置文件提供了 label，则覆盖菜单项的显示文本（简易覆盖逻辑）
          if (m.label && m.menu.length > 0) {
              return m.menu.map(item => ({ ...item, label: m.label }))
          }
          return m.menu
      }

      // 情况 2: 生成的菜单（外部模块）
      if (m.externalUrl && m.label) {
        const path = m.integrationMode === 'new-tab' ? m.externalUrl : `/ext/${m.key}`
        return [{
          label: m.label,
          path: path,
        }]
      }
      return []
    })
  }

  async loadConfig(url: string) {
    try {
      const response = await fetch(url)
      if (!response.ok) {
        console.warn(`加载模块配置失败: ${response.statusText}`)
        return
      }
      const configItems: SimHubModule[] = await response.json()
      
      this.activeModules = [] // 重置活跃模块列表

      configItems.forEach(item => {
        if (item.integrationMode === 'internal') {
            // 激活内部模块实现
            const impl = this.implementations.get(item.key)
            if (impl) {
                // 将配置项（如 label）合并到代码实现中
                const merged = { ...impl, ...item } 
                // 注意：保留实现中的 routes/menu，但覆盖 label 等顶层属性
                this.activeModules.push(merged)
            } else {
                console.warn(`未找到内部模块 '${item.key}' 的代码实现。`)
            }
        } else {
            // 激活外部模块
            this.activeModules.push(item)
        }
      })
      console.log(`已成功加载 ${this.activeModules.length} 个活跃模块`)
    } catch (e) {
      console.error("加载模块配置时发生异常", e)
    }
  }
}

export const moduleManager = new ModuleManager()
