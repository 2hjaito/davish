// What each kind of widget looks like at a glance: a color and an icon, so the
// widgets on a desktop can be told apart without reading them. The colors are
// not the ones that mean a state (green, amber and red stay for that).
import type { IconName } from '../ui/icons'
import type { WidgetKind } from './kinds'

export const LOOK: Record<WidgetKind, { accent: string; icon: IconName }> = {
  monitor: { accent: 'var(--widget-monitor)', icon: 'monitor' },
  docker: { accent: 'var(--widget-docker)', icon: 'brandDocker' },
  services: { accent: 'var(--widget-services)', icon: 'services' },
  disk: { accent: 'var(--widget-disk)', icon: 'disk' },
}
