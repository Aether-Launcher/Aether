<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';
  import { DownloadAndUpdate } from '../../wailsjs/go/main/App.js';

  type Status = {
    phase: 'none' | 'checking' | 'available' | 'downloading' | 'ready' | 'error';
    version?: string;
    message?: string;
    progress?: number;
  };

  let status: Status = { phase: 'none' };
  let dismissed = false;

  function apply(payload: unknown) {
    status = (payload as Status) || { phase: 'none' };
  }

  function applyProgress(payload: unknown) {
    const p = payload as { progress?: number };
    if (status.phase === 'downloading' && p && typeof p.progress === 'number') {
      status = { ...status, progress: p.progress };
    }
  }

  async function update() {
    try {
      await DownloadAndUpdate();
    } catch (e) {
      status = { phase: 'error', message: String(e) };
    }
  }

  onMount(() => {
    EventsOn('update:status', apply);
    EventsOn('update:progress', applyProgress);
  });

  onDestroy(() => {
    EventsOff('update:status');
    EventsOff('update:progress');
  });
</script>

{#if !dismissed && status.phase !== 'none' && status.phase !== 'checking'}
  <div class="update-banner {status.phase}">
    {#if status.phase === 'available'}
      <div class="upd-title">Update to v{status.version} available</div>
      <button class="btn btn-primary btn-sm" on:click={update}>Update</button>
    {:else if status.phase === 'downloading'}
      <div class="upd-title">Downloading v{status.version}...</div>
      {#if status.progress != null}
        <div class="upd-progress">
          <div class="upd-progress-fill" style="width: {status.progress}%"></div>
        </div>
      {/if}
    {:else if status.phase === 'ready'}
      <div class="upd-title">Restarting...</div>
    {:else if status.phase === 'error'}
      <div class="upd-title">Update failed: {status.message}</div>
    {/if}

    {#if status.phase === 'available' || status.phase === 'error'}
      <button class="upd-dismiss" on:click={() => (dismissed = true)} title="Dismiss">×</button>
    {/if}
  </div>
{/if}

<style>
  .update-banner {
    position: relative;
    margin: 8px 0;
    padding: 10px 12px;
    border-radius: var(--border-radius);
    background: rgba(79, 156, 249, 0.12);
    border: 1px solid rgba(79, 156, 249, 0.35);
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 12px;
    color: var(--text-primary);
  }

  .update-banner.error {
    background: rgba(232, 17, 35, 0.1);
    border-color: rgba(232, 17, 35, 0.4);
  }

  .upd-title {
    line-height: 1.35;
    padding-right: 16px;
    overflow-wrap: anywhere;
  }

  .upd-progress {
    height: 4px;
    border-radius: 2px;
    background: rgba(255, 255, 255, 0.1);
    overflow: hidden;
  }

  .upd-progress-fill {
    height: 100%;
    border-radius: 2px;
    background: var(--accent, #4f9cf9);
    transition: width 0.3s ease;
  }

  .upd-dismiss {
    position: absolute;
    top: 4px;
    right: 6px;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    font-size: 16px;
    line-height: 1;
    cursor: pointer;
    padding: 2px 6px;
  }

  .upd-dismiss:hover {
    color: var(--text-primary);
  }
</style>