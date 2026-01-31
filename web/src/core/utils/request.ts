import axios, { AxiosInstance, AxiosResponse, AxiosError } from 'axios'
import { ElMessage } from 'element-plus'

export interface RequestInstance extends AxiosInstance {
  <T = any, R = T, D = any>(config: import('axios').AxiosRequestConfig<D>): Promise<R>;
  <T = any, R = T, D = any>(url: string, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
  get<T = any, R = T, D = any>(url: string, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
  delete<T = any, R = T, D = any>(url: string, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
  head<T = any, R = T, D = any>(url: string, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
  options<T = any, R = T, D = any>(url: string, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
  post<T = any, R = T, D = any>(url: string, data?: D, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
  put<T = any, R = T, D = any>(url: string, data?: D, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
  patch<T = any, R = T, D = any>(url: string, data?: D, config?: import('axios').AxiosRequestConfig<D>): Promise<R>;
}

const request = axios.create({
  baseURL: '/',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request Interceptor
request.interceptors.request.use(
  (config) => {
    // Add Auth Token
    const token = localStorage.getItem('simhub_token')
    if (token) config.headers.Authorization = `Bearer ${token}`
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response Interceptor
request.interceptors.response.use(
  (response: AxiosResponse) => {
    // Return the standard data object from the response
    return response.data
  },
  (error: AxiosError) => {
    const errorData = error.response?.data as any
    const message = errorData?.error || errorData?.message || error.message || '未知服务请求异常'
    
    // Global Error Notification
    ElMessage.error(message)
    
    return Promise.reject(error)
  }
)

export default (request as unknown) as RequestInstance
