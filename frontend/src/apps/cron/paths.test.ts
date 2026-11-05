import { describe, expect, it } from 'vitest'

import { commandPaths, folderOf, MAX_PATHS } from './paths'

const HOME = '/home/d'

describe('commandPaths', () => {
  it.each([
    ['/opt/backup/run.sh', ['/opt/backup/run.sh']],
    ['/opt/run.sh >> /var/log/run.log 2>&1', ['/opt/run.sh', '/var/log/run.log']],
    ['bash /opt/run.sh', ['/opt/run.sh']],
    ['/usr/bin/python3 /srv/job.py --fast', ['/srv/job.py']],
    ['cd /srv/app && ./run.sh', ['/srv/app', '/srv/app/run.sh']],
    ['cd /srv/app && php artisan.php now', ['/srv/app', '/srv/app/artisan.php']],
    ['cd /srv/app/sub && ../run.sh', ['/srv/app/sub', '/srv/app/run.sh']],
    ['./nightly.sh', ['/home/d/nightly.sh']],
    ['~/bin/x.sh', ['/home/d/bin/x.sh']],
    ['"/srv/my app/run.sh"', ['/srv/my app/run.sh']],
    ["'/srv/my app/run.sh' > /dev/null", ['/srv/my app/run.sh']],
    ['/srv/my\\ app/run.sh', ['/srv/my app/run.sh']],
    ['flock -n /tmp/x.lock /opt/run.sh', ['/tmp/x.lock', '/opt/run.sh']],
    ['PATH=/opt/bin /opt/run.sh', ['/opt/run.sh']],
    ['/opt/a.sh; /opt/b.sh', ['/opt/a.sh', '/opt/b.sh']],
    ['/opt/a.sh && /opt/a.sh', ['/opt/a.sh']],
  ])('%s', (command, want) => {
    expect(commandPaths(command, HOME)).toEqual(want)
  })

  it('stops at the first unescaped %', () => {
    expect(commandPaths('/opt/a.sh %/opt/stdin.txt', HOME)).toEqual(['/opt/a.sh'])
    expect(commandPaths('/opt/run.sh \\%Y', HOME)).toEqual(['/opt/run.sh'])
  })

  it('leaves out programs, urls, options and expansions', () => {
    expect(commandPaths('curl -s https://dangth.dev/ping', HOME)).toEqual([])
    expect(commandPaths('/usr/sbin/logrotate --state=/x', HOME)).toEqual([])
    expect(commandPaths('find $HOME/logs -delete', HOME)).toEqual([])
    expect(commandPaths('rm /tmp/*.log', HOME)).toEqual([])
    expect(commandPaths('sync', HOME)).toEqual([])
    expect(commandPaths('', HOME)).toEqual([])
  })

  it('knows no relative path without a home', () => {
    expect(commandPaths('./run.sh ~/x', '')).toEqual([])
  })

  it('keeps the list short', () => {
    const many = Array.from({ length: 20 }, (_, i) => `/opt/${i}`).join(' ')
    expect(commandPaths(`/opt/x ${many}`, HOME)).toHaveLength(MAX_PATHS)
  })
})

describe('folderOf', () => {
  it('gives the parent', () => {
    expect(folderOf('/etc/cron.d/backup')).toBe('/etc/cron.d')
    expect(folderOf('/x')).toBe('/')
  })
})
