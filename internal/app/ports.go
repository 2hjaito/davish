package app

import "davish/internal/ports"

// PortsList returns the server's listening TCP ports.
func (a *App) PortsList(serverID string) (ports.List, error) {
	return a.ports.List(a.ctx, serverID)
}

// PortsProbe fetches http://host:port/ from the server itself.
func (a *App) PortsProbe(serverID, host string, port int, hostHeader string) (string, error) {
	if err := a.lock.Check(); err != nil {
		return "", err
	}
	return a.ports.Probe(a.ctx, serverID, host, port, hostHeader)
}

// TunnelStart forwards a server port to 127.0.0.1 on this machine.
func (a *App) TunnelStart(serverID, host string, port int, hostHeader string) (ports.Tunnel, error) {
	if err := a.lock.Check(); err != nil {
		return ports.Tunnel{}, err
	}
	return a.tunnels.Start(serverID, host, port, hostHeader)
}

// TunnelStop closes a tunnel.
func (a *App) TunnelStop(id string) {
	a.tunnels.Stop(id)
}

// TunnelList returns the running tunnels.
func (a *App) TunnelList() []ports.Tunnel {
	return a.tunnels.List()
}
