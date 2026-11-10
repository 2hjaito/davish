import { describe, expect, it } from 'vitest'
import { looksSecret, type Snippet, visibleOn } from './snippets'

const s = (id: string, name: string, serverID = '', command = 'ls'): Snippet => ({
  id,
  name,
  command,
  serverID,
})

describe('visibleOn', () => {
  const list = [
    s('1', 'tail nginx', 'a', 'tail -f /var/log/nginx/error.log'),
    s('2', 'disk'),
    s('3', 'other', 'b'),
  ]
  it('keeps the server’s own and the shared ones, by name', () => {
    expect(visibleOn(list, 'a').map((x) => x.id)).toEqual(['2', '1'])
  })
  it('searches name and command', () => {
    expect(visibleOn(list, 'a', 'error.log').map((x) => x.id)).toEqual(['1'])
    expect(visibleOn(list, 'a', 'DISK').map((x) => x.id)).toEqual(['2'])
  })
})

describe('looksSecret', () => {
  it.each([
    'mysql -u root -pS3cret db',
    'export API_KEY=abcd1234',
    'curl -H "Authorization: Bearer xyz" https://x',
    'psql postgres://app:hunter2@localhost/app',
    'echo ghp_abcdefghijklmnop1234 | gh auth login --with-token',
    'PASSWORD=foo ./run.sh',
  ])('flags %s', (cmd) => expect(looksSecret(cmd)).toBe(true))

  it.each([
    'docker compose -p shop up -d',
    'tail -f /var/log/nginx/error.log',
    'git push origin main',
    'ssh -p 2222 host',
    'grep password /etc/ssh/sshd_config',
  ])('leaves %s alone', (cmd) => expect(looksSecret(cmd)).toBe(false))
})
