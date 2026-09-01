package instance

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
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

type cfManifest struct {
	Minecraft struct {
		Version    string `json:"version"`
		ModLoaders []struct {
			ID string `json:"id"`
		} `json:"modLoaders"`
	} `json:"minecraft"`
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Author    string   `json:"author"`
	Files     []cfFile `json:"files"`
	Overrides string   `json:"overrides"`
}

type cfFile struct {
	ProjectID int  `json:"projectID"`
	FileID    int  `json:"fileID"`
	Required  bool `json:"required"`
}

type cfFileInfo struct {
	Data struct {
		FileName    string `json:"fileName"`
		DownloadURL string `json:"downloadUrl"`
	} `json:"data"`
}

// ModpackInstallProgress is called after each file is written to disk.
type ModpackInstallProgress func(done, total int, file string)

// InstallMrpack downloads a .mrpack or CurseForge .zip archive from packURL,
// creates a new Aether instance named packName (falls back to the pack's own name if empty),
// downloads all declared mod files, extracts the overrides layer, and returns the created instance.
//
// The returned instance is NOT yet Minecraft-installed — the caller should
// queue the launcher's Install pipeline for the new instance afterwards.
func InstallMrpack(ctx context.Context, packURL, packName, targetRoot string, onProgress ModpackInstallProgress) (*Instance, error) {
	// ── 1. Download the archive to a temp file ────────────────────────────
	tmpFile, err := os.CreateTemp("", "aether-pack-*.zip")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, packURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
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
		return nil, fmt.Errorf("write pack archive: %w", err)
	}
	tmpFile.Close()

	// ── 2. Open zip, detect pack type ─────────────────────────────────────
	zr, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("not a valid pack archive: %w", err)
	}
	defer zr.Close()

	var mrIdx mrpackIndex
	var cfIdx cfManifest
	isMrpack := false
	isCurseForge := false

	for _, f := range zr.File {
		if f.Name == "modrinth.index.json" {
			rc, openErr := f.Open()
			if openErr == nil {
				if json.NewDecoder(rc).Decode(&mrIdx) == nil && mrIdx.Game != "" {
					isMrpack = true
				}
				rc.Close()
			}
		} else if f.Name == "manifest.json" {
			rc, openErr := f.Open()
			if openErr == nil {
				if json.NewDecoder(rc).Decode(&cfIdx) == nil && cfIdx.Minecraft.Version != "" {
					isCurseForge = true
				}
				rc.Close()
			}
		}
	}

	if !isMrpack && !isCurseForge {
		return nil, fmt.Errorf("archive contains neither modrinth.index.json nor valid CurseForge manifest.json")
	}

	var mcVersion, loader, defaultName string
	var overrideDirName string

	if isMrpack {
		mcVersion = mrIdx.Dependencies["minecraft"]
		if mcVersion == "" {
			return nil, fmt.Errorf("pack declares no 'minecraft' dependency")
		}
		loader = "Vanilla"
		for dep := range mrIdx.Dependencies {
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
		defaultName = mrIdx.Name
		overrideDirName = "overrides/"
	} else {
		mcVersion = cfIdx.Minecraft.Version
		loader = "Vanilla"
		if len(cfIdx.Minecraft.ModLoaders) > 0 {
			rawLoader := strings.ToLower(cfIdx.Minecraft.ModLoaders[0].ID)
			if strings.Contains(rawLoader, "fabric") {
				loader = "Fabric"
			} else if strings.Contains(rawLoader, "neoforge") {
				loader = "NeoForge"
			} else if strings.Contains(rawLoader, "forge") {
				loader = "Forge"
			} else if strings.Contains(rawLoader, "quilt") {
				loader = "Quilt"
			}
		}
		defaultName = cfIdx.Name
		overrideDirName = "overrides/"
		if cfIdx.Overrides != "" {
			overrideDirName = strings.TrimSuffix(cfIdx.Overrides, "/") + "/"
		}
	}

	// ── 3. Create the Aether instance ─────────────────────────────────────
	name := packName
	if name == "" {
		name = defaultName
	}
	inst, err := Create(name, mcVersion, loader)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}
	instanceDir := filepath.Join(targetRoot, inst.ID)

	var filesToDownload []mrpackFile

	if isMrpack {
		filesToDownload = mrIdx.Files
	} else {
		// Convert CurseForge files to download tasks via proxy
		for _, file := range cfIdx.Files {
			apiURL := fmt.Sprintf("https://curseforge-proxy.cribest7890.workers.dev/v1/mods/%d/files/%d", file.ProjectID, file.FileID)
			req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
			if reqErr != nil {
				continue
			}
			req.Header.Set("X-Launcher-Token", "f2269b54edcf1439afa4675c23f8d3aefb3828ef622e3c9741171501cf5e1bfe")
			req.Header.Set("User-Agent", "Mozilla/5.0")
			resp, respErr := http.DefaultClient.Do(req)
			if respErr != nil {
				continue
			}
			var info cfFileInfo
			if json.NewDecoder(resp.Body).Decode(&info) == nil && info.Data.DownloadURL != "" {
				fileName := info.Data.FileName
				if fileName == "" {
					fileName = fmt.Sprintf("mod-%d-%d.jar", file.ProjectID, file.FileID)
				}
				filesToDownload = append(filesToDownload, mrpackFile{
					Path:      filepath.Join("mods", fileName),
					Downloads: []string{info.Data.DownloadURL},
				})
			}
			resp.Body.Close()
		}
	}

	total := len(filesToDownload) + 1
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

	// ── 4. Download mod files concurrently (up to 4 at a time) ───────────
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	for _, mf := range filesToDownload {
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
				fmt.Printf("[Pack] skipping unsafe path: %s\n", mf.Path)
				progress(mf.Path)
				return
			}
			if mkErr := os.MkdirAll(filepath.Dir(destPath), 0755); mkErr == nil {
				if dlErr := mrpackDownloadFile(ctx, mf.Downloads[0], destPath); dlErr != nil {
					fmt.Printf("[Pack] failed to download %s: %v\n", mf.Path, dlErr)
				}
			}
			progress(mf.Path)
		}()
	}
	wg.Wait()

	// ── 5. Extract overrides layer from the zip ───────────────────────────
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, overrideDirName) || f.FileInfo().IsDir() {
			continue
		}
		rel := strings.TrimPrefix(f.Name, overrideDirName)
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

func mrpackDownloadFile(ctx context.Context, targetURL, dest string) error {
	parsedURL, _ := neturl.Parse(targetURL)

	doDownload := func(u string) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
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

	dlErr := doDownload(targetURL)
	if dlErr != nil && parsedURL != nil && parsedURL.Hostname() != "" {
		host := parsedURL.Hostname()
		if host == "edge.forgecdn.net" || host == "media.forgecdn.net" {
			altHost := "media.forgecdn.net"
			if host == "media.forgecdn.net" {
				altHost = "edge.forgecdn.net"
			}
			altURL := strings.Replace(targetURL, host, altHost, 1)
			return doDownload(altURL)
		}
	}
	return dlErr
}
