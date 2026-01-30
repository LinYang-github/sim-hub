import { describe, it, expect, vi, beforeEach } from 'vitest'
import { moduleManager } from '../moduleManager'
import request from '../utils/request'

// Mock request utility
vi.mock('../utils/request', () => ({
  default: {
    get: vi.fn()
  }
}))

describe('ModuleManager', () => {
  beforeEach(() => {
    // Reset registries before each test
    moduleManager.viewRegistry.value = new Map()
    moduleManager.actionRegistry.value = new Map()
    moduleManager.viewerRegistry.value = new Map()
  })

  it('should register and resolve views correctly', () => {
    const mockView = { key: 'test-view', label: 'Test View', icon: 'Picture' }
    moduleManager.registerView(mockView)
    
    const resolved = moduleManager.resolveView('test-view')
    expect(resolved.label).toBe('Test View')
    
    const fallback = moduleManager.resolveView('unknown-view')
    expect(fallback.key).toBe('unknown-view')
    expect(fallback.icon).toBe('Document')
  })

  it('should register and resolve viewers correctly', () => {
    const mockViewer = { key: 'GLB', label: '3D Viewer', path: '/viewer/glb' }
    moduleManager.registerViewer(mockViewer)
    
    expect(moduleManager.resolveViewer('GLB')).toBe('/viewer/glb')
    expect(moduleManager.resolveViewer('Unknown')).toBe('Unknown')
  })

  it('should auto-register components from config metadata', async () => {
    const mockConfig = [
      {
        type_key: 'scenario',
        type_name: 'Scenario Module',
        meta_data: {
          viewer: { key: 'FolderPreview', label: 'Folder', path: 'FolderPreview' },
          supported_views: [
            { key: 'gallery', label: 'Gallery', icon: 'Grid' }
          ],
          custom_actions: [
            { key: 'deploy', label: 'Deploy', icon: 'Aim' }
          ]
        }
      }
    ]

    vi.mocked(request.get).mockResolvedValue(mockConfig)

    await moduleManager.loadConfig()

    // Verify auto-registration
    expect(moduleManager.resolveViewer('FolderPreview')).toBe('FolderPreview')
    expect(moduleManager.resolveView('gallery').label).toBe('Gallery')
    expect(moduleManager.resolveAction('deploy').label).toBe('Deploy')
  })

  it('should handle nested modules and fallbacks', async () => {
     const mockConfig = [
        {
          type_key: 'simple_res',
          type_name: 'Simple',
          integration_mode: 'internal'
        }
     ]
     vi.mocked(request.get).mockResolvedValue(mockConfig)

     await moduleManager.loadConfig()
     
     const modules = moduleManager.getActiveModules().value
     expect(modules).toHaveLength(1)
     expect(modules[0].key).toBe('simple_res')
     // Should have auto-generated fallback menu/route
     expect(modules[0].menu?.[0].path).toBe('/res/simple_res')
  })
})
