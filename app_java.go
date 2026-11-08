// app_java.go - Java runtime status and download Wails bindings.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/Aether-Launcher/Aether/pkg/fs"
	"github.com/Aether-Launcher/Aether/pkg/instance"
	"github.com/Aether-Launcher/Aether/pkg/java"
)

// JavaRuntimeStatus describes the status of a managed or system Java runtime.
type JavaRuntimeStatus struct {
	Version   int    `json:"version"`
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
	IsSystem  bool   `json:"isSystem"`
}

// javaStatusVersions returns the Java majors the settings page should show:
// the base majors Aether knows about, every managed runtime on disk, and every
// major required by an installed instance. The instance version.json value is
// authoritative because modded versions can override it.
func javaStatusVersions() []int {
	seen := map[int]bool{8: true, 17: true, 21: true}

	for _, v := range java.ManagedJavaVersions() {
		seen[v] = true
	}

	for _, inst := range instance.GetInstances() {
		seen[java.RequiredJavaVersion(inst.Version)] = true

		versionPath := filepath.Join(fs.GetDataDir(), "instances", inst.ID, "version.json")
		data, err := os.ReadFile(versionPath)
		if err != nil {
			continue
		}

		var info struct {
			JavaVersion struct {
				MajorVersion int `json:"majorVersion"`
			} `json:"javaVersion"`
		}
		if err := json.Unmarshal(data, &info); err == nil && info.JavaVersion.MajorVersion > 0 {
			seen[info.JavaVersion.MajorVersion] = true
		}
	}

	versions := make([]int, 0, len(seen))
	for v := range seen {
		versions = append(versions, v)
	}
	sort.Ints(versions)
	return versions
}

// GetJavaStatus returns the installation status for each required Java version.
func (a *App) GetJavaStatus() []JavaRuntimeStatus {
	var statuses []JavaRuntimeStatus
	for _, v := range javaStatusVersions() {
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
