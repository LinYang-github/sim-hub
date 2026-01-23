import { describe, it, expect, vi, beforeEach, type Mocked } from 'vitest'
import { ref } from 'vue'
import { useResourceList } from '../useResourceList'
import axios from 'axios'

// Mock axios
vi.mock('axios')
const mockedAxios = axios as Mocked<typeof axios>

// Mock ElMessage
vi.mock('element-plus', () => ({
  ElMessage: {
    error: vi.fn(),
    success: vi.fn()
  }
}))

describe('useResourceList', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetchList should call API with correct params for PRIVATE scope', async () => {
    const selectedCategoryId = ref('all')
    const { fetchList, activeScope } = useResourceList('model', true, selectedCategoryId)
    
    // Set scope to PRIVATE
    activeScope.value = 'PRIVATE'

    // Mock Response
    mockedAxios.get.mockResolvedValue({ data: { items: [] } })

    await fetchList()

    expect(mockedAxios.get).toHaveBeenCalledWith('/api/v1/resources', {
      params: { 
        type: 'model', 
        name: '',
        scope: 'PRIVATE',
        owner_id: 'admin'
      }
    })
  })

  it('fetchList should include category_id if selected', async () => {
    const selectedCategoryId = ref('cat-123')
    const { fetchList } = useResourceList('model', true, selectedCategoryId)
    
    mockedAxios.get.mockResolvedValue({ data: { items: [] } })

    await fetchList()

    expect(mockedAxios.get).toHaveBeenCalledWith('/api/v1/resources', expect.objectContaining({
      params: expect.objectContaining({
        category_id: 'cat-123'
      })
    }))
  })

  it('fetchList should include search query', async () => {
    const selectedCategoryId = ref('all')
    const { fetchList, searchQuery } = useResourceList('model', true, selectedCategoryId)
    
    searchQuery.value = 'tank'
    mockedAxios.get.mockResolvedValue({ data: { items: [] } })

    await fetchList()

    expect(mockedAxios.get).toHaveBeenCalledWith('/api/v1/resources', expect.objectContaining({
      params: expect.objectContaining({
        name: 'tank'
      })
    }))
  })

  it('syncFromStorage should call sync API and refresh list', async () => {
    const selectedCategoryId = ref('all')
    const { syncFromStorage, fetchList } = useResourceList('model', true, selectedCategoryId)
    
    mockedAxios.post.mockResolvedValue({ data: { count: 5 } })
    mockedAxios.get.mockResolvedValue({ data: { items: [] } })

    await syncFromStorage()

    expect(mockedAxios.post).toHaveBeenCalledWith('/api/v1/resources/sync')
    // Should call fetchList internally after sync
    expect(mockedAxios.get).toHaveBeenCalled()
  })
})
