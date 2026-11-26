// The panes of Settings, in sidebar order, and the words each one answers
// to in the sidebar search.
import { t } from '../../lib/i18n'
import { STAT_IDS } from '../../lib/shell/statIds'

export const PANES = [
  'account',
  'general',
  'appearance',
  'performance',
  'menubar',
  'dock',
  'terminal',
  'editor',
  'files',
  'alerts',
  'security',
  'about',
] as const

export type Pane = (typeof PANES)[number]

export function paneLabel(p: Pane): string {
  return t.settings[p]
}

/** Everything a pane shows a title for: its name, then its rows. */
function words(p: Pane): string[] {
  const s = t.settings
  switch (p) {
    case 'account':
      return [s.profileName, s.avatarPick, s.lock, s.lockAfter, s.lockChange]
    case 'general':
      return [s.language, s.startup, ...Object.values(s.startups)]
    case 'appearance':
      return [
        s.theme,
        ...Object.values(s.themes),
        s.wallpaper,
        s.wallpaperPicture,
        s.wallpaperDim,
        s.monoFont,
      ]
    case 'performance':
      return [
        s.perfPreset,
        s.dragTitle,
        s.dragHint,
        s.dragFill,
        s.perfAdvanced,
        s.perfBlur,
        s.perfShadow,
        s.perfOpaque,
        s.perfWallpaper,
        s.perfWallpaperImage,
        s.perfMotion,
        s.perfNoDmabuf,
        s.perfNoCompositing,
      ]
    case 'menubar':
      return [s.menubarStats, ...STAT_IDS.map((id) => t.menubar.stats.names[id])]
    case 'dock':
      return [s.dockSize, s.dockPosition, s.dockTransparent, s.dockAutohide]
    case 'terminal':
      return [
        s.fontSize,
        s.termScheme,
        s.termFont,
        s.termCursor,
        s.termBlink,
        s.prompt,
        s.gpu,
        s.termSolid,
        s.scrollback,
      ]
    case 'editor':
      return [
        s.editorFont,
        s.editorWrap,
        s.editorLineNumbers,
        s.editorWhitespace,
        s.editorIndent,
        s.backup,
      ]
    case 'files':
      return [s.filesHidden]
    case 'alerts':
      return [s.alertMem, s.alertDisk, s.alertHold]
    case 'security':
      return [s.sudoForget]
    case 'about':
      return ['Davish', s.updates, s.autoUpdate, s.checkNow]
  }
}

const fold = (s: string) =>
  s.normalize('NFD').replace(/\p{M}/gu, '').replace(/đ/g, 'd').toLowerCase()

/** The panes whose name or rows contain query, ignoring case and accents. */
export function searchPanes(query: string): Pane[] {
  const q = fold(query.trim())
  if (!q) return [...PANES]
  return PANES.filter((p) => [paneLabel(p), ...words(p)].some((w) => fold(w).includes(q)))
}
