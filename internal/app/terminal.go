package app

// TermStart opens a shell on the server and returns the session ID; output
// arrives as term:data and the end as term:exit. prompt asks for Davish's
// prompt on bash. A non-empty dir is where the shell starts.
func (a *App) TermStart(serverID string, cols, rows int, prompt bool, dir string) (string, error) {
	if err := a.lock.Check(); err != nil {
		return "", err
	}
	return a.terms.Start(a.ctx, serverID, cols, rows, prompt, dir)
}

// TermWrite sends input to the shell.
func (a *App) TermWrite(id, data string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.terms.Write(id, data)
}

// TermResize updates the PTY size.
func (a *App) TermResize(id string, cols, rows int) error {
	return a.terms.Resize(id, cols, rows)
}

// TermClose ends the session.
func (a *App) TermClose(id string) error {
	return a.terms.Close(id)
}
