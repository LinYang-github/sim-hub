import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import App from './App.vue'
import { createRouter, createWebHistory } from 'vue-router'
import Workstation from './components/Workstation.vue'
import { moduleManager } from './core/moduleManager'
import resourceModule from './modules/resource'
import scenarioModule from './modules/scenario'

// Register Built-in Modules Implementations
moduleManager.registerImplementation(resourceModule)
moduleManager.registerImplementation(scenarioModule)

const routes = [
  { path: '/', component: Workstation },
  // Module routes are added dynamically
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

const app = createApp(App)
app.use(ElementPlus, {
  locale: zhCn,
})
app.use(router)

const initApp = async () => {
  // Load External Modules Config
  await moduleManager.loadConfig('/modules.json')
  
  // Install Modules (after router is ready & config loaded)
  moduleManager.install(app, router)
  
  app.mount('#app')
}

initApp()
