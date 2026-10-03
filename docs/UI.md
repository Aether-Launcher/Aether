# UI Layout and Components

Aether is a desktop launcher, so launching Minecraft and managing instances should always be easy to find. Extensions add optional features without crowding the core experience.

## Sidebar

The 220px sidebar links to Home, Instances, Marketplace, and Settings, with connection status and the signed-in account below. Extension pages share one **Active Extensions** group by default. An extension can opt to pin its pages to the sidebar. A slim accent bar marks the current page.

## Home

Home puts the selected instance and its Play button first. It also shows the Minecraft version, mod loader, and launch state, with a link to instance settings. Depending on their preferences, users may also see recent instances, screenshots, service status, news, and extension updates. Optional Home sections can be changed in Settings.

## Instances

On the Instances page, users can create or import an instance. Each card shows its name, Minecraft version, mod loader, last-played time, avatar, and Play button. The gear button opens instance details and settings.

## Instance details

Instance details collect the selected instance's settings, game files, and launch options. Group related controls together and make the main action easy to spot.

## Extensions

The Extensions page has **Installed** and **Gallery** views. Users can install a local `.aex` package or browse the gallery, check for updates, reload extensions, update them, and uninstall them. Cards show an extension's name, version, author, status, trust label, and description. Developer mode adds runtime details.

Extensions can register pages that Aether displays in an iframe. Pages appear under **Active Extensions** unless the extension asks to pin them. An extension cannot replace launcher navigation or window controls, display content outside its frame, or modify another extension.

## Settings

Settings are grouped by topic, such as launcher behavior, appearance, Java, updates, and advanced options. Keep related choices together and describe them in plain language.

## Empty states, dialogs, and feedback

An empty page should explain what belongs there and offer a useful next step. Keep each dialog focused on one decision, with a clear way to cancel. Use notifications sparingly, and show progress during downloads, installs, and launches.

## Desktop layout

The window's minimum width is 1100px. The layout should remain comfortable at that size. Let content areas scroll while the sidebar and title bar stay in place.
