import { SimHubModule } from '../../core/types'
import { Folder } from '@element-plus/icons-vue'

const scenarioModule: SimHubModule = {
  key: 'scenario',
  label: '想定资源库', // Default label, can be overridden by config
  menu: [
    {
      label: '想定资源库',
      path: '/res/scenario',
      icon: Folder
    }
  ],
  routes: [
    {
      path: '/res/scenario',
      component: () => import('../../components/resource/ResourceList.vue'),
      props: { 
        typeKey: 'scenario', 
        typeName: '想定',
        uploadMode: 'folder-zip',
        enableScope: true,
        viewer: 'FolderPreview'
      }
    }
  ]
}

export default scenarioModule
