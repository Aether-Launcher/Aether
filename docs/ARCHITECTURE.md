# Architecture

## Backend
The launcher backend is written in Go to ensure high performance, memory safety, and native compilation across Windows, macOS, and Linux. The backend operates completely headlessly and serves the frontend via Wails bindings. The frontend is a lightweight web view rendering the UI.

## Go Packages
- `main.go`: The main entry point.
- `pkg/instance`: Logic for managing Minecraft instances, resolving dependencies, and constructing launch arguments.
- `pkg/java`: Discovery, installation, and management of Java Runtimes (JRE/JDK).
- `pkg/auth`: Offline account storage and deterministic offline UUID generation.
- `pkg/extensions`: The extension manager and sandbox environment.

## Extension Manager
The Extension Manager (`pkg/extensions`) is responsible for discovering, installing, and executing extensions. Manifest parsing and install-time ID checks are implemented; a broader extension validator is planned.
Extensions are executed in an isolated JavaScript runtime (Goja). They do not have direct access to the host OS. Any action an extension wants to perform must be requested through the launcher's API, which enforces the capability-based permission model.

## Planned Features
An updater is described in the longer-term architecture but is not implemented in the current codebase. Launcher and extension updates are currently manual.

## Launcher Pipeline
1. **Resolution**: Determine the Minecraft version, loader (Fabric, Forge, etc.), and required libraries.
2. **Verification**: Check if all assets, libraries, and the Java runtime are present. Download missing files.
3. **Authentication**: Load the selected offline account and deterministic UUID.
4. **Extension Load**: `LoadAll()` performs fast metadata scan; sandbox creation deferred via `reloadSandboxes()` background goroutine. `UpdateExtension`/`ReloadExtensions` calls `ReloadAsync()` which returns immediately, triggers sandbox script reload with 10s timeout.
5. **Execution**: Construct the massive Java command line and spawn the child process.
6. **Monitoring**: Emit process state and log events to the Wails frontend.

## Diagrams

### Launcher Startup & Extension Loading

```mermaid
sequenceDiagram
    participant OS as Operating System
    participant Core as Aether Core (Go)
    participant UI as Aether UI (Svelte)
    participant ExtManager as Extension Manager
    participant Sandbox as Goja Sandbox

    OS->>Core: Launch Aether
    Core->>UI: Start Wails Webview
    Core->>ExtManager: Initialize()
    
    loop For each Extension
        ExtManager->>ExtManager: Read manifest.json
        ExtManager->>Sandbox: Create new isolated runtime
        ExtManager->>Sandbox: Inject permitted Aether APIs
        ExtManager->>Sandbox: Execute main.js
        Sandbox->>ExtManager: Aether.ui.registerSidebarPage()
        ExtManager->>UI: Emit 'extension:sidebar:add' event
    end
    
    UI-->>User: Render fully loaded UI
```

### Permission Validation

```mermaid
sequenceDiagram
    participant Sandbox as Goja Sandbox
    participant API as Aether Core API
    
    Sandbox->>API: Aether.instances.installMod(id, file, url)
    API->>API: Check Extension Manifest
    
    alt Has granular mod permission
        API-->>UI: Request user confirmation
        UI-->>API: Approve or reject
        API-->>Sandbox: Continue or throw error
    else Missing permission
        API-->>Sandbox: Throw Error ("Permission Denied")
    end
```
