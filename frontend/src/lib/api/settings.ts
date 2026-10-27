// Typed wrappers for the settings bindings (internal/app/settings.go).
import * as App from '../../../wailsjs/go/app/App'

/** UI state from settings.json: JSON text by key. */
export const settingsLoad = (): Promise<Record<string, string>> => App.SettingsLoad()

export const settingsSave = (key: string, json: string): Promise<void> =>
  App.SettingsSave(key, json)
