import { newId, type Snippet } from '../../apps/snippets/snippets'
import { load, save } from './persist'

const KEY = 'snippets'

/** A command to save, handed to the Saved commands window by another app. */
export interface SnippetDraft {
  command: string
  serverID: string
}

/** Saved commands, on this computer only (settings.json, never the server). */
class Snippets {
  list = $state<Snippet[]>(load(KEY, []))
  /** Set by Command log's "Save as a command"; the window takes it. */
  draft = $state<SnippetDraft | null>(null)

  add(s: Omit<Snippet, 'id'>): Snippet {
    const made = { ...s, id: newId() }
    this.list = [...this.list, made]
    save(KEY, this.list)
    return made
  }

  update(id: string, s: Omit<Snippet, 'id'>) {
    this.list = this.list.map((x) => (x.id === id ? { ...s, id } : x))
    save(KEY, this.list)
  }

  remove(id: string) {
    this.list = this.list.filter((x) => x.id !== id)
    save(KEY, this.list)
  }
}

export const snippets = new Snippets()
