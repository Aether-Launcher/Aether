package instance

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"Aether/pkg/auth"
	"Aether/pkg/fs"
	"Aether/pkg/java"
	"Aether/pkg/mojang"
	"Aether/pkg/settings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Launch spawns the Minecraft process with the correct arguments
func Launch(ctx context.Context, inst *Instance) error {
	instanceDir := filepath.Join(fs.GetDataDir(), "instances", inst.ID)
	assetsDir := fs.GetAssetsDir()

	// Determine the required Java version for this Minecraft version
	requiredJava := java.RequiredJavaVersion(inst.Version)
	fmt.Printf("[Launcher] Minecraft %s requires Java >= %d\n", inst.Version, requiredJava)

	var javaPath string

	// Fast path: use already-managed JRE if present
	if java.IsManagedJavaInstalled(requiredJava) {
		javaPath = java.GetManagedJavaPath(requiredJava)
		fmt.Printf("[Launcher] Using managed JRE: %s\n", javaPath)
	} else {
		// Try to find a compatible system Java
		systemJava, err := java.FindJava(requiredJava)
		if err == nil {
			javaPath = systemJava
			fmt.Printf("[Launcher] Using system Java: %s\n", javaPath)
		} else {
			// Download a managed JRE from Adoptium
			fmt.Printf("[Launcher] No compatible Java found, downloading Java %d...\n", requiredJava)
			if dlErr := java.DownloadJava(ctx, requiredJava); dlErr != nil {
				return fmt.Errorf("failed to download Java %d: %w", requiredJava, dlErr)
			}
			javaPath = java.GetManagedJavaPath(requiredJava)
		}
	}

	fmt.Printf("[Launcher] Using Java: %s\n", javaPath)

	// Load saved version.json
	versionPath := filepath.Join(instanceDir, "version.json")
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		return fmt.Errorf("version.json not found — is the instance installed? %w", err)
	}

	var versionInfo mojang.VersionInfo
	if err := json.Unmarshal(versionData, &versionInfo); err != nil {
		return fmt.Errorf("failed to parse version.json: %w", err)
	}

	// Build classpath
	classpath := buildClasspath(instanceDir, &versionInfo)

	// Build native directory path
	nativesDir := filepath.Join(instanceDir, "natives")

	// Build argument variable replacements
	activeAccount := auth.GetActiveAccount()
	username := "Player"
	uuid := auth.GenerateOfflineUUID(username)
	accessToken := "0"
	userType := "legacy"

	if activeAccount != nil {
		username = activeAccount.Username
		uuid = activeAccount.ID
		if activeAccount.Type == auth.TypeMicrosoft {
			userType = "msa"
			// Check if token is expired or close to expiration (within 5 minutes)
			if activeAccount.ExpiresAt == 0 || time.Now().Unix() > (activeAccount.ExpiresAt-300) {
				fmt.Println("[Launcher] Microsoft access token expired or expiring soon, refreshing...")
				refreshed, err := auth.RefreshMicrosoftToken(ctx, activeAccount)
				if err == nil {
					_ = auth.AddMicrosoftAccount(*refreshed)
					activeAccount = refreshed
				} else {
					fmt.Printf("[Launcher] Failed to refresh token: %v\n", err)
				}
			}
			accessToken = activeAccount.AccessToken
		}
	}

	vars := map[string]string{
		"${auth_player_name}":  username,
		"${version_name}":      versionInfo.ID,
		"${game_directory}":    instanceDir,
		"${assets_root}":       assetsDir,
		"${assets_index_name}": versionInfo.Assets,
		"${auth_uuid}":         uuid,
		"${auth_access_token}": accessToken,
		"${clientid}":          "0",
		"${auth_xuid}":         "0",
		"${user_type}":         userType,
		"${version_type}":      versionInfo.Type,
		"${natives_directory}": nativesDir,
		"${launcher_name}":     "Aether",
		"${launcher_version}":  "1.0",
		"${classpath}":         classpath,
		"${path}":              filepath.Join(instanceDir, versionInfo.Logging.Client.File.ID),
	}

	// Mod Loader Interception
	mainClass := versionInfo.MainClass
	cpArray := strings.Split(classpath, string(os.PathListSeparator))

	if inst.Loader != "" && inst.Loader != "Vanilla" && ModLoaderHook != nil {
		fmt.Printf("[Launcher] Intercepting launch with mod loader: %s\n", inst.Loader)

		hookCtx := map[string]interface{}{
			"instancePath": instanceDir,
			"mcVersion":    inst.Version,
			"classpath":    cpArray,
			"mainClass":    mainClass,
		}

		modified, err := ModLoaderHook(strings.ToLower(inst.Loader), hookCtx)
		if err != nil {
			return fmt.Errorf("mod loader failed: %w", err)
		}

		// Extract modified values
		if mc, ok := modified["mainClass"].(string); ok {
			mainClass = mc
		}

		if cpIface, ok := modified["classpath"].([]interface{}); ok {
			cpArray = make([]string, len(cpIface))
			for i, v := range cpIface {
				cpArray[i] = fmt.Sprint(v)
			}
		}

		// Rebuild classpath variable for substitution
		vars["${classpath}"] = strings.Join(cpArray, string(os.PathListSeparator))
	}

	// Resolve JVM arguments from version JSON
	jvmArgs := mojang.ResolveArguments(versionInfo.Arguments.JVM)
	jvmArgs = substituteVars(jvmArgs, vars)

	// Load global settings for fallbacks
	globalSettings := settings.Load()

	// Add memory flags
	memory := inst.Memory
	if memory == "" || memory == "Default" {
		memory = globalSettings.DefaultMemory
	}

	// If memory string doesn't specify M or G (like "4096"), append M
	if !strings.HasSuffix(strings.ToUpper(memory), "M") && !strings.HasSuffix(strings.ToUpper(memory), "G") {
		memory += "M"
	}

	prependedArgs := []string{"-Xmx" + memory, "-Xms512M"}

	// Add Garbage Collector selection if specified
	switch strings.ToUpper(globalSettings.GarbageCollector) {
	case "ZGC":
		prependedArgs = append(prependedArgs, "-XX:+UseZGC")
	case "SHENANDOAH":
		prependedArgs = append(prependedArgs, "-XX:+UseShenandoahGC")
	case "PARALLEL":
		prependedArgs = append(prependedArgs, "-XX:+UseParallelGC")
	case "G1GC":
		prependedArgs = append(prependedArgs, "-XX:+UseG1GC")
	}

	// Add custom JVM arguments if set
	if globalSettings.CustomJVMArgs != "" {
		customFields := strings.Fields(globalSettings.CustomJVMArgs)
		prependedArgs = append(prependedArgs, customFields...)
	}

	jvmArgs = append(prependedArgs, jvmArgs...)

	// Add log4j config if available
	if versionInfo.Logging.Client.File.URL != "" {
		logConfigPath := filepath.Join(instanceDir, versionInfo.Logging.Client.File.ID)
		if _, err := os.Stat(logConfigPath); err == nil {
			logArg := strings.Replace(versionInfo.Logging.Client.Argument, "${path}", logConfigPath, 1)
			jvmArgs = append(jvmArgs, logArg)
		}
	}

	// Resolve game arguments from version JSON
	gameArgs := mojang.ResolveArguments(versionInfo.Arguments.Game)
	gameArgs = substituteVars(gameArgs, vars)

	// Construct full command: java [jvm args] mainClass [game args]
	args := append(jvmArgs, mainClass)
	args = append(args, gameArgs...)

	fmt.Printf("[Launcher] Command: %s %s\n", javaPath, strings.Join(args, " "))

	cmd := exec.Command(javaPath, args...)
	cmd.Dir = instanceDir

	// Get pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Minecraft: %w", err)
	}

	// Notify frontend that the instance is running
	runtime.EventsEmit(ctx, "instance:state", map[string]interface{}{
		"id":    inst.ID,
		"state": "Running",
	})

	if globalSettings.CloseOnLaunch {
		runtime.WindowHide(ctx)
	}

	// Async log readers
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			runtime.EventsEmit(ctx, "instance:log", scanner.Text())
			fmt.Println("[MC]", scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			runtime.EventsEmit(ctx, "instance:log", scanner.Text())
			fmt.Println("[MC]", scanner.Text())
		}
	}()

	// Wait for process to exit in a goroutine
	go func() {
		err := cmd.Wait()
		state := "Stopped"
		if err != nil {
			state = "Crashed"
			fmt.Printf("[Launcher] Minecraft exited with error: %v\n", err)
		}

		if globalSettings.CloseOnLaunch {
			runtime.WindowShow(ctx)
		}

		runtime.EventsEmit(ctx, "instance:state", map[string]interface{}{
			"id":    inst.ID,
			"state": state,
		})
	}()

	return nil
}

// buildClasspath constructs the Java classpath from installed libraries + client jar
func buildClasspath(instanceDir string, info *mojang.VersionInfo) string {
	var paths []string

	for _, lib := range info.Libraries {
		if !mojang.IsLibraryAllowed(lib) {
			continue
		}
		if lib.Downloads.Artifact.Path == "" {
			continue
		}

		libPath := filepath.Join(instanceDir, "libraries", lib.Downloads.Artifact.Path)
		if _, err := os.Stat(libPath); err == nil {
			paths = append(paths, libPath)
		}
	}

	// Add the client jar
	clientJar := filepath.Join(instanceDir, "bin", info.ID+".jar")
	paths = append(paths, clientJar)

	return strings.Join(paths, string(os.PathListSeparator))
}

// substituteVars replaces ${variable} placeholders in arguments
func substituteVars(args []string, vars map[string]string) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		for k, v := range vars {
			arg = strings.ReplaceAll(arg, k, v)
		}
		result[i] = arg
	}
	return result
}
