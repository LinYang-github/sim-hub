
import { moduleManager } from './moduleManager'
import { viewMeta as TableMeta } from '../components/resource/views/ResourceTableView.vue'
import { viewMeta as CardMeta } from '../components/resource/views/ResourceCardView.vue'
import { viewMeta as GridMeta } from '../components/resource/views/ResourceDataGrid.vue'

export const registerStandardViews = () => {
    [TableMeta, CardMeta, GridMeta].forEach(meta => {
        moduleManager.registerView(meta)
    })
}
