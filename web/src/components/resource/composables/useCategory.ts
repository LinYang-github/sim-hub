import { ref, computed, Ref, watch } from 'vue'
import request from '../../../core/utils/request'
import { ElMessage, ElMessageBox } from 'element-plus'
import { buildTree } from '../../../core/utils/tree'
import type { Category, CategoryNode } from '../../../core/types/resource'

export function useCategory(typeKey: Ref<string>) {
  const categories = ref<Category[]>([])
  const selectedCategoryId = ref('all')

  const fetchCategories = async () => {
    try {
      const res = await request.get<Category[]>('/api/v1/categories', { params: { type: typeKey.value } })
      categories.value = res || []
    } catch (e: any) {
    }
  }

  // 监听资源类型变化：重置选中项并刷新分类树
  watch(typeKey, () => {
    selectedCategoryId.value = 'all'
    fetchCategories()
  }, { immediate: true })

  const categoryTree = computed<CategoryNode[]>(() => {
    const tree = buildTree(categories.value) as CategoryNode[]
    return [
      { id: 'all', name: '全部分类' } as CategoryNode,
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
      await request.post('/api/v1/categories', {
        type_key: typeKey.value,
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
      await request.delete(`/api/v1/categories/${id}`)
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
