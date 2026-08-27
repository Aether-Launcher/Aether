import { writable } from 'svelte/store';
import { GetActiveThemeCSS, GetActiveThemeAssets } from '../../wailsjs/go/main/App.js';

/**
 * Only these keys are ever recognized — this mirrors the backend whitelist
 * in pkg/theme/protected.go. A theme can never supply "app-icon",
 * "tray-icon", or "launcher-name": those aren't runtime-swappable assets at
 * all, so components should always fall back to Aether's built-in image and
 * hardcoded "Aether" name when a key is missing here.
 */
export interface ThemeAssets {
  'sidebar-logo'?: string;
  'titlebar-logo'?: string;
  background?: string;
}

export const themeAssets = writable<ThemeAssets>({});

const STYLE_TAG_ID = 'aether-theme-overwrite';

function injectCSS(css: string) {
  let styleTag = document.getElementById(STYLE_TAG_ID) as HTMLStyleElement | null;
  if (!styleTag) {
    styleTag = document.createElement('style');
    styleTag.id = STYLE_TAG_ID;
    document.head.appendChild(styleTag);
  }
  styleTag.textContent = css;
}

/**
 * Loads the currently active theme (if any) from the backend, injects its
 * sanitized CSS as a <style> tag appended after the base stylesheet (so its
 * rules win on equal specificity — mainly :root variable overrides), and
 * populates the asset store so components can swap in whitelisted images.
 *
 * Safe to call again after the user switches themes in Settings.
 */
export async function applyActiveTheme(): Promise<void> {
  try {
    const [css, assets] = await Promise.all([
      GetActiveThemeCSS().catch(() => ''),
      GetActiveThemeAssets().catch(() => ({} as Record<string, string>)),
    ]);
    injectCSS(css || '');
    themeAssets.set((assets as ThemeAssets) || {});
  } catch (e) {
    console.error('Failed to apply active theme:', e);
    injectCSS('');
    themeAssets.set({});
  }
}
