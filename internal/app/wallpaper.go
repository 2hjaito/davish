package app

// WallpaperGet returns the desktop picture the person chose as a data URL,
// or "" when there is none.
func (a *App) WallpaperGet() (string, error) {
	return a.wallpaper.Get()
}

// WallpaperSet keeps a picture (a JPEG, PNG or WebP data URL) as the desktop background.
func (a *App) WallpaperSet(dataURL string) error {
	return a.wallpaper.Set(dataURL)
}

// WallpaperClear removes the chosen picture.
func (a *App) WallpaperClear() error {
	return a.wallpaper.Clear()
}
