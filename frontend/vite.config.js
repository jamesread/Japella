import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'

export default defineConfig({
	plugins: [
        Components({
          dirs: "resources/vue/",
          extensions: ['vue'],
          deep: true,
          dts: false,
        }),
		vue(),
	],
})
