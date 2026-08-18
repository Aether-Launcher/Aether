package mojang

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
)

const ManifestURL = "https://launchermeta.mojang.com/mc/game/version_manifest_v2.json"

var versionManifestURL = ManifestURL

type VersionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []VersionEntry `json:"versions"`
}

type VersionEntry struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// VersionInfo represents the full version metadata from Mojang
type VersionInfo struct {
	ID        string `json:"id"`
	MainClass string `json:"mainClass"`
	Type      string `json:"type"`
	Downloads struct {
		Client struct {
			URL  string `json:"url"`
			Size int64  `json:"size"`
			Sha1 string `json:"sha1"`
		} `json:"client"`
	} `json:"downloads"`
	Libraries          []Library  `json:"libraries"`
	AssetIndex         AssetIndex `json:"assetIndex"`
	Assets             string     `json:"assets"`
	MinecraftArguments string     `json:"minecraftArguments"`
	Arguments          struct {
		Game []json.RawMessage `json:"game"`
		JVM  []json.RawMessage `json:"jvm"`
	} `json:"arguments"`
	JavaVersion struct {
		Component    string `json:"component"`
		MajorVersion int    `json:"majorVersion"`
	} `json:"javaVersion"`
	Logging struct {
		Client struct {
			Argument string `json:"argument"`
			File     struct {
				ID   string `json:"id"`
				Sha1 string `json:"sha1"`
				Size int64  `json:"size"`
				URL  string `json:"url"`
			} `json:"file"`
			Type string `json:"type"`
		} `json:"client"`
	} `json:"logging"`
}

type AssetIndex struct {
	ID        string `json:"id"`
	Sha1      string `json:"sha1"`
	Size      int64  `json:"size"`
	TotalSize int64  `json:"totalSize"`
	URL       string `json:"url"`
}

type Library struct {
	Name      string `json:"name"`
	Downloads struct {
		Artifact struct {
			Path string `json:"path"`
			URL  string `json:"url"`
			Size int64  `json:"size"`
			Sha1 string `json:"sha1"`
		} `json:"artifact"`
	} `json:"downloads"`
	Rules []Rule `json:"rules,omitempty"`
}

type Rule struct {
	Action string `json:"action"`
	OS     *struct {
		Name string `json:"name,omitempty"`
		Arch string `json:"arch,omitempty"`
	} `json:"os,omitempty"`
	Features map[string]bool `json:"features,omitempty"`
}

// ArgumentRule represents a conditional argument entry in the arguments arrays
type ArgumentRule struct {
	Rules []Rule          `json:"rules"`
	Value json.RawMessage `json:"value"` // can be string or []string
}

// ruleMatchesPlatform reports whether a Mojang rule applies to this process.
func ruleMatchesPlatform(rule Rule) bool {
	if len(rule.Features) > 0 {
		return false
	}
	if rule.OS == nil {
		return true
	}
	if rule.OS.Name != "" && rule.OS.Name != getOSName() {
		return false
	}
	if rule.OS.Arch != "" && rule.OS.Arch != runtime.GOARCH && !(rule.OS.Arch == "x86" && runtime.GOARCH == "386") {
		return false
	}
	return true
}

// IsLibraryAllowed checks if a library should be included based on its rules and the current OS.
func IsLibraryAllowed(lib Library) bool {
	if len(lib.Rules) == 0 {
		return true // No rules means always allowed
	}

	osName := getOSName()
	allowed := false

	for _, rule := range lib.Rules {
		if rule.OS == nil {
			// Rule without OS applies to all platforms
			if rule.Action == "allow" {
				allowed = true
			} else {
				allowed = false
			}
			continue
		}

		if rule.OS.Name == osName {
			if rule.Action == "allow" {
				allowed = true
			} else {
				allowed = false
			}
		}
	}

	return allowed
}

// getOSName returns the Mojang-format OS name
func getOSName() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "osx"
	case "linux":
		return "linux"
	default:
		return runtime.GOOS
	}
}

// ResolveArguments parses a raw argument array (game or jvm) and returns the resolved string arguments.
// It evaluates OS rules and skips conditional arguments that don't apply.
func ResolveArguments(rawArgs []json.RawMessage) []string {
	var result []string
	for _, raw := range rawArgs {
		// Try as plain string first
		var str string
		if err := json.Unmarshal(raw, &str); err == nil {
			result = append(result, str)
			continue
		}

		// Try as a conditional argument with rules
		var argRule ArgumentRule
		if err := json.Unmarshal(raw, &argRule); err != nil {
			continue
		}

		allowed := len(argRule.Rules) == 0
		for _, rule := range argRule.Rules {
			if ruleMatchesPlatform(rule) {
				allowed = rule.Action == "allow"
			}
		}
		if !allowed {
			continue
		}

		// Value can be a string or []string
		var singleVal string
		if err := json.Unmarshal(argRule.Value, &singleVal); err == nil {
			result = append(result, singleVal)
			continue
		}

		var multiVal []string
		if err := json.Unmarshal(argRule.Value, &multiVal); err == nil {
			result = append(result, multiVal...)
		}
	}

	return result
}

// GetVersionManifest fetches the master manifest
func GetVersionManifest() (*VersionManifest, error) {
	resp, err := http.Get(versionManifestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var manifest VersionManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// GetVersionInfo fetches the specific JSON for a version
func GetVersionInfo(versionID string) (*VersionInfo, error) {
	manifest, err := GetVersionManifest()
	if err != nil {
		return nil, err
	}

	var targetURL string
	for _, v := range manifest.Versions {
		if v.ID == versionID {
			targetURL = v.URL
			break
		}
	}

	if targetURL == "" {
		return nil, fmt.Errorf("version %s not found in manifest", versionID)
	}

	resp, err := http.Get(targetURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
