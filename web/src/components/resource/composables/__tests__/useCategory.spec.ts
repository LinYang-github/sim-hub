import { describe, it, expect, vi, beforeEach, type Mocked } from 'vitest'
import { ref } from 'vue'
import { useCategory } from '../useCategory'
import request from '../../../../core/utils/request'
import { ElMessageBox } from 'element-plus'
import { ROOT_CATEGORY_ID } from '../../../../core/constants/resource'

// Mock dependencies
vi.mock('../../../../core/utils/request', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn()
  }
}))

vi.mock('element-plus', () => ({
  ElMessage: {
    success: vi.fn(),
    error: vi.fn()
  },
  ElMessageBox: {
    prompt: vi.fn(),
    confirm: vi.fn()
  }
}))

const mockedRequest = request as Mocked<typeof request>
const mockedMessageBox = ElMessageBox as Mocked<typeof ElMessageBox>

describe('useCategory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetchCategories should load and populate data', async () => {
    const typeKey = ref('model')
    const { fetchCategories, categories, categoryTree } = useCategory(typeKey)
    
    const mockData = [
      { id: '1', name: 'Vehicles', parent_id: '' },
      { id: '2', name: 'Tank', parent_id: '1' }
    ]
    mockedRequest.get.mockResolvedValue(mockData)

    await fetchCategories()

    expect(categories.value).toEqual(mockData)
    // Tree should contain root 'all' + built tree
    expect(categoryTree.value).toHaveLength(2) // ROOT + Vehicles
    expect((categoryTree.value[1] as any).children).toHaveLength(1) // Vehicles has Tank
  })

  it('promptAddCategory should call API on confirm', async () => {
    const typeKey = ref('model')
    const { promptAddCategory } = useCategory(typeKey)
    
    mockedMessageBox.prompt.mockResolvedValue({ value: 'New Cat', action: 'confirm' } as any)
    mockedRequest.post.mockResolvedValue({})

    await promptAddCategory()

    expect(mockedRequest.post).toHaveBeenCalledWith('/api/v1/categories', {
      type_key: 'model',
      name: 'New Cat',
      parent_id: ''
    })
  })

  it('confirmDeleteCategory should call API and callback', async () => {
    const typeKey = ref('model')
    const { confirmDeleteCategory, selectedCategoryId, categories } = useCategory(typeKey)
    const onSuccess = vi.fn()
    
    mockedMessageBox.confirm.mockResolvedValue('confirm' as any)
    mockedRequest.delete.mockResolvedValue({})

    // Initialize categories so find() works
    categories.value = [{ id: 'cat-1', name: 'Cat 1', parent_id: '' }]

    // Simulate deleting the currently selected category
    selectedCategoryId.value = 'cat-1'
    await confirmDeleteCategory('cat-1', onSuccess)

    expect(mockedRequest.delete).toHaveBeenCalledWith('/api/v1/categories/cat-1')
    expect(selectedCategoryId.value).toBe(ROOT_CATEGORY_ID)
    expect(onSuccess).toHaveBeenCalled()
  })
})
