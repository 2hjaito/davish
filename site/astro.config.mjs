// davish.app (ADR 0025): the landing page in src/pages, the docs
// under /docs with Starlight. Static HTML; it never ships with the app.
import starlight from '@astrojs/starlight'
import { defineConfig } from 'astro/config'

const group = (en, vi, zh, dir) => ({
  label: en,
  translations: { vi, 'zh-CN': zh },
  items: [{ autogenerate: { directory: `docs/${dir}` } }],
})

export default defineConfig({
  site: 'https://davish.app',
  // Screenshots come from the repo's assets/, outside this folder.
  vite: { server: { fs: { allow: ['..'] } } },
  integrations: [
    starlight({
      title: { en: 'Davish Docs', vi: 'Tài liệu Davish', 'zh-CN': 'Davish 文档' },
      favicon: '/favicon.svg',
      defaultLocale: 'root',
      locales: {
        root: { label: 'English', lang: 'en' },
        vi: { label: 'Tiếng Việt', lang: 'vi' },
        zh: { label: '简体中文', lang: 'zh-CN' },
      },
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/2hjaito/davish' },
        { icon: 'discord', label: 'Discord', href: 'https://discord.com/invite/eG5txdWaqr' },
      ],
      editLink: { baseUrl: 'https://github.com/2hjaito/davish/edit/main/site/' },
      customCss: ['./src/styles/docs.css'],
      // The header's parts, matched to the landing page's nav.
      components: {
        SiteTitle: './src/components/docs/SiteTitle.astro',
        SocialIcons: './src/components/docs/SocialIcons.astro',
        ThemeSelect: './src/components/docs/ThemeSelect.astro',
        LanguageSelect: './src/components/docs/LanguageSelect.astro',
      },
      sidebar: [
        group('Get started', 'Bắt đầu', '入门', 'start'),
        group('On the server', 'Trên server', '服务器上', 'on-the-server'),
        group('Services and apps', 'Dịch vụ và ứng dụng', '服务与应用', 'services-and-apps'),
        group('Network and security', 'Mạng và bảo mật', '网络与安全', 'network-and-security'),
        group('The desktop', 'Desktop', '桌面', 'the-desktop'),
      ],
    }),
  ],
})
