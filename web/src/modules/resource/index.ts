import { SimHubModule } from '../../core/types'
import ResourceList from './views/ResourceList.vue'

const resourceModule: SimHubModule = {
  key: 'resource',
  routes: [
    { 
      path: '/resources', 
      component: ResourceList 
    }
  ],
  menu: [
    { label: '资源库', path: '/resources' }
  ]
}

export default resourceModule
