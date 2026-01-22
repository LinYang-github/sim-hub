import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import App from './App.vue'
import { createRouter, createWebHistory } from 'vue-router'
import ResourcesList from './components/ResourcesList.vue'
import Workstation from './components/Workstation.vue'

const routes = [
  { path: '/', component: Workstation },
  { path: '/resources', component: ResourcesList },
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
app.mount('#app')
