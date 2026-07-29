package mojang

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"Aether/pkg/netutil"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DownloadEngine handles fetching assets concurrently
type DownloadEngine struct {
	ctx      context.Context
	instance string
	basePath string
}

func NewDownloadEngine(ctx context.Context, instanceID, basePath string) *DownloadEngine {
	return &DownloadEngine{
		ctx:      ctx,
		instance: instanceID,
		basePath: basePath,
	}
}

// Install processes the VersionInfo and downloads everything needed to launch
func (e *DownloadEngine) Install(info *VersionInfo, assetsDir string) error {
	// Count only the libraries to download + client jar
	allowedLibs := []Library{}
	for _, lib := range info.Libraries {
		if lib.Downloads.Artifact.URL != "" && IsLibraryAllowed(lib) {
			allowedLibs = append(allowedLibs, lib)
		}
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(allowedLibs)+5)

	totalItems := len(allowedLibs) + 1 // +1 for client jar
	var completed int
	var mu sync.Mutex

	reportProgress := func(name string, phase string) {
		mu.Lock()
		completed++
		pct := float64(completed) / float64(totalItems) * 100
		mu.Unlock()

		runtime.EventsEmit(e.ctx, "instance:progress", map[string]interface{}{
			"id":       e.instance,
			"progress": pct,
			"status":   fmt.Sprintf("[%s] %s", phase, name),
		})
	}

	// Download Client Jar
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientPath := filepath.Join(e.basePath, "bin", fmt.Sprintf("%s.jar", info.ID))
		if err := netutil.DownloadFile(e.ctx, info.Downloads.Client.URL, clientPath, nil, info.Downloads.Client.Sha1); err != nil {
			errors <- err
			return
		}
		reportProgress("client.jar", "Core")
	}()

	// Download Libraries concurrently
	sem := make(chan struct{}, 10)

	for _, lib := range allowedLibs {
		wg.Add(1)
		go func(l Library) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			libPath := filepath.Join(e.basePath, "libraries", l.Downloads.Artifact.Path)
			if err := netutil.DownloadFile(e.ctx, l.Downloads.Artifact.URL, libPath, nil, l.Downloads.Artifact.Sha1); err != nil {
				errors <- err
				return
			}
			reportProgress(l.Name, "Libraries")
		}(lib)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return fmt.Errorf("download error: %w", err)
		}
	}

	// Download Assets
	runtime.EventsEmit(e.ctx, "instance:progress", map[string]interface{}{
		"id":       e.instance,
		"progress": 50,
		"status":   "Downloading game assets...",
	})

	if err := e.DownloadAssets(e.ctx, info.AssetIndex, assetsDir); err != nil {
		return fmt.Errorf("asset download error: %w", err)
	}

	// Extract Native Libraries
	runtime.EventsEmit(e.ctx, "instance:progress", map[string]interface{}{
		"id":       e.instance,
		"progress": 90,
		"status":   "Extracting native libraries...",
	})

	nativesDir := filepath.Join(e.basePath, "natives")
	librariesDir := filepath.Join(e.basePath, "libraries")
	if err := ExtractNatives(info.Libraries, librariesDir, nativesDir); err != nil {
		return fmt.Errorf("native extraction error: %w", err)
	}

	// Download Log4j config file
	if info.Logging.Client.File.URL != "" {
		logConfigPath := filepath.Join(e.basePath, info.Logging.Client.File.ID)
		if err := netutil.DownloadFile(e.ctx, info.Logging.Client.File.URL, logConfigPath, nil); err != nil {
			fmt.Printf("Warning: failed to download log config: %v\n", err)
			// Non-fatal, continue
		}
	}

	// Save version.json to disk (required for launcher to resolve arguments)
	versionData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal version info: %w", err)
	}
	versionPath := filepath.Join(e.basePath, "version.json")
	if err := os.WriteFile(versionPath, versionData, 0644); err != nil {
		return fmt.Errorf("failed to save version.json: %w", err)
	}

	runtime.EventsEmit(e.ctx, "instance:progress", map[string]interface{}{
		"id":       e.instance,
		"progress": 100,
		"status":   "Installation Complete",
	})

	return nil
}


