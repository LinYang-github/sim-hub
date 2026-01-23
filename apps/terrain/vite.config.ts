import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  return {
    plugins: [vue()],
    // 开发环境下使用根路径，预览/构建环境下根据后端嵌入路径配置为 /terrain/
    base: mode === 'development' ? '/' : '/terrain/',
    server: {
      port: 30032,
      host: '0.0.0.0'
    }
  }
})
