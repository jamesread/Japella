import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import mkcert from 'vite-plugin-mkcert'

export default defineConfig({
  optimizeDeps: {
    // Keep a single module instance for the popup store (app code + host component).
    exclude: ['picocrank/vue/composables/useNotificationPopups.js'],
  },
  server: {
    allowedHosts: ['baneling.teratan.net'],
    proxy: {
      // Backend API + static files that only exist on the Go server (or in dist/).
      // /assets must be proxied: Vite itself 404s hashed production bundles when
      // requested as scripts (e.g. after a service-worker or cached dist index.html).
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/lang': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/assets': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/upload': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/media': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/oauth2callback': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/mcp': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/llms.txt': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },
  plugins: [
    mkcert(),
    Components({
      dirs: "resources/vue/",
      extensions: ['vue'],
      deep: true,
      dts: false,
    }),
    vue({
      template: {
        compilerOptions: {
          isCustomElement: (tag) => tag === 'emoji-picker',
        },
      },
    }),
  ],
})
