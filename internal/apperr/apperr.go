// Package apperr defines errors with stable codes that the frontend maps to
// user-facing messages.
package apperr

import "errors"

// Code is a stable error identifier shared with the frontend.
type Code string

// Known error codes. Keep in sync with frontend/src/lib/api/errors.ts.
const (
	Internal             Code = "INTERNAL"
	ConnHostKeyChanged   Code = "CONN_HOSTKEY_CHANGED"
	ConnAuthFailed       Code = "CONN_AUTH_FAILED"
	ConnHostKeyUnknown   Code = "CONN_HOSTKEY_UNKNOWN"
	ConnUnreachable      Code = "CONN_UNREACHABLE"
	ConnSecretMissing    Code = "CONN_SECRET_MISSING"
	ConnAgentUnavailable Code = "CONN_AGENT_UNAVAILABLE"
	ConnKeyUnreadable    Code = "CONN_KEY_UNREADABLE"
	ConnNotConnected     Code = "CONN_NOT_CONNECTED"
	TermClosed           Code = "TERM_CLOSED"
	FSPermissionDenied   Code = "FS_PERMISSION_DENIED"
	FSExists             Code = "FS_EXISTS"
	FSTooLarge           Code = "FS_TOO_LARGE"
	FSStale              Code = "FS_STALE"
	CmdNeedsRoot         Code = "CMD_NEEDS_ROOT"
	InvalidInput         Code = "INVALID_INPUT"
	NotFound             Code = "NOT_FOUND"
	StoreCorrupt         Code = "STORE_CORRUPT"
	KeychainUnavailable  Code = "KEYCHAIN_UNAVAILABLE"
)

// Error is the only error type allowed to cross into the frontend.
// Message must never contain secrets or key paths.
type Error struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	cause   error
}

// New creates an Error without an underlying cause.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap creates an Error that keeps cause for errors.Is/As and logging,
// without exposing it to the frontend.
func Wrap(code Code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, cause: cause}
}

func (e *Error) Error() string { return string(e.Code) + ": " + e.Message }

func (e *Error) Unwrap() error { return e.cause }

// Format converts any error into the payload sent to the frontend. Errors
// that are not *Error become Internal so system details do not leak.
func Format(err error) any {
	var ae *Error
	if errors.As(err, &ae) {
		return ae
	}
	return New(Internal, "internal error")
}
