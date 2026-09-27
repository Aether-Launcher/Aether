<script lang="ts">
  import { onMount } from 'svelte';
  import { GetConnectivityStatus } from '../../wailsjs/go/main/App';

  let connectivity: any = null;
  let checking = false;

  async function refresh() {
    if (checking) return;
    checking = true;
    try {
      connectivity = await GetConnectivityStatus();
    } catch (e) {
      console.error(e);
      connectivity = { overall: 'unknown', services: [] };
    } finally {
      checking = false;
    }
  }

  function getStatusLabel(overall: string | undefined): string {
    if (!overall || overall === 'checking') return 'Checking';
    if (overall === 'online') return 'Operational';
    if (overall === 'degraded') return 'Degraded';
    return 'Offline';
  }

  onMount(() => {
    refresh();
  });
</script>

<div class="widget-card card">
  <div class="widget-header">
    <div class="widget-title">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
      </svg>
      <span>Live Services</span>
    </div>
    <div class="header-right">
      <span class="status-pill {connectivity?.overall || 'checking'}">
        <span class="status-dot"></span>
        <span>{getStatusLabel(connectivity?.overall)}</span>
      </span>
      <button class="refresh-btn" on:click={refresh} disabled={checking} title="Probe endpoints">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class:spin={checking}>
          <polyline points="23 4 23 10 17 10"/>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
        </svg>
      </button>
    </div>
  </div>

  {#if !connectivity || !connectivity.services || connectivity.services.length === 0}
    <div class="loading-state">
      <span>Probing endpoints...</span>
    </div>
  {:else}
    <div class="services-grid">
      {#each connectivity.services as svc}
        <div class="service-chip" title="{svc.name}: {svc.reachable ? svc.latencyMs + 'ms' : 'Unreachable'}">
          <div class="chip-left">
            <span class="svc-dot" class:online={svc.reachable} class:offline={!svc.reachable}></span>
            <span class="svc-name">{svc.name}</span>
          </div>
          <span class="svc-latency" class:offline={!svc.reachable}>
            {svc.reachable ? svc.latencyMs + 'ms' : 'down'}
          </span>
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

  .header-right {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 10px;
    font-weight: 600;
    padding: 2px 7px;
    border-radius: 999px;
    border: 1px solid rgba(255, 255, 255, 0.08);
  }

  .status-pill .status-dot {
    width: 5px;
    height: 5px;
    border-radius: 50%;
    background: currentColor;
  }

  .status-pill.online {
    color: #22c55e;
    background: rgba(34, 197, 94, 0.1);
    border-color: rgba(34, 197, 94, 0.25);
  }

  .status-pill.degraded {
    color: #f59e0b;
    background: rgba(245, 158, 11, 0.1);
    border-color: rgba(245, 158, 11, 0.25);
  }

  .status-pill.offline {
    color: #ef4444;
    background: rgba(239, 68, 68, 0.1);
    border-color: rgba(239, 68, 68, 0.25);
  }

  .status-pill.checking {
    color: #94a3b8;
    background: rgba(255, 255, 255, 0.04);
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

  .spin {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .loading-state {
    padding: 10px;
    text-align: center;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .services-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
  }

  .service-chip {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 5px 8px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.035);
    border-radius: 5px;
    gap: 6px;
    min-width: 0;
  }

  .chip-left {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    overflow: hidden;
  }

  .svc-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #6b7280;
    flex-shrink: 0;
  }

  .svc-dot.online {
    background: #22c55e;
  }

  .svc-dot.offline {
    background: #ef4444;
  }

  .svc-name {
    font-size: 11px;
    color: rgba(255, 255, 255, 0.8);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .svc-latency {
    font-size: 10px;
    color: var(--text-secondary);
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    flex-shrink: 0;
  }

  .svc-latency.offline {
    color: #ef4444;
  }
</style>
