# Extension API & Sandbox

Aether executes extension backend scripts inside an isolated **Goja JavaScript runtime**. 

> [!WARNING]
> This backend environment is **NOT** a browser or a Node.js environment. The following APIs are unavailable:
> - `window`
> - `document`
> - `fetch` (unless explicitly provided via `Aether.http.get`)
> - `localStorage`
> - `require()`
> - `process`, `fs`, `child_process` (and all other Node.js modules)

Instead, your script interacts with the launcher core via the injected `Aether` global object.

## The `Aether` Global Object

When your extension's `main.js` is executed, the `Aether` object is injected into the global scope. The capabilities attached to this object depend strictly on the permissions requested in your `manifest.json`.

### UI Registration (`ui:sidebar`)
Allows the extension to register a frontend UI tab.

- `Aether.ui.registerSidebarPage(options)`
  - **options** (Object):
    - `id` (String): A unique identifier for the tab.
    - `label` (String): The text displayed on the tab.
    - `url` (String): The path to your UI HTML file, relative to your extension's root directory (e.g., `"ui/index.html"`).

### Dialogs (`ui:dialogs`)
The capability is present for compatibility, but `Aether.ui.openDialog()` is currently a stub and does not open a launcher dialog yet.

### Instance and Mod Management
`instances:list` allows the extension to query instances. The separate `mods:list`, `mods:install`, `mods:delete`, and `mods:toggle` permissions control access to files in their `mods` directories. Sensitive mod operations require confirmation in the launcher UI.

- `Aether.instances.list()`
  - Returns an array of objects representing all installed instances: `[{ id, name, version, loader }]`.
- `Aether.instances.installMod(instanceId, jarName, downloadURL)`
  - **instanceId** (String): The ID of the instance to modify.
  - **jarName** (String): The filename to save the mod as (e.g. `fabric-api.jar`).
  - **downloadURL** (String): The URL to download the mod from (must be allowed in `hosts`).
- `Aether.instances.listMods(instanceId)`
  - **instanceId** (String): The ID of the instance.
  - Returns an array of strings representing the filenames in the `mods` folder.
- `Aether.instances.deleteMod(instanceId, jarName)`
  - Deletes the specified mod file from the instance.
- `Aether.instances.toggleMod(instanceId, jarName, enable)`
  - **enable** (Boolean): True to enable, false to disable.
  - Disabling a mod renames it to `.jar.disabled`. Enabling it renames it back to `.jar`.

### Mod Loader Registration (`launcher:modloader`)
Allows the extension to register a custom mod loader that Aether can use to launch instances.

- `Aether.launcher.registerModLoader(config)`
  - **config** (Object):
    - `id` (String): A unique identifier for the loader.
    - `name` (String): The display name of the loader.
    - `description` (String): A brief description of the loader.
    - `onLaunch` (Function): A callback executed when an instance with this loader is launched.

### Skin Management (`skin:export`)
Allows the extension to write base64 encoded skins to the local filesystem.

- `Aether.skins.export(base64Data, filename)`
  - **base64Data** (String): The skin image encoded as a base64 string.
  - **filename** (String): The name to save the skin as (e.g., `skin.png`). Returns the saved file path.

## Network Access
By default, the backend Sandbox cannot access the network. To make requests, you must request `network:http` in your permissions and use the provided `Aether.http.get(url)` API. Backend extension requests require HTTPS, an allowed hostname, and are limited to 10 MiB responses.
Direct browser `fetch()` is unavailable in the backend sandbox. The provided HTTP API applies host allow-listing; logging and rate limiting are not currently implemented.

*Note: An extension's frontend iframe can use the browser's normal `fetch()` behavior. That request is separate from the backend sandbox API and is not covered by the backend host allow-list.*

## File System Access
By default, the backend Sandbox cannot download arbitrary files.
- `Aether.fs.download(url, dest)` (requires `fs:download` permission)
  - Downloads a file from the given URL (must be allowed in `hosts`) into Aether's shared `libraries` directory. This capability also requires HTTPS; mod installation additionally requires a `.jar` file of at most 100 MiB.

## Communication with the Frontend (Iframe)
Your frontend UI runs in an `<iframe>` served by a local HTTP server. Because it's isolated, it cannot call the `Aether` Go API directly.

To communicate between your UI and the backend Goja sandbox, use the built-in IPC bridge.

### 1. In your Backend Script (`main.js`)
You can register a listener using `Aether.ui.onMessage`. Any data returned by this function is automatically sent back to the frontend UI as a response. You can also push messages down to the frontend UI without a prompt using `Aether.ui.postMessage`.

```javascript
// Listen for messages from the frontend
Aether.ui.onMessage((payload) => {
    if (payload.action === 'download_mod') {
        const path = Aether.instances.installMod(payload.instanceId, payload.jarName, payload.url);
        return { status: 'success', path: path }; // Sent back to frontend
    }
});

// Push a message to the frontend unconditionally
Aether.ui.postMessage({ type: 'download_progress', percent: 50 });
```

### 2. In your Frontend UI (`index.html`)
Because the frontend runs in an isolated `<iframe>`, you use standard Web APIs (`window.postMessage`) to talk to the bridge, and listen for responses via `window.addEventListener('message')`.

```javascript
// Send a message to your backend script
// Uses computed target origin instead of '*' to prevent message spoofing
const targetOrigin = window.location.protocol + '//' + window.location.hostname + (window.location.port ? ':' + window.location.port : '');
window.parent.postMessage({
    action: 'download_mod',
    instanceId: 'fabric-1.20',
    jarName: 'my-mod.jar',
    url: 'https://example.com/mod.jar',
    __aether: true // Marker so the backend only responds to our messages
}, targetOrigin);

// Listen for responses or pushed messages from the backend script
// Only accept messages from the launcher parent with our __aether marker
window.addEventListener('message', (event) => {
    if (event.source !== window.parent || event.data.__aether !== true) return;
    if (event.data.status === 'success') {
        console.log("Mod downloaded to: ", event.data.path);
    }
});
```

## Lifecycle

```mermaid
graph TD
    A[Launcher Starts] --> B[Read manifest.json]
    B --> C[Validate manifest]
    C --> D[Create Goja runtime]
    D --> E[Inject Aether API]
    E --> F[Execute main.js]
    F --> G[Register UI]
    G --> H[Ready]
```

### Lifecycle Callbacks (Planned)
Future API versions will introduce explicit lifecycle callbacks so your extension can run setup or cleanup logic predictively:
- `onLoad()`
- `onEnable()`
- `onDisable()`
- `onUnload()`
- `onUpdate()`

## Events
Extensions can subscribe to core launcher events (requires `discord:presence` or `instances:list`):

- `Aether.events.on('instance:state', (evt) => { ... })`
  - `evt` is `{ id: string, state: "Running" | "Stopped" | "Crashed" }` – fired when a game process starts/stops.
  - Example for Rich Presence:
    ```js
    Aether.events.on('instance:state', (e) => {
      if (e.state === 'Running') {
        const inst = Aether.instances.list().find(i => i.id === e.id);
        Aether.discord.setActivity({
          details: inst.name,
          state: `${inst.version} \u2022 ${inst.loader}`,
          largeImageKey: "aether-logo",
          smallImageKey: inst.loader,
          startTimestamp: Date.now()
        });
      } else {
        Aether.discord.clearActivity();
      }
    });
    ```
  - `Aether.events.off('instance:state')` removes all handlers for the event.

### Discord Rich Presence (`discord:presence`)
Requires `discord:presence` permission. Works only if Discord desktop is running.

- `Aether.discord.setActivity(opts)`
  - `opts`: `{ details, state, largeImageKey, largeText, smallImageKey, smallText, startTimestamp }`
  - `startTimestamp` is milliseconds since epoch (`Date.now()`).
- `Aether.discord.clearActivity()` – back to Idle.

Legacy planned names `instance:launch`/`instance:stop` remain supported as aliases for `instance:state` with `Running`/`Stopped`.

## API Version Negotiation
Extensions may declare an `api` version in their manifest. The current launcher does not negotiate API versions or enforce `minApi` and `maxApi` ranges; those fields are planned compatibility metadata.

## Extended Permissions Model

Current permissions recognized by the runtime:
- `ui:sidebar`
- `ui:dialogs` (stub only)
- `instances:list`
- `mods:list`
- `mods:install`
- `mods:delete`
- `mods:toggle`
- `network:http`
- `fs:download`
- `launcher:modloader`
- `skin:export`
- `discord:presence`

The legacy `instances:patch` permission is still recognized for migration and grants the current instance/mod capabilities. New extensions should use the granular permissions above.

Confirmation requests and decisions are recorded in Aether's extension security log.

Future granular permissions:
- `instances:read`, `instances:write`
- `settings:read`, `settings:write`
- `launcher:launch`, `launcher:stop`
- `downloads:start`, `downloads:cancel`
- `notifications:show`
- `clipboard:read`, `clipboard:write`
- `extensions:list`
