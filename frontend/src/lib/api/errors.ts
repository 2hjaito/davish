import { t } from '../i18n'

// Keep in sync with internal/apperr.
export type ErrorCode = keyof typeof t.errors

export interface UserError {
  code: ErrorCode
  message: string
}

function isErrorCode(code: unknown): code is ErrorCode {
  return typeof code === 'string' && code in t.errors
}

/** Maps a rejected binding call to a Vietnamese message for the UI. */
export function toUserError(e: unknown): UserError {
  const code =
    typeof e === 'object' && e !== null && 'code' in e && isErrorCode(e.code) ? e.code : 'INTERNAL'
  return { code, message: t.errors[code] }
}
