/** Covers the app with the sleep screen until the next click or key press. */
class Sleep {
  active = $state(false)

  start() {
    this.active = true
  }

  wake() {
    this.active = false
  }
}

export const sleep = new Sleep()
