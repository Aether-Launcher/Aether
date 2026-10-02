<script lang="ts">
  import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

  export let settings: any;
  export let currentVersion = '';
  export let onCheck: () => void;
  export let onDevModeToggle: () => void;
  export let onOpenLogs: () => void;
</script>

<div class="settings-card card">
  <div class="card-header-row">
    <h2>Updates</h2>
    {#if currentVersion}
      <span class="version-badge">{currentVersion === 'dev' ? 'dev' : 'v' + currentVersion.replace(/^v+/i, '')}</span>
    {/if}
  </div>

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

  <div class="update-action-row">
    <button class="btn btn-secondary" on:click={onCheck}>
      Check for Updates Now
    </button>
  </div>
</div>

<div class="settings-card card">
  <h2>Advanced</h2>

  <div class="form-group checkbox-group">
    <label class="checkbox-label" for="developer-mode">
      <input id="developer-mode" type="checkbox" bind:checked={settings.developerMode} on:change={onDevModeToggle} />
      <span class="custom-checkbox"></span>
      <div class="label-content">
        <div class="label-title">Developer Mode</div>
        <div class="label-desc">Enable developer tools, live terminal logs, and advanced extension debugging features.</div>
      </div>
    </label>
    {#if settings.developerMode}
      <button class="btn btn-secondary btn-sm" on:click={onOpenLogs} style="margin-left: auto;">
        &gt;_ View Terminal Logs (Ctrl+Shift+L)
      </button>
    {/if}
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

  .form-group.warning .label-title {
    color: #ef4444;
  }

  .form-group.vertical {
    flex-direction: column;
    align-items: stretch;
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

  .btn-sm {
    padding: 4px 10px;
    font-size: 12px;
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

  .card-header-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-md, 16px);
  }

  .card-header-row h2 {
    margin: 0;
  }

  .version-badge {
    padding: 2px 8px;
    border-radius: 12px;
    background: rgba(255, 255, 255, 0.08);
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .update-action-row {
    margin-top: 14px;
    padding-top: 14px;
    border-top: 1px solid rgba(255, 255, 255, 0.06);
    display: flex;
    justify-content: flex-end;
  }
</style>
