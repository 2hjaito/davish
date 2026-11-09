import { t } from '../i18n'

// Keep in sync with internal/apperr.
export type ErrorCode = keyof typeof t.errors

export interface UserError {
  code: ErrorCode
  message: string
  /** The backend's own English text ("download answered 504 …"), for a bug report. */
  detail: string
}

function isErrorCode(code: unknown): code is ErrorCode {
  return typeof code === 'string' && code in t.errors
}

/** Maps a rejected binding call to a Vietnamese message for the UI. */
export function toUserError(e: unknown): UserError {
  const code =
    typeof e === 'object' && e !== null && 'code' in e && isErrorCode(e.code) ? e.code : 'INTERNAL'
  const detail =
    typeof e === 'object' && e !== null && 'message' in e && typeof e.message === 'string'
      ? e.message
      : String(e)
  return { code, message: t.errors[code], detail }
}
