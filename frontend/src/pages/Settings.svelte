<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { GetSettings, SaveSettings, GetJavaStatus, DownloadJavaRuntime, GetThemes, SelectAndInstallTheme, SetActiveTheme, UninstallTheme } from '../../wailsjs/go/main/App.js';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
  import Dropdown from '../components/Dropdown.svelte';
  import { applyActiveTheme } from '../stores/theme';
  import { toast } from '../stores/toast';

  let settings = {
    defaultMemory: '4096',
    closeOnLaunch: false,
    developerMode: false,
    disableExtensions: false,
    garbageCollector: 'G1GC',
    customJvmArgs: '',
    autoCheckUpdates: true,
    includeBetaUpdates: false,
  };

  let saving = false;
  let saveSuccess = false;
  let saveTimeout: ReturnType<typeof setTimeout> | null = null;
  let javaStatuses: any[] = [];
  let javaDownloading: Record<number, string> = {};

  let themes: any[] = [];
  let installingTheme = false;
  let themeBusyId = '';

  const memoryOptions = [
    { label: '2 GB', value: '2048' },
    { label: '4 GB', value: '4096' },
    { label: '6 GB', value: '6144' },
    { label: '8 GB', value: '8192' },
    { label: '12 GB', value: '12288' },
    { label: '16 GB', value: '16384' },
  ];

  const gcOptions = [
    { label: 'G1GC (Default)', value: 'G1GC' },
    { label: 'ZGC (Ultra Low Latency)', value: 'ZGC' },
    { label: 'Shenandoah GC', value: 'Shenandoah' },
    { label: 'Parallel GC', value: 'Parallel' },
  ];

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
    <!-- General Section -->
    <div class="settings-card card">
      <h2>General</h2>
      
      <div class="form-group">
        <div class="field-label">
          <div class="label-title">Default Memory Allocation</div>
          <div class="label-desc">Memory used for new instances or instances set to 'Default'.</div>
        </div>
        <div class="control-wrap">
          <Dropdown options={memoryOptions} bind:value={settings.defaultMemory} />
        </div>
      </div>

      <div class="form-group checkbox-group">
        <label class="checkbox-label" for="close-on-launch">
          <input id="close-on-launch" type="checkbox" bind:checked={settings.closeOnLaunch} />
          <span class="custom-checkbox"></span>
          <div class="label-content">
            <div class="label-title">Close launcher on game start</div>
            <div class="label-desc">Aether will hide itself when Minecraft opens and reappear when it closes.</div>
          </div>
        </label>
      </div>
    </div>

    <!-- Appearance Section -->
    <div class="settings-card card">
      <h2>Appearance</h2>

      <div class="form-group vertical">
        <div class="field-label">
          <div class="label-title">Themes</div>
          <div class="label-desc">
            Install a <code>.theme</code> package to restyle Aether. Themes are a CSS overwrite plus optional
            logo/background images — they can't touch the launcher's app icon or its name.
          </div>
        </div>

        <div class="theme-list">
          {#if themes.length === 0}
            <p class="theme-empty">No themes installed yet.</p>
          {/if}
          {#each themes as t (t.id)}
            <div class="theme-item">
              <div class="theme-item-info">
                {#if t.iconUrl}
                  <img src={t.iconUrl} alt="" class="theme-icon" />
                {/if}
                <div>
                  <div class="theme-name">
                    {t.name}
                    <span class="theme-version">v{t.version}</span>
                  </div>
                  {#if t.description}<div class="theme-desc">{t.description}</div>{/if}
                </div>
              </div>
              <div class="theme-item-actions">
                {#if t.active}
                  <span class="badge badge-verified">Active</span>
                  <button class="btn btn-secondary btn-sm" disabled={themeBusyId === t.id} on:click={() => activateTheme('')}>
                    Disable
                  </button>
                {:else}
                  <button class="btn btn-secondary btn-sm" disabled={!!themeBusyId} on:click={() => activateTheme(t.id)}>
                    Apply
                  </button>
                {/if}
                <button class="btn btn-danger btn-sm" disabled={themeBusyId === t.id} on:click={() => removeTheme(t.id)}>
                  Remove
                </button>
              </div>
            </div>
          {/each}
        </div>

        <button class="btn btn-secondary" disabled={installingTheme} on:click={installTheme}>
          {installingTheme ? 'Installing…' : 'Install Theme (.theme)'}
        </button>
      </div>
    </div>

    <!-- Java & Performance Section -->
    <div class="settings-card card">
      <h2>Java & Performance</h2>

      <div class="form-group vertical">
        <div class="field-label">
          <div class="label-title">Managed Java Runtimes</div>
          <div class="label-desc">Aether manages Java runtimes required by different Minecraft versions automatically.</div>
        </div>
        <div class="java-list">
          {#each javaStatuses as js}
            <div class="java-item">
              <div class="java-item-info">
                <span class="java-name">Java {js.version}</span>
                <span class="java-target">
                  {js.version === 8 ? '(Minecraft < 1.17)' : js.version === 17 ? '(Minecraft 1.17 – 1.20.4)' : '(Minecraft 1.20.5+)'}
                </span>
              </div>
              <div class="java-item-status">
                {#if js.installed}
                  <span class="badge badge-installed" title={js.path}>
                    ✓ Installed {js.isSystem ? '(System)' : '(Managed)'}
                  </span>
                {:else if javaDownloading[js.version]}
                  <span class="java-progress">{javaDownloading[js.version]}</span>
                {:else}
                  <button class="btn btn-secondary btn-sm" on:click={() => downloadJava(js.version)}>Download JRE</button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </div>

      <div class="form-group">
        <div class="field-label">
          <div class="label-title">Garbage Collector</div>
          <div class="label-desc">Select the JVM garbage collection algorithm. ZGC is recommended for high memory (>= 8GB).</div>
        </div>
        <div class="control-wrap">
          <Dropdown options={gcOptions} bind:value={settings.garbageCollector} />
        </div>
      </div>

      <div class="form-group vertical">
        <div class="field-label">
          <div class="label-title">Custom JVM Arguments</div>
          <div class="label-desc">Additional flags passed to the Java runtime on launch (e.g. -XX:+UnlockExperimentalVMOptions).</div>
        </div>
        <input type="text" class="custom-args-input" bind:value={settings.customJvmArgs} placeholder="e.g. -XX:+UnlockExperimentalVMOptions" />
      </div>
    </div>

    <!-- Updates Section -->
    <div class="settings-card card">
      <h2>Updates</h2>

      <div class="form-group checkbox-group">
        <label class="checkbox-label" for="auto-check-updates">
          <input id="auto-check-updates" type="checkbox" bind:checked={settings.autoCheckUpdates} />
          <span class="custom-checkbox"></span>
          <div class="label-content">
            <div class="label-title">Check for updates automatically</div>
            <div class="label-desc">Aether checks for new releases on startup and offers to update itself in place.</div>
          </div>
        </label>
      </div>

      <div class="form-group checkbox-group">
        <label class="checkbox-label" for="include-beta-updates">
          <input id="include-beta-updates" type="checkbox" bind:checked={settings.includeBetaUpdates} disabled={!settings.autoCheckUpdates} />
          <span class="custom-checkbox"></span>
          <div class="label-content">
            <div class="label-title">Include beta releases</div>
            <div class="label-desc">Also offer pre-release versions in the updater. Beta versions may be unstable.</div>
          </div>
        </label>
      </div>
    </div>

    <!-- Advanced Section -->
    <div class="settings-card card">
      <h2>Advanced</h2>

      <div class="form-group checkbox-group">
        <label class="checkbox-label" for="developer-mode">
          <input id="developer-mode" type="checkbox" bind:checked={settings.developerMode} />
          <span class="custom-checkbox"></span>
          <div class="label-content">
            <div class="label-title">Developer Mode</div>
            <div class="label-desc">Enable developer tools, logs, and advanced extension debugging features.</div>
          </div>
        </label>
      </div>

      <div class="form-group checkbox-group warning">
        <label class="checkbox-label" for="disable-extensions">
          <input id="disable-extensions" type="checkbox" bind:checked={settings.disableExtensions} />
          <span class="custom-checkbox"></span>
          <div class="label-content">
            <div class="label-title">Disable Extensions Completely</div>
            <div class="label-desc">Prevents all extensions from loading. Requires an app restart to take effect.</div>
          </div>
        </label>
      </div>
    </div>

    <!-- Help & Support Section -->
    <div class="settings-card card">
      <h2>Help &amp; Support</h2>

      <div class="form-group vertical">
        <div class="field-label">
          <div class="label-title">Report a Bug</div>
          <div class="label-desc">Found something broken? Join our Discord and tell us — include your launcher version and what you were doing when it happened.</div>
        </div>
        <button class="btn btn-secondary bug-report-btn" on:click={() => BrowserOpenURL('https://discord.gg/hyPWTs9FfM')}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
            <path d="M20.317 4.37a19.79 19.79 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.058a.082.082 0 0 0 .031.056 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128c.126-.094.252-.192.372-.291a.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z"/>
          </svg>
          Report a Bug on Discord
        </button>
      </div>
    </div>

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

  .grid-top {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--spacing-md);
    align-items: start;
  }

  .java-bottom-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--spacing-xl);
    margin-top: var(--spacing-md);
  }

  .settings-card {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-lg);
  }

  .settings-card h2 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
    color: var(--text-primary);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding-bottom: var(--spacing-md);
  }

  .form-group {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--spacing-md);
  }

  .form-group.warning .label-title {
    color: #ef4444;
  }

  .control-wrap {
    width: 160px;
    flex-shrink: 0;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .label-title {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .label-desc {
    font-size: 12px;
    color: var(--text-meta);
    line-height: 1.4;
  }

  /* Custom Checkbox */
  .checkbox-group {
    justify-content: flex-start;
  }

  .checkbox-label {
    display: flex;
    flex-direction: row;
    align-items: flex-start;
    gap: 12px;
    cursor: pointer;
    user-select: none;
  }

  .checkbox-label input {
    position: absolute;
    opacity: 0;
    cursor: pointer;
    height: 0;
    width: 0;
  }

  .custom-checkbox {
    width: 18px;
    height: 18px;
    border: 2px solid rgba(255, 255, 255, 0.2);
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-fast);
    flex-shrink: 0;
    margin-top: 2px;
  }

  .checkbox-label:hover .custom-checkbox {
    border-color: rgba(255, 255, 255, 0.4);
  }

  .checkbox-label input:checked ~ .custom-checkbox {
    background-color: var(--accent-color);
    border-color: var(--accent-color);
  }

  .checkbox-label input:checked ~ .custom-checkbox:after {
    content: '';
    width: 4px;
    height: 8px;
    border: solid white;
    border-width: 0 2px 2px 0;
    transform: rotate(45deg);
    margin-bottom: 2px;
  }

  .label-content {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--spacing-md);
  }

  .form-group.vertical {
    flex-direction: column;
    align-items: stretch;
  }

  .custom-args-input {
    background: rgba(0, 0, 0, 0.25);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: var(--border-radius-md);
    padding: 10px 12px;
    color: var(--text-primary);
    font-family: monospace;
    font-size: 13px;
    width: 100%;
    box-sizing: border-box;
    transition: border-color var(--transition-fast);
  }

  .custom-args-input:focus {
    outline: none;
    border-color: var(--accent-color);
  }

  .java-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
    width: 100%;
  }

  .java-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: var(--border-radius-md);
  }

  .java-item-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .java-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary);
  }

  .java-target {
    font-size: 12px;
    color: var(--text-meta);
  }

  .badge-installed {
    background: rgba(34, 197, 94, 0.15);
    color: #4ade80;
    border: 1px solid rgba(34, 197, 94, 0.3);
    padding: 3px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
  }

  .java-progress {
    font-size: 12px;
    color: #60a5fa;
  }

  .theme-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
    width: 100%;
  }

  .theme-empty {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0;
  }

  .theme-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 14px;
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: var(--border-radius-md);
  }

  .theme-item-info {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .theme-icon {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .theme-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary);
    display: flex;
    align-items: baseline;
    gap: 6px;
  }

  .theme-version {
    font-weight: 400;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .theme-desc {
    font-size: 12px;
    color: var(--text-meta);
    max-width: 42ch;
  }

  .theme-item-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .btn-sm {
    padding: 4px 10px;
    font-size: 12px;
  }

  .save-btn {
    min-width: 140px;
  }

  .bug-report-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    align-self: flex-start;
    color: #5865F2;
    border-color: rgba(88, 101, 242, 0.35);
  }

  .bug-report-btn:hover:not(:disabled) {
    border-color: #5865F2;
    background: rgba(88, 101, 242, 0.1);
  }
</style>
