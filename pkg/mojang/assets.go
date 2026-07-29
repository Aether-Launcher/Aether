package mojang

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"Aether/pkg/netutil"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"context"
)

// AssetIndexData represents the parsed asset index JSON
type AssetIndexData struct {
	Objects map[string]AssetObject `json:"objects"`
}

type AssetObject struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// DownloadAssets fetches the asset index and downloads all game assets to a shared directory.
func (e *DownloadEngine) DownloadAssets(ctx context.Context, assetIndex AssetIndex, assetsDir string) error {
	// Download the asset index JSON
	indexDir := filepath.Join(assetsDir, "indexes")
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return err
	}

	indexPath := filepath.Join(indexDir, assetIndex.ID+".json")
	if err := netutil.DownloadFile(ctx, assetIndex.URL, indexPath, nil); err != nil {
		return fmt.Errorf("failed to download asset index: %w", err)
	}

	// Parse the asset index
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	var indexData AssetIndexData
	if err := json.Unmarshal(data, &indexData); err != nil {
		return fmt.Errorf("failed to parse asset index: %w", err)
	}

	// Deduplicate assets by hash to avoid concurrent write collisions on identical files
	uniqueAssets := make(map[string]AssetObject)
	for _, obj := range indexData.Objects {
		uniqueAssets[obj.Hash] = obj
	}

	// Download all unique asset objects concurrently
	objectsDir := filepath.Join(assetsDir, "objects")
	totalAssets := len(uniqueAssets)
	var completed int
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := make(chan error, totalAssets)
	sem := make(chan struct{}, 15) // Limit concurrency

	for hash, obj := range uniqueAssets {
		prefix := hash[:2]
		objPath := filepath.Join(objectsDir, prefix, hash)

		// Skip if already downloaded
		if info, err := os.Stat(objPath); err == nil && info.Size() == obj.Size {
			mu.Lock()
			completed++
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(h string, assetObj AssetObject, dest string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			url := fmt.Sprintf("https://resources.download.minecraft.net/%s/%s", h[:2], h)
			if err := netutil.DownloadFile(ctx, url, dest, nil, h); err != nil {
				errors <- fmt.Errorf("asset %s: %w", h, err)
				return
			}

			mu.Lock()
			completed++
			pct := float64(completed) / float64(totalAssets) * 100
			mu.Unlock()

			// Only emit progress every ~5% to avoid flooding the event bus
			if int(pct)%5 == 0 {
				runtime.EventsEmit(ctx, "instance:progress", map[string]interface{}{
					"id":       e.instance,
					"progress": pct,
					"status":   fmt.Sprintf("Downloading assets (%d/%d)", completed, totalAssets),
				})
			}
		}(hash, obj, objPath)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			return err
		}
	}

	return nil
}
