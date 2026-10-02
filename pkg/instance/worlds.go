package instance

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Aether-Launcher/Aether/pkg/fs"
	"github.com/Aether-Launcher/Aether/pkg/servers"
)

// WorldInfo describes one singleplayer world (a folder under saves/).
type WorldInfo struct {
	ID         string `json:"id"` // saves folder name
	Name       string `json:"name"`
	LastPlayed int64  `json:"lastPlayed,omitempty"`
	GameMode   int32  `json:"gameMode,omitempty"`
}

// ListWorlds returns the singleplayer worlds of an instance, most recently
// played first. Worlds without a readable level.dat still appear (folder
// name only) so a corrupt world never hides the rest.
func ListWorlds(instanceID string) ([]WorldInfo, error) {
	instanceDir, err := fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), instanceID)
	if err != nil {
		return nil, err
	}
	return listWorldsIn(instanceDir)
}

func listWorldsIn(instanceDir string) ([]WorldInfo, error) {
	savesDir := filepath.Join(instanceDir, "saves")
	entries, err := os.ReadDir(savesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []WorldInfo{}, nil
		}
		return nil, err
	}
	var out []WorldInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info := WorldInfo{ID: e.Name(), Name: e.Name()}
		if level, err := readLevelDat(filepath.Join(savesDir, e.Name(), "level.dat")); err == nil {
			if level.Name != "" {
				info.Name = level.Name
			}
			info.LastPlayed = level.LastPlayed
			info.GameMode = level.GameMode
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastPlayed > out[j].LastPlayed })
	return out, nil
}

type levelSummary struct {
	Name       string
	LastPlayed int64
	GameMode   int32
}

// readLevelDat extracts the display name and metadata from a level.dat.
// Vanilla writes gzip-compressed NBT; very old files may be raw NBT.
func readLevelDat(path string) (levelSummary, error) {
	var out levelSummary
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if len(raw) > 8*1024*1024 {
		return out, fmt.Errorf("level.dat exceeds 8 MiB")
	}
	data := raw
	if len(raw) >= 2 && raw[0] == 0x1f && raw[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return out, err
		}
		defer zr.Close()
		data, err = io.ReadAll(io.LimitReader(zr, 8*1024*1024+1))
		if err != nil {
			return out, err
		}
	}
	root, err := servers.ParseRootCompound(data)
	if err != nil {
		return out, err
	}
	dataTag, _ := root["Data"].(map[string]any)
	if dataTag == nil {
		return out, fmt.Errorf("level.dat has no Data compound")
	}
	if name, ok := dataTag["LevelName"].(string); ok {
		out.Name = name
	}
	switch v := dataTag["LastPlayed"].(type) {
	case int64:
		out.LastPlayed = v
	case int32:
		out.LastPlayed = int64(v)
	}
	switch v := dataTag["GameType"].(type) {
	case int32:
		out.GameMode = v
	case byte:
		out.GameMode = int32(v)
	}
	return out, nil
}

// SupportsQuickPlay reports whether a Minecraft version accepts the
// --quickPlaySingleplayer flag (added in 1.20). Unknown version strings
// return false so old clients never receive flags they can't parse.
func SupportsQuickPlay(version string) bool {
	version = strings.TrimSpace(version)
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	return major > 1 || (major == 1 && minor >= 20)
}

// validateLaunchHost rejects empty hosts and values that could be parsed as
// extra CLI flags once appended to the game arguments.
func validateLaunchHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("server address is empty")
	}
	if strings.ContainsAny(host, " \t\r\n") {
		return "", fmt.Errorf("server address contains whitespace: %q", host)
	}
	if strings.HasPrefix(host, "-") {
		return "", fmt.Errorf("server address must not start with '-': %q", host)
	}
	return host, nil
}

// validateWorldFolder confines the world to a direct child of saves/ — no
// separators, no dot entries, no flag-like names — and requires level.dat.
func validateWorldFolder(instanceDir, world string) (string, error) {
	if world == "" || world == "." || world == ".." {
		return "", fmt.Errorf("invalid world name: %q", world)
	}
	if world != filepath.Base(world) {
		return "", fmt.Errorf("invalid world name: %q", world)
	}
	if strings.HasPrefix(world, "-") {
		return "", fmt.Errorf("invalid world name: %q", world)
	}
	worldDir, err := fs.ContainedPath(filepath.Join(instanceDir, "saves"), world)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(filepath.Join(worldDir, "level.dat")); err != nil || st.IsDir() {
		return "", fmt.Errorf("world %q has no level.dat", world)
	}
	return world, nil
}

func findInstanceByID(id string) (*Instance, error) {
	all := GetInstances()
	for i := range all {
		if all[i].ID == id {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("instance not found: %s", id)
}

func instanceDirOf(inst *Instance) (string, error) {
	return fs.ContainedPath(filepath.Join(fs.GetDataDir(), "instances"), inst.ID)
}

// LaunchToServer launches an instance and auto-connects to host:port via the
// vanilla --server/--port game flags (supported on all versions).
func LaunchToServer(ctx context.Context, instanceID, host string, port int) error {
	inst, err := findInstanceByID(instanceID)
	if err != nil {
		return err
	}
	host, err = validateLaunchHost(host)
	if err != nil {
		return err
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}
	return launch(ctx, inst, []string{"--server", host, "--port", strconv.Itoa(port)})
}

// LaunchToWorld launches an instance and auto-loads a singleplayer world via
// Mojang's Quick Play flags (requires Minecraft 1.20+).
func LaunchToWorld(ctx context.Context, instanceID, world string) error {
	inst, err := findInstanceByID(instanceID)
	if err != nil {
		return err
	}
	if !SupportsQuickPlay(inst.Version) {
		return fmt.Errorf("joining a world directly needs Minecraft 1.20 or newer (instance is %s)", inst.Version)
	}
	instanceDir, err := instanceDirOf(inst)
	if err != nil {
		return err
	}
	world, err = validateWorldFolder(instanceDir, world)
	if err != nil {
		return err
	}
	quickPlayDir := filepath.Join(instanceDir, "quickPlay")
	if err := os.MkdirAll(quickPlayDir, 0755); err != nil {
		return err
	}
	return launch(ctx, inst, []string{
		"--quickPlayPath", filepath.Join(quickPlayDir, "aether-quickplay.json"),
		"--quickPlaySingleplayer", world,
	})
}
