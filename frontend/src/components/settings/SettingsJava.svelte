<script lang="ts">
  import Dropdown from '../Dropdown.svelte';

  export let settings: any;
  export let javaStatuses: any[] = [];
  export let javaDownloading: Record<number, string> = {};
  export let onDownload: (version: number) => void;

  const gcOptions = [
    { label: 'G1GC (Default)', value: 'G1GC' },
    { label: 'ZGC (Ultra Low Latency)', value: 'ZGC' },
    { label: 'Shenandoah GC', value: 'Shenandoah' },
    { label: 'Parallel GC', value: 'Parallel' },
  ];
</script>

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
              <button class="btn btn-secondary btn-sm" on:click={() => onDownload(js.version)}>Download JRE</button>
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

<style>
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

  .form-group.vertical {
    flex-direction: column;
    align-items: stretch;
  }

  .control-wrap {
    width: 160px;
    flex-shrink: 0;
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

  .custom-args-input {
    background: rgba(0, 0, 0, 0.25);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: var(--border-radius);
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
    border-radius: var(--border-radius);
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
</style>
