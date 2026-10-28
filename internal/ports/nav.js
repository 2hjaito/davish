// Added by Davish's proxy to every page shown in its Browser app
// (docs/adr/0008-in-app-browser.md). It tells Davish where the page is, and
// takes back, forward and reload from Davish only.
;(function () {
  var parent = window.parent
  if (parent === window) return
  function report() {
    parent.postMessage({ davish: 'nav', href: location.href, title: document.title }, '*')
  }
  report()
  addEventListener('load', report)
  addEventListener('popstate', report)
  addEventListener('hashchange', report)
  var h = history
  var push = h.pushState
  var replace = h.replaceState
  h.pushState = function () {
    push.apply(h, arguments)
    report()
  }
  h.replaceState = function () {
    replace.apply(h, arguments)
    report()
  }
  addEventListener('message', function (e) {
    if (e.source !== parent) return
    var cmd = e.data && e.data.davish
    if (cmd === 'back') history.back()
    else if (cmd === 'forward') history.forward()
    else if (cmd === 'reload') location.reload()
  })
})()
