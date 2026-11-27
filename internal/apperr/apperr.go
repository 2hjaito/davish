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
	ConnKeyPPK           Code = "CONN_KEY_PPK"
	ConnPassphraseWrong  Code = "CONN_PASSPHRASE_WRONG"
	ConnNotConnected     Code = "CONN_NOT_CONNECTED"
	ConnControlSocket    Code = "CONN_CONTROL_SOCKET"
	// ConnAgentSignTimeout: the SSH agent did not sign the login in time, for
	// instance because it waits for the person to confirm the key (#168).
	ConnAgentSignTimeout Code = "CONN_AGENT_SIGN_TIMEOUT"
	// ConnProxyUnconfirmed: the server reaches its host through a command
	// that runs on this computer and the user has not confirmed it (ADR 0031).
	ConnProxyUnconfirmed Code = "CONN_PROXY_UNCONFIRMED"
	TermClosed           Code = "TERM_CLOSED"
	FSPermissionDenied   Code = "FS_PERMISSION_DENIED"
	FSExists             Code = "FS_EXISTS"
	FSTooLarge           Code = "FS_TOO_LARGE"
	FSStale              Code = "FS_STALE"
	FSChanged            Code = "FS_CHANGED"
	FSProtected          Code = "FS_PROTECTED"
	CmdNeedsRoot         Code = "CMD_NEEDS_ROOT"
	CmdMissing           Code = "CMD_MISSING"
	CmdTooOld            Code = "CMD_TOO_OLD"
	CmdFailed            Code = "CMD_FAILED"
	CmdTimeout           Code = "CMD_TIMEOUT"
	DockerDown           Code = "DOCKER_DOWN"
	InvalidInput         Code = "INVALID_INPUT"
	NotFound             Code = "NOT_FOUND"
	StoreCorrupt         Code = "STORE_CORRUPT"
	KeychainUnavailable  Code = "KEYCHAIN_UNAVAILABLE"
	AppLocked            Code = "APP_LOCKED"
	LockWrongPassword    Code = "LOCK_WRONG_PASSWORD"
	LockTooManyAttempts  Code = "LOCK_TOO_MANY_ATTEMPTS"
	LockBiometricFailed  Code = "LOCK_BIOMETRIC_FAILED"
	GitUnsafeOwner       Code = "GIT_UNSAFE_OWNER"
	GitDirty             Code = "GIT_DIRTY"
	PlatformUnsupported  Code = "PLATFORM_UNSUPPORTED"
	GitAuthFailed        Code = "GIT_AUTH_FAILED"
	GitNotFastForward    Code = "GIT_NOT_FAST_FORWARD"
	GitNoUpstream        Code = "GIT_NO_UPSTREAM"
	GitRemoteFailed      Code = "GIT_REMOTE_FAILED"
	ShortcutChanged      Code = "SHORTCUT_CHANGED"
	UpdateUnavailable    Code = "UPDATE_UNAVAILABLE"
	UpdateUnsigned       Code = "UPDATE_UNSIGNED"
	UpdateBadSignature   Code = "UPDATE_BAD_SIGNATURE"
	UpdateNoWrite        Code = "UPDATE_NO_WRITE"
	UpdateBroken         Code = "UPDATE_BROKEN"
	UpdateDevBuild       Code = "UPDATE_DEV_BUILD"
	UpdateDownloadFailed Code = "UPDATE_DOWNLOAD_FAILED"
	DBAuthRequired       Code = "DB_AUTH_REQUIRED"
	DBAuthFailed         Code = "DB_AUTH_FAILED"
	DBSystem             Code = "DB_SYSTEM"
	SudoPasswordRequired Code = "SUDO_PASSWORD_REQUIRED"
	SudoPasswordWrong    Code = "SUDO_PASSWORD_WRONG"
	// Turning password logins off (ADR 0029): the key could not log in
	// before the change, or could not after it, which was then undone.
	SecKeyLoginFailed Code = "SEC_KEY_LOGIN_FAILED"
	SecKeyLoginLost   Code = "SEC_KEY_LOGIN_LOST"
	// Blocking or banning an address that would cut Davish off: too wide a
	// range, or the address Davish connects from.
	SecWouldLockOut Code = "SEC_WOULD_LOCK_OUT"
	// An Ansible inventory in YAML, or a script, which only INI import reads.
	InventoryYAMLUnsupported   Code = "INVENTORY_YAML_UNSUPPORTED"
	InventoryScriptUnsupported Code = "INVENTORY_SCRIPT_UNSUPPORTED"
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
