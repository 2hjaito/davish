/** Which shell-level dialog or overlay is open. */
type ServerDialog = { mode: 'add' } | { mode: 'edit'; id: string } | null

class Ui {
  serverDialog = $state<ServerDialog>(null)
  spotlight = $state(false)
  launchpad = $state(false)

  toggleSpotlight() {
    this.spotlight = !this.spotlight
  }

  addServer() {
    this.serverDialog = { mode: 'add' }
  }

  editServer(id: string) {
    this.serverDialog = { mode: 'edit', id }
  }

  closeServerDialog() {
    this.serverDialog = null
  }
}

export const ui = new Ui()
