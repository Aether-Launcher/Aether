# Aether SDK

> The CLI and `@aethermc/sdk` package are maintained outside this core launcher repository. The Go runtime in this repository remains the source of truth for which APIs are actually injected into extension sandboxes.

The Aether SDK (`@aethermc/sdk`) is the development companion for building Aether extensions. It is intended as a development-time package and does not ship inside an extension at runtime.

## CLI vs SDK

These are two separate but connected tools:

- **CLI** (`aether-cli`) — what you run in your terminal. Scaffolds projects, runs dev mode, packages extensions.
- **SDK** (`@aethermc/sdk`) — what you import inside your extension's code. Provides types and helper utilities.

## The Problem It Solves

The `Aether` global object is injected directly into every extension's sandbox by the Go runtime. Without the SDK, the developer experience is rough:

- No autocompletion — editors don't know what `Aether.ui` or `Aether.instances` contains.
- No type safety — a typo like `Aether.instances.patcH()` is only caught at runtime.
- No friendly error messages — raw sandbox errors are cryptic.

The SDK fixes all of this without changing the runtime.

## What the SDK Contains

### 1. TypeScript Definitions

The most important part. A `.d.ts` file that tells your editor exactly what every Aether API accepts and returns. No code from this runs at runtime — it is purely for your editor:

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

Thin wrappers that make common patterns cleaner and catch mistakes earlier:

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

## Versioning

The `"api"` field in `manifest.json` ties directly to the SDK version:

```json
{
  "api": "1.0"
}
```

When you install `@aethermc/sdk@1.0`, you get types and helpers that match Aether's runtime API at that version exactly. When Aether ships a new API version, `aether-cli migrate` updates your manifest and you run:

```
npm install @aethermc/sdk@2.0
```

## What the SDK Does NOT Do

- It does not ship any JavaScript into your packaged extension.
- It does not replace the `Aether` global — that is always injected by the Go runtime.
- It does not add a runtime dependency to your `.aex` file.

The Go runtime is the single source of truth for what APIs actually exist. The SDK is a development-time mirror of that truth.

## Full Developer Flow

```
aether-cli init
  └─ scaffolds project, installs @aethermc/sdk

Edit main.js
  └─ full autocomplete and type checking via SDK

aether dev
  └─ connects to a running Aether instance, hot-reloads on save

aether-cli validate
  └─ checks your manifest against the declared SDK/API version

aether-cli build
  └─ bundles into a .aex file, SDK types stripped automatically
```

## Installation

```bash
npm install --save-dev @aethermc/sdk
```

Use the SDK package from the SDK repository for local development. The TypeScript definitions in that repository should mirror the Go runtime APIs exposed by this launcher.
