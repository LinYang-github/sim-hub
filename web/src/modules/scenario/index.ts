import { SimHubModule } from '../../core/types'
import { Folder } from '@element-plus/icons-vue'

const scenarioModule: SimHubModule = {
  key: 'scenario',
  label: '想定资源库', // Default label, can be overridden by config
  menu: [
    {
      label: '想定资源库',
      path: '/scenarios',
      icon: Folder
    }
  ],
  routes: [
    {
      path: '/scenarios',
      component: () => import('./views/ScenarioList.vue')
    }
  ]
}

export default scenarioModule
