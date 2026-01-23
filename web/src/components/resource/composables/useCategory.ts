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

  const fetchCategories = async () => {
    try {
      const res = await axios.get('/api/v1/categories', { params: { type: typeKey } })
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
