import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import { useResourceList } from '../useResourceList'
import request from '../../../../core/utils/request'
import { RESOURCE_SCOPE, ROOT_CATEGORY_ID, DEFAULT_ADMIN_ID } from '../../../../core/constants/resource'

// Mock request module
vi.mock('../../../../core/utils/request', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn()
  }
}))

describe('useResourceList', () => {
  // Clear mocks before each test
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should initialize with default state', () => {
    const typeKey = ref('model_glb')
    const enableScope = ref(true)
    const selectedCategoryId = ref(ROOT_CATEGORY_ID)

    const { resources, loading, activeScope, searchQuery, syncing } = useResourceList(
      typeKey,
      enableScope,
      selectedCategoryId
    )

    expect(resources.value).toEqual([])
    expect(loading.value).toBe(true) // immediate watch triggers loading
    expect(activeScope.value).toBe(RESOURCE_SCOPE.ALL)
    expect(searchQuery.value).toBe('')
    expect(syncing.value).toBe(false)
  })

  it('should fetch resources on initialization', async () => {
    const mockData = { items: [{ id: '1', name: 'Res 1' }] }
    ;(request.get as any).mockResolvedValue(mockData)

    const typeKey = ref('model_glb')
    const enableScope = ref(false)
    const selectedCategoryId = ref(ROOT_CATEGORY_ID)

    const { resources, loading } = useResourceList(
      typeKey,
      enableScope,
      selectedCategoryId
    )

    // Wait for the async watcher
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(request.get).toHaveBeenCalledWith('/api/v1/resources', expect.any(Object))
    expect(resources.value).toEqual(mockData.items)
    expect(loading.value).toBe(false)
  })

  it('should construct correct parameters for filtering', async () => {
    const typeKey = ref('map_terrain')
    const enableScope = ref(true)
    const selectedCategoryId = ref('cat-123')
    
    ;(request.get as any).mockResolvedValue({ items: [] })

    const { fetchList, activeScope, searchQuery } = useResourceList(
      typeKey,
      enableScope,
      selectedCategoryId
    )

    // Set various filters
    searchQuery.value = 'test-query'
    activeScope.value = RESOURCE_SCOPE.PRIVATE
    
    await fetchList()

    expect(request.get).toHaveBeenCalledWith('/api/v1/resources', {
      params: {
        type: 'map_terrain',
        query: 'test-query',
        category_id: 'cat-123',
        scope: RESOURCE_SCOPE.PRIVATE,
        owner_id: DEFAULT_ADMIN_ID
      }
    })
  })

  it('should handle scope changes based on enableScope logic', async () => {
    const typeKey = ref('model_glb')
    const enableScope = ref(false)
    const selectedCategoryId = ref(ROOT_CATEGORY_ID)

    const { activeScope } = useResourceList(typeKey, enableScope, selectedCategoryId)
    
    // Initial state for enableScope = false
    expect(activeScope.value).toBe(RESOURCE_SCOPE.ALL)

    // Toggle enableScope to true
    enableScope.value = true
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(activeScope.value).toBe(RESOURCE_SCOPE.ALL)

    // Toggle back
    enableScope.value = false
    await new Promise(resolve => setTimeout(resolve, 0))
    expect(activeScope.value).toBe(RESOURCE_SCOPE.ALL)
  })

  it('should handle race conditions properly', async () => {
    const typeKey = ref('model_glb')
    const enableScope = ref(true)
    const selectedCategoryId = ref(ROOT_CATEGORY_ID)
    
    const { fetchList, resources } = useResourceList(typeKey, enableScope, selectedCategoryId)

    // Setup two promises, one slow and one fast
    let resolveSlow: Function
    const slowReq = new Promise(resolve => { resolveSlow = resolve })
    
    const fastData = { items: [{ id: 'fast', name: 'Fast' }] }
    const slowData = { items: [{ id: 'slow', name: 'Slow' }] }

    // First call (Slow)
    ;(request.get as any).mockReturnValue(slowReq)
    const p1 = fetchList()

    // Second call (Fast)
    ;(request.get as any).mockResolvedValue(fastData)
    const p2 = fetchList()

    await p2
    expect(resources.value).toEqual(fastData.items) // Should update to fast data

    // Now resolve the slow one
    resolveSlow!(slowData)
    await p1

    // Should NOT update to slow data, remain fast
    expect(resources.value).toEqual(fastData.items)
  })

  it('should sync from storage and refresh list', async () => {
    const typeKey = ref('model_glb')
    const enableScope = ref(true)
    const selectedCategoryId = ref(ROOT_CATEGORY_ID)
    
    const { syncFromStorage } = useResourceList(typeKey, enableScope, selectedCategoryId)

    ;(request.post as any).mockResolvedValue({ count: 5 })
    ;(request.get as any).mockResolvedValue({ items: [] })

    await syncFromStorage()

    expect(request.post).toHaveBeenCalledWith('/api/v1/resources/sync')
    expect(request.get).toHaveBeenCalled() // fetchList should be called after sync
  })

  it('should trigger fetch on search query change', async () => {
    const typeKey = ref('model_glb')
    const enableScope = ref(true)
    const selectedCategoryId = ref(ROOT_CATEGORY_ID)
    
    ;(request.get as any).mockResolvedValue({ items: [] })

    const { searchQuery } = useResourceList(typeKey, enableScope, selectedCategoryId)
    
    // Initial fetch happens immediately
    await new Promise(resolve => setTimeout(resolve, 0))
    vi.clearAllMocks()

    searchQuery.value = 'test-search'
    await new Promise(resolve => setTimeout(resolve, 0))

    expect(request.get).toHaveBeenCalledWith('/api/v1/resources', expect.objectContaining({
      params: expect.objectContaining({ query: 'test-search' })
    }))
  })

  it('should handle fetch error gracefully', async () => {
    const typeKey = ref('model_glb')
    const enableScope = ref(true)
    const selectedCategoryId = ref(ROOT_CATEGORY_ID)

    ;(request.get as any).mockRejectedValue(new Error('Network Error'))

    const { loading, resources } = useResourceList(typeKey, enableScope, selectedCategoryId)
    
    // Initial fetch triggered by watch
    await new Promise(resolve => setTimeout(resolve, 0))
    
    expect(loading.value).toBe(false)
    expect(resources.value).toEqual([])
  })
})
