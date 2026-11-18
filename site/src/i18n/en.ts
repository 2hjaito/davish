// The page's words in English; vi.ts and zh.ts follow the same shape.
export const en = {
  lang: 'en',
  title: 'Davish: a desktop for your Linux servers',
  description:
    'Terminal, files, Docker, logs, databases and more over plain SSH, in one desktop app for macOS, Windows and Linux. Nothing to install on the server.',
  nav: { features: 'Features', security: 'Security', download: 'Download', docs: 'Docs', theme: 'Light or dark', changelog: 'Changelog' },
  hero: {
    title: 'A desktop for your Linux servers.',
    lead: 'Terminal, files, services, Docker, logs, databases and git over plain SSH, in one place that feels like an operating system.',
    tags: ['No web panel', 'No login', 'No agent'],
    download: 'Download for',
    other: 'Other platforms',
    github: 'View on GitHub',
    version: (v: string) => `Version ${v} · Free and open source (GPL-3.0)`,
    shotAlt: 'The Davish desktop: Terminal, Files and Monitor on one server',
    badge: (v: string) => `What’s new in ${v}`,
    docs: 'Read the docs',
    platforms: 'Works on macOS, Windows and Linux',
    hint: 'Pick an app to look inside',
    tabs: { desktop: 'Desktop', servers: 'All servers', docker: 'Docker', database: 'Database', git: 'Git', security: 'Security' },
  },
  why: {
    title: 'Why Davish',
    items: [
      {
        title: 'Open it and you’re in',
        text: 'No login screen, no web panel. Davish connects with your SSH key and puts you on the server’s desktop. Nothing new listens on the server.',
      },
      {
        title: 'Agentless',
        text: 'Only sshd and SFTP. Everything runs from your machine; nothing is copied to, installed on or left behind on the server.',
      },
      {
        title: 'You see every command',
        text: 'Anything destructive shows the exact command first. The command log keeps all of it and exports the changes as a script.',
      },
      {
        title: 'Many servers, one desktop',
        text: 'Each server is a workspace that remembers its windows. Drag files from one server to another.',
      },
      {
        title: 'Your ssh config just works',
        text: 'Import hosts from ~/.ssh/config or an Ansible inventory, keys and ProxyJump included.',
      },
      {
        title: 'Updates itself',
        text: 'A new version shows what changed; Davish downloads it, checks its signature and restarts into it.',
      },
    ],
  },
  features: {
    title: 'Everything a server needs, in windows you already know',
    lead: 'More than twenty apps share one desktop: drag, snap and switch between them like on your own computer.',
    show: [
      {
        key: 'servers',
        title: 'All your servers at a glance',
        text: 'Health and alerts for every server in one window, and a map of what each one serves: domain, nginx, port, container.',
      },
      {
        key: 'multi-run',
        title: 'One command, many servers',
        text: 'Run a command on several servers at once, each in its own terminal tab, with a stop button that really stops.',
      },
      {
        key: 'git',
        title: 'Git on the server',
        text: 'Status, diffs and the commit graph of the repos on your server, with fetch and pull.',
      },
      {
        key: 'docker',
        title: 'Docker, clearly',
        text: 'Containers by compose project, images and their layers, volumes, logs and a shell in one click.',
      },
      {
        key: 'sqlite-rows',
        title: 'Databases without a web panel',
        text: 'MariaDB, PostgreSQL, MongoDB and SQLite: create, dump and restore, and read SQLite tables page by page.',
      },
      {
        key: 'security',
        title: 'Who is knocking',
        text: 'Failed logins per hour and by address, open sessions, ufw and fail2ban, without leaving the desktop.',
      },
    ],
    more: 'Also inside',
    list: [
      'Terminal',
      'Files (SFTP)',
      'Editor',
      'Logs',
      'Monitor',
      'Services',
      'Cron',
      'Nginx',
      'Packages',
      'Ports & tunnels',
      'Browser',
      'Disk',
      'SSH keys',
      'Saved commands',
      'Command log',
      'Spotlight & Launchpad',
    ],
    all: 'See every feature',
  },
  security: {
    title: 'Built to be trusted with root',
    items: [
      'Passwords and passphrases live only in your OS keychain.',
      'A sudo password is never saved: asked once per session, kept in memory, forgotten after 15 minutes.',
      'Locks with its own password or Touch ID when you step away.',
      'Updates are verified: the updater checks the release signature before it installs anything.',
    ],
    more: 'Read the security notes',
  },
  download: {
    title: 'Download Davish',
    lead: 'Free for macOS, Windows and Linux. Your server only needs OpenSSH with SFTP.',
    platforms: {
      macos: { name: 'macOS', note: 'macOS 12 or later, Apple silicon and Intel' },
      windows: { name: 'Windows', note: 'Windows 10 or later, 64-bit' },
      linux: { name: 'Linux', note: 'GTK 3 and WebKitGTK 4.1, x86-64' },
    },
    installer: 'Installer',
    archive: 'Portable',
    checksums: 'Checksums',
    notes: 'Release notes',
    unsigned:
      'Davish is not code-signed yet, so macOS and Windows warn the first time. The install guide on GitHub shows how to open it.',
    guide: 'Install guide',
  },
  stats: {
    downloads: 'downloads of the app',
    apps: 'apps on one desktop',
    version: (date: string) => `latest version, ${date}`,
    stars: 'stars on GitHub',
    free: 'Free',
    freeNote: 'open source, GPL-3.0',
  },
  docs: {
    title: 'Learn it in ten minutes',
    lead: 'Every app has its own page in the docs: what it does, what it runs on the server, and where your data stays.',
    start: 'Get started',
    links: [
      { label: 'Install Davish', href: 'start/install/' },
      { label: 'Connect your first server', href: 'start/first-server/' },
      { label: 'What the server needs', href: 'start/requirements/' },
      { label: 'Keyboard shortcuts', href: 'start/shortcuts/' },
    ],
  },
  community: { discord: 'Join us on Discord', discordNote: 'Questions, ideas and early builds.' },
  thanks: {
    title: 'Made with people who use it',
    lead: 'Code, bug reports, testing on their own machines and ideas: thank you.',
    code: 'code',
    join: 'Join them on GitHub',
  },
  footer: {
    license: 'Davish is free software under the GPL-3.0.',
    changelog: 'Changelog',
    issues: 'Report an issue',
    language: 'Language',
  },
}

export type Dict = typeof en
