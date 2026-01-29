import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  base: './',
  server: {
    port: 30032,
    proxy: {
      '/api': {
        target: 'http://localhost:30030',
        changeOrigin: true,
      }
    }
  }
})
