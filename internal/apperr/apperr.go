// Package apperr defines errors with stable codes that the frontend maps to
// user-facing messages.
package apperr

import "errors"

// Code is a stable error identifier shared with the frontend.
type Code string

// Known error codes. Keep in sync with frontend/src/lib/api/errors.ts.
const (
	Internal           Code = "INTERNAL"
	ConnHostKeyChanged Code = "CONN_HOSTKEY_CHANGED"
	ConnAuthFailed     Code = "CONN_AUTH_FAILED"
	FSPermissionDenied Code = "FS_PERMISSION_DENIED"
	CmdNeedsRoot       Code = "CMD_NEEDS_ROOT"
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
