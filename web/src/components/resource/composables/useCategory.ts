import { ref, computed } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { buildTree } from '../../../core/utils/tree'

export interface Category {
  id: string
  name: string
  parent_id?: string
}

export function useCategory(typeKey: string) {
  const categories = ref<Category[]>([])
  const selectedCategoryId = ref('all')

  /* 
   * Watch typeKey to reset selection and re-fetch.
   * Note: typeKey is passed as a string value in current usage, 
   * but if it's reactive (ref), we should watch it.
   * Assuming it's reactive or the hook is re-called.
   * Actually, the hook is called ONCE per component setup. 
   * If the parent prop updates, the typeKey argument (string) might not be reactive here 
   * unless passed as a ref.
   * 
   * Wait, in ResourceList.vue:
   * useCategory(props.typeKey)
   * 
   * Since 'props.typeKey' is a primitive string when accessed, 
   * this hook won't know when it changes unless we pass a Ref or use watch inside ResourceList.
   * 
   * ResourceList.vue DOES watch props.typeKey and re-inits data.
   * So we just need to expose a reset method or rely on the parent to reset 'selectedCategoryId'.
   * 
   * In ResourceList.vue:
   * watch(() => props.typeKey, () => { selectedCategoryId.value = 'all'; initData(); })
   * 
   * But wait, 'categories' need to be re-fetched too with the NEW typeKey.
   * The current closure 'typeKey' is STALE if the component is reused!
   * 
   * CORRECT FIX: useCategory should accept a MaybeRef or Ref, OR we expose a method to update it.
   * OR simpler: ResourceList handles the re-fetch logic correctly? 
   * No, 'fetchCategories' uses the closed-over 'typeKey'.
   * 
   * If ResourceList component is reused (same instance), 'useCategory' state persists.
   * But 'typeKey' param is fixed to the initial value.
   * 
   * We need 'typeKey' to be dynamic.
   */
  const fetchCategories = async (overrideTypeKey?: string) => {
    try {
      const key = overrideTypeKey || typeKey
      const res = await axios.get('/api/v1/categories', { params: { type: key } })
      categories.value = res.data || []
    } catch (e: any) {
      console.error('Fetch categories failed', e)
    }
  }

  const categoryTree = computed(() => {
    const tree = buildTree(categories.value)
    return [
      { id: 'all', name: '全部分类' },
      ...tree
    ]
  })

  const currentCategoryName = computed(() => {
    if (selectedCategoryId.value === 'all') return '全部'
    const cat = categories.value.find(c => c.id === selectedCategoryId.value)
    return cat ? cat.name : ''
  })

  const promptAddCategory = () => {
    return ElMessageBox.prompt('请输入分类名称', '新建分类', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
    }).then(async ({ value }) => {
      if (!value) return
      await axios.post('/api/v1/categories', {
        type_key: typeKey,
        name: value,
        parent_id: ''
      })
      ElMessage.success('创建成功')
      fetchCategories()
    })
  }

  const confirmDeleteCategory = (id: string, onSuccess: () => void) => {
    const categoryToDelete = categories.value.find(c => c.id === id);
    const categoryName = categoryToDelete ? categoryToDelete.name : '该分类';

    return ElMessageBox.confirm(`确定要删除分类 "${categoryName}" 吗？`, '警告', {
      type: 'warning'
    }).then(async () => {
      await axios.delete(`/api/v1/categories/${id}`)
      ElMessage.success('删除成功')
      if (selectedCategoryId.value === id) {
        selectedCategoryId.value = 'all'
        onSuccess() // callback to refresh list
      }
      fetchCategories()
    })
  }

  return {
    categories,
    selectedCategoryId,
    categoryTree,
    currentCategoryName,
    fetchCategories,
    promptAddCategory,
    confirmDeleteCategory
  }
}
