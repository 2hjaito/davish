import { describe, expect, it } from 'vitest'

import { detectLanguage } from './languages'

describe('detectLanguage', () => {
  it.each([
    ['/srv/app/package.json', '', 'json'],
    ['/srv/app/docker-compose.yml', '', 'yaml'],
    ['/etc/netplan/50-cloud.yaml', '', 'yaml'],
    ['/srv/README.md', '', 'markdown'],
    ['/opt/deploy.sh', '', 'shell'],
    ['/home/u/.bashrc', '', 'shell'],
    ['/home/u/.profile', '', 'shell'],
    ['/srv/app/.env', '', 'env'],
    ['/srv/app/.env.local', '', 'env'],
    ['/srv/app/prod.env', '', 'env'],
    ['/etc/nginx/nginx.conf', '', 'nginx'],
    ['/etc/nginx/sites-available/default', '', 'nginx'],
    ['/etc/nginx/conf.d/app.conf', '', 'nginx'],
    ['/etc/ssh/sshd_config', '', 'conf'],
    ['/etc/mysql/my.cnf', '', 'conf'],
    ['/etc/sysctl.conf', '', 'conf'],
    ['/etc/hosts', '', 'conf'],
    ['/usr/local/bin/backup', '#!/usr/bin/env bash', 'shell'],
    ['/usr/local/bin/run', '#!/bin/sh -e', 'shell'],
    ['/usr/local/bin/tool', '#!/usr/bin/env python3', 'plain'],
    ['/srv/notes.txt', '', 'plain'],
    ['/srv/Makefile', '', 'plain'],
  ])('%s → %s', (path, first, want) => {
    expect(detectLanguage(path, first)).toBe(want)
  })
})
