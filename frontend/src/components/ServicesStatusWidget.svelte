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

  onMount(() => {
    refresh();
  });
</script>

<div class="widget-card card">
  <div class="widget-header">
    <div class="widget-title">
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
      </svg>
      <span>Live Services Health</span>
    </div>
    <div class="header-right">
      <span class="status-pill {connectivity?.overall || 'checking'}">
        <span class="status-dot"></span>
        <span>{connectivity?.overall ? connectivity.overall.toUpperCase() : 'CHECKING'}</span>
      </span>
      <button class="refresh-btn" on:click={refresh} disabled={checking} title="Probe services now">
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
    <div class="services-list">
      {#each connectivity.services as svc}
        <div class="service-row">
          <div class="service-info">
            <span class="svc-dot" class:online={svc.reachable} class:offline={!svc.reachable}></span>
            <span class="svc-name">{svc.name}</span>
          </div>
          <div class="service-metrics">
            {#if svc.reachable}
              <span class="svc-latency">{svc.latencyMs}ms</span>
            {:else}
              <span class="svc-offline">Unreachable</span>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .widget-card {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .widget-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .widget-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 10px;
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 999px;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .status-pill .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }

  .status-pill.online {
    color: #22c55e;
    background: rgba(34, 197, 94, 0.12);
    border-color: rgba(34, 197, 94, 0.3);
  }

  .status-pill.degraded {
    color: #f59e0b;
    background: rgba(245, 158, 11, 0.12);
    border-color: rgba(245, 158, 11, 0.3);
  }

  .status-pill.offline {
    color: #ef4444;
    background: rgba(239, 68, 68, 0.12);
    border-color: rgba(239, 68, 68, 0.3);
  }

  .status-pill.checking {
    color: #94a3b8;
    background: rgba(255, 255, 255, 0.05);
  }

  .refresh-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    padding: 4px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 0.15s ease, background 0.15s ease;
  }

  .refresh-btn:hover {
    color: var(--text-primary);
    background: rgba(255, 255, 255, 0.08);
  }

  .spin {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }

  .loading-state {
    padding: 18px;
    text-align: center;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .services-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .service-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 5px 8px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.02);
  }

  .service-info {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .svc-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: #6b7280;
    flex-shrink: 0;
  }

  .svc-dot.online {
    background: #22c55e;
    box-shadow: 0 0 6px rgba(34, 197, 94, 0.4);
  }

  .svc-dot.offline {
    background: #ef4444;
    box-shadow: 0 0 6px rgba(239, 68, 68, 0.4);
  }

  .svc-name {
    font-size: 12px;
    color: rgba(255, 255, 255, 0.85);
  }

  .service-metrics {
    display: flex;
    align-items: center;
  }

  .svc-latency {
    font-size: 11px;
    color: var(--text-secondary);
    font-family: monospace;
  }

  .svc-offline {
    font-size: 11px;
    color: #ef4444;
    font-weight: 500;
  }
</style>
