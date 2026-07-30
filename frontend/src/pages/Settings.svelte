<script lang="ts">
  import { onMount } from 'svelte';
  import { GetSettings, SaveSettings, GetJavaStatus, DownloadJavaRuntime } from '../../wailsjs/go/main/App.js';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';
  import Dropdown from '../components/Dropdown.svelte';

  let settings = {
    defaultMemory: '4096',
    closeOnLaunch: false,
    developerMode: false,
    disableExtensions: false,
    garbageCollector: 'G1GC',
    customJvmArgs: '',
  };

  let saving = false;
  let saveSuccess = false;
  let javaStatuses: any[] = [];
  let javaDownloading: Record<number, string> = {};

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

  onMount(async () => {
    try {
      const s = await GetSettings();
      settings = { ...s };
      await loadJavaStatuses();
    } catch (e) {
      console.error("Failed to load settings:", e);
    }

    EventsOn('java:status', (data: any) => {
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

  async function save() {
    saving = true;
    saveSuccess = false;
    try {
      await SaveSettings(settings);
      saveSuccess = true;
      setTimeout(() => {
        saveSuccess = false;
      }, 2000);
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

  .btn-sm {
    padding: 4px 10px;
    font-size: 12px;
  }

  .save-btn {
    min-width: 140px;
  }
</style>
