import { SimHubModule } from '../types'
import { Folder, Box, Location } from '@element-plus/icons-vue'

export function createResourceModule(
    key: string, 
    label: string, 
    typeName: string, 
    path: string, 
    icon: any,
    uploadMode: 'single' | 'folder-zip' = 'single',
    accept?: string
): SimHubModule {
    return {
        key,
        label,
        menu: [
            {
                label,
                path,
                icon
            }
        ],
        routes: [
            {
                path,
                component: () => import('../../components/resource/ResourceList.vue'),
                props: { 
                    typeKey: key, 
                    typeName,
                    uploadMode,
                    accept
                }
            }
        ]
    }
}
