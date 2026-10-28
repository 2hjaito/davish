// Following one log source, through the Go streams or a demo in the plain
// browser preview.
import { onLogsEnd, onLogsLine, type LogsEnd } from '../../lib/api/events'
import { type LogSource, type LogsStarted, logsStart, logsStop } from '../../lib/api/logs'
import { hasRuntime } from '../../lib/api/runtime'

export interface Handlers {
  lines: (batch: string[]) => void
  end: (e: Omit<LogsEnd, 'streamID'>) => void
}

export interface Following {
  stop: () => void
  /** The follow command with sudo, offered when the source stays silent. */
  sudo: string
}

/** Starts following src; resolves to how to stop it. */
export async function follow(serverID: string, src: LogSource, h: Handlers): Promise<Following> {
  if (!hasRuntime) return demo(src, h)
  // Lines can arrive before logsStart answers, so events are collected
  // from the start and sorted out once the stream's ID is known.
  let id: string | null = null
  const early: { lines?: string[]; end?: LogsEnd }[] = []
  let ended = false
  const offLine = onLogsLine((p) => {
    if (id === null) early.push({ lines: p.lines })
    else if (p.streamID === id) h.lines(p.lines)
  })
  const offEnd = onLogsEnd((p) => {
    if (id === null) early.push({ end: p })
    else if (p.streamID === id) finish(p)
  })
  const finish = (p: LogsEnd) => {
    if (ended) return
    ended = true
    offLine()
    offEnd()
    h.end({ error: p.error, sudo: p.sudo })
  }
  let started: LogsStarted
  try {
    started = await logsStart(serverID, src)
  } catch (e) {
    offLine()
    offEnd()
    throw e
  }
  id = started.streamID
  // Early events of other streams are dropped with the rest of the list.
  for (const ev of early) {
    if (ev.lines) h.lines(ev.lines)
    if (ev.end?.streamID === id) finish(ev.end)
  }
  const mine = id
  return {
    stop: () => {
      if (!ended) void logsStop(mine)
    },
    sudo: started.sudo,
  }
}

function demo(src: LogSource, h: Handlers): Following {
  const at = () => new Date().toISOString().slice(0, 19)
  const sample = [
    () => `${at()} vps app[812]: GET /api/orders 200 12ms`,
    () => `${at()} vps app[812]: GET /health 200 1ms`,
    () => `${at()} vps app[812]: WARN slow query took 1200ms`,
    () => `${at()} vps app[812]: Error: cannot open /etc/nginx/nginx.conf: no such file`,
    () => `${at()} vps app[812]:   File "/srv/app/worker.py", line 42, in run`,
  ]
  if (src.target === '/etc/shadow') {
    setTimeout(
      () =>
        h.end({
          error: { code: 'CMD_NEEDS_ROOT', message: '' },
          sudo: `sudo tail -F -n 200 -- ${src.target}`,
        }),
      200,
    )
    return { stop: () => {}, sudo: '' }
  }
  const sudo = `sudo journalctl -u ${src.target} -f -n 200 -o short-iso --no-pager`
  // A unit with nothing in the journal, to preview the silent state.
  if (src.target === 'quiet.service') return { stop: () => {}, sudo }
  h.lines(Array.from({ length: 40 }, (_, i) => sample[i % sample.length]!()))
  let i = 0
  const timer = setInterval(() => h.lines([sample[i++ % sample.length]!()]), 700)
  return { stop: () => clearInterval(timer), sudo }
}
