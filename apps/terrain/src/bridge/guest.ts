export class SimHubGuestBridge {
  private handlers: Map<string, (payload: any) => void> = new Map()
  private pendingRequests: Map<string, { resolve: (val: any) => void, reject: (err: any) => void }> = new Map()

  constructor() {
    window.addEventListener('message', this.handleMessage.bind(this))
  }

  /**
   * 向主应用发送请求（支持 Promise 回调）
   */
  async callHost<T = any>(type: string, payload?: any): Promise<T> {
    const id = Math.random().toString(36).substring(2)
    const message = { id, type, payload, timestamp: Date.now() }

    return new Promise((resolve, reject) => {
      this.pendingRequests.set(id, { resolve, reject })
      window.parent.postMessage(message, '*')
      
      // 30秒超时处理
      setTimeout(() => {
        if (this.pendingRequests.has(id)) {
          this.pendingRequests.delete(id)
          reject(new Error(`SimHubBridge: Request timeout [${type}]`))
        }
      }, 30000)
    })
  }

  /**
   * 发送无需回复的消息
   */
  emit(type: string, payload?: any) {
    const message = { id: 'evt', type, payload, timestamp: Date.now() }
    window.parent.postMessage(message, '*')
  }

  /**
   * 监听来自主应用的广播
   */
  on(type: string, handler: (payload: any) => void) {
    this.handlers.set(type, handler)
  }

  private handleMessage(event: MessageEvent) {
    const data = event.data
    
    // 处理 RPC 响应
    if (data.id && this.pendingRequests.has(data.id)) {
      const { resolve, reject } = this.pendingRequests.get(data.id)!
      this.pendingRequests.delete(data.id)
      if (data.success) {
        resolve(data.data)
      } else {
        reject(new Error(data.error))
      }
      return
    }

    // 处理来自主应用的广播 (Event)
    if (data.type && this.handlers.has(data.type)) {
      this.handlers.get(data.type)!(data.payload)
    }
  }

  // 快捷方法
  getAuthToken() {
    return this.callHost<{ token: string }>('AUTH_TOKEN_GET')
  }

  notify(config: { type?: 'success' | 'warning' | 'info' | 'error', title?: string, message: string }) {
    this.emit('NOTIFY', config)
  }
}

export const bridge = new SimHubGuestBridge()
