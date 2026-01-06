import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import mkcert from 'vite-plugin-mkcert'

export default defineConfig({
  server: {
    allowedHosts: ['baneling.teratan.net'],
    proxy: {
      '/api': {
        target: 'http://localhost:8234',
        changeOrigin: true,
        secure: false,
      },
      '/lang': {
        target: 'http://localhost:8234',
        changeOrigin: true,
        secure: false,
      }
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
    vue(),
  ],
})
