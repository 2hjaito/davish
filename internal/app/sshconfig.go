package app

import "davish/internal/sshconfig"

// SSHConfigHosts lists the concrete hosts in ~/.ssh/config, marking those
// already saved.
func (a *App) SSHConfigHosts() ([]sshconfig.Candidate, error) {
	hosts, _, err := sshconfig.LoadDefault()
	if err != nil {
		return nil, err
	}
	saved, err := a.store.Servers(a.ctx)
	if err != nil {
		return nil, err
	}
	return sshconfig.Candidates(hosts, saved), nil
}

// ImportSSHHosts saves the chosen ~/.ssh/config hosts, and the jump hosts
// they need, and returns the servers added.
func (a *App) ImportSSHHosts(aliases []string) ([]ServerView, error) {
	if err := a.lock.Check(); err != nil {
		return nil, err
	}
	hosts, localUser, err := sshconfig.LoadDefault()
	if err != nil {
		return nil, err
	}
	added, err := sshconfig.Import(a.ctx, a.store, hosts, aliases, localUser)
	views := make([]ServerView, 0, len(added))
	for _, s := range added {
		v, verr := a.view(s)
		if verr != nil {
			return views, verr
		}
		views = append(views, v)
	}
	return views, err
}
