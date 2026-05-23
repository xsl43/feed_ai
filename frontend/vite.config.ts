import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/account': 'http://localhost:8080',
      '/video': 'http://localhost:8080',
      '/like': 'http://localhost:8080',
      '/comment': 'http://localhost:8080',
      '/social': 'http://localhost:8080',
      '/feed': 'http://localhost:8080',
      '/message': 'http://localhost:8080',
      '/notification': 'http://localhost:8080',
      '/ai': 'http://localhost:8080',
      '/media': 'http://localhost:8080',
      '/static': 'http://localhost:8080',
      '/review': 'http://localhost:8080',
    }
  }
})
