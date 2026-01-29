import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  base: './',
  server: {
    port: 30031,
    cors: true
  },
  build: {
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        'demo-repo': resolve(__dirname, 'demo-repo/index.html'),
        'demo-preview': resolve(__dirname, 'demo-preview/index.html'),
      },
    },
    outDir: 'dist'
  }
})
