package app

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"davish/internal/transfer"
)

// TransferPickFiles asks the user for files to upload; empty when canceled.
func (a *App) TransferPickFiles(title string) ([]string, error) {
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
}

// TransferPickFolder asks the user for a folder to upload; empty when canceled.
func (a *App) TransferPickFolder(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
}

// TransferPlanUpload reports which local paths clash with names in dir,
// and what replacing each would run.
func (a *App) TransferPlanUpload(serverID, dir string, locals []string) ([]transfer.UploadItem, error) {
	return a.transfers.PlanUpload(a.ctx, serverID, dir, locals)
}

// TransferUpload starts uploads into dir; progress arrives as events.
func (a *App) TransferUpload(serverID, dir string, reqs []transfer.UploadRequest) ([]transfer.Job, error) {
	return a.transfers.Upload(a.ctx, serverID, dir, reqs)
}

// TransferDownload starts downloads into the user's Downloads folder.
func (a *App) TransferDownload(serverID string, paths []string) ([]transfer.Job, error) {
	return a.transfers.Download(a.ctx, serverID, paths)
}

// TransferCancel stops a job and removes its partial files.
func (a *App) TransferCancel(jobID string) error {
	return a.transfers.Cancel(jobID)
}

// TransferReveal shows a finished download in Finder, Explorer or the
// Linux file manager.
func (a *App) TransferReveal(path string) error {
	return a.transfers.RevealDownload(a.ctx, path)
}
