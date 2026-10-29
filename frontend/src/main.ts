import '@fontsource-variable/inter'
import '@fontsource-variable/jetbrains-mono'
import './lib/styles/fonts.css'
import './lib/styles/tokens.css'
import './lib/styles/reset.css'
import './lib/styles/glass.css'
import './lib/styles/controls.css'

import { mount } from 'svelte'

import { reportFrontendErrors } from './lib/api/diag'
import { hasRuntime } from './lib/api/runtime'
import { initPersist } from './lib/stores/persist'

const target = document.getElementById('app')
if (!target) throw new Error('missing #app')

reportFrontendErrors(hasRuntime)

// Stores read saved UI state when their modules load, so the app is only
// imported once settings.json has arrived.
await initPersist()
const { default: App } = await import('./App.svelte')

export default mount(App, { target })
