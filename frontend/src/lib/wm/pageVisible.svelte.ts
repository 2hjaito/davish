/** Whether the Davish window can be seen: false while hidden or minimized to the OS. */
class PageVisible {
  visible = $state(typeof document === 'undefined' ? true : !document.hidden)

  constructor() {
    if (typeof document === 'undefined') return
    document.addEventListener('visibilitychange', () => (this.visible = !document.hidden))
  }
}

export const pageVisible = new PageVisible()
