import { svelte } from '@sveltejs/vite-plugin-svelte'
import { readFileSync } from 'node:fs'
import { defineConfig } from 'vitest/config'

// The version shown in About comes from wails.json, the one the build stamps.
const wails = JSON.parse(readFileSync(new URL('../wails.json', import.meta.url), 'utf8')) as {
  info: { productVersion: string }
}

export default defineConfig({
  plugins: [svelte()],
  // The manifest lets scripts/bundle-budget.mjs tell the first screen's
  // chunks from those loaded later.
  build: { manifest: true },
  define: { __APP_VERSION__: JSON.stringify(wails.info.productVersion) },
  // Lets *.svelte.test.ts load Svelte's client build (see below); the
  // other tests keep the server one.
  resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,
  test: {
    setupFiles: ['src/test-setup.ts'],
    projects: [
      {
        extends: true,
        test: {
          name: 'unit',
          include: ['src/**/*.test.ts'],
          exclude: ['src/**/*.svelte.test.ts'],
        },
      },
      // Tests of effects: under the default environment Svelte loads its
      // server build, where effects never run.
      {
        extends: true,
        test: {
          name: 'runes',
          include: ['src/**/*.svelte.test.ts'],
          environment: './src/test-env-client.ts',
        },
      },
    ],
  },
})
