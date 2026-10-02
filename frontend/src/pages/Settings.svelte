<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { GetSettings, SaveSettings, GetJavaStatus, DownloadJavaRuntime, GetThemes, SelectAndInstallTheme, SetActiveTheme, UninstallTheme, GetLauncherVersion } from '../../wailsjs/go/main/App.js';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';
  import { applyActiveTheme } from '../stores/theme';
  import { toast } from '../stores/toast';
  import SettingsGeneral from '../components/settings/SettingsGeneral.svelte';
  import SettingsAppearance from '../components/settings/SettingsAppearance.svelte';
  import SettingsJava from '../components/settings/SettingsJava.svelte';
  import SettingsSystem from '../components/settings/SettingsSystem.svelte';

  let settings = {
    defaultMemory: '4096',
    closeOnLaunch: false,
    developerMode: false,
    disableExtensions: false,
    garbageCollector: 'G1GC',
    customJvmArgs: '',
    autoCheckUpdates: true,
    includeBetaUpdates: false,
    showRecentInstances: false,
    showScreenshots: false,
    showServicesHealth: false,
    showNewsFeed: false,
  };

  let saving = false;
  let saveSuccess = false;
  let saveTimeout: ReturnType<typeof setTimeout> | null = null;
  let javaStatuses: any[] = [];
  let javaDownloading: Record<number, string> = {};

  let themes: any[] = [];
  let installingTheme = false;
  let themeBusyId = '';
  let currentVersion = '';

  function triggerUpdateCheck() {
    window.dispatchEvent(new CustomEvent('aether:check-updates'));
  }

  function openTerminalLogs() {
    window.dispatchEvent(new CustomEvent('aether:open-terminal-logs'));
  }

  function onDevModeToggle() {
    window.dispatchEvent(new CustomEvent('aether:settings-updated'));
  }

  async function loadJavaStatuses() {
    try {
      javaStatuses = await GetJavaStatus();
    } catch (e) {
      console.error("Failed to fetch Java statuses:", e);
    }
  }

  async function downloadJava(version: number) {
    javaDownloading[version] = 'Starting...';
    javaDownloading = { ...javaDownloading };
    try {
      await DownloadJavaRuntime(version);
      await loadJavaStatuses();
    } catch (e: any) {
      console.error(e);
      javaDownloading[version] = 'Error: ' + e;
      javaDownloading = { ...javaDownloading };
    }
  }

  let javaStatusUnsub: (() => void) | null = null;

onMount(async () => {
    try {
      const s = await GetSettings();
      settings = { ...settings, ...s };
      currentVersion = await GetLauncherVersion();
      await loadJavaStatuses();
      await loadThemes();
    } catch (e) {
      console.error("Failed to load settings:", e);
    }

    javaStatusUnsub = EventsOn('java:status', (data: any) => {
      if (data && data.version) {
        if (data.phase === 'done') {
          delete javaDownloading[data.version];
          javaDownloading = { ...javaDownloading };
          loadJavaStatuses();
        } else {
          javaDownloading[data.version] = data.message || `${data.phase}...`;
          javaDownloading = { ...javaDownloading };
        }
      }
    });
  });

  onDestroy(() => {
    if (javaStatusUnsub) javaStatusUnsub();
    clearTimeout(saveTimeout);
  });

  async function loadThemes() {
    try {
      themes = (await GetThemes()) || [];
    } catch (e) {
      console.error('Failed to load themes:', e);
    }
  }

  async function installTheme() {
    if (installingTheme) return;
    installingTheme = true;
    try {
      const result = await SelectAndInstallTheme();
      if (result && result.Manifest) {
        toast.success(`Installed theme "${result.Manifest.name}"`);
        if (result.Warnings && result.Warnings.length) {
          for (const w of result.Warnings) toast.info(w, 6000);
        }
        await loadThemes();
      }
    } catch (e: any) {
      console.error('Failed to install theme:', e);
      toast.error('Could not install theme: ' + (e?.message || e));
    } finally {
      installingTheme = false;
    }
  }

  async function activateTheme(id: string) {
    if (themeBusyId) return;
    themeBusyId = id;
    try {
      await SetActiveTheme(id);
      await loadThemes();
      await applyActiveTheme();
      toast.success(id ? 'Theme applied.' : 'Reverted to the default look.');
    } catch (e: any) {
      console.error('Failed to activate theme:', e);
      toast.error('Could not apply theme: ' + (e?.message || e));
    } finally {
      themeBusyId = '';
    }
  }

  async function removeTheme(id: string) {
    if (themeBusyId) return;
    themeBusyId = id;
    try {
      await UninstallTheme(id);
      await loadThemes();
      await applyActiveTheme();
      toast.success('Theme removed.');
    } catch (e: any) {
      console.error('Failed to remove theme:', e);
      toast.error('Could not remove theme: ' + (e?.message || e));
    } finally {
      themeBusyId = '';
    }
  }

  async function save() {
    saving = true;
    saveSuccess = false;
    clearTimeout(saveTimeout);
    saveTimeout = setTimeout(() => {
      saveSuccess = false;
    }, 2000);
    try {
      await SaveSettings(settings);
      saveSuccess = true;
      window.dispatchEvent(new CustomEvent('aether:settings-updated'));
    } catch (e) {
      console.error("Failed to save settings:", e);
    } finally {
      saving = false;
    }
  }
</script>

<div class="page page-enter">
  <div class="header">
    <h1>Settings</h1>
    <p class="subtitle">Global preferences for Aether</p>
  </div>

  <div class="settings-grid">
    <SettingsGeneral bind:settings />

    <SettingsAppearance
      {themes}
      {installingTheme}
      {themeBusyId}
      onInstall={installTheme}
      onApply={activateTheme}
      onDisable={() => activateTheme('')}
      onRemove={removeTheme}
    />

    <SettingsJava
      bind:settings
      {javaStatuses}
      {javaDownloading}
      onDownload={downloadJava}
    />

    <SettingsSystem
      bind:settings
      {currentVersion}
      onCheck={triggerUpdateCheck}
      onDevModeToggle={onDevModeToggle}
      onOpenLogs={openTerminalLogs}
    />

    <div class="actions">
      <button class="btn btn-primary save-btn" on:click={save} disabled={saving}>
        {#if saving}
          Saving...
        {:else if saveSuccess}
          Saved!
        {:else}
          Save Changes
        {/if}
      </button>
    </div>
  </div>
</div>

<style>
  .page {
    padding: var(--spacing-xl);
    height: 100%;
    box-sizing: border-box;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .header {
    width: 100%;
    margin-bottom: var(--spacing-md);
  }

  h1 {
    font-size: 24px;
    margin: 0 0 4px 0;
    color: var(--text-primary);
  }

  .subtitle {
    color: var(--text-secondary);
    margin: 0;
    font-size: 14px;
  }

  .settings-grid {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
    padding-bottom: var(--spacing-xl);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-md);
  }

  .save-btn {
    min-width: 140px;
  }
</style>
