// Reading ufw status lines, which Davish shows as ufw prints them.

const addr = (tok: string) =>
  /^[0-9a-fA-F.:]+(\/\d+)?$/.test(tok) && /[.:]/.test(tok) && /\d/.test(tok)

/** The addresses a DENY or REJECT rule blocks for good, such as
 *  "Anywhere   DENY   193.24.211.219". ufw stops at the first rule that
 *  matches, so a deny listed after an ALLOW rule does not count: the
 *  allowed port stays open to that address. */
export function deniedIPs(lines: readonly string[]): Set<string> {
  const out = new Set<string>()
  let allowed = false
  for (const l of lines) {
    if (/\bALLOW\b/.test(l)) allowed = true
    if (allowed || !/\b(DENY|REJECT)\b/.test(l)) continue
    for (const tok of l.split(/\s+/)) if (addr(tok)) out.add(tok)
  }
  return out
}

/** Blocks ip before any other rule. An older deny of the same address,
 *  maybe listed after an ALLOW where it does nothing, is removed first;
 *  ufw only says so when there is none. */
export const blockCommand = (ip: string) =>
  `sudo ufw delete deny from ${ip} ; sudo ufw insert 1 deny from ${ip}`
