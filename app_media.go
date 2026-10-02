// app_media.go - screenshots, instance icons and folder Wails bindings.

package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/Aether-Launcher/Aether/pkg/fs"
	"github.com/Aether-Launcher/Aether/pkg/instance"
)

// ScreenshotItem represents an in-game screenshot taken in an instance.
type ScreenshotItem struct {
	InstanceID   string `json:"instanceId"`
	InstanceName string `json:"instanceName"`
	FileName     string `json:"fileName"`
	DataURL      string `json:"dataUrl"`
	ModTime      string `json:"modTime"`
}

// GetRecentScreenshots finds the newest screenshots across all instances.
func (a *App) GetRecentScreenshots(limit int) ([]ScreenshotItem, error) {
	if limit <= 0 {
		limit = 4
	}
	instances := instance.GetInstances()
	type fileEntry struct {
		instanceID   string
		instanceName string
		fileName     string
		fullPath     string
		modTime      time.Time
	}
	var allFiles []fileEntry

	for _, inst := range instances {
		instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), inst.ID)
		if err != nil {
			continue
		}
		screenshotsDir := filepath.Join(instanceDir, "screenshots")
		entries, err := os.ReadDir(screenshotsDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			allFiles = append(allFiles, fileEntry{
				instanceID:   inst.ID,
				instanceName: inst.Name,
				fileName:     e.Name(),
				fullPath:     filepath.Join(screenshotsDir, e.Name()),
				modTime:      info.ModTime(),
			})
		}
	}

	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].modTime.After(allFiles[j].modTime)
	})

	if len(allFiles) > limit {
		allFiles = allFiles[:limit]
	}

	results := make([]ScreenshotItem, 0, len(allFiles))
	for _, f := range allFiles {
		data, err := os.ReadFile(f.fullPath)
		if err != nil {
			continue
		}
		mime := "image/png"
		if strings.HasSuffix(strings.ToLower(f.fileName), ".jpg") || strings.HasSuffix(strings.ToLower(f.fileName), ".jpeg") {
			mime = "image/jpeg"
		}
		dataURL := fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(data))
		results = append(results, ScreenshotItem{
			InstanceID:   f.instanceID,
			InstanceName: f.instanceName,
			FileName:     f.fileName,
			DataURL:      dataURL,
			ModTime:      f.modTime.Format(time.RFC3339),
		})
	}

	return results, nil
}

// OpenScreenshot opens a specific screenshot file using the default operating system viewer.
func (a *App) OpenScreenshot(instanceID, fileName string) error {
	fileName = filepath.Base(fileName)
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return err
	}
	imgPath, err := fs.ContainedPath(filepath.Join(instanceDir, "screenshots"), fileName)
	if err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", imgPath)
	case "darwin":
		cmd = exec.Command("open", imgPath)
	default:
		cmd = exec.Command("xdg-open", imgPath)
	}
	return cmd.Start()
}

// PickInstanceIcon opens a file dialog for a PNG/JPEG image, validates it,
// and saves it as the instance's custom icon. Returns the icon as a data:
// URL for immediate display.
func (a *App) PickInstanceIcon(instanceID string) (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose Instance Icon",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Images (*.png;*.jpg;*.jpeg)",
				Pattern:     "*.png;*.jpg;*.jpeg",
			},
		},
	})
	if err != nil {
		return "", err
	}
	if file == "" {
		// User cancelled
		return "", nil
	}
	if err := instance.SetIconFromFile(instanceID, file); err != nil {
		return "", err
	}
	return instance.GetInstanceIcon(instanceID)
}

// RemoveInstanceIcon deletes an instance's custom icon, reverting to default art.
func (a *App) RemoveInstanceIcon(instanceID string) error {
	return instance.RemoveInstanceIcon(instanceID)
}

// GetInstanceIcon returns the instance's custom icon as a data: URL,
// or "" when it uses default art.
func (a *App) GetInstanceIcon(instanceID string) string {
	url, _ := instance.GetInstanceIcon(instanceID)
	return url
}

// GetInstanceIcons returns id -> data URL for every instance with a custom
// icon, so lists can render icons with a single backend call.
func (a *App) GetInstanceIcons() map[string]string {
	return instance.GetInstanceIcons()
}

// OpenInstanceFolder opens the instance's root folder in Explorer / Finder.
func (a *App) OpenInstanceFolder(instanceID string) error {
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return err
	}
	if st, err := os.Stat(instanceDir); err != nil || !st.IsDir() {
		return fmt.Errorf("instance folder not found: %s", instanceID)
	}
	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", instanceDir)
	case "darwin":
		cmd = exec.Command("open", instanceDir)
	default:
		cmd = exec.Command("xdg-open", instanceDir)
	}
	return cmd.Start()
}

// OpenScreenshotsFolder opens the screenshots folder of an instance in Explorer / Finder.
func (a *App) OpenScreenshotsFolder(instanceID string) error {
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return err
	}
	screenshotsDir := filepath.Join(instanceDir, "screenshots")
	if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
		return err
	}
	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", screenshotsDir)
	case "darwin":
		cmd = exec.Command("open", screenshotsDir)
	default:
		cmd = exec.Command("xdg-open", screenshotsDir)
	}
	return cmd.Start()
}
