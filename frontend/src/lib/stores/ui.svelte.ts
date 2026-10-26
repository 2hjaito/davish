/** Which shell-level dialog is open. */
type ServerDialog = { mode: 'add' } | { mode: 'edit'; id: string } | null

class Ui {
  serverDialog = $state<ServerDialog>(null)

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
