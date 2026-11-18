// The landing page at davish.dangth.dev (ADR 0025). Static HTML, no client
// framework; it never ships with the app.
import sitemap from '@astrojs/sitemap'
import { defineConfig } from 'astro/config'

export default defineConfig({
  site: 'https://davish.dangth.dev',
  i18n: {
    defaultLocale: 'en',
    locales: ['en', 'vi', 'zh'],
    routing: { prefixDefaultLocale: false },
  },
  // Screenshots come from the repo's assets/, outside this folder.
  vite: { server: { fs: { allow: ['..'] } } },
  integrations: [
    sitemap({
      i18n: { defaultLocale: 'en', locales: { en: 'en', vi: 'vi', zh: 'zh-CN' } },
    }),
  ],
})
