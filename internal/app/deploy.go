package app

import "davish/internal/deploy"

// DeployPlan plans the deploy wizard's answers: its steps, the line that
// runs them in a terminal, what they create and how to remove it.
func (a *App) DeployPlan(spec deploy.Spec) (deploy.Plan, error) {
	return deploy.Make(spec)
}
