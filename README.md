# Aether Launcher

<p align="center">
  <img src="frontend/public/logo.png" alt="Aether Logo" width="120" />
</p>

<p align="center">
  <em>A minimal, extensible, lightning-fast Minecraft launcher.</em>
</p>

## Overview

Aether is designed with a single core principle: **This launcher exists to launch Minecraft, and nothing more.**

Every feature that is not essential to launching the game—like downloading mods, managing servers, or viewing logs—is implemented as an extension. By keeping the core launcher intentionally minimal, Aether remains fast, predictable, and free from bloat. Think of it as the VS Code of Minecraft launchers. You install only the features you actually want.

## Features

- **Blazing Fast**: Cold starts in under 2 seconds with an idle memory footprint below 100MB.
- **Minimalist UI**: Clean, native-feeling, elegant design that gets out of your way, complete with smooth custom toast notifications.
- **Snapshot Support**: Effortlessly toggle between stable releases and latest snapshots when creating instances.
- **Extensible Architecture**: Everything from Modrinth integration to server browsers is an extension.
- **Capability-Based Extensions**: Extension backend code runs in a restricted Goja runtime and can only use the launcher APIs granted by its manifest permissions.

## Documentation

All project documentation is located in the `docs/` directory. If you are looking to contribute, build an extension, or just understand how Aether works, start here:

- **[Project Philosophy & Design](docs/DESIGN.md)** - The core principles guiding Aether's development.
- **[Architecture](docs/ARCHITECTURE.md)** - Overview of the Go backend, Extension Manager, and launcher pipeline.
- **[API & Interoperability](docs/API.md)** - Details on the extension API and permission model.
- **[Extensions Guide](docs/EXTENSIONS.md)** - How to build, package, and publish extensions for Aether.
  - *Looking for the official extension registry? Visit the [Aether-Extensions](https://github.com/wayback09/Aether-Extensions) repository.*
- **[Security & Sandboxing](docs/SECURITY.md)** - Threat models, capability isolation, and review guidelines.
- **[UI Specifications](docs/UI.md)** - Layout rules, components, and empty states.
- **[Styleguide](docs/STYLEGUIDE.md)** - Visual language, typography, colors, and animations.

## Developer Tooling

The following tools are available for building Aether extensions. They are maintained outside this core launcher repository and are available for terminal installation:

- **Aether SDK** (`@aether/sdk`) - TypeScript type definitions and helper utilities for extension development. Install with `npm install --save-dev @aether/sdk`.
- **Aether CLI** (`aether`) - Scaffold, validate, and package extensions into `.aex` format. Install with `go install github.com/wayback09/aether-cli@latest`.

## Getting Started

Download the latest release for your platform from the [Releases](../../releases) page.

| Platform | File |
|---|---|
| Windows (x64) | `Aether-windows-amd64-installer.exe` |
| macOS (Apple Silicon) | `Aether-macos-arm64.dmg` |
| Linux (x64) | `Aether-linux-amd64` |

## Building from Source

If you'd like to build Aether from source, ensure you have the following installed:
- [Go 1.22+](https://go.dev/doc/install)
- [Node.js 20+](https://nodejs.org/)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)

```bash
# Clone the repository
git clone https://github.com/wayback09/Aether.git
cd Aether
# Run in development mode (live reload)
wails dev

# Or build a production binary
wails build
```

## License

The Aether launcher and first-party tooling are licensed under the GNU GPL v3.0-only. See [LICENSE](LICENSE).
