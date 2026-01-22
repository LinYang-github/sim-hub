import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import App from './App.vue'
import { createRouter, createWebHistory } from 'vue-router'
import Workstation from './components/Workstation.vue'
import { moduleManager } from './core/moduleManager'
import resourceModule from './modules/resource'
import bingModule from './modules/external_example'

// Register Modules
moduleManager.register(resourceModule)
moduleManager.register(bingModule)

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

// Install Modules (after router is ready)
moduleManager.install(app, router)

app.mount('#app')
