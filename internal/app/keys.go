package app

import "davish/internal/sshkeys"

// KeysAuthorized lists the keys allowed to log in as the server's user.
func (a *App) KeysAuthorized(serverID string) (sshkeys.Authorized, error) {
	return a.keys.Authorized(a.ctx, serverID)
}

// KeysPlanAdd describes adding a public key line to authorized_keys.
func (a *App) KeysPlanAdd(serverID, text string) (sshkeys.KeyPlan, error) {
	return a.keys.PlanAdd(a.ctx, serverID, text)
}

// KeysAdd runs the confirmed command of KeysPlanAdd.
func (a *App) KeysAdd(serverID, text, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.keys.Add(a.ctx, serverID, text, command)
}

// KeysPlanRemove describes removing a key from authorized_keys.
func (a *App) KeysPlanRemove(serverID, fingerprint string) (sshkeys.KeyPlan, error) {
	return a.keys.PlanRemove(a.ctx, serverID, fingerprint)
}

// KeysRemove runs the confirmed plan of KeysPlanRemove.
func (a *App) KeysRemove(serverID string, plan sshkeys.KeyPlan) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.keys.Remove(a.ctx, serverID, plan)
}

// KeysServer lists the key pairs in the server user's ~/.ssh.
func (a *App) KeysServer(serverID string) ([]sshkeys.ServerKey, error) {
	return a.keys.ServerKeys(a.ctx, serverID)
}

// KeysPlanCreate describes making a key pair on the server.
func (a *App) KeysPlanCreate(serverID, name, comment string) (sshkeys.CreatePlan, error) {
	return a.keys.PlanCreate(a.ctx, serverID, name, comment)
}

// KeysCreate runs the confirmed command of KeysPlanCreate.
func (a *App) KeysCreate(serverID, name, comment, command string) error {
	if err := a.lock.Check(); err != nil {
		return err
	}
	return a.keys.Create(a.ctx, serverID, name, comment, command)
}

// KeysPlanGitTest describes testing a server key against a git host.
func (a *App) KeysPlanGitTest(serverID, host, name string) (sshkeys.GitTestPlan, error) {
	return a.keys.PlanGitTest(a.ctx, serverID, host, name)
}

// KeysGitTest runs the confirmed command of KeysPlanGitTest.
func (a *App) KeysGitTest(serverID, host, name, command string) (sshkeys.GitTestResult, error) {
	if err := a.lock.Check(); err != nil {
		return sshkeys.GitTestResult{}, err
	}
	return a.keys.GitTest(a.ctx, serverID, host, name, command)
}

// KeysLocal lists the key pairs in ~/.ssh on this machine.
func (a *App) KeysLocal() ([]sshkeys.LocalKey, error) {
	dir, err := sshkeys.LocalDir()
	if err != nil {
		return nil, err
	}
	return sshkeys.LocalKeys(dir)
}

// KeysGenerateLocal makes an ed25519 pair in ~/.ssh on this machine.
func (a *App) KeysGenerateLocal(name, comment, passphrase string) (sshkeys.LocalKey, error) {
	if err := a.lock.Check(); err != nil {
		return sshkeys.LocalKey{}, err
	}
	dir, err := sshkeys.LocalDir()
	if err != nil {
		return sshkeys.LocalKey{}, err
	}
	return sshkeys.GenerateLocal(dir, name, comment, passphrase)
}
