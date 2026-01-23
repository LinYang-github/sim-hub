import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import App from './App.vue'
import { createRouter, createWebHistory } from 'vue-router'
import Workstation from './components/Workstation.vue'
import { moduleManager } from './core/moduleManager'
import scenarioModule from './modules/scenario'

// 注册内部模块实现 (此处仅注册有特殊定制需求的模块)
moduleManager.registerImplementation(scenarioModule)
// model_glb 将通过 modules.json 中的元数据自动使用 ResourceList 兜底

const routes = [
  { path: '/', component: Workstation },
]

const initApp = async () => {
  const app = createApp(App)
  
  app.use(ElementPlus, {
    locale: zhCn,
  })

  // 1. 加载模块配置（异步）
  await moduleManager.loadConfig('/modules.json')
  
  // 2. 初始化 Router
  const router = createRouter({
    history: createWebHistory(),
    routes,
  })

  // 3. 安装已加载模块的路由
  moduleManager.install(app, router)
  
  app.use(router)
  app.mount('#app')
}

initApp()
