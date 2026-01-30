import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import { createRouter, createWebHistory } from 'vue-router'
import Workstation from './components/Workstation.vue'
import { moduleManager } from './core/moduleManager'
import scenarioModule from './modules/scenario'
import VNetworkGraph from "v-network-graph"
import "v-network-graph/lib/style.css"

import { registerStandardViews } from './core/registerStandardViews'

// 注册标准视图
registerStandardViews()

// 暴露全局 API 给外部应用 (demo-view 等)
;(window as any).SimHub = {
    registerView: (meta: any) => moduleManager.registerView(meta),
    registerViewer: (meta: any) => moduleManager.registerViewer(meta)
}

// 注册内部模块实现 (此处仅注册有特殊定制需求的模块)
moduleManager.registerImplementation(scenarioModule)
// model_glb 将通过 modules.json 中的元数据自动使用 ResourceList 兜底

// 动态插件发现机制：根据活跃模块的配置自动加载外部视图注册脚本
const loadExternalPlugins = () => {
  moduleManager.getActiveModules().value.forEach(m => {
    // 只有在开发模式或显式指定了 devUrl 时才尝试加载开发中的插件脚本
    const baseUrl = (import.meta.env.DEV && m.devUrl) ? m.devUrl : m.externalUrl
    if (baseUrl) {
      const script = document.createElement('script')
      // 约定：如果一个外部应用支持注册到主应用，其入口脚本必须位于 /src/main.ts (Vite Dev)
      script.src = `${baseUrl}/src/main.ts`
      script.type = 'module'
      document.head.appendChild(script)
      console.log(`[PluginLoader] Loading plugin for module: ${m.key} from ${script.src}`)
    }
  })
}

const routes = [
  { path: '/', component: Workstation },
  { path: '/res/map_terrain', redirect: '/res/map_service' },
]

const initApp = async () => {
  const app = createApp(App)
  
  app.use(ElementPlus, {
    locale: zhCn,
  })
  app.use(VNetworkGraph)

  for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
  }

  // 1. 加载模块配置（异步）
  await moduleManager.loadConfig()
  
  // 2. 初始化 Router
  const router = createRouter({
    history: createWebHistory(),
    routes,
  })

  // 3. 安装已加载模块的路由
  moduleManager.install(app, router)
  
  // 4. 加载插件
  loadExternalPlugins()
  
  app.use(router)
  app.mount('#app')
}

initApp()
