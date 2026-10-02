# Skin Selector

Browse skins from public databases (Mojang official + NameMC) and manage the
skins and capes attached to the currently logged-in Microsoft account.

## Tabs

- **Gallery**: enter a username, preview the 3D skin, download the PNG, or
  apply it directly to your account (classic/slim selectable).
- **My Skins**: lists `GET /minecraft/profile` skins, previews them, uploads
  a PNG (`POST /minecraft/profile/skins`), or re-applies an owned URL.
- **My Capes**: lists owned capes, Equip (`PUT /capes/active`) or Hide
  (`DELETE /capes/active`).

Offline accounts are view-only. Tokens never leave Go; the sandbox only sees
`{signedIn, id, username, type}` plus skin/cape metadata. Upload/equip/hide
require launcher confirmation.

## 3D rendering

The preview uses [skin3d](https://github.com/cosmic-fi/skin3d) (MIT),
vendored with its dependencies under `ui/vendor/`:

- `skin3d` by cosmic-fi (MIT) — player model, cape, orbit controls
- `three.js` (MIT) — WebGL renderer (r156, bundled by esbuild)
- `skinview-utils` by bs-community (MIT) — texture loading helpers
