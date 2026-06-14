import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), vueJsx(), vueDevTools()],
  build: {
    outDir: 'dist',
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    headers: {
      // Required for Firebase signInWithPopup — allows the Google OAuth popup
      // to communicate back to the app window.
      'Cross-Origin-Opener-Policy': 'same-origin-allow-popups',
    },
    proxy: {
      '/api': process.env.VITE_API_BASE || 'http://localhost:8080',
      '/ws': {
        target: (process.env.VITE_API_BASE || 'http://localhost:8080').replace(/^http/, 'ws'),
        ws: true,
      },
    },
  },
})
