<script lang="ts">
  import { onMount } from 'svelte';
  import { GetRecentScreenshots, OpenScreenshot, OpenScreenshotsFolder } from '../../wailsjs/go/main/App';

  let screenshots: Array<{
    instanceId: string;
    instanceName: string;
    fileName: string;
    dataUrl: string;
    modTime: string;
  }> = [];
  let loading = true;
  let activePreview: any = null;

  async function loadScreenshots() {
    loading = true;
    try {
      screenshots = (await GetRecentScreenshots(4)) || [];
    } catch (e) {
      console.error('Failed to load recent screenshots:', e);
      screenshots = [];
    } finally {
      loading = false;
    }
  }

  function openPreview(sc: any) {
    activePreview = sc;
  }

  function closePreview() {
    activePreview = null;
  }

  async function handleOpenInViewer() {
    if (!activePreview) return;
    try {
      await OpenScreenshot(activePreview.instanceId, activePreview.fileName);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleOpenFolder() {
    if (!activePreview) return;
    try {
      await OpenScreenshotsFolder(activePreview.instanceId);
    } catch (e) {
      console.error(e);
    }
  }

  function formatTime(iso: string): string {
    if (!iso) return '';
    try {
      const d = new Date(iso);
      return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
    } catch {
      return '';
    }
  }

  onMount(() => {
    loadScreenshots();
  });
</script>

<div class="widget-card card">
  <div class="widget-header">
    <div class="widget-title">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
        <circle cx="12" cy="13" r="4"/>
      </svg>
      <span>Screenshots</span>
    </div>
    {#if screenshots.length > 0}
      <button class="refresh-btn" on:click={loadScreenshots} title="Refresh screenshots">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="23 4 23 10 17 10"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
        </svg>
      </button>
    {/if}
  </div>

  {#if loading}
    <div class="loading-wrap">
      <span class="loading-text">Scanning screenshots...</span>
    </div>
  {:else if screenshots.length === 0}
    <div class="empty-compact">
      <div class="empty-left">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
          <circle cx="8.5" cy="8.5" r="1.5"/>
          <polyline points="21 15 16 10 5 21"/>
        </svg>
        <span class="empty-text">No screenshots yet</span>
      </div>
      <span class="empty-hint">Press <kbd>F2</kbd> in-game</span>
    </div>
  {:else}
    <div class="screenshots-grid">
      {#each screenshots as sc (sc.instanceId + sc.fileName)}
        <button class="screenshot-tile" on:click={() => openPreview(sc)} title="{sc.fileName} • {sc.instanceName}">
          <img src={sc.dataUrl} alt={sc.fileName} class="thumb-img" loading="lazy" />
          <div class="tile-overlay">
            <span class="tile-inst">{sc.instanceName}</span>
            <span class="tile-date">{formatTime(sc.modTime)}</span>
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>

{#if activePreview}
  <div
    class="lightbox-backdrop"
    role="button"
    tabindex="0"
    on:click={closePreview}
    on:keydown={(e) => { if (e.key === 'Escape' || e.key === 'Enter') closePreview(); }}
  >
    <!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
    <div
      class="lightbox-modal"
      role="dialog"
      aria-modal="true"
      tabindex="-1"
      on:click|stopPropagation
      on:keydown|stopPropagation
    >
      <div class="lightbox-header">
        <div class="lightbox-title-wrap">
          <span class="lightbox-title">{activePreview.fileName}</span>
          <span class="lightbox-subtitle">{activePreview.instanceName} · {formatTime(activePreview.modTime)}</span>
        </div>
        <div class="lightbox-actions">
          <button class="btn btn-secondary btn-sm" on:click={handleOpenInViewer} title="Open in Windows Photo Viewer">
            Open in App
          </button>
          <button class="btn btn-secondary btn-sm" on:click={handleOpenFolder} title="Show file in folder">
            Show in Folder
          </button>
          <button class="close-btn" on:click={closePreview} title="Close preview">✕</button>
        </div>
      </div>
      <div class="lightbox-body">
        <img src={activePreview.dataUrl} alt={activePreview.fileName} class="lightbox-img" />
      </div>
    </div>
  </div>
{/if}

<style>
  .widget-card {
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    background: var(--panel-bg);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: var(--border-radius);
  }

  .widget-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .widget-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }

  .refresh-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    padding: 3px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.15s ease, background 0.15s ease;
  }

  .refresh-btn:hover {
    color: var(--text-primary);
    background: rgba(255, 255, 255, 0.06);
  }

  .loading-wrap {
    padding: 14px;
    text-align: center;
  }

  .loading-text {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .empty-compact {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: rgba(255, 255, 255, 0.015);
    border: 1px dashed rgba(255, 255, 255, 0.07);
    border-radius: 6px;
  }

  .empty-left {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--text-secondary);
    font-size: 11px;
  }

  .empty-hint {
    font-size: 11px;
    color: rgba(255, 255, 255, 0.4);
  }

  kbd {
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 3px;
    padding: 1px 4px;
    font-size: 10px;
    color: var(--text-primary);
  }

  .screenshots-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }

  .screenshot-tile {
    position: relative;
    border-radius: 6px;
    overflow: hidden;
    height: 82px;
    background: #000;
    border: 1px solid rgba(255, 255, 255, 0.05);
    padding: 0;
    cursor: pointer;
    transition: transform 0.15s ease, border-color 0.15s ease;
  }

  .screenshot-tile:hover {
    transform: translateY(-1px);
    border-color: rgba(255, 255, 255, 0.2);
  }

  .thumb-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .tile-overlay {
    position: absolute;
    inset: 0;
    background: linear-gradient(to top, rgba(0, 0, 0, 0.8) 0%, transparent 60%);
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
    padding: 5px 8px;
    opacity: 0;
    transition: opacity 0.15s ease;
  }

  .screenshot-tile:hover .tile-overlay {
    opacity: 1;
  }

  .tile-inst {
    font-size: 10px;
    font-weight: 600;
    color: #ffffff;
    max-width: 65%;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tile-date {
    font-size: 10px;
    color: rgba(255, 255, 255, 0.6);
  }

  /* Lightbox */
  .lightbox-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.85);
    backdrop-filter: blur(8px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    padding: 24px;
    box-sizing: border-box;
  }

  .lightbox-modal {
    background: #181818;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    max-width: 900px;
    width: 100%;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    box-shadow: 0 16px 40px rgba(0, 0, 0, 0.6);
  }

  .lightbox-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    background: #141414;
  }

  .lightbox-title-wrap {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .lightbox-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .lightbox-subtitle {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .lightbox-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    font-size: 16px;
    padding: 4px 8px;
    cursor: pointer;
    border-radius: 4px;
  }

  .close-btn:hover {
    color: #ffffff;
    background: rgba(255, 255, 255, 0.08);
  }

  .lightbox-body {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #0d0d0d;
    overflow: hidden;
    padding: 16px;
  }

  .lightbox-img {
    max-width: 100%;
    max-height: 70vh;
    object-fit: contain;
    border-radius: 6px;
  }
</style>
