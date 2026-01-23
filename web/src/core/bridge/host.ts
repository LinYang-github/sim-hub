import type { SimHubMessage, SimHubResponse, SimHubMessageType } from './types'
import { ElNotification } from 'element-plus'

export class SimHubHostBridge {
  private iframes: Set<HTMLIFrameElement> = new Set()

  constructor() {
    window.addEventListener('message', this.handleMessage.bind(this))
  }

  /**
   * 注册一个需要通信的 iframe
   */
  register(iframe: HTMLIFrameElement) {
    this.iframes.add(iframe)
  }

  unregister(iframe: HTMLIFrameElement) {
    this.iframes.delete(iframe)
  }

  /**
   * 向所有子应用广播消息 (例如主题更新)
   */
  broadcast(type: SimHubMessageType, payload: unknown) {
    const message: SimHubMessage = {
      id: Math.random().toString(36).substring(2),
      type,
      payload,
      timestamp: Date.now()
    }

    this.iframes.forEach(iframe => {
      if (iframe.contentWindow) {
        iframe.contentWindow.postMessage(message, '*')
      }
    })
  }

  /**
   * 处理来自子应用的消息
   */
  private handleMessage(event: MessageEvent) {
    const message = event.data as SimHubMessage
    if (!message || !message.type) return

    console.log('[SimHubBridge] Received message from guest:', message)

    switch (message.type) {
      case 'AUTH_TOKEN_GET':
        this.sendResponse(event.source as Window, message.id, {
          token: localStorage.getItem('token') || 'dev-token'
        })
        break

      case 'NOTIFY':
        if (!message.payload) break
        const { type = 'info', title, message: text } = message.payload
        ElNotification({
          type,
          title: title || '来自模块的消息',
          message: text,
          position: 'bottom-right'
        })
        break

      case 'NAVIGATE':
        // 可以在这里处理路由跳转，假设我们已经导出了 router
        // window.location.hash = message.payload.path
        console.warn('Navigation not implemented yet in Host Bridge', message.payload)
        break
    }
  }

  private sendResponse(target: Window, requestId: string, data: unknown) {
    const response: SimHubResponse = {
      id: requestId,
      success: true,
      data
    }
    target.postMessage(response, '*')
  }
}

// 单例模式，全局共享一个消息中心
export const hostBridge = new SimHubHostBridge()
