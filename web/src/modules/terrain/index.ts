import { SimHubModule } from '../../core/types'
import { Location } from '@element-plus/icons-vue'

const terrainModule: SimHubModule = {
  key: 'map_terrain',
  label: '高程地形库',
  menu: [
    {
      label: '高程地形库',
      path: '/terrain',
      icon: Location
    }
  ],
  routes: [
    {
      path: '/terrain',
      component: () => import('./TerrainList.vue')
    }
  ]
}

export default terrainModule
