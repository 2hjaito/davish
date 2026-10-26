// A tiny line-editing shell over mock data, so the Terminal can be typed into
// before real PTY sessions exist (M3). It speaks the same byte stream as a PTY:
// input() receives keystrokes from xterm, emit() writes ANSI output back.
import { join, mockFs } from '../files/mockFs'

const GREEN = '\x1b[32m'
const BLUE = '\x1b[34m'
const YELLOW = '\x1b[33m'
const BOLD_BLUE = '\x1b[1;34m'
const RESET = '\x1b[0m'
const CRLF = '\r\n'

export interface ShellOptions {
  user: string
  host: string
  banner: string
  emit: (data: string) => void
  onCwd?: (cwd: string) => void
  onExit?: () => void
}

export class MockShell {
  cwd: string
  #line = ''
  #history: string[] = []
  #historyIndex = 0
  #home: string
  #opts: ShellOptions

  constructor(opts: ShellOptions) {
    this.#opts = opts
    this.#home = `/home/${opts.user}`
    this.cwd = this.#home
  }

  start() {
    this.#opts.emit(this.#opts.banner + CRLF + CRLF)
    this.#prompt()
  }

  /** Shown in tab titles: the cwd with $HOME abbreviated. */
  get displayCwd(): string {
    if (this.cwd === this.#home) return '~'
    return this.cwd.startsWith(this.#home + '/')
      ? '~' + this.cwd.slice(this.#home.length)
      : this.cwd
  }

  input(data: string) {
    // Arrow keys arrive as escape sequences; everything else is handled per char.
    if (data === '\x1b[A') return this.#recall(-1)
    if (data === '\x1b[B') return this.#recall(1)
    if (data.startsWith('\x1b')) return

    for (const ch of data) {
      if (ch === '\r' || ch === '\n') {
        this.#opts.emit(CRLF)
        this.#run(this.#line)
        this.#line = ''
      } else if (ch === '\x7f' || ch === '\b') {
        if (this.#line.length > 0) {
          this.#line = this.#line.slice(0, -1)
          this.#opts.emit('\b \b')
        }
      } else if (ch === '\x03') {
        this.#opts.emit('^C' + CRLF)
        this.#line = ''
        this.#prompt()
      } else if (ch === '\x0c') {
        this.#opts.emit('\x1b[2J\x1b[H')
        this.#prompt()
        this.#opts.emit(this.#line)
      } else if (ch === '\x15') {
        this.#replaceLine('')
      } else if (ch >= ' ') {
        this.#line += ch
        this.#opts.emit(ch)
      }
    }
  }

  #prompt() {
    const { user, host, emit } = this.#opts
    emit(`${GREEN}${user}@${host}${RESET}:${BLUE}${this.displayCwd}${RESET}$ `)
  }

  #replaceLine(text: string) {
    this.#opts.emit('\b \b'.repeat(this.#line.length) + text)
    this.#line = text
  }

  #recall(step: number) {
    if (this.#history.length === 0) return
    this.#historyIndex = Math.min(this.#history.length, Math.max(0, this.#historyIndex + step))
    this.#replaceLine(this.#history[this.#historyIndex] ?? '')
  }

  #resolve(path: string | undefined): string {
    if (!path || path === '~') return this.#home
    const abs = path.startsWith('/')
      ? path
      : path.startsWith('~/')
        ? this.#home + path.slice(1)
        : join(this.cwd, path)
    const parts: string[] = []
    for (const p of abs.split('/')) {
      if (p === '' || p === '.') continue
      if (p === '..') parts.pop()
      else parts.push(p)
    }
    return '/' + parts.join('/')
  }

  #run(line: string) {
    const trimmed = line.trim()
    if (trimmed) {
      this.#history.push(trimmed)
      this.#historyIndex = this.#history.length
    }
    const [cmd, ...args] = trimmed.split(/\s+/)
    const out = (s = '') => this.#opts.emit(s + CRLF)

    switch (cmd) {
      case undefined:
      case '':
        break
      case 'help':
        out('Lệnh có sẵn: ls, cd, pwd, cat, echo, clear, whoami, hostname, date, uname,')
        out('             history, docker ps, systemctl status <unit>, exit')
        break
      case 'clear':
        this.#opts.emit('\x1b[2J\x1b[H')
        break
      case 'pwd':
        out(this.cwd)
        break
      case 'whoami':
        out(this.#opts.user)
        break
      case 'hostname':
        out(this.#opts.host)
        break
      case 'echo':
        out(args.join(' '))
        break
      case 'date':
        out(new Date().toString())
        break
      case 'uname':
        out(
          args.includes('-a')
            ? `Linux ${this.#opts.host} 6.8.0-45-generic x86_64 GNU/Linux`
            : 'Linux',
        )
        break
      case 'history':
        this.#history.forEach((h, i) => out(`${String(i + 1).padStart(5)}  ${h}`))
        break
      case 'cd': {
        const target = this.#resolve(args[0])
        if (mockFs[target]) {
          this.cwd = target
          this.#opts.onCwd?.(this.displayCwd)
        } else {
          out(`bash: cd: ${args[0]}: No such file or directory`)
        }
        break
      }
      case 'ls': {
        const dir = this.#resolve(args.find((a) => !a.startsWith('-')) ?? this.cwd)
        const entries = mockFs[dir]
        if (!entries) {
          out(`ls: cannot access '${args[0]}': No such file or directory`)
        } else {
          out(entries.map((e) => (e.dir ? `${BOLD_BLUE}${e.name}${RESET}` : e.name)).join('  '))
        }
        break
      }
      case 'cat': {
        const path = this.#resolve(args[0])
        const parent = path.slice(0, path.lastIndexOf('/')) || '/'
        const name = path.slice(path.lastIndexOf('/') + 1)
        const entry = mockFs[parent]?.find((e) => e.name === name)
        if (!args[0]) out('cat: thiếu tên file')
        else if (!entry) out(`cat: ${args[0]}: No such file or directory`)
        else if (entry.dir) out(`cat: ${args[0]}: Is a directory`)
        else out(`# ${path} (nội dung mô phỏng)`)
        break
      }
      case 'docker':
        if (args[0] === 'ps') {
          out('NAMES              STATUS')
          out('grafana            Up 12 days')
          out('loki               Up 12 days')
          out('collabora          Up 3 days')
          out(`lightrag           ${YELLOW}Restarting (1) 8 seconds ago${RESET}`)
          out('license-server     Up 12 days')
        } else {
          out('Usage:  docker ps')
        }
        break
      case 'systemctl': {
        const unit = args[1] ?? 'nginx'
        if (args[0] !== 'status') {
          out('Chỉ hỗ trợ: systemctl status <unit>')
          break
        }
        out(`${GREEN}●${RESET} ${unit}.service`)
        out(`     Loaded: loaded (/lib/systemd/system/${unit}.service; enabled)`)
        out(`     Active: ${GREEN}active (running)${RESET} since Mon 2026-10-12`)
        break
      }
      case 'exit':
      case 'logout':
        out('logout')
        this.#opts.onExit?.()
        return
      default:
        out(`bash: ${cmd}: command not found`)
    }
    this.#prompt()
  }
}
