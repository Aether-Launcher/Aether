# Aether SDK

> The CLI and `@aethermc/sdk` live in separate repositories. This launcher repository defines the runtime API: the Go code determines which capabilities are available in an extension sandbox.

The Aether SDK (`@aethermc/sdk`) provides editor support while you build an extension. It is a development dependency and is not bundled into the extension at runtime.

## CLI vs SDK

The [CLI](https://github.com/Aether-Launcher/aether-cli) and [SDK](https://github.com/Aether-Launcher/Aether-SDK) serve different purposes:

- **CLI** (`aether-cli`) is the terminal tool for scaffolding projects, running development mode, and packaging extensions.
- **SDK** (`@aethermc/sdk`) is the package you import in extension code for type definitions and helper utilities.

Choose sidebar placement in the extension manifest, not in the SDK call. Set `"pinToSidebar": true` to keep registered pages directly in the sidebar. Otherwise, Aether groups them under **Active Extensions**. See the [Extensions Guide](EXTENSIONS.md#manifest) for an example.

## Why use the SDK?

The Go runtime adds an `Aether` object to each extension sandbox. You can use it without the SDK, but your editor will not know its API types. That means:

- Editors cannot autocomplete `Aether.ui` or `Aether.instances`.
- Typos such as `Aether.instances.patcH()` are only caught at runtime.
- Sandbox errors can be difficult to interpret.

The SDK supplies those editor aids without changing the runtime itself.

## What the SDK Contains

### 1. TypeScript Definitions

The SDK's TypeScript declarations tell your editor what the Aether APIs accept and return. They are used for type checking and autocomplete; they do not run in the extension:

```typescript
declare global {
  const Aether: {
    ui: {
      registerSidebarPage(opts: { id: string; label: string; url: string }): void;
      openDialog(opts: any): void;
    };
    instances: {
      list(): { id: string; name: string; version: string; loader: string }[];
      installMod(instanceId: string, jarName: string, downloadURL: string): string;
    };
    http: {
      get(url: string): string;
    };
    fs: {
      download(url: string, destPath: string): string;
    };
    launcher: {
      registerModLoader(config: { id: string; name: string; description: string; onLaunch: (ctx: any) => any }): void;
    };
    skins: {
      export(base64Data: string, filename: string): string;
    };
  };
}
```

### 2. Helper Utilities

The SDK also includes small helpers for common tasks:

```javascript
import { onReady, createLogger } from '@aethermc/sdk';

const log = createLogger('my-extension');

onReady(() => {
  log.info('Extension started');

  Aether.ui.registerSidebarPage({
    id: 'my-page',
    label: 'My Page',
    url: 'ui/index.html',
  });
});
```

Available helpers:

| Helper | Purpose |
|---|---|
| `onReady(fn)` | Runs your function once the sandbox is fully initialised |
| `createLogger(name)` | Returns a namespaced logger (`log.info`, `log.warn`, `log.error`) |
| `defineProvider(spec)` | Registers a typed Loader Provider with validation |
| `assertPermission(perm)` | Throws a clear error if a required permission was not declared |
| `createIframeBridge(timeoutMs?)` | Handles sidebar iframe requests and responses, including `requestId` matching, timeouts, and `postMessage` details |

## API compatibility

The `"api"` field in `manifest.json` describes the extension API version:

```json
{
  "api": "1.0"
}
```

The SDK package has its own version number, separate from the manifest's `api` value. This launcher does not currently negotiate API versions or enforce compatibility ranges. Check the SDK release notes and this launcher's API documentation when choosing a version.

## What the SDK does not do

- It does not ship any JavaScript into your packaged extension.
- It does not replace the `Aether` global. The Go runtime injects that object.
- It does not add a runtime dependency to your `.aex` file.

The Go runtime remains the source of truth for available APIs. The SDK mirrors that API for development; if the two disagree, follow the runtime.

## Full Developer Flow

```
aether-cli init
  └─ scaffolds project, installs @aethermc/sdk

Edit main.js
  └─ full autocomplete and type checking via SDK

aether-cli dev
  └─ connects to a running Aether instance, hot-reloads on save

aether-cli validate
  └─ checks the manifest and required project files

aether-cli build
  └─ packages the project into a .aex file without node_modules
```

## Installation

```bash
npm install --save-dev @aethermc/sdk
```

Install the SDK from its [repository](https://github.com/Aether-Launcher/Aether-SDK) when developing an extension. Its TypeScript definitions are intended to mirror the APIs exposed by this launcher's Go runtime.
