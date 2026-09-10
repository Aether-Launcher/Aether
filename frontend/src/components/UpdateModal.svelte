<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';
  import { CheckForUpdates, DownloadAndUpdate } from '../../wailsjs/go/main/App.js';
  import { toast } from '../stores/toast.js';
  import Icon from './Icon.svelte';

  type Status = {
    phase: 'none' | 'checking' | 'available' | 'downloading' | 'ready' | 'error';
    version?: string;
    notes?: string;
    message?: string;
    progress?: number;
  };

  let status: Status = { phase: 'none' };
  let showModal = false;
  let isChecking = false;
  let skippedVersion = localStorage.getItem('aether_skipped_version') || '';

  function handleStatus(payload: unknown) {
    const s = (payload as Status) || { phase: 'none' };
    status = s;

    if (s.phase === 'available') {
      // If the user previously skipped this exact version tag, don't force auto-modal
      // unless they manually clicked "Check for Updates".
      if (s.version && s.version === skippedVersion) {
        return;
      }
      showModal = true;
    } else if (s.phase === 'downloading' || s.phase === 'ready' || s.phase === 'error') {
      showModal = true;
    }
  }

  function handleProgress(payload: unknown) {
    const p = payload as { progress?: number };
    if (p && typeof p.progress === 'number') {
      status = { ...status, progress: p.progress };
    }
  }

  export async function checkManually() {
    if (isChecking) return;
    isChecking = true;
    status = { phase: 'checking' };
    try {
      const info = await CheckForUpdates();
      if (info && info.version) {
        // Reset skipped version if user manually checked
        skippedVersion = '';
        localStorage.removeItem('aether_skipped_version');
        status = {
          phase: 'available',
          version: info.version,
          notes: info.releaseNotes,
        };
        showModal = true;
      } else {
        status = { phase: 'none' };
        toast.info("You're using the latest version of Aether Launcher!");
      }
    } catch (e: any) {
      status = { phase: 'error', message: String(e?.message || e) };
      showModal = true;
    } finally {
      isChecking = false;
    }
  }

  export function openModal() {
    if (status.phase !== 'none') {
      showModal = true;
    } else {
      checkManually();
    }
  }

  async function startUpdate() {
    try {
      status = { ...status, phase: 'downloading', progress: 0 };
      await DownloadAndUpdate();
    } catch (e: any) {
      status = { phase: 'error', message: String(e?.message || e) };
    }
  }

  function remindLater() {
    showModal = false;
  }

  function skipThisVersion() {
    if (status.version) {
      skippedVersion = status.version;
      localStorage.setItem('aether_skipped_version', status.version);
    }
    showModal = false;
  }

  function handleOpenEvent() {
    openModal();
  }

  onMount(() => {
    EventsOn('update:status', handleStatus);
    EventsOn('update:progress', handleProgress);
    window.addEventListener('aether:check-updates', handleOpenEvent);
  });

  onDestroy(() => {
    EventsOff('update:status');
    EventsOff('update:progress');
    window.removeEventListener('aether:check-updates', handleOpenEvent);
  });
</script>

{#if showModal && status.phase !== 'none'}
  <div
    class="modal-backdrop"
    role="presentation"
    on:click|self={remindLater}
    on:keydown={(e) => e.key === 'Escape' && remindLater()}
  >
    <div
      class="modal-card"
      role="dialog"
      aria-modal="true"
      aria-labelledby="update-title"
      on:click={(e) => e.stopPropagation()}
      on:keydown={(e) => e.stopPropagation()}
    >
      <div class="modal-header">
        <div class="update-icon-badge">
          <Icon name="package" size={20} color="#4f9cf9" />
        </div>
        <div class="header-text">
          <h2 id="update-title">
            {#if status.phase === 'available'}
              New Update Available
            {:else if status.phase === 'downloading'}
              Downloading Update...
            {:else if status.phase === 'ready'}
              Restarting Launcher...
            {:else if status.phase === 'error'}
              Update Failed
            {:else}
              Checking for Updates...
            {/if}
          </h2>
          {#if status.version}
            <span class="version-tag">Aether v{status.version}</span>
          {/if}
        </div>
        {#if status.phase === 'available' || status.phase === 'error'}
          <button class="close-btn" on:click={remindLater} title="Close">&times;</button>
        {/if}
      </div>

      <div class="modal-body">
        {#if status.phase === 'available'}
          <p class="body-intro">
            A new version of Aether Launcher is available to install. Update now for the latest features, performance improvements, and bug fixes.
          </p>

          {#if status.notes}
            <div class="release-notes-box">
              <div class="notes-header">Release Notes</div>
              <pre class="notes-content">{status.notes}</pre>
            </div>
          {/if}
        {:else if status.phase === 'downloading'}
          <div class="progress-section">
            <p>Downloading latest release package...</p>
            <div class="progress-bar-track">
              <div
                class="progress-bar-fill"
                style="width: {status.progress ?? 0}%"
              ></div>
            </div>
            <div class="progress-percentage">
              {status.progress != null ? `${status.progress}%` : 'Starting download...'}
            </div>
          </div>
        {:else if status.phase === 'ready'}
          <div class="ready-section">
            <div class="spinner"></div>
            <p>Installing update and restarting Aether Launcher...</p>
          </div>
        {:else if status.phase === 'error'}
          <div class="error-section">
            <div class="error-badge">!</div>
            <div class="error-message">
              {status.message || 'An unknown error occurred during update.'}
            </div>
          </div>
        {/if}
      </div>

      <div class="modal-footer">
        {#if status.phase === 'available'}
          <button class="btn btn-secondary" on:click={skipThisVersion}>
            Skip This Version
          </button>
          <button class="btn btn-secondary" on:click={remindLater}>
            Remind Me Later
          </button>
          <button class="btn btn-primary" on:click={startUpdate}>
            Update Now
          </button>
        {:else if status.phase === 'error'}
          <button class="btn btn-secondary" on:click={remindLater}>
            Dismiss
          </button>
          <button class="btn btn-primary" on:click={startUpdate}>
            Try Again
          </button>
        {:else if status.phase === 'downloading' || status.phase === 'ready'}
          <span class="updating-label">Updating Aether...</span>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(4px);
    z-index: 9999;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    box-sizing: border-box;
    animation: fadeIn 0.2s ease-out;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal-card {
    width: 100%;
    max-width: 520px;
    background: var(--card-bg, #1a1b23);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: var(--border-radius, 12px);
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    color: var(--text-primary, #ffffff);
    animation: slideUp 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes slideUp {
    from { transform: translateY(16px) scale(0.97); opacity: 0; }
    to { transform: translateY(0) scale(1); opacity: 1; }
  }

  .modal-header {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 20px 24px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    position: relative;
  }

  .update-icon-badge {
    width: 44px;
    height: 44px;
    border-radius: 10px;
    background: rgba(79, 156, 249, 0.15);
    border: 1px solid rgba(79, 156, 249, 0.3);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    flex-shrink: 0;
  }

  .header-text {
    flex: 1;
    min-width: 0;
  }

  .header-text h2 {
    margin: 0;
    font-size: 17px;
    font-weight: 700;
    line-height: 1.3;
    color: var(--text-primary, #fff);
  }

  .version-tag {
    display: inline-block;
    margin-top: 3px;
    padding: 2px 8px;
    border-radius: 12px;
    background: rgba(79, 156, 249, 0.18);
    color: var(--accent, #4f9cf9);
    font-size: 11px;
    font-weight: 600;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-secondary, #999);
    font-size: 16px;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 6px;
    transition: background 0.15s, color 0.15s;
  }

  .close-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-primary, #fff);
  }

  .modal-body {
    padding: 20px 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    max-height: 340px;
    overflow-y: auto;
  }

  .body-intro {
    margin: 0;
    font-size: 13px;
    line-height: 1.5;
    color: var(--text-secondary, #cccccc);
  }

  .release-notes-box {
    background: rgba(0, 0, 0, 0.25);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 8px;
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .notes-header {
    font-size: 11px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted, #888);
  }

  .notes-content {
    margin: 0;
    font-family: inherit;
    font-size: 12px;
    line-height: 1.5;
    color: var(--text-primary, #ddd);
    white-space: pre-wrap;
    word-break: break-word;
    max-height: 180px;
    overflow-y: auto;
  }

  .progress-section {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 10px 0;
  }

  .progress-section p {
    margin: 0;
    font-size: 13px;
    color: var(--text-secondary, #ccc);
  }

  .progress-bar-track {
    height: 8px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.1);
    overflow: hidden;
  }

  .progress-bar-fill {
    height: 100%;
    border-radius: 4px;
    background: var(--accent, #4f9cf9);
    transition: width 0.2s ease;
  }

  .progress-percentage {
    font-size: 12px;
    color: var(--text-muted, #aaa);
    text-align: right;
  }

  .ready-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: 20px 0;
    text-align: center;
  }

  .spinner {
    width: 28px;
    height: 28px;
    border: 3px solid rgba(255, 255, 255, 0.15);
    border-top-color: var(--accent, #4f9cf9);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-section {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 14px;
    background: rgba(232, 17, 35, 0.12);
    border: 1px solid rgba(232, 17, 35, 0.35);
    border-radius: 8px;
  }

  .error-badge {
    font-size: 20px;
  }

  .error-message {
    font-size: 13px;
    line-height: 1.4;
    color: #ff6b6b;
  }

  .modal-footer {
    padding: 16px 24px;
    background: rgba(0, 0, 0, 0.15);
    border-top: 1px solid rgba(255, 255, 255, 0.08);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 10px;
  }

  .btn {
    padding: 8px 16px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    border: 1px solid transparent;
    transition: background 0.15s, border-color 0.15s, opacity 0.15s;
  }

  .btn-secondary {
    background: rgba(255, 255, 255, 0.06);
    color: var(--text-primary, #fff);
    border-color: rgba(255, 255, 255, 0.12);
  }

  .btn-secondary:hover {
    background: rgba(255, 255, 255, 0.12);
    border-color: rgba(255, 255, 255, 0.2);
  }

  .btn-primary {
    background: var(--accent, #4f9cf9);
    color: #ffffff;
  }

  .btn-primary:hover {
    filter: brightness(1.1);
  }

  .updating-label {
    font-size: 12px;
    color: var(--text-muted, #aaa);
    font-style: italic;
  }
</style>
