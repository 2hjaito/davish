import js from '@eslint/js'
import prettier from 'eslint-config-prettier'
import svelte from 'eslint-plugin-svelte'
import globals from 'globals'
import ts from 'typescript-eslint'

import svelteConfig from './svelte.config.js'

export default ts.config(
  { ignores: ['dist/', 'wailsjs/', 'node_modules/'] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs.recommended,
  prettier,
  ...svelte.configs.prettier,
  {
    languageOptions: { globals: { ...globals.browser } },
    rules: {
      // Leading underscore marks parameters required by a signature but unused.
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    },
  },
  {
    files: ['**/*.svelte', '**/*.svelte.ts'],
    languageOptions: {
      parserOptions: {
        projectService: true,
        extraFileExtensions: ['.svelte'],
        parser: ts.parser,
        svelteConfig,
      },
    },
  },
  {
    // Only lib/api may talk to the generated Wails bindings.
    files: ['src/**/*.{ts,svelte}'],
    ignores: ['src/lib/api/**'],
    rules: {
      'no-restricted-imports': [
        'error',
        { patterns: [{ group: ['**/wailsjs/**'], message: 'Import from lib/api instead.' }] },
      ],
    },
  },
)
