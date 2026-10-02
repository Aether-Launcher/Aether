<script lang="ts">
  import Dropdown from '../Dropdown.svelte';

  export let settings: any;

  const memoryOptions = [
    { label: '2 GB', value: '2048' },
    { label: '4 GB', value: '4096' },
    { label: '6 GB', value: '6144' },
    { label: '8 GB', value: '8192' },
    { label: '12 GB', value: '12288' },
    { label: '16 GB', value: '16384' },
  ];
</script>

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

<div class="settings-card card">
  <h2>Home Dashboard</h2>
  <p class="section-hint">Choose which widgets appear on the Home dashboard overview.</p>

  <div class="form-group checkbox-group">
    <label class="checkbox-label" for="show-recent-instances">
      <input id="show-recent-instances" type="checkbox" bind:checked={settings.showRecentInstances} />
      <span class="custom-checkbox"></span>
      <div class="label-content">
        <div class="label-title">Recent Instances & Quick-Play</div>
        <div class="label-desc">Show quick-launch cards for your recently played instances.</div>
      </div>
    </label>
  </div>

  <div class="form-group checkbox-group">
    <label class="checkbox-label" for="show-screenshots">
      <input id="show-screenshots" type="checkbox" bind:checked={settings.showScreenshots} />
      <span class="custom-checkbox"></span>
      <div class="label-content">
        <div class="label-title">Screenshot Showcase</div>
        <div class="label-desc">Show a gallery of in-game screenshots with a full-size lightbox viewer.</div>
      </div>
    </label>
  </div>

  <div class="form-group checkbox-group">
    <label class="checkbox-label" for="show-services-health">
      <input id="show-services-health" type="checkbox" bind:checked={settings.showServicesHealth} />
      <span class="custom-checkbox"></span>
      <div class="label-content">
        <div class="label-title">Live Services Status</div>
        <div class="label-desc">Show reachability and latency indicators for Mojang & Minecraft servers.</div>
      </div>
    </label>
  </div>

  <div class="form-group checkbox-group">
    <label class="checkbox-label" for="show-news-feed">
      <input id="show-news-feed" type="checkbox" bind:checked={settings.showNewsFeed} />
      <span class="custom-checkbox"></span>
      <div class="label-content">
        <div class="label-title">News & Updates Feed</div>
        <div class="label-desc">Show official Minecraft patch notes and Aether launcher releases.</div>
      </div>
    </label>
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

  .section-hint {
    font-size: 13px;
    color: var(--text-secondary);
    margin: -8px 0 4px 0;
  }

  .form-group {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--spacing-md);
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
</style>
