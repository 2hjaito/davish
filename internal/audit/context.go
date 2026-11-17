package audit

import "context"

type ctxKey int

const (
	changingKey ctxKey = iota
	passwordKey
)

// Changing marks the commands run with ctx as part of an action that
// changes the server, so the Command log can export them as a script
// (docs/adr/0022). Action bindings call services with it.
func Changing(ctx context.Context) context.Context {
	return context.WithValue(ctx, changingKey, true)
}

// IsChanging reports whether ctx came from Changing.
func IsChanging(ctx context.Context) bool {
	v, _ := ctx.Value(changingKey).(bool)
	return v
}

// PasswordStdin marks a command whose only stdin is a sudo password, so
// its entry does not claim content was sent.
func PasswordStdin(ctx context.Context) context.Context {
	return context.WithValue(ctx, passwordKey, true)
}

func passwordStdin(ctx context.Context) bool {
	v, _ := ctx.Value(passwordKey).(bool)
	return v
}
