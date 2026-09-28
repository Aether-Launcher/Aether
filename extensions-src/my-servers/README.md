# My Servers (official)

Pick an instance, see your saved multiplayer servers with live status, and
launch the game straight into a server — or jump directly into one of your
singleplayer worlds.

## What it does

- **Servers**: reads the selected instance's `servers.dat`, pings each entry,
  and shows MOTD, player counts, and latency. One click on **Play** launches
  the game and auto-connects (vanilla `--server`/`--port`).
- **Worlds**: lists singleplayer worlds from `saves/` with names read from
  `level.dat`. One click on **Play** launches the game and auto-loads the
  world via Mojang Quick Play (requires Minecraft 1.20+).

## Permissions

- `ui:sidebar` — the "My Servers" sidebar page.
- `instances:list` — populate the instance picker.
- `instances:launch` — start the game (install-time grant, no per-click prompt).
- `servers:list` — read `servers.dat` and ping entries.
- `saves:list` — read world metadata from `level.dat` (names only).

## Notes

- Launching an uninstalled or first-run instance may take a while (Java /
  game files download in the background) — the UI says so instead of
  erroring on a timeout.
- Direct world join needs Minecraft 1.20 or newer; older instances show the
  launcher's error.
