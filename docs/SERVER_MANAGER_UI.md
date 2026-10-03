# Server Manager UI Spec

This guide describes how to build an extension for managing local Paper,
Purpur, Vanilla, or Fabric servers. The panel runs in the extension's iframe.
Use the bridge methods, events, error handling, and interface guidance below
as the implementation contract.

For the available APIs, see [API](API.md). For manifest fields and
permissions, see [Extensions](EXTENSIONS.md).

## 1. Architecture

```
┌─ Extension iframe (ui/index.html + script.js + style.css) ─────────┐
│  window.parent.postMessage  ⇄  window 'message' listener            │
└───────────────────────────────┬────────────────────────────────────┘
                                │ sandbox IPC bridge (requestId)
┌─ Backend sandbox (main.js) ───▼────────────────────────────────────┐
│  Aether.ui.onMessage  →  Aether.servers.*  →  Aether.ui.postMessage │
└────────────────────────────────────────────────────────────────────┘
```

The UI does not call Go directly. Each request travels from the UI through
`main.js` to `Aether.servers.*`, and the response returns through the bridge.
The `server-list` and `server-host` examples in `dist-extensions/` show this
pattern in practice.

## 2. Manifest

```json
{
  "id": "your-server-manager",
  "name": "Server Manager",
  "version": "1.0.0",
  "author": "You",
  "description": "Host and manage local Minecraft servers.",
  "icon": "icon.png",
  "main": "main.js",
  "api": "1.0",
  "permissions": [
    "ui:sidebar",
    "servers:list",
    "servers:manage",
    "servers:process"
  ]
}
```

- `servers:list`: read `servers.dat` and ping servers.
- `servers:manage`: create or delete `servers/<id>/` directories and read or write files.
- `servers:process`: start, stop, inspect status, send console commands, and
  manage EULA acceptance. This permission can control running processes.
- Request only the permissions the panel needs. Avoid `servers:process`
  unless the extension actually hosts servers.

## 3. IPC protocol (UI ↔ main.js)

Every request should carry an incrementing `requestId`, which the response
echoes along with either a result or an `error`. Keep these two bridge details
in mind:

1. **`targetOrigin` MUST be `"*"`.** Inside the iframe, `window.location`
   is the *iframe's own* origin (`http://127.0.0.1:port`) while
   `window.parent` is the Wails webview (`wails://…`). A computed origin
   never matches, so `postMessage` silently drops every request and all
   you see is `Request timed out`. Use `requestId` to match each response
   with its request, as the shipped Modrinth UI does.
2. **Do not require a marker on inbound messages.** `ExtensionView`
   forwards backend payloads as-is. Responses carry no `__aether`
   marker. Filtering on one drops every reply. Match responses by
   `requestId` instead.

```javascript
const pending = {};
let reqCounter = 0;

function sendMessage(payload, timeoutMs) {
  const ms = timeoutMs || 15000;
  return new Promise((resolve, reject) => {
    const id = ++reqCounter;
    payload.requestId = id;
    pending[id] = { resolve, reject };
    // Use "*" here, as described in rule 1 above.
    window.parent.postMessage(payload, "*");
    setTimeout(() => {
      if (pending[id]) {
        delete pending[id];
        reject(new Error('Request timed out'));
      }
    }, ms);
  });
}

window.addEventListener('message', (e) => {
  const msg = e.data;
  // No marker check; see rule 2 above.
  if (!msg || msg.requestId == null) return;
  const p = pending[msg.requestId];
  if (!p) return;
  delete pending[msg.requestId];
  if (msg.error || msg.success === false) p.reject(new Error(msg.error || 'failed'));
  else p.resolve(msg);
});
```

In `main.js`, handle each message and send a reply with its `requestId`.
Catch exceptions and return them as errors rather than letting them escape:

```javascript
Aether.ui.onMessage(function (msg) {
  try {
    if (msg.type === 'server_status') {
      var st = Aether.servers.status(msg.id);
      Aether.ui.postMessage({ type: 'server_status_result', requestId: msg.requestId, success: true, status: st });
      return;
    }
    // ... one branch per message below ...
    Aether.ui.postMessage({ type: 'x', requestId: msg.requestId, success: false, error: 'Unknown message type: ' + msg.type });
  } catch (e) {
    Aether.ui.postMessage({ type: 'x', requestId: msg.requestId, success: false, error: String(e && e.message ? e.message : e) });
  }
});
```

## 4. Screens

### 4.1 Server list (home view)

- Load: `host_list` → `Aether.servers.listServers()` → `[{ id, name }]`.
- Per server, call `status` (see 4.2) to render the state dot:
  green running / gray stopped.
- **Poll `status()` every 3 seconds, but only while at least one server is
  running.** Stop polling when all servers are stopped to avoid unnecessary work.
- Actions per row: **Open console**, **Start**/**Stop** (toggle by state),
  **Delete** (backend fires the launcher confirmation dialog; on denial the
  bridge throws `user denied server deletion`; show this as information, not
  an error).
- Empty state: "No servers yet. Create one below." Include the create form.

### 4.2 Start flow (with options + EULA gate)

Start form fields (all optional except the server itself):

| Field | Maps to | Rules |
|---|---|---|
| MC version | `mcVersion` | `"1.21.1"` style. **Required** when the jar filename carries no version (notably `fabric-server-launch.jar`). Otherwise auto-detected from `paper-<mc>-*` / `purpur-<mc>-*` names. |
| Memory (MB) | `memoryMB` | Default 2048, clamped server-side to 512–16384. Prefill 2048. |
| Jar file | `jarName` | Optional. Defaults to auto-detecting `paper-*.jar`, then `purpur-*.jar`, then `server.jar`. `fabric-server-launch.jar` is also recognized. Keep the selected jar visible in the form; `status().mcVersion` reports only the game version. |
| Extra JVM args | `extraArgs[]` | Optional list, max 32 entries, 500 chars each (server rejects the rest). One per line in a textarea. |

Sequence:

1. Call `eulaStatus` first.
2. If `false`, show an inline message that the Mojang EULA must be accepted before the first start, with a **Review & Accept** button. Call `acceptEula`; Aether shows a confirmation dialog with your extension's name. If the user declines (`user denied EULA acceptance`), keep them on the form and do not show an error.
3. Call `start`. Disable the Start button and show spinner until it resolves.
4. On success → navigate to the console view for that server.

### 4.3 Console view (the core of the panel)

- On open: call `recentLogs(id, 100)` and render top-to-bottom, oldest first.
- Subscribe live: `window.addEventListener('message')` handler for
  `server:log` events. **Payload shape:** `{ id, line }` (note: these arrive
  as pushed events from the backend. They carry no `requestId`, so route
  them by `msg.id`, not through the `pending` map).
- Also listen for `server:state` `{ id, state }` (`"running"` / `"stopped"`)
  to flip the header dot and enable/disable the input.
- **Cap the DOM at about 300 lines**, dropping older lines from the top. A busy modded server
  will otherwise grow the iframe without bound.
- **Auto-scroll**: stick to bottom only while already at bottom; if the user
  scrolled up, show a "Jump to latest" pill instead of yanking.
- Command input + Send button → `send(id, command)` (4 KB cap server-side;
  enforce `maxlength="4000"` client-side too). Disable while stopped.
- Header: show the server name and state dot. Player counts are not available here; console
  parsing for player lists is out of scope for v1.

### 4.4 Stop flow

- `stop(id)` sends `stop` to the console and waits up to 10 s for graceful
  shutdown before the backend kills the process. Show "Stopping…" state;
  the `server:state → stopped` event is the source of truth, not the call
  resolving.
- Never expose kill/force-stop in v1 UI.

### 4.5 Delete flow

- `delete(id)` triggers the launcher confirmation dialog automatically.
  Handle both outcomes: success → remove the card + toast; denial error
  (`user denied server deletion`) → silent/info, no red error styling.

### 4.6 File editor (server.properties and friends)

- `readFile(id, "server.properties")` returns UTF-8 text (5 MiB cap).
- `writeFile(id, path, base64)` needs a UTF-8-safe base64 helper
  (`TextEncoder` → bytes → `btoa`), **not** raw `btoa()` (breaks on unicode
  MOTDs):

```javascript
function toB64(str) {
  const bytes = new TextEncoder().encode(str);
  let bin = '';
  bytes.forEach((b) => { bin += String.fromCharCode(b); });
  return btoa(bin);
}
```

- After saving `server.properties`, note in the UI that changes apply on
  next start (or offer Restart = stop + start).
- Keep paths inside `servers/<id>/`. The backend rejects traversal;
  mirror that client-side by disallowing `..` in any path input.

## 5. Backend message catalog (main.js reference)

| UI msg `type` | Bridge call | Returns |
|---|---|---|
| `host_list` | `Aether.servers.listServers()` | `{ servers: [{id, name}] }` |
| `host_create` | `Aether.servers.create(id, name?)` | `{ server: {id, name} }` |
| `host_delete` | `Aether.servers.delete(id)` | `{}` (confirm dialog may deny) |
| `host_read` | `Aether.servers.readFile(id, path)` | `{ content }` |
| `host_write` | `Aether.servers.writeFile(id, path, b64)` | `{}` |
| `server_status` | `Aether.servers.status(id)` | `{ status: {id, running, pid?, startedAt?, port?, mcVersion?} }` |
| `server_start` | `Aether.servers.start(id, opts?)` | `{ status }` (`opts`: `{mcVersion?, memoryMB?, jarName?, extraArgs?[]}`) |
| `server_stop` | `Aether.servers.stop(id)` | `{}` |
| `server_send` | `Aether.servers.send(id, command)` | `{}` |
| `server_eula` | `Aether.servers.eulaStatus(id)` | `{ accepted: bool }` |
| `server_accept_eula` | `Aether.servers.acceptEula(id)` | `{}` (confirm dialog may deny) |
| `server_logs` | `Aether.servers.recentLogs(id, n?)` | `{ lines: [] }`, newest last, default 100 |
| `list_servers` | `Aether.servers.list(instanceId)` | `{ instances: [{instanceId, instanceName, servers: [{name, ip, hidden, hasIcon}]}] }` (server-list tab) |
| `ping_server` | `Aether.servers.ping(host)` | `{ result: {online, host, port, motd, playersOnline, playersMax, version, protocol, latencyMs} }` |

## 6. Event catalog (pushed, no requestId)

| Event | Payload | Notes |
|---|---|---|
| `server:log` | `{ id, line }` | One per console line, last-500 ring per server |
| `server:state` | `{ id, state }` | `"running"` on start, `"stopped"` on exit/crash |

## 7. Error catalog (exact strings → UI treatment)

| Backend error contains | Meaning | UI treatment |
|---|---|---|
| `server EULA not accepted (eula.txt)` | First start without EULA | EULA banner + Review & Accept flow (§4.2). Info, not red. |
| `user denied EULA acceptance` / `user denied server deletion` | User cancelled the launcher dialog | Silent/info. Never an error toast. |
| `server "x" is already running` | Double start | Refresh status; info level. |
| `server limit reached (2 concurrent max)` | Cap hit | Error toast, suggest stopping another server. |
| `port NNNN is already in use` | Conflict | Error toast naming the port; suggest editing `server-port`. |
| `no server jar found in …` | Empty/misnamed dir | Error + hint listing expected names (`paper-*.jar`, `purpur-*.jar`, `fabric-server-launch.jar`, `server.jar`). |
| `could not determine Minecraft version` | Unversioned jar, no `mcVersion` passed | Error + focus the MC version field. |
| `server "x" is not running` | Stop/send/status race | Refresh status; info level. |
| `server "x" does not exist` | Deleted elsewhere | Reload list. |
| `command too long` | >4 KB input | Client-side `maxlength` should prevent; treat as validation. |
| `file exceeds the 5 MB limit` / `invalid base64 data` | Editor abuse/edge | Error toast. |
| `instance.json`-style: any `not found` / `does not exist` on read | Missing file | Empty editor with "new file" hint, not an error. |

## 8. Security and input handling

1. **Escape every interpolated string.** Use an `esc()` helper for `&<>"'` on
   *all* server names, MOTDs, log lines, and file contents rendered via
   `innerHTML`, or build DOM with `textContent`. Console output is
   attacker-influenced (any player can print into a server log via chat).
2. **CSP meta** on every `ui/*.html`:
   `default-src 'none'; img-src https: data:; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'`.
3. **IPC hygiene**: use `"*"` as `postMessage`'s `targetOrigin` because the
   iframe and Wails webview use different origins. Do not require a marker on
   replies; the bridge forwards payloads as-is. Correlate replies by
   `requestId`, as described in §3.
4. **No `eval`/`Function`/`innerHTML` with unescaped data** anywhere.
5. **15 s default IPC timeout** (3–5 min only for bulk downloads); always
   re-enable buttons in `finally`.
6. Client-side mirrors of server limits: id regex
   `^[a-z0-9][a-z0-9._-]{0,64}$`, path inputs reject `..`, memory inputs
   clamp 512–16384.

## 9. UX requirements

- While an action is running, disable its button and show progress. Re-enable
  it in `finally`, including when the request fails.
- Status polling: 3 s cadence **only while ≥1 server reports running**.
- Log DOM cap ~300 lines; auto-scroll stickiness per §4.3.
- Denied confirmations are *information*, never red errors.
- Empty states everywhere (no servers / no logs yet / file missing).
- Dark-first styling matching the launcher (`--bg-color` family); reuse the
  `.card/.btn/.badge` vocabulary from `dist-extensions/server-list`.

## 10. Out of scope for v1 (do not build)

Player lists (there is no API for them; they would require log parsing or
RCON), scheduled restarts and backups, multi-user permissions, remote
process management, and log search or filtering.

## 11. Acceptance checklist (what "done" means)

- [ ] Create → appears in list → EULA gate → Start → console streams →
      Send `list` → response appears → Stop → `stopped` state, all without reload.
- [ ] Double-Start and Stop-when-idle produce info notices, not errors.
- [ ] Port conflict surfaces the port number with remediation hint.
- [ ] Delete fires the launcher confirm; denial is silent.
- [ ] `server.properties` round-trips unicode MOTDs byte-identical.
- [ ] `npm run check`-equivalent: no console errors; buttons never stick disabled.
- [ ] Meets the Aether Extensions registry's package and permission
      requirements, including archive-path and file-size checks.
