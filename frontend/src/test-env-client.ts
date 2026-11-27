// Node, with modules built as for the browser: Svelte's client runtime and
// runes compiled for it, so effects run. vite.config.ts gives it to the
// *.svelte.test.ts files.
import type { Environment } from 'vitest/runtime'

export default {
  name: 'client',
  viteEnvironment: 'client',
  setup: () => ({ teardown: () => {} }),
} satisfies Environment
