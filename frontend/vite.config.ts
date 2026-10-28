import { svelte } from '@sveltejs/vite-plugin-svelte'
import { readFileSync } from 'node:fs'
import { defineConfig } from 'vitest/config'

// The version shown in About comes from wails.json, the one the build stamps.
const wails = JSON.parse(readFileSync(new URL('../wails.json', import.meta.url), 'utf8')) as {
  info: { productVersion: string }
}

export default defineConfig({
  plugins: [svelte()],
  define: { __APP_VERSION__: JSON.stringify(wails.info.productVersion) },
  test: {
    include: ['src/**/*.test.ts'],
  },
})
