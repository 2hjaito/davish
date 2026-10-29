/**
 * For a load that may be started again before the last one answered:
 * each call marks a new attempt and returns whether that attempt is still
 * the newest. A retry must never wait for a server that does not answer,
 * and the late answer of an abandoned attempt must not overwrite the new.
 */
export function supersedes(): () => () => boolean {
  let n = 0
  return () => {
    const mine = ++n
    return () => mine === n
  }
}
