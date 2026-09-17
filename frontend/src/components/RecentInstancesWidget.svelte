<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { gameStore } from '../stores/gameStore';

  export let instances: any[] = [];
  export let activeInstanceId: string = '';

  const dispatch = createEventDispatcher();

  function formatLastPlayed(ts: number | undefined): string {
    if (!ts || ts === 0) return 'Never played';
    const now = Math.floor(Date.now() / 1000);
    const diff = now - ts;
    if (diff < 60) return 'Just now';
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
    return new Date(ts * 1000).toLocaleDateString();
  }

  function instanceGradient(name: string): string {
    const gradients = [
      'linear-gradient(135deg, #3b82f6, #1d4ed8)',
      'linear-gradient(135deg, #8b5cf6, #6d28d9)',
      'linear-gradient(135deg, #06b6d4, #0284c7)',
      'linear-gradient(135deg, #10b981, #047857)',
      'linear-gradient(135deg, #f59e0b, #b45309)',
      'linear-gradient(135deg, #ec4899, #be185d)',
    ];
    let hash = 0;
    for (let i = 0; i < (name || '').length; i++) hash = (name || '').charCodeAt(i) + ((hash << 5) - hash);
    return gradients[Math.abs(hash) % gradients.length];
  }

  $: candidateList = (() => {
    const list = (instances || []).filter((i) => i.id !== activeInstanceId);
    list.sort((a, b) => (b.lastPlayed || 0) - (a.lastPlayed || 0));
    return list.slice(0, 3);
  })();

  function handlePlay(inst: any) {
    dispatch('play', inst);
  }

  function handleViewAll() {
    dispatch('navigate', 'instances');
  }
</script>

<div class="widget-card card">
  <div class="widget-header">
    <div class="widget-title">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"/>
        <polyline points="12 6 12 12 16 14"/>
      </svg>
      <span>Recent Instances</span>
    </div>
    <button class="view-all-btn" on:click={handleViewAll}>View all</button>
  </div>

  {#if candidateList.length === 0}
    <div class="empty-compact">
      <span class="empty-text">No other instances created</span>
      <button class="btn btn-secondary btn-sm new-btn" on:click={() => window.dispatchEvent(new CustomEvent('aether:open-create-instance'))}>
        + New Instance
      </button>
    </div>
  {:else}
    <div class="recent-list">
      {#each candidateList as inst (inst.id)}
        {@const grad = instanceGradient(inst.name)}
        <div class="recent-row">
          <div class="recent-art" style="background: {grad};">
            <span>{inst.name.charAt(0).toUpperCase()}</span>
          </div>

          <div class="recent-info">
            <div class="recent-name" title={inst.name}>{inst.name}</div>
            <div class="recent-meta">
              <span class="meta-tag">{inst.version}</span>
              <span class="meta-dot">·</span>
              <span class="meta-tag">{inst.loader || 'Vanilla'}</span>
              <span class="meta-dot">·</span>
              <span class="meta-time">{formatLastPlayed(inst.lastPlayed)}</span>
            </div>
          </div>

          <button
            class="btn btn-secondary btn-sm play-btn"
            on:click={() => handlePlay(inst)}
            disabled={!inst.installed || ($gameStore.instanceId === inst.id && ($gameStore.state === 'Starting...' || $gameStore.state === 'Running'))}
            title="Launch {inst.name}"
          >
            {#if !inst.installed}
              Install
            {:else if $gameStore.instanceId === inst.id && $gameStore.state === 'Starting...'}
              Starting...
            {:else if $gameStore.instanceId === inst.id && $gameStore.state === 'Running'}
              Running
            {:else}
              Play
            {/if}
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>

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

  .view-all-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    font-size: 11px;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
    transition: color 0.15s ease, background 0.15s ease;
  }

  .view-all-btn:hover {
    color: var(--text-primary);
    background: rgba(255, 255, 255, 0.06);
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

  .empty-text {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .new-btn {
    padding: 4px 10px;
    font-size: 11px;
  }

  .recent-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .recent-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 7px 10px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.035);
    border-radius: 6px;
    transition: background 0.15s ease, border-color 0.15s ease;
  }

  .recent-row:hover {
    background: rgba(255, 255, 255, 0.04);
    border-color: rgba(255, 255, 255, 0.07);
  }

  .recent-art {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 700;
    color: #ffffff;
    flex-shrink: 0;
  }

  .recent-info {
    flex: 1;
    min-width: 0;
  }

  .recent-name {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recent-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 10px;
    color: var(--text-secondary);
    margin-top: 1px;
  }

  .meta-tag {
    color: rgba(255, 255, 255, 0.7);
  }

  .meta-dot {
    color: rgba(255, 255, 255, 0.2);
  }

  .meta-time {
    color: var(--text-secondary);
  }

  .play-btn {
    padding: 4px 10px;
    font-size: 11px;
    font-weight: 600;
    flex-shrink: 0;
  }
</style>
