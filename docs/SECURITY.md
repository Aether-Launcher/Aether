# Security

## Sandbox
Extension backend scripts run in a Goja JavaScript runtime. They do not receive Node.js modules, Go APIs, shell access, or direct access to the host filesystem. The supported integration surface is the injected `Aether` object, whose capabilities are added according to the permissions in the manifest.

This is a capability boundary, not a complete OS-level security boundary. Extensions with instance or download permissions can change shared launcher data through those APIs.

## Whitelisted Networking
By default, extensions cannot make outbound network requests. 
If an extension needs to communicate with an external API, it must declare allowed hostnames in its `manifest.json` under `hosts`. Requests are limited to HTTP(S) URLs whose hostname matches an allowed host or one of its subdomains. The launcher does not currently show a separate host approval screen during installation.

```json
"hosts": [
    "api.modrinth.com"
]
```

## Capability Model
The runtime uses a capability-based model. An extension only receives the API objects associated with its declared permissions. Calls to unavailable capabilities fail in the JavaScript runtime. The current instance capability is limited to listing instances and installing, listing, deleting, or toggling mods; it does not provide general instance JSON or log access.

## Registry Trust
The extension gallery can assign trust labels such as Official, Verified, Community, or Local. These labels are registry metadata displayed by the launcher; the current application does not perform automated code analysis, quarantine extensions, or enforce a maintainer review workflow.

## Threat Model
**Expected Threats:**
- Malicious extensions attempting to steal Minecraft session tokens.
- Extensions attempting to download and execute arbitrary binaries (malware).
- Extensions attempting to read arbitrary files on the user's system (e.g., SSH keys, browser cookies).
- Extensions attempting to execute arbitrary shell commands.
- Extensions attempting to escape the Goja Sandbox.

**Mitigation:**
Every privileged operation is strictly mediated by the launcher. Extensions explicitly **CANNOT**:
- Execute arbitrary shell commands (e.g., via `os/exec`).
- Access the filesystem directly (they can only use scoped `Aether.fs` APIs).
- Read launcher memory.
- Escape the Goja runtime through Node.js or Go bindings.
- Access Go APIs directly.

- Authentication supports offline accounts and Microsoft account sign-in. Extensions are not given account credentials, access tokens, or refresh tokens through the `Aether` API.
- File access is abstracted through scoped APIs, but the permitted locations are shared launcher directories such as instance `mods`, `libraries`, and `skins`; they are not isolated per extension.
- Network access is HTTPS-only and host-allow-listed. Requests are not currently rate-limited or security-logged, but backend responses and mod downloads have size limits.

Sensitive extension confirmation requests and their decisions are recorded as JSON lines in `logs/extension-security.log`.

## Security Boundary and Commitments

Aether can promise that backend extension code executes inside a Goja JavaScript runtime without Node.js, Go, shell, or direct host-filesystem APIs. Manifest permissions control which launcher bridge objects are injected, HTTPS requests are host-allow-listed, and privileged mod/file operations can require user confirmation.

Aether cannot promise that an installed extension is trustworthy, that shared launcher data is isolated per extension, or that the Goja runtime is an operating-system security boundary. Extensions granted install, delete, toggle, download, or mod-loader permissions can affect shared launcher or instance state. Frontend extension iframes are a separate browser surface and may have browser networking behavior outside the backend sandbox policy.

Treat extensions as code with the permissions shown in their manifest. Review Local and Community extensions, keep permissions minimal, and do not describe the sandbox as malware-proof or equivalent to a separate process/container. The current implementation does not provide automated code analysis, quarantine, network rate limiting, or a host approval prompt.
