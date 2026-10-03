# Security

## Extension sandbox
Each extension backend runs in a Goja JavaScript runtime. It does not receive Node.js modules, Go APIs, shell access, or direct access to the host filesystem. Instead, it can use the APIs exposed through `Aether`, based on the permissions in its manifest.

This limits what an extension can ask Aether to do; it is not a complete operating-system security boundary. For example, an extension with instance or download permissions can change shared launcher data through those APIs.

## Network access
Backend extensions have no network access by default. To call an external API, an extension must request `network:http` and list allowed hostnames under `hosts` in `manifest.json`. Requests must use HTTPS, and the hostname must match an allowed host or one of its subdomains. Aether does not currently show a separate host approval prompt during installation.

```json
"hosts": [
    "api.modrinth.com"
]
```

## Capability Model
An extension only receives the API objects associated with its declared permissions. Calls to unavailable APIs fail in the runtime. The current instance APIs cover listing instances and installing, listing, deleting, or toggling mods; they do not expose general instance JSON or logs. `saves:list` exposes singleplayer world names, while `instances:launch` can start the game and optionally connect to a server or world chosen by the user. These permissions are granted at install time and do not expose credentials or arbitrary files.

## Registry Trust
The gallery can attach Official, Verified, Community, or Local labels to extensions. Aether displays these labels as registry metadata. The launcher does not currently analyze extension code, quarantine extensions, or enforce a maintainer review process, so a badge is not a security guarantee.

## What the protections cover
The launcher mediates privileged operations. Backend extensions cannot run arbitrary shell commands, access the filesystem directly, read launcher memory, or call Go APIs. They can use only the scoped APIs exposed through `Aether`.

Those APIs still affect shared data. Depending on its permissions, an extension may change files in shared locations such as instance `mods`, `libraries`, and `skins`. Extensions are not isolated from one another at the data-directory level.

Authentication supports offline and Microsoft accounts. The `Aether` API does not expose account credentials or access and refresh tokens. HTTPS requests are host-allow-listed, but are not currently rate-limited or written to a security log. Backend responses and mod downloads have size limits. Modrinth icon URLs are restricted to `https://cdn.modrinth.com` and its subdomains, and user-controlled text is rendered as text rather than inserted as HTML.

The extension UI and backend communicate through an iframe bridge. Requests and replies are correlated by `requestId`; the UI must use `"*"` as `postMessage`'s target origin because the iframe and Wails webview have different origins. Replies are forwarded as-is and do not carry a special marker. See the IPC guidance in [API](API.md) and [Server Manager UI](SERVER_MANAGER_UI.md).

Sensitive extension confirmation requests and their decisions are recorded as JSON lines in `logs/extension-security.log`.

## Security Boundary and Commitments

Aether can provide a Goja runtime without Node.js, Go, shell, or direct host-filesystem APIs. Manifest permissions control which launcher APIs are available. HTTPS requests are host-allow-listed, and some sensitive mod and file operations require user confirmation.

Aether cannot guarantee that an installed extension is trustworthy, that each extension's data is isolated, or that Goja acts as an operating-system security boundary. Extensions with install, delete, toggle, download, or mod-loader permissions can affect shared launcher or instance state. An extension's iframe is a separate browser surface and may make browser requests outside the backend sandbox's network policy.

Treat an extension as code with the permissions listed in its manifest. Review its source when possible and prefer the smallest set of permissions it needs. The sandbox is not malware-proof and is not equivalent to a separate process or container. Aether does not currently provide automated code analysis, quarantine, network rate limiting, or a host approval prompt.
