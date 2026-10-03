# Extensions Guide

> `.aex` packages and the Goja backend API described here are implemented. An `.aex` is a zip archive with an Aether-specific file extension; install `.aex` files through the launcher rather than using a `.zip` file. The CLI and SDK are maintained in separate repositories. Install the CLI with `go install github.com/Aether-Launcher/aether-cli@latest` and add the SDK to an extension project with `npm install --save-dev @aethermc/sdk`.
>
> Want to change Aether's appearance without writing an extension? Themes are CSS and image packages. See the [Themes Guide](THEMES.md).

## Architecture Overview
An extension has two parts:

1. **Backend (`main.js`)** runs in a Goja JavaScript runtime. It has no DOM, and can call only the launcher APIs allowed by its manifest permissions.
2. **Frontend (`ui/index.html`)** is shown in an iframe inside Aether. A local HTTP server serves the extension's UI files.

The runtime limits which launcher APIs an extension can use, but it is not an operating-system security boundary. Read the [Security Guide](SECURITY.md) before relying on it to protect sensitive data.

## How to Build an Extension

1. Create a project directory and add a `manifest.json`.
2. Write the backend entry point in `main.js`.
3. Add the interface files under `ui/`, starting with `ui/index.html`.
4. From `main.js`, use `Aether.ui.registerSidebarPage()` to make the page available in Aether.

```javascript
// main.js
Aether.ui.registerSidebarPage({
    id: "my-custom-page",
    label: "My Page",
    url: "ui/index.html" // Relative to the extension folder
});
```

## Packaging and Installation

An extension package can include documentation, an icon, and other assets alongside its required files. For example:

```text
my-extension.aex
├── manifest.json
├── main.js
├── README.md
├── LICENSE
├── CHANGELOG.md
├── icon.png
├── ui/
│   ├── index.html
│   ├── style.css
│   └── script.js
└── assets/
```

Install an `.aex` package from the Extensions page or Extension Gallery. Although the package uses the zip format internally, the launcher expects the `.aex` extension. During development, you can place an extracted extension folder in Aether's `extensions` data directory. Aether uses a local `.aether` directory when one is present; otherwise, it uses the platform's configuration directory.

## Manifest
Each extension needs a `manifest.json` at its root. For example:
```json
{
  "id": "com.example.myextension",
  "name": "My Extension",
  "version": "1.0.0",
  "author": "Your Name",
  "description": "Adds a cool new feature to Aether.",
  "main": "main.js",
  "api": "1.0",
  "pinToSidebar": false,
  "permissions": [
    "ui:sidebar",
    "instances:list",
    "mods:install"
  ]
}
```

`pinToSidebar` is optional and defaults to `false`. Registered pages without an approved pin stay under **Active Extensions**. Set it to `true` to ask for permanent links in the main sidebar; the extension must also request `ui:sidebar` to register pages. Aether asks the user when the extension is installed or updated. Approval or denial is remembered for that version, and a later version can ask again. Declining the pin does not remove the extension's page from **Active Extensions**.

### Optional Registry Metadata

The registry may also store metadata such as compatibility ranges and project links:

```json
{
  "minApi": "1.0",
  "maxApi": "2.0",
  "homepage": "https://example.com/myextension",
  "repository": "https://github.com/example/myextension",
  "license": "MIT",
  "keywords": ["mods", "fabric"]
}
```

The registry can store these fields for discovery and future tooling. The launcher does not currently enforce the API ranges or use the project links when it loads an extension.

## Permissions
Extensions only receive the APIs they request in `manifest.json`. Ask for the permissions your extension needs, and avoid requesting unrelated access.
- `ui:sidebar`: Register extension pages that render your `ui/index.html` in an iframe. If the manifest also sets `pinToSidebar` to `true`, Aether asks the user whether to pin those pages directly in the sidebar when the extension is installed or updated.
- `ui:dialogs`: Exposes the current dialog stub; a functional dialog API is planned.
- `instances:list`: List installed instances.
- `mods:list`: List mods in an instance.
- `mods:install`: Install a mod after launcher confirmation.
- `mods:delete`: Delete a mod after launcher confirmation.
- `mods:toggle`: Enable or disable a mod after launcher confirmation.
- `network:http`: Make backend HTTP GET requests to allowed hosts.
- `fs:download`: Download files to the shared `libraries` directory.
- `launcher:modloader`: Register a launch-time mod-loader callback.
- `skin:export`: Write base64 data below the shared `skins` directory.
- `account:read`: Read the safe active-account subset (`signedIn`, `id`, `username`, `type`). Tokens are never exposed.
- `skin:manage`: List owned Mojang skins (`Aether.skins.listMine`) and upload/apply skins (`upload`, `applyUrl`) after user confirmation.
- `cape:manage`: List owned Mojang capes and equip/hide the active cape after user confirmation.
- `discord:presence`: Update Discord Rich Presence via `Aether.discord.setActivity` / `clearActivity` and subscribe to `Aether.events.on('instance:state')`.
- `servers:list`: Read `servers.dat` from instances and ping servers (`Aether.servers.list` / `Aether.servers.ping`).
- `servers:manage`: Create, list, delete, and read/write files inside `servers/<id>/` directories. Deleting a server asks for user confirmation.
- `servers:process`: Start, stop, query, and send console input to supervised server processes (`Aether.servers.start` / `stop` / `status` / `send`), with log streaming via `server:log`. EULA acceptance requires explicit user confirmation (`Aether.servers.acceptEula`). Max 2 concurrent servers.
- `saves:list`: List singleplayer worlds of an instance (`Aether.instances.listWorlds`). Read-only world metadata (names, last played, game mode).
- `instances:launch`: Launch the game, optionally quick-connecting to a server or world (`Aether.instances.launchToServer` / `launchToWorld`). Granted at install time; launching is a visible user action so no per-click confirmation is shown.

The older `instances:patch` permission remains available for compatibility and grants the current instance and mod capabilities. New extensions should request the more specific permissions listed above.

## Extension UI Rules
Your extension UI runs in an `<iframe>`, so you can build it with React, Vue, Svelte, Solid, Lit, or plain HTML and CSS. Aether does not require a particular frontend framework.

For a more seamless experience, consider using colors and patterns that sit comfortably alongside Aether's dark interface.

## Examples
The `extensions-src/` directory contains sample extensions, including the Modrinth Browser and Fabric mod loader.

## Trust Tiers
The registry can attach a trust label to an extension, and Aether displays that label in the interface. A label describes registry metadata; it is not a security guarantee or a substitute for reviewing the code:

1. **Official**: The registry identifies the extension as maintained by the Aether team.
2. **Verified**: The registry assigns this label. It should not be read as a promise that the current launcher performed a security audit.
3. **Community**: The registry lists the extension as community-maintained. Review its permissions and source before installing.
4. **Local**: Installed from a local `.aex` file. If its manifest ID matches a registry entry, Aether may display the registry's label instead.

The launcher does not currently enforce a maintainer review process or automated code analysis. Treat all extensions according to their source and requested permissions, regardless of badge.

## Aether CLI

The CLI is maintained in a separate repository. It currently supports four project commands: `init`, `dev`, `validate`, and `build`, plus `help`. You can run them as `aether`, `aet`, or `aether-cli`.

Install it with npm:

```bash
npm install -g @aethermc/cli
```

Or install it with Go:

```bash
go install github.com/Aether-Launcher/aether-cli/...@latest
```

### Start a project

Create an extension project by providing its display name and ID:

```bash
aether init my-extension com.example.myextension
```

To create a theme instead, add `--theme`:

```bash
aether init my-theme com.example.mytheme --theme
```

### Develop and validate

Run `aether dev` from the project directory to deploy the extension to a local Aether installation and watch for file changes. Aether must be running.

Run `aether validate` to check an extension project. For a theme, use `aether validate --theme`. The CLI can detect a theme when it finds `package.json` and no `manifest.json`.

### Build a package

Run `aether build` for an extension or `aether build --theme` for a theme. The CLI validates the project and creates a zip-format `.aex` or `.theme` package. It excludes `.git/`, `node_modules/`, and existing archives.

For command options and the latest behavior, see the [Aether CLI repository](https://github.com/Aether-Launcher/aether-cli).
