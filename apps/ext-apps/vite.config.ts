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
        'demo-view': resolve(__dirname, 'demo-view/index.html'),
        'demo-form': resolve(__dirname, 'demo-form/index.html'),
        'demo-preview': resolve(__dirname, 'demo-preview/index.html'),
      },
    },
    outDir: 'dist',
    rollupOptions: {
      // ... input config ...
      external: [
          // IMPORTANT: Do NOT externalize for standalone app logic, 
          // BUT if we intend to ship components as library, we should.
          // However, here we are exposing source via DevServer which is raw ESM.
          // The issue is likely double-loading Vue.
          // For remote components, we must ensure we use the HOST's Vue.
          // But since we are loading valid ESM from dev server, it imports its OWN Vue from node_modules.
      ]
    }
  }
})
