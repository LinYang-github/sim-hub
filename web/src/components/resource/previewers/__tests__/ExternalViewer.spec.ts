import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ExternalViewer from '../ExternalViewer.vue'

describe('ExternalViewer.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Mock getComputedStyle for theme tokens
    // @ts-ignore
    global.getComputedStyle = vi.fn().mockReturnValue({
      getPropertyValue: vi.fn().mockReturnValue(' #409eff ')
    })
  })

  it('should send theme and data on iframe load', async () => {
    const mockResource = { id: 'res-1', name: 'Test Resource' }
    const wrapper = mount(ExternalViewer, {
      props: {
        url: 'viewer',
        resource: mockResource
      }
    })

    const iframe = wrapper.find('iframe').element as HTMLIFrameElement
    // Mock contentWindow.postMessage
    const postMessageSpy = vi.fn()
    Object.defineProperty(iframe, 'contentWindow', {
        value: { postMessage: postMessageSpy },
        writable: true
    })

    // Simulate iframe load
    await wrapper.find('iframe').trigger('load')

    expect(wrapper.vm.loading).toBe(false)
    // Should send THEME_UPDATE and PREVIEW_DATA
    expect(postMessageSpy).toHaveBeenCalledWith(
        expect.objectContaining({ type: 'THEME_UPDATE' }),
        '*'
    )
    expect(postMessageSpy).toHaveBeenCalledWith(
        expect.objectContaining({ 
            type: 'PREVIEW_DATA',
            payload: expect.objectContaining({ resource: mockResource })
        }),
        '*'
    )
  })

  it('should handle GUEST_READY message and re-sync', async () => {
    const wrapper = mount(ExternalViewer, {
      props: { url: 'viewer', resource: { id: '1' } }
    })

    const iframe = wrapper.find('iframe').element as HTMLIFrameElement
    const postMessageSpy = vi.fn()
    Object.defineProperty(iframe, 'contentWindow', {
        value: { postMessage: postMessageSpy },
        writable: true
    })

    // Simulate GUEST_READY postMessage
    const messageEvent = new MessageEvent('message', {
      data: { type: 'GUEST_READY' },
      source: iframe.contentWindow
    })
    window.dispatchEvent(messageEvent)

    // Verify it bubbled up the event
    expect(wrapper.emitted('ready')).toBeTruthy()
    // Should have re-sent theme and data
    expect(postMessageSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'THEME_UPDATE' }), '*')
  })

  it('should remove event listener on unmount', () => {
    const removeSpy = vi.spyOn(window, 'removeEventListener')
    const wrapper = mount(ExternalViewer, {
      props: { url: 'viewer' }
    })

    wrapper.unmount()
    expect(removeSpy).toHaveBeenCalledWith('message', expect.any(Function))
  })
})
