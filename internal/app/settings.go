package app

// SettingsLoad returns the UI state saved in settings.json: JSON text by key.
func (a *App) SettingsLoad() (map[string]string, error) {
	return a.settings.All()
}

// SettingsSave stores one key of UI state as JSON text.
func (a *App) SettingsSave(key, value string) error {
	return a.settings.Set(key, value)
}
