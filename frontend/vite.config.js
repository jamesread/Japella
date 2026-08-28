import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import mkcert from 'vite-plugin-mkcert'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createReadStream, existsSync } from 'node:fs'
import { stat } from 'node:fs/promises'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const supplementalThemesDir = path.join(__dirname, 'node_modules/picocrank/supplemental-themes')
const includeSupplementalThemes = process.env.VITE_INCLUDE_SUPPLEMENTAL_THEMES !== 'false'

function contentTypeFor(filePath) {
  if (filePath.endsWith('.css')) return 'text/css; charset=utf-8'
  if (filePath.endsWith('.json')) return 'application/json; charset=utf-8'
  if (filePath.endsWith('.png')) return 'image/png'
  if (filePath.endsWith('.jpg') || filePath.endsWith('.jpeg')) return 'image/jpeg'
  if (filePath.endsWith('.webp')) return 'image/webp'
  if (filePath.endsWith('.svg')) return 'image/svg+xml'
  return 'application/octet-stream'
}

function copyDirectoryRecursive(sourceDir, targetDir) {
  fs.mkdirSync(targetDir, { recursive: true })
  for (const entry of fs.readdirSync(sourceDir, { withFileTypes: true })) {
    const sourcePath = path.join(sourceDir, entry.name)
    const targetPath = path.join(targetDir, entry.name)
    if (entry.isDirectory()) {
      copyDirectoryRecursive(sourcePath, targetPath)
      continue
    }
    fs.copyFileSync(sourcePath, targetPath)
  }
}

function supplementalThemesPlugin() {
  return {
    name: 'japella-supplemental-themes',
    configureServer(server) {
      if (!includeSupplementalThemes || !fs.existsSync(supplementalThemesDir)) {
        return
      }

      server.middlewares.use('/supplemental-themes', (req, res, next) => {
        const requestPath = decodeURIComponent((req.url || '/').split('?')[0])
        const relativePath = requestPath.replace(/^\/+/, '')
        const filePath = path.resolve(supplementalThemesDir, relativePath)

        if (!filePath.startsWith(path.resolve(supplementalThemesDir))) {
          res.statusCode = 403
          res.end('Forbidden')
          return
        }

        if (!existsSync(filePath)) {
          next()
          return
        }

        stat(filePath).then((fileStat) => {
          if (!fileStat.isFile()) {
            next()
            return
          }
          res.setHeader('Content-Type', contentTypeFor(filePath))
          createReadStream(filePath).pipe(res)
        }).catch(() => {
          next()
        })
      })
    },
    closeBundle() {
      if (!includeSupplementalThemes || !fs.existsSync(supplementalThemesDir)) {
        return
      }
      const outDir = path.resolve(__dirname, 'dist', 'supplemental-themes')
      copyDirectoryRecursive(supplementalThemesDir, outDir)
    },
  }
}

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
    supplementalThemesPlugin(),
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
