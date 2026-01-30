import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import FolderPreview from '../FolderPreview.vue'
import JSZip from 'jszip'

// Mock JSZip
vi.mock('jszip', () => {
  return {
    default: {
      loadAsync: vi.fn()
    }
  }
})

// Mock URL and Fetch
global.URL.createObjectURL = vi.fn(() => 'blob:mock-url')
global.URL.revokeObjectURL = vi.fn()
global.fetch = vi.fn()

describe('FolderPreview.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should use cover_url from metadata if priority', async () => {
    const wrapper = mount(FolderPreview, {
      props: {
        url: 'http://test.com/file.zip',
        metaData: { cover_url: 'http://test.com/cover.jpg' }
      }
    })

    // Wait for the immediate watch to trigger
    await new Promise(resolve => setTimeout(resolve, 0))
    
    expect(wrapper.find('.preview-img').exists()).toBe(true)
    const img = wrapper.get('.preview-img')
    expect(img.attributes('src')).toBe('http://test.com/cover.jpg')
    expect(global.fetch).not.toHaveBeenCalled()
  })

  it('should extract cover from ZIP with priority (preview > root > any)', async () => {
    // Mock Fetch response
    const mockBuffer = new ArrayBuffer(8)
    vi.mocked(global.fetch).mockResolvedValue({
      ok: true,
      arrayBuffer: () => Promise.resolve(mockBuffer)
    } as any)

    // Mock JSZip structure
    const mockFiles = {
      'folder/other.txt': { dir: false, async: vi.fn() },
      'inner/pic.png': { dir: false, async: vi.fn(() => Promise.resolve(new Blob())) },
      'root_pic.jpg': { dir: false, async: vi.fn(() => Promise.resolve(new Blob())) },
      'sub/preview_it.png': { dir: false, async: vi.fn(() => Promise.resolve(new Blob())) }
    }
    
    vi.mocked(JSZip.loadAsync).mockResolvedValue({
      files: mockFiles
    } as any)

    const wrapper = mount(FolderPreview, {
      props: {
        url: 'http://test.com/file.zip'
      }
    })

    await new Promise(resolve => setTimeout(resolve, 50)) // Wait for async logic
    
    expect(global.fetch).toHaveBeenCalledWith('http://test.com/file.zip', expect.any(Object))
    expect(JSZip.loadAsync).toHaveBeenCalledWith(mockBuffer)
    
    // In our mock, 'sub/preview_it.png' contains 'preview', so it should be chosen
    expect(mockFiles['sub/preview_it.png'].async).toHaveBeenCalledWith('blob')
    expect(wrapper.get('.preview-img').attributes('src')).toBe('blob:mock-url')
  })

  it('should handle fetch errors and show fallback', async () => {
    vi.mocked(global.fetch).mockRejectedValue(new TypeError('Failed to fetch'))

    const wrapper = mount(FolderPreview, {
      props: {
        url: 'http://test.com/bad.zip',
        typeName: 'Scenario'
      }
    })

    await new Promise(resolve => setTimeout(resolve, 50))
    
    expect(wrapper.get('.folder-name').text()).toBe('Scenario')
    expect(wrapper.find('.cors-hint').exists()).toBe(true)
  })
})
