package instance

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// mrpackIndex mirrors the modrinth.index.json format inside a .mrpack archive.
type mrpackIndex struct {
	FormatVersion int               `json:"formatVersion"`
	Game          string            `json:"game"`
	Name          string            `json:"name"`
	Summary       string            `json:"summary"`
	Files         []mrpackFile      `json:"files"`
	Dependencies  map[string]string `json:"dependencies"`
}

type mrpackFile struct {
	Path      string            `json:"path"`
	Hashes    map[string]string `json:"hashes"`
	Downloads []string          `json:"downloads"`
	FileSize  int64             `json:"fileSize"`
}

// ModpackInstallProgress is called after each file is written to disk.
type ModpackInstallProgress func(done, total int, file string)

// InstallMrpack downloads a .mrpack archive from packURL, creates a new Aether
// instance named packName (falls back to the pack's own name if empty),
// downloads all declared mod files from the index concurrently, extracts the
// overrides layer, and returns the created instance.
//
// The returned instance is NOT yet Minecraft-installed — the caller should
// queue the launcher's Install pipeline for the new instance afterwards.
func InstallMrpack(ctx context.Context, packURL, packName, targetRoot string, onProgress ModpackInstallProgress) (*Instance, error) {
	// ── 1. Download the .mrpack to a temp file ────────────────────────────
	tmpFile, err := os.CreateTemp("", "aether-mrpack-*.mrpack")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, packURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: HTTP %s", resp.Status)
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("write mrpack: %w", err)
	}
	tmpFile.Close()

	// ── 2. Open zip, parse modrinth.index.json ────────────────────────────
	zr, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("not a valid mrpack archive: %w", err)
	}
	defer zr.Close()

	var idx mrpackIndex
	for _, f := range zr.File {
		if f.Name == "modrinth.index.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			decErr := json.NewDecoder(rc).Decode(&idx)
			rc.Close()
			if decErr != nil {
				return nil, fmt.Errorf("invalid modrinth.index.json: %w", decErr)
			}
			break
		}
	}
	if idx.Game == "" {
		return nil, fmt.Errorf("modrinth.index.json not found in pack archive")
	}

	// ── 3. Resolve Minecraft version + mod loader ─────────────────────────
	mcVersion := idx.Dependencies["minecraft"]
	if mcVersion == "" {
		return nil, fmt.Errorf("pack declares no 'minecraft' dependency")
	}
	loader := "Vanilla"
	for dep := range idx.Dependencies {
		switch strings.ToLower(dep) {
		case "fabric-loader":
			loader = "Fabric"
		case "forge":
			loader = "Forge"
		case "quilt-loader":
			loader = "Quilt"
		case "neoforge":
			loader = "NeoForge"
		}
	}

	// ── 4. Create the Aether instance ─────────────────────────────────────
	name := packName
	if name == "" {
		name = idx.Name
	}
	inst, err := Create(name, mcVersion, loader)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}
	instanceDir := filepath.Join(targetRoot, inst.ID)
	total := len(idx.Files) + 1 // +1 for the overrides pass
	var done int
	var mu sync.Mutex

	progress := func(file string) {
		mu.Lock()
		done++
		d := done
		mu.Unlock()
		if onProgress != nil {
			onProgress(d, total, file)
		}
	}

	// ── 5. Download mod files concurrently (up to 4 at a time) ───────────
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, mf := range idx.Files {
		if len(mf.Downloads) == 0 {
			progress(mf.Path)
			continue
		}
		mf := mf
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			destPath := filepath.Join(instanceDir, filepath.FromSlash(mf.Path))
			rel, relErr := filepath.Rel(instanceDir, destPath)
			if relErr != nil || strings.HasPrefix(rel, "..") {
				fmt.Printf("[Mrpack] skipping unsafe path: %s\n", mf.Path)
				progress(mf.Path)
				return
			}
			if mkErr := os.MkdirAll(filepath.Dir(destPath), 0755); mkErr == nil {
				if dlErr := mrpackDownloadFile(ctx, mf.Downloads[0], destPath); dlErr != nil {
					fmt.Printf("[Mrpack] failed to download %s: %v\n", mf.Path, dlErr)
				}
			}
			progress(mf.Path)
		}()
	}
	wg.Wait()

	// ── 6. Extract overrides layer from the zip ───────────────────────────
	const overridePrefix = "overrides/"
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, overridePrefix) || f.FileInfo().IsDir() {
			continue
		}
		rel := strings.TrimPrefix(f.Name, overridePrefix)
		destPath := filepath.Join(instanceDir, filepath.FromSlash(rel))
		relCheck, relErr := filepath.Rel(instanceDir, destPath)
		if relErr != nil || strings.HasPrefix(relCheck, "..") {
			continue
		}
		if mkErr := os.MkdirAll(filepath.Dir(destPath), 0755); mkErr != nil {
			continue
		}
		rc, openErr := f.Open()
		if openErr != nil {
			continue
		}
		out, createErr := os.Create(destPath)
		if createErr != nil {
			rc.Close()
			continue
		}
		io.Copy(out, rc)
		out.Close()
		rc.Close()
	}
	progress("overrides")

	return inst, nil
}

func mrpackDownloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
