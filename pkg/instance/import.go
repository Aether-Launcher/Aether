package instance

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"Aether/pkg/fs"
)

// Format identifies the launcher an instance folder came from.
type Format string

const (
	FormatNative     Format = "native"
	FormatMultiMC    Format = "multimc"
	FormatCurseForge Format = "curseforge"
	FormatUnknown    Format = ""
)

// ImportProgress is called with the number of files copied so far, the total
// file count, and the current file's mapped destination path.
type ImportProgress func(done, total int, file string)

// mmcPack mirrors the Prism/MultiMC mmc-pack.json structure.
type mmcPack struct {
	Components []struct {
		UID     string `json:"uid"`
		Version string `json:"version"`
	} `json:"components"`
}

// curseforgeManifest mirrors the CurseForge app manifest.json structure.
type curseforgeManifest struct {
	Minecraft struct {
		Version    string `json:"version"`
		ModLoaders []struct {
			ID string `json:"id"`
		} `json:"modLoaders"`
	} `json:"minecraft"`
}

// DetectFormat identifies which launcher produced the given instance folder.
func DetectFormat(source string) Format {
	if _, err := os.Stat(filepath.Join(source, "instance.json")); err == nil {
		return FormatNative
	}
	if _, err := os.Stat(filepath.Join(source, "mmc-pack.json")); err == nil {
		return FormatMultiMC
	}
	if data, err := os.ReadFile(filepath.Join(source, "manifest.json")); err == nil {
		var m curseforgeManifest
		if json.Unmarshal(data, &m) == nil && m.Minecraft.Version != "" {
			return FormatCurseForge
		}
	}
	return FormatUnknown
}

// ImportInstance imports an instance folder from any supported launcher into
// targetRoot (the Aether instances directory). User content (mods, config,
// saves) is preserved; launcher-managed binaries are dropped so the Install
// flow re-downloads them into Aether's layout. It returns the created
// instance manifest.
func ImportInstance(source, targetRoot string, onProgress ImportProgress) (*Instance, error) {
	format := DetectFormat(source)
	if format == FormatUnknown {
		return nil, fmt.Errorf("this doesn't look like an Aether, Prism/MultiMC, or CurseForge instance folder")
	}

	inst := &Instance{Loader: "Vanilla", Memory: "4G", LastPlayed: "Never"}
	baseName := filepath.Base(source)

	switch format {
	case FormatNative:
		data, err := os.ReadFile(filepath.Join(source, "instance.json"))
		if err != nil {
			return nil, fmt.Errorf("invalid instance: missing instance.json: %w", err)
		}
		if err := json.Unmarshal(data, inst); err != nil {
			return nil, fmt.Errorf("invalid instance manifest: %w", err)
		}
		if inst.ID == "" {
			inst.ID = baseName
		}
		if inst.Name == "" {
			inst.Name = baseName
		}
		if inst.Version == "" {
			return nil, fmt.Errorf("instance manifest must include a version")
		}
	case FormatMultiMC:
		version, loader, name, memory, err := parseMultiMC(source)
		if err != nil {
			return nil, err
		}
		inst.Version = version
		inst.Loader = loader
		inst.Name = name
		if name == "" {
			inst.Name = baseName
		}
		if memory != "" {
			inst.Memory = memory
		}
		inst.ID = baseName
	case FormatCurseForge:
		version, loader, err := parseCurseForge(source)
		if err != nil {
			return nil, err
		}
		inst.Version = version
		inst.Loader = loader
		inst.Name = baseName
		inst.ID = baseName
	}

	if inst.Version == "" {
		return nil, fmt.Errorf("could not determine the Minecraft version for this instance")
	}

	inst.ID = uniqueInstanceID(inst.ID, targetRoot)
	target := filepath.Join(targetRoot, inst.ID)
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, err
	}

	if err := copyInstanceTree(source, target, copyPlanFor(format), onProgress); err != nil {
		_ = os.RemoveAll(target)
		return nil, fmt.Errorf("failed to import instance: %w", err)
	}

	// Installed state is recomputed from bin/<version>.jar on load; foreign
	// imports always re-download binaries via the Install flow.
	inst.Installed = false

	data, err := json.MarshalIndent(inst, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(target, "instance.json"), data, 0644); err != nil {
		return nil, err
	}
	return inst, nil
}

// parseMultiMC reads mmc-pack.json (version + loader components) and
// instance.cfg (display name, memory).
func parseMultiMC(source string) (version, loader, name, memory string, err error) {
	data, err := os.ReadFile(filepath.Join(source, "mmc-pack.json"))
	if err != nil {
		return "", "", "", "", err
	}
	var pack mmcPack
	if err := json.Unmarshal(data, &pack); err != nil {
		return "", "", "", "", fmt.Errorf("invalid mmc-pack.json: %w", err)
	}
	loader = "Vanilla"
	for _, c := range pack.Components {
		switch c.UID {
		case "net.minecraft":
			version = c.Version
		case "net.fabricmc.fabric-loader":
			loader = "fabric"
		case "net.minecraftforge":
			loader = "forge"
		case "net.neoforged":
			loader = "neoforge"
		case "net.quiltmc.quilt-loader":
			loader = "quilt"
		}
	}

	if data, err := os.ReadFile(filepath.Join(source, "instance.cfg")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name=") {
				name = strings.TrimSpace(strings.TrimPrefix(line, "Name="))
			}
			if strings.HasPrefix(line, "Memory=") {
				memory = strings.TrimSpace(strings.TrimPrefix(line, "Memory="))
			}
			if strings.HasPrefix(line, "MaxMemAlloc=") {
				memory = strings.TrimSpace(strings.TrimPrefix(line, "MaxMemAlloc="))
			}
		}
	}
	return version, loader, name, memory, nil
}

// parseCurseForge reads the CurseForge app manifest.json (version + first
// mod loader, e.g. "forge-47.2.0" -> "forge").
func parseCurseForge(source string) (version, loader string, err error) {
	data, err := os.ReadFile(filepath.Join(source, "manifest.json"))
	if err != nil {
		return "", "", err
	}
	var m curseforgeManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return "", "", fmt.Errorf("invalid manifest.json: %w", err)
	}
	version = m.Minecraft.Version
	if len(m.Minecraft.ModLoaders) > 0 {
		id := strings.ToLower(strings.TrimSpace(m.Minecraft.ModLoaders[0].ID))
		if i := strings.IndexByte(id, '-'); i > 0 {
			loader = id[:i]
		} else if id != "" {
			loader = id
		}
	}
	return version, loader, nil
}

// uniqueInstanceID returns a folder ID that does not collide with any
// existing instance, appending -2, -3, ... when needed.
func uniqueInstanceID(base, targetRoot string) string {
	id := slug(base)
	if _, err := os.Stat(filepath.Join(targetRoot, id)); os.IsNotExist(err) {
		return id
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", id, i)
		if _, err := os.Stat(filepath.Join(targetRoot, candidate)); os.IsNotExist(err) {
			return candidate
		}
	}
}

// slug converts a folder name into a safe instance ID.
func slug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return "imported-instance"
	}
	// Keep only allow-list characters; other runs become a single dash.
	s = regexpMustCompile(`[^a-z0-9._-]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-._")
	s = regexpMustCompile(`-+`).ReplaceAllString(s, "-")
	if s == "" {
		return "imported-instance"
	}
	if s[0] == '.' || s[0] == '_' || s[0] == '-' {
		s = "a" + s
	}
	if len(s) > 65 {
		s = s[:65]
		s = strings.TrimRight(s, "-._")
	}
	return s
}

// regexpMustCompile is a tiny helper to avoid importing regexp at top for a single use.
// Defined here to keep import list minimal for tests that don't need regexp.
func regexpMustCompile(expr string) *regexp.Regexp {
	r, _ := regexp.Compile(expr)
	return r
}

// copyPlanFor returns the copy rules for a launcher format.
func copyPlanFor(format Format) copyPlan {
	switch format {
	case FormatNative:
		// Native instances keep everything, including bin/ and libraries/.
		return copyPlan{remap: func(rel string) (string, bool) { return rel, true }}
	case FormatMultiMC:
		skipped := map[string]bool{
			"libraries": true, "patches": true, "cache": true,
			"logs": true, "crash-reports": true, "bin": true,
		}
		remapped := map[string]string{
			"mods":               "mods",
			"config":             "config",
			"resourcepacks":      "resourcepacks",
			"saves":              "saves",
			"options.txt":        "options.txt",
			"servers.dat":        "servers.dat",
			"usernamecache.json": "usernamecache.json",
		}
		return copyPlan{remap: func(rel string) (string, bool) {
			root := rel
			if i := strings.IndexByte(rel, '/'); i > 0 {
				root = rel[:i]
			}
			if skipped[root] {
				return "", false
			}
			if root == "minecraft" {
				rest := strings.TrimPrefix(rel, "minecraft/")
				for prefix, mapped := range remapped {
					if rest == prefix {
						return mapped, true
					}
					if strings.HasPrefix(rest, prefix+"/") {
						return mapped + "/" + strings.TrimPrefix(rest, prefix+"/"), true
					}
				}
				return "", false // jar, natives, logs — re-downloaded
			}
			return rel, true
		}}
	case FormatCurseForge:
		skipped := map[string]bool{"logs": true, "crash-reports": true}
		return copyPlan{remap: func(rel string) (string, bool) {
			root := rel
			if i := strings.IndexByte(rel, '/'); i > 0 {
				root = rel[:i]
			}
			if skipped[root] {
				return "", false
			}
			// Extracted modpack zips keep their overrides/ separate; merge
			// them into the instance root like the CurseForge app does.
			if rel == "overrides" {
				return "", false
			}
			if strings.HasPrefix(rel, "overrides/") {
				return strings.TrimPrefix(rel, "overrides/"), true
			}
			return rel, true
		}}
	}
	return copyPlan{remap: func(rel string) (string, bool) { return rel, true }}
}

// copyPlan decides how source paths map into the target instance.
type copyPlan struct {
	remap func(rel string) (string, bool)
}

// copyInstanceTree streams the source tree into target, applying the plan's
// skip/remap rules. Symlinks pointing inside the source are followed; links
// to external locations (e.g. shared libraries) are skipped with a warning.
func copyInstanceTree(source, target string, plan copyPlan, onProgress ImportProgress) error {
	total := 0
	_ = filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			total++
		}
		return nil
	})

	done := 0
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		mapped, keep := plan.remap(rel)
		if !keep {
			return nil
		}
		dest, err := fs.ContainedPath(target, mapped)
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return os.MkdirAll(dest, 0755)
		}

		if entry.Type()&os.ModeSymlink != 0 {
			return copySymlink(source, path, dest)
		}
		if err := copyFile(path, dest); err != nil {
			return err
		}
		done++
		if onProgress != nil {
			onProgress(done, total, mapped)
		}
		return nil
	})
}

// copySymlink follows a symlink when it resolves to a regular file inside the
// source tree; external links are skipped.
func copySymlink(source, path, dest string) error {
	resolved := path
	if link, err := os.Readlink(path); err == nil {
		if filepath.IsAbs(link) {
			resolved = link
		} else {
			resolved = filepath.Join(filepath.Dir(path), link)
		}
	}
	eval, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		fmt.Printf("[Import] skipping broken symlink %s\n", path)
		return nil
	}
	rel, err := filepath.Rel(source, eval)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		fmt.Printf("[Import] skipping external symlink %s\n", path)
		return nil
	}
	info, err := os.Stat(eval)
	if err != nil || info.IsDir() {
		fmt.Printf("[Import] skipping directory/broken symlink %s\n", path)
		return nil
	}
	return copyFile(eval, dest)
}

// copyFile streams src into dest without loading it into memory.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
