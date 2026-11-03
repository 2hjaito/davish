//go:build dev

package update

// devBuild is set by wails dev, whose app is rebuilt and removed with the
// dev server: it never updates itself.
const devBuild = true
