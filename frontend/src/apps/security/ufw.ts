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

/** Every blocked address, by ufw, fail2ban or both, in address order. */
export function blockedIPs(
  denied: ReadonlySet<string>,
  banned: ReadonlySet<string>,
): { ip: string; via: string[] }[] {
  return [...new Set([...denied, ...banned])]
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
    .map((ip) => ({
      ip,
      via: [denied.has(ip) && 'ufw', banned.has(ip) && 'fail2ban'].filter((v) => v !== false),
    }))
}
