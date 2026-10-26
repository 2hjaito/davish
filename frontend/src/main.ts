import '@fontsource-variable/inter'
import '@fontsource-variable/jetbrains-mono'
import './lib/styles/tokens.css'
import './lib/styles/reset.css'
import './lib/styles/glass.css'

import { mount } from 'svelte'

import App from './App.svelte'

const target = document.getElementById('app')
if (!target) throw new Error('missing #app')

export default mount(App, { target })
