
import { moduleManager } from './moduleManager'
import { viewMeta as TableMeta } from '../components/resource/views/ResourceTableView.vue'
import { viewMeta as CardMeta } from '../components/resource/views/ResourceCardView.vue'
import { viewMeta as GridMeta } from '../components/resource/views/ResourceDataGrid.vue'
import { viewMeta as GalleryMeta } from '../components/resource/views/ResourceGalleryView.vue'

export const registerStandardViews = () => {
    [TableMeta, CardMeta, GridMeta, GalleryMeta].forEach(meta => {
        moduleManager.registerView(meta)
    })
}
