package java

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// RequiredJavaVersion returns the minimum Java major version needed
// for a given Minecraft version string.
func RequiredJavaVersion(mcVersion string) int {
	parts := strings.Split(mcVersion, ".")
	if len(parts) < 2 {
		return 8
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 8
	}

	// 1.20.5+ requires Java 21
	if minor >= 20 {
		if len(parts) >= 3 {
			patch, _ := strconv.Atoi(parts[2])
			if minor > 20 || patch >= 5 {
				return 21
			}
		}
		// 1.20.0–1.20.4 still needs 17
		return 17
	}
	// 1.17–1.20.4 requires Java 17
	if minor >= 17 {
		return 17
	}
	// 1.16 and below requires Java 8
	return 8
}

// GetMajorVersion returns the major version number of the given java binary.
// e.g. "1.8.0_301" → 8, "17.0.5" → 17, "21.0.1" → 21
func GetMajorVersion(javaPath string) (int, error) {
	cmd := newHiddenCommand(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, err
	}

	raw := string(output)

	// Match both: version "1.8.0_301" and version "21.0.1"
	re := regexp.MustCompile(`version "(\d+)(?:\.(\d+))?`)
	m := re.FindStringSubmatch(raw)
	if m == nil {
		return 0, fmt.Errorf("could not parse java version from: %s", raw)
	}

	major, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, err
	}

	// Java 8 reports as "1.8.x" — major will be 1, minor will be 8
	if major == 1 && len(m) > 2 {
		minor, err := strconv.Atoi(m[2])
		if err == nil {
			return minor, nil
		}
	}
	return major, nil
}

// FindJava locates a working Java binary that satisfies minVersion.
// Checks managed runtimes first, then JAVA_HOME, then PATH, then
// common Windows install directories.
func FindJava(minVersion int) (string, error) {
	// Check JAVA_HOME
	javaHome := os.Getenv("JAVA_HOME")
	if javaHome != "" {
		for _, name := range []string{"java.exe", "java"} {
			candidate := filepath.Join(javaHome, "bin", name)
			if isCompatible(candidate, minVersion) {
				return candidate, nil
			}
		}
	}

	// Check PATH
	if pathJava, err := exec.LookPath("java"); err == nil {
		if isCompatible(pathJava, minVersion) {
			return pathJava, nil
		}
	}

	// Common Windows install paths
	for _, base := range []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
	} {
		if base == "" {
			continue
		}
		for _, candidate := range findJavaInDir(base) {
			if isCompatible(candidate, minVersion) {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("no Java >= %d found on this system", minVersion)
}

// isCompatible returns true if the binary exists, runs, and meets minVersion.
func isCompatible(path string, minVersion int) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	v, err := GetMajorVersion(path)
	if err != nil {
		return false
	}
	return v >= minVersion
}

// findJavaInDir searches for java.exe in common Java installation subdirectories.
func findJavaInDir(baseDir string) []string {
	var results []string
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return results
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.ContainsAny(name, "j") &&
			(strings.Contains(name, "java") || strings.Contains(name, "jdk") ||
				strings.Contains(name, "jre") || strings.Contains(name, "adoptium") ||
				strings.Contains(name, "zulu") || strings.Contains(name, "graalvm") ||
				strings.Contains(name, "temurin")) {
			candidate := filepath.Join(baseDir, entry.Name(), "bin", "java.exe")
			if _, err := os.Stat(candidate); err == nil {
				results = append(results, candidate)
			}
		}
	}
	return results
}
