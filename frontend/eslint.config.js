import pluginVue from 'eslint-plugin-vue'

export default [
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      'resources/javascript/gen/**',
      'resources/javascript/main.ts',
    ],
  },
  ...pluginVue.configs['flat/essential'],
  {
    files: ['**/*.{js,vue}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
    },
    rules: {
      'vue/multi-word-component-names': 'off',
      'vue/require-v-for-key': 'off',
      'vue/no-textarea-mustache': 'off',
    },
  },
]
