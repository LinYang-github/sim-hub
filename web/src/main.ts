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
import axios from 'axios'
import { registerStandardViews } from './core/registerStandardViews'
import { useAuth } from './core/auth'

// 注册标准视图
registerStandardViews()

// 暴露全局 API 给外部应用 (demo-view 等)
;(window as any).SimHub = {
    registerView: (meta: any) => moduleManager.registerView(meta),
    registerViewer: (meta: any) => moduleManager.registerViewer(meta),
    registerAction: (meta: any) => moduleManager.registerAction(meta)
}

// 注册内部模块实现 (此处仅注册有特殊定制需求的模块)
moduleManager.registerImplementation(scenarioModule)
// model_glb 将通过 modules.json 中的元数据自动使用 ResourceList 兜底

// 动态插件发现机制：根据活跃模块的配置自动加载外部视图注册脚本
const loadExternalPlugins = () => {
    // External plugins are now registered via configuration (modules.yaml)
    // No need to inject scripts manually.
}

const routes = [
  { path: '/login', component: () => import('./views/Login.vue'), meta: { isPublic: true } },
  { path: '/', component: Workstation },
  { 
    path: '/res/:typeKey', 
    component: () => import('./components/resource/ResourceList.vue'),
    props: (route: any) => {
      const module = moduleManager.getActiveModules().value.find(m => m.key === route.params.typeKey)
      if (!module) return { typeKey: route.params.typeKey }
      
      return {
        ...module,
        typeKey: route.params.typeKey,
        // Resolve views and actions to full objects for the component
        supportedViews: module.supportedViews?.map(v => moduleManager.resolveView(v)),
        customActions: module.customActions?.map(a => moduleManager.resolveAction(a))
      }
    }
  },
  {
    path: '/settings/tokens',
    component: () => import('./components/settings/TokenSettings.vue'),
    name: 'TokenSettings'
  }
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

  // 2. 初始化 Axios 拦截器 (必须在请求之前)
  axios.interceptors.request.use(config => {
    const token = localStorage.getItem('simhub_token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  })

  // 3. 初始化用户信息与权限指令
  const { hasPermission, fetchCurrentUser } = useAuth()
  app.directive('auth', {
    mounted(el, binding) {
      if (!hasPermission(binding.value)) {
        el.style.display = 'none'
      }
    }
  })

  const token = localStorage.getItem('simhub_token')
  if (token) {
    await fetchCurrentUser()
  }
  
  // 4. 初始化 Router
  const router = createRouter({
    history: createWebHistory(),
    routes,
  })

  // 路由守卫
  router.beforeEach((to, from, next) => {
    const token = localStorage.getItem('simhub_token')
    if (to.path === '/login') {
      if (token) return next('/')
      return next()
    }
    
    if (!token && !to.meta.isPublic) {
      return next('/login')
    }
    next()
  })

  axios.interceptors.response.use(
    response => response,
    error => {
      if (error.response?.status === 401) {
        localStorage.removeItem('simhub_token')
        router.push('/login')
      }
      return Promise.reject(error)
    }
  )

  // 5. 安装模块
  moduleManager.install(app, router)
  app.use(router)
  app.mount('#app')
}

initApp()
