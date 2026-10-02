// app_java.go - Java runtime status and download Wails bindings.

package main

import (
	"github.com/Aether-Launcher/Aether/pkg/java"
)

// JavaRuntimeStatus describes the status of a managed or system Java runtime.
type JavaRuntimeStatus struct {
	Version   int    `json:"version"`
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
	IsSystem  bool   `json:"isSystem"`
}

// GetJavaStatus returns the installation status for each required Java version.
func (a *App) GetJavaStatus() []JavaRuntimeStatus {
	versions := []int{8, 17, 21}
	var statuses []JavaRuntimeStatus
	for _, v := range versions {
		installed := java.IsManagedJavaInstalled(v)
		path := ""
		isSystem := false
		if installed {
			path = java.GetManagedJavaPath(v)
		} else if sysPath, err := java.FindJava(v); err == nil {
			installed = true
			path = sysPath
			isSystem = true
		}
		statuses = append(statuses, JavaRuntimeStatus{
			Version:   v,
			Installed: installed,
			Path:      path,
			IsSystem:  isSystem,
		})
	}
	return statuses
}

// DownloadJavaRuntime downloads a managed JRE for the given major version.
func (a *App) DownloadJavaRuntime(version int) error {
	return java.DownloadJava(a.ctx, version)
}
