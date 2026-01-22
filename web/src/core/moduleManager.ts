import { App } from 'vue'
import { Router } from 'vue-router'
import { SimHubModule } from './types'
import IframeContainer from './views/IframeContainer.vue'

class ModuleManager {
  private modules: SimHubModule[] = []

  register(module: SimHubModule) {
    this.modules.push(module)
  }

  install(app: App, router: Router) {
    this.modules.forEach(m => {
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
    return this.modules.flatMap(m => {
      // Case 1: Explicit Menu (Internal)
      if (m.menu) return m.menu

      // Case 2: Generated Menu (External)
      if (m.externalUrl && m.label) {
        const path = m.integrationMode === 'new-tab' ? m.externalUrl : `/ext/${m.key}`
        return [{
          label: m.label,
          path: path,
          // TODO: handle new-tab click logic in menu if needed
        }]
      }
      return []
    })
  }
}

export const moduleManager = new ModuleManager()
