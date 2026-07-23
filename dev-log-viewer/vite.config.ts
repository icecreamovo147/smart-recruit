import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  root: 'web',
  plugins: [tailwindcss()],
  server: {
    host: '127.0.0.1',
    port: 8091,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
      '/healthz': {
        target: 'http://127.0.0.1:8090',
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: '127.0.0.1',
    port: 8091,
    strictPort: true,
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  test: {
    globals: true,
    environment: 'node',
    include: ['src/test/**/*.test.ts'],
  },
})
