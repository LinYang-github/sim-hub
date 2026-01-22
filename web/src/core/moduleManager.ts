import { App } from 'vue'
import { Router } from 'vue-router'
import { SimHubModule } from './types'
import IframeContainer from './views/IframeContainer.vue'

class ModuleManager {
  // Configured (Active) Modules
  private activeModules: SimHubModule[] = []
  
  // Available Code Implementations (Internal Modules)
  private implementations: Map<string, SimHubModule> = new Map()

  /**
   * Register a code module (Internal)
   * This makes the module available to be activated by modules.json
   */
  registerImplementation(module: SimHubModule) {
    this.implementations.set(module.key, module)
  }

  install(app: App, router: Router) {
    this.activeModules.forEach(m => {
      // 1. Internal Routes
      if (m.routes) {
        m.routes.forEach(r => router.addRoute(r))
      }

      // 2. External Routes (Auto-generate if iframe mode)
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
      // Case 1: Explicit Menu (Internal) - Override label if config exists
      if (m.menu) {
          // If config provided a label, override the first menu item (Simple override logic)
          if (m.label && m.menu.length > 0) {
              return m.menu.map(item => ({ ...item, label: m.label }))
          }
          return m.menu
      }

      // Case 2: Generated Menu (External)
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
        console.warn(`Failed to load module config: ${response.statusText}`)
        return
      }
      const configItems: SimHubModule[] = await response.json()
      
      this.activeModules = [] // Reset active modules

      configItems.forEach(item => {
        if (item.integrationMode === 'internal') {
            // Activate Internal Module
            const impl = this.implementations.get(item.key)
            if (impl) {
                // Merge config (label) into implementation
                const merged = { ...impl, ...item } 
                // Note: We keep impl routes/menu but override top-level props like label
                this.activeModules.push(merged)
            } else {
                console.warn(`Internal module implementation '${item.key}' not found.`)
            }
        } else {
            // Activate External Module
            this.activeModules.push(item)
        }
      })
      console.log(`Loaded ${this.activeModules.length} active modules`)
    } catch (e) {
      console.error("Failed to load module configuration", e)
    }
  }
}

export const moduleManager = new ModuleManager()
