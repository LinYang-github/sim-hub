import { describe, it, expect, vi, beforeEach, type Mocked } from 'vitest'
import { useCategory } from '../useCategory'
import axios from 'axios'
import { ElMessageBox } from 'element-plus'

// Mock dependencies
vi.mock('axios')
const mockedAxios = axios as Mocked<typeof axios>

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

const mockedMessageBox = ElMessageBox as Mocked<typeof ElMessageBox>

describe('useCategory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetchCategories should load and populate data', async () => {
    const { fetchCategories, categories, categoryTree } = useCategory('model')
    
    const mockData = [
      { id: '1', name: 'Vehicles', parent_id: '' },
      { id: '2', name: 'Tank', parent_id: '1' }
    ]
    mockedAxios.get.mockResolvedValue({ data: mockData })

    await fetchCategories()

    expect(categories.value).toEqual(mockData)
    // Verify tree building logic (flat -> tree)
    // Tree should contain root 'all' + built tree
    expect(categoryTree.value).toHaveLength(2) // 'all' + 'Vehicles'
    expect((categoryTree.value[1] as any).children).toHaveLength(1) // 'Vehicles' has 'Tank'
  })

  it('promptAddCategory should call API on confirm', async () => {
    const { promptAddCategory } = useCategory('model')
    
    // Type casting logic for message box return is complex, simplified for test
    mockedMessageBox.prompt.mockResolvedValue({ value: 'New Cat', action: 'confirm' } as any)
    mockedAxios.post.mockResolvedValue({})

    await promptAddCategory()

    expect(mockedAxios.post).toHaveBeenCalledWith('/api/v1/categories', {
      type_key: 'model',
      name: 'New Cat',
      parent_id: ''
    })
  })

  it('confirmDeleteCategory should call API and callback', async () => {
    const { confirmDeleteCategory, selectedCategoryId } = useCategory('model')
    const onSuccess = vi.fn()
    
    mockedMessageBox.confirm.mockResolvedValue('confirm' as any)
    mockedAxios.delete.mockResolvedValue({})

    // Simulate deleting the currently selected category
    selectedCategoryId.value = 'cat-1'
    await confirmDeleteCategory('cat-1', onSuccess)

    expect(mockedAxios.delete).toHaveBeenCalledWith('/api/v1/categories/cat-1')
    expect(selectedCategoryId.value).toBe('all')
    expect(onSuccess).toHaveBeenCalled()
  })
})
