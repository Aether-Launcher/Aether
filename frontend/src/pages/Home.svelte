<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { GetActiveInstance, GetInstances, LaunchInstance, InstallInstance, GetExtensions } from '../../wailsjs/go/main/App.js';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';
  import { gameStore } from '../stores/gameStore.js';
  import EmptyState from '../components/EmptyState.svelte';

  const dispatch = createEventDispatcher();

  // When navigated from instance creation, this will be set to the new instance ID
  export let activeInstanceId: string = '';

  let currentInstance: any = null;
  let recentInstances: any[] = [];
  let extensions: any[] = [];
  let installProgress = 0;
  let installStatusText = '';
  let javaStatus: { phase: string; message: string; progress?: number } | null = null;

  // Derive from global store — survives navigation
  $: launchState = ($gameStore.instanceId === currentInstance?.id) ? $gameStore.state : 'Idle';
  $: logs = ($gameStore.instanceId === currentInstance?.id) ? $gameStore.logs.slice(-10) : [];

  onMount(async () => {
    await loadHome();

    // instance:state and instance:log are handled globally in gameStore.
    // Install progress is page-local — only relevant while Home is mounted.
    EventsOn("instance:progress", (data: any) => {
      if (currentInstance && data.id === currentInstance.id) {
        installProgress = data.progress;
        installStatusText = data.status;
        if (data.progress >= 100) {
          currentInstance.installed = true;
          installStatusText = '';
        }
      }
    });

    EventsOn("java:status", (data: any) => {
      if (data.phase === 'done') {
        javaStatus = null;
      } else {
        javaStatus = data;
      }
    });
  });

  // Reactive: when activeInstanceId changes (e.g. after instance creation), reload and install
  $: if (activeInstanceId) {
    loadAndInstall(activeInstanceId);
  }

  async function loadHome() {
    const all = await GetInstances();
    // If we have a forced activeInstanceId use it, otherwise use GetActiveInstance
    if (activeInstanceId) {
      currentInstance = all.find((i: any) => i.id === activeInstanceId) || await GetActiveInstance();
    } else {
      currentInstance = await GetActiveInstance();
    }

    recentInstances = all
      .filter((i: any) => !currentInstance || i.id !== currentInstance.id)
      .sort((a: any, b: any) => {
        if (!a.lastPlayed && !b.lastPlayed) return 0;
        if (!a.lastPlayed) return 1;
        if (!b.lastPlayed) return -1;
        return b.lastPlayed.localeCompare(a.lastPlayed);
      })
      .slice(0, 5);

    const exts = await GetExtensions();
    extensions = exts || [];
  }

  async function loadAndInstall(id: string) {
    const all = await GetInstances();
    const inst = all.find((i: any) => i.id === id);
    if (!inst) return;
    currentInstance = inst;
    recentInstances = all.filter((i: any) => i.id !== id).slice(0, 5);
    // Begin installation immediately if not already installed
    if (!inst.installed) {
      await handleInstall();
    }
  }

  async function handlePlay() {
    if (!currentInstance) return;
    gameStore.update(s => ({ ...s, instanceId: currentInstance.id, state: 'Starting...' }));
    try {
      await LaunchInstance(currentInstance.id);
    } catch (err) {
      gameStore.update(s => ({ ...s, state: 'Error' }));
      console.error(err);
    }
  }

  async function handleInstall() {
    if (!currentInstance) return;
    installStatusText = 'Installing...';
    installProgress = 0;
    try {
      await InstallInstance(currentInstance.id);
    } catch (err) {
      installStatusText = 'Error';
      console.error(err);
    }
  }

  async function handleQuickPlay(inst: any) {
    try {
      await LaunchInstance(inst.id);
    } catch (err) {
      console.error('Quick play failed:', err);
    }
  }

  function instanceGradient(name: string): string {
    const gradients = [
      'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)',
      'linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%)',
      'linear-gradient(135deg, #06b6d4 0%, #0284c7 100%)',
      'linear-gradient(135deg, #10b981 0%, #047857 100%)',
      'linear-gradient(135deg, #f59e0b 0%, #b45309 100%)',
      'linear-gradient(135deg, #ec4899 0%, #be185d 100%)',
    ];
    let h = 0;
    for (let i = 0; i < name.length; i++) h = name.charCodeAt(i) + ((h << 5) - h);
    return gradients[Math.abs(h) % gradients.length];
  }

  $: artGradient = currentInstance ? instanceGradient(currentInstance.name) : '';

  function formatLastPlayed(dateStr: string): string {
    if (!dateStr) return 'Never';
    try {
      const d = new Date(dateStr);
      const now = new Date();
      const diffMs = now.getTime() - d.getTime();
      const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
      if (diffDays === 0) return 'Today';
      if (diffDays === 1) return 'Yesterday';
      if (diffDays < 7) return `${diffDays} days ago`;
      if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
      return d.toLocaleDateString();
    } catch {
      return dateStr;
    }
  }

  /**
   * Normalise a raw memory string from the instance config into a human-friendly label.
   * Handles: '4G', '4g', '4096', '4096M', '4096m', '2048', etc.
   */
  function formatMemory(raw: string): string {
    if (!raw || raw === 'Default') return 'Default';
    const upper = raw.trim().toUpperCase();
    // Already in GB form — e.g. '4G' or '4GB'
    const gbMatch = upper.match(/^(\d+(?:\.\d+)?)\s*G(?:B)?$/);
    if (gbMatch) return `${gbMatch[1]} GB`;
    // In MB form — e.g. '4096M' or '4096MB'
    const mbMatch = upper.match(/^(\d+(?:\.\d+)?)\s*M(?:B)?$/);
    if (mbMatch) {
      const mb = parseFloat(mbMatch[1]);
      return mb >= 1024 ? `${(mb / 1024).toFixed(mb % 1024 === 0 ? 0 : 1)} GB` : `${mb} MB`;
    }
    // Plain number — assume MB
    const numMatch = upper.match(/^(\d+(?:\.\d+)?)$/);
    if (numMatch) {
      const mb = parseFloat(numMatch[1]);
      return mb >= 1024 ? `${(mb / 1024).toFixed(mb % 1024 === 0 ? 0 : 1)} GB` : `${mb} MB`;
    }
    return raw;
  }
</script>

<div class="page">
  {#if currentInstance}
    <div class="content-wrapper">
      
      <!-- ── Active Instance ─────────────────────────────────── -->
      <div class="section">
        <div class="section-label">{launchState === 'Running' ? 'Now Playing' : 'Current Instance'}</div>

        <div class="launch-container">
          <div class="instance-header">
            <div class="instance-art" style="background: {artGradient};">
              <span class="instance-art-letter">
                {currentInstance.name.charAt(0).toUpperCase()}
              </span>
            </div>

            <div class="instance-info">
              <div class="instance-name">{currentInstance.name}</div>
              <div class="instance-meta">
                <span>{currentInstance.version}</span> •
                <span>{currentInstance.loader}</span> •
                <span>{formatMemory(currentInstance.memory)}</span>
              </div>
            </div>
          </div>

          <div class="actions-row">
            {#if currentInstance.installed}
              <button
                class="btn btn-primary play-btn"
                on:click={handlePlay}
                disabled={launchState === 'Running'}
              >
                {launchState === 'Running' ? 'Running' : 'Play'}
              </button>
              <button class="btn btn-secondary" on:click={() => dispatch('navigate', `instance-details:${currentInstance.id}`)}>
                Settings
              </button>
            {:else}
              <div class="install-col">
                <button
                  class="btn btn-primary play-btn"
                  on:click={handleInstall}
                  disabled={installStatusText === 'Installing...' || (installProgress > 0 && installProgress < 100)}
                >
                  {installStatusText === '' || installStatusText === 'Error' ? 'Install' : 'Installing...'}
                </button>
                {#if installProgress > 0 && installProgress < 100}
                  <div class="progress-track">
                    <div class="progress-fill" style="width: {installProgress}%; background: {artGradient};"></div>
                  </div>
                {/if}
              </div>
            {/if}

            {#if javaStatus}
              <div class="java-status">
                <span class="java-status-label">{javaStatus.message}</span>
                {#if javaStatus.progress !== undefined}
                  <div class="progress-track">
                    <div class="progress-fill" style="width: {javaStatus.progress}%; background: linear-gradient(90deg, #f59e0b, #d97706);"></div>
                  </div>
                {/if}
              </div>
            {/if}

            {#if launchState !== 'Idle' && launchState !== ''}
              <span class="status-label">{launchState}</span>
            {/if}
          </div>

          {#if logs.length > 0}
            <div class="log-panel">
              {#each logs as log}
                <div class="log-line">{log}</div>
              {/each}
            </div>
          {/if}
        </div>
      </div>

      <div class="divider"></div>

      <!-- ── Recently Played ──────────────────────────────────── -->
      {#if recentInstances.length > 0}
        <div class="section">
          <div class="section-label">Recently Played</div>

          <div class="recent-list">
            {#each recentInstances as inst}
              {@const grad = instanceGradient(inst.name)}
              <div class="recent-row">
                <div class="recent-art" style="background: {grad};">
                  <span class="recent-art-letter">{inst.name.charAt(0).toUpperCase()}</span>
                </div>

                <div class="recent-info">
                  <div class="recent-name">{inst.name}</div>
                  <div class="recent-meta">
                    {inst.version} • {inst.loader}
                    <span class="recent-time">· {formatLastPlayed(inst.lastPlayed)}</span>
                  </div>
                </div>

                <button
                  class="btn btn-secondary recent-play-btn"
                  on:click={() => handleQuickPlay(inst)}
                  title="Launch {inst.name}"
                  disabled={!inst.installed}
                >
                  {inst.installed ? 'Play' : 'Not installed'}
                </button>
              </div>
            {/each}
          </div>
        </div>
        
        <div class="divider"></div>
      {/if}

      <!-- ── Extension Updates ────────────────────────────────── -->
      <div class="section">
        <div class="section-label">Extension Updates</div>
        <div class="updates-box">
          {#if extensions.length === 0}
            <span class="updates-text">No extensions installed.</span>
            <button class="btn btn-secondary btn-sm" on:click={() => dispatch('navigate', 'extensions')}>Browse Gallery</button>
          {:else}
            <span class="updates-text">All extensions are up to date.</span>
          {/if}
        </div>
      </div>

    </div>
  {:else}
    <EmptyState
      icon="play"
      title="Nothing to play"
      description="Select or create an instance from the Instances tab."
      actionLabel="Go to Instances"
      on:action={() => dispatch('navigate', 'instances')}
    />
  {/if}
</div>

<style>
  .page {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: var(--spacing-xl);
    box-sizing: border-box;
    overflow-y: auto;
  }

  /* Wraps the entire layout to keep it max-width and handle spacing */
  .content-wrapper {
    display: flex;
    flex-direction: column;
    max-width: 560px;
    width: 100%;
  }

  /* ── Structural Separator ── */
  .divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.08);
    width: 100%;
    margin: var(--spacing-xl) 0;
  }

  /* ── Section ── */
  .section {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
  }

  .section-label {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  /* ── Active Instance ── */
  .launch-container {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
  }

  .instance-header {
    display: flex;
    align-items: center;
    gap: var(--spacing-lg);
  }

  .instance-art {
    width: 64px;
    height: 64px;
    border-radius: 14px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.35);
  }

  .instance-art-letter {
    font-size: 28px;
    font-weight: 800;
    color: rgba(255, 255, 255, 0.9);
    line-height: 1;
  }

  .instance-info {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .instance-name {
    font-size: 32px;
    font-weight: 700;
    letter-spacing: -1px;
    line-height: 1;
  }

  .instance-meta {
    font-size: 13px;
    color: var(--text-meta);
    font-weight: 500;
  }

  .actions-row {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    /* Prevent status-text changes from reflowing button positions */
    flex-wrap: nowrap;
  }

  .install-col {
    display: flex;
    flex-direction: column;
    gap: 8px;
    /* Fixed width prevents the column from growing/shrinking with status text */
    width: 200px;
    min-width: 200px;
    flex-shrink: 0;
  }

  .play-btn {
    font-size: 14px;
    padding: 11px 36px;
    text-transform: uppercase;
    letter-spacing: 1.5px;
    font-weight: 700;
    box-shadow: 0 4px 16px rgba(59, 130, 246, 0.3);
    transition: box-shadow var(--transition-fast), transform 80ms ease;
  }

  .play-btn:hover:not(:disabled) {
    box-shadow: 0 6px 22px rgba(59, 130, 246, 0.5);
    transform: translateY(-1px);
  }

  .status-label {
    font-size: 13px;
    color: var(--text-secondary);
    /* Fixed-width container: status text never pushes buttons wider */
    min-width: 0;
    max-width: 200px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex-shrink: 1;
  }

  .java-status {
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
  }

  .java-status-label {
    font-size: 12px;
    color: #f59e0b;
    font-weight: 500;
  }

  .progress-track {
    width: 100%;
    height: 3px;
    background: rgba(255, 255, 255, 0.08);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    border-radius: 2px;
    transition: width 0.2s ease;
  }

  .log-panel {
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 11px;
    color: var(--text-secondary);
    background: rgba(0, 0, 0, 0.25);
    padding: 10px 14px;
    border-radius: var(--border-radius);
    border: 1px solid rgba(255, 255, 255, 0.05);
    width: 100%;
  }

  .log-line { line-height: 1.6; }

  /* ── Recently Played ── */
  .recent-list {
    display: flex;
    flex-direction: column;
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: var(--border-radius);
    overflow: hidden;
  }

  .recent-row {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    padding: 12px var(--spacing-md);
    background: var(--panel-bg);
    transition: background var(--transition-fast);
  }

  .recent-row:not(:last-child) {
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  }

  .recent-row:hover {
    background: rgba(255, 255, 255, 0.04);
  }

  .recent-art {
    width: 38px;
    height: 38px;
    border-radius: 8px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 3px 8px rgba(0, 0, 0, 0.3);
  }

  .recent-art-letter {
    font-size: 16px;
    font-weight: 800;
    color: rgba(255, 255, 255, 0.9);
    line-height: 1;
  }

  .recent-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .recent-name {
    font-size: 14px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .recent-meta {
    font-size: 12px;
    color: var(--text-meta);
  }

  .recent-time {
    color: var(--text-secondary);
  }

  .recent-play-btn {
    flex-shrink: 0;
    font-size: 12px;
    padding: 6px 14px;
  }

  /* ── Extension Updates ── */
  .updates-box {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-md);
    background: var(--panel-bg);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: var(--border-radius);
  }

  .updates-text {
    font-size: 13px;
    color: var(--text-secondary);
  }
  
  .btn-sm {
    padding: 6px 12px;
    font-size: 12px;
  }
</style>
