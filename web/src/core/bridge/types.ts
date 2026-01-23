export type SimHubMessageType = 
  | 'THEME_UPDATE'       // Host -> Guest
  | 'AUTH_TOKEN_GET'     // Guest -> Host
  | 'NOTIFY'             // Guest -> Host
  | 'NAVIGATE'           // Guest -> Host
  | 'VIEWPORT_SYNC'      // Hybrid (Optional: 3D视口同步)

export interface SimHubMessage<T = any> {
  id: string
  type: SimHubMessageType
  payload?: T
  timestamp: number
}

export interface SimHubResponse<T = any> {
  id: string              // Matching request id
  success: boolean
  data?: T
  error?: string
}
