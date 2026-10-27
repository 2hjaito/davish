// Moves an element to the end of <body>. Windows are positioned with
// translate, which makes them the containing block of position: fixed
// descendants; overlays that must sit at viewport coordinates escape that.
export function portal(node: HTMLElement) {
  document.body.appendChild(node)
  return {
    destroy() {
      node.remove()
    },
  }
}
