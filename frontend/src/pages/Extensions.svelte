<script lang="ts">
  import { onMount } from 'svelte';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';
  import { GetExtensions, SelectAndInstallExtension, DownloadAndInstallExtension, GetSettings, UninstallExtension, GetExtensionUpdates, UpdateExtension, ReloadExtensions, GetLauncherVersion } from '../../wailsjs/go/main/App.js';
  import EmptyState from '../components/EmptyState.svelte';
  import ConfirmDialog from '../lib/components/ConfirmDialog.svelte';
  import { toast } from '../stores/toast';

  let installedExtensions: any[] = [];
  let galleryExtensions: any[] = [];
  let isInstalling = false;
  let activeTab = 'installed'; // 'installed' or 'gallery'
  let galleryError = '';
  let galleryLoading = false;
  let gallerySearch = '';

  // Real GitHub URL for the Aether Extension Registry
  const GALLERY_INDEX_URL = 'https://raw.githubusercontent.com/wayback09/Aether-Extensions/main/index.json';

  let isDevMode = false;
  let confirmDialog: any;
  let pendingUninstall: any = null;

  let updates: any[] = [];
  let checkingUpdates = false;
  let updatingId = '';
  let reloading = false;
  let launcherVersion = 'dev'; // will be loaded on mount

  $: filteredGalleryExtensions = galleryExtensions.filter((ext) => {
    const query = gallerySearch.trim().toLowerCase();
    if (!query) return true;
    return [ext.name, ext.author, ext.id, ext.description].some((value) => String(value || '').toLowerCase().includes(query));
  });

  function compareVersions(left: string, right: string): number {
    const parse = (version: string) => {
      const match = String(version || '').trim().replace(/^v/i, '').match(/^(\d+)(?:\.(\d+))?(?:\.(\d+))?/);
      return match ? [Number(match[1]), Number(match[2] || 0), Number(match[3] || 0)] : null;
    };
    const a = parse(left), b = parse(right);
    if (!a || !b) return 0;
    for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return a[i] > b[i] ? 1 : -1;
    return 0;
  }

  function installedFor(ext: any): any { return installedExtensions.find((installed) => installed.id === ext.id); }
  function updateFor(ext: any): any { return updates.find((u) => u.id === ext.id); }
  function hasUpdate(ext: any): boolean {
    const installed = installedFor(ext);
    return !!installed && compareVersions(ext.version, installed.version) > 0;
  }

  // Returns the required min version string if the current launcher is too old, otherwise null.
  function requiresNewerLauncher(ext: any): string | null {
    const min = ext.minLauncherVersion;
    if (!min) return null;
    // "dev" builds are treated as compatible (local dev)
    if (launcherVersion === 'dev') return null;
    return compareVersions(launcherVersion, min) < 0 ? min : null;
  }

  async function fetchUpdates(): Promise<any[]> {
    try {
      return (await GetExtensionUpdates()) || [];
    } catch (e: any) {
      throw new Error(String(e || 'Could not reach the extension registry'));
    }
  }

  async function checkForUpdates() {
    if (checkingUpdates) return;
    checkingUpdates = true;
    try {
      updates = await fetchUpdates();
      if (updates.length === 0) toast.info('No updates available.');
    } catch (e: any) {
      updates = [];
      console.error('Failed to check for updates:', e);
      toast.error('Could not check for updates: ' + (e?.message || e));
    } finally {
      checkingUpdates = false;
    }
  }

  async function handleUpdate(ext: any) {
    if (updatingId) return;
    updatingId = ext.id;
    try {
      const updated = await UpdateExtension(ext.id);
      await loadInstalled();
      try {
        updates = await fetchUpdates();
      } catch { /* registry unreachable — keep current list */ }
      toast.success(`Updated ${updated.name} to v${updated.newVersion}`);
    } catch (e: any) {
      console.error('Update failed:', e);
      toast.error('Update failed: ' + e);
    } finally {
      updatingId = '';
    }
  }

  async function handleReload() {
    if (reloading) return;
    reloading = true;
    try {
      await ReloadExtensions();
      await loadInstalled();
      try {
        updates = await fetchUpdates();
      } catch { /* registry unreachable — keep current list */ }
      toast.success('Extensions reloaded.');
    } catch (e: any) {
      console.error('Reload failed:', e);
      toast.error('Failed to reload extensions: ' + e);
    } finally {
      reloading = false;
    }
  }

  async function loadInstalled() {
    try {
      const [exts, sets] = await Promise.all([GetExtensions(), GetSettings()]);
      installedExtensions = exts || [];
      isDevMode = sets.developerMode;
    } catch (e) {
      console.error(e);
    }
  }

  async function loadGallery() {
    galleryLoading = true;
    galleryError = '';
    try {
      const cacheBuster = new Date().getTime();
      const res = await fetch(`${GALLERY_INDEX_URL}?t=${cacheBuster}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      galleryExtensions = await res.json();
    } catch (e) {
      console.error('Failed to load gallery:', e);
      galleryError = 'Could not reach the Extension Gallery. Make sure you are connected to the internet.';
    } finally {
      galleryLoading = false;
    }
  }

  async function handleLocalInstall() {
    if (isInstalling) return;
    isInstalling = true;
    try {
      const installed = await SelectAndInstallExtension();
      if (installed) {
        await loadInstalled();
        activeTab = 'installed';
      }
    } catch (e: any) {
      console.error('Installation failed:', e);
      toast.error('Installation failed: ' + e);
    } finally {
      isInstalling = false;
    }
  }

  let installingId = '';
  async function handleRemoteInstall(url: string, extId: string) {
    if (isInstalling) return;
    isInstalling = true;
    installingId = extId;
    try {
      const installed = await DownloadAndInstallExtension(url);
      if (installed) {
        await loadInstalled();
        toast.success('Extension installed successfully!');
      }
    } catch (e: any) {
      console.error('Remote installation failed:', e);
      toast.error('Failed to install extension: ' + e);
    } finally {
      isInstalling = false;
      installingId = '';
    }
  }

  async function handleUninstall(ext: any) {
    pendingUninstall = ext;
    confirmDialog.open(
      `Uninstall ${ext.name}?`,
      `Are you sure you want to uninstall "${ext.name}"? Its extension files and sidebar pages will be removed.`,
      true
    );
  }

  async function handleUninstallConfirm(event: CustomEvent<boolean>) {
    const ext = pendingUninstall;
    pendingUninstall = null;
    if (!event.detail || !ext) return;
    try {
      await UninstallExtension(ext.id);
      await loadInstalled();
      toast.success('Extension uninstalled.');
    } catch (e: any) {
      toast.error('Failed to uninstall extension: ' + e);
    }
  }

  onMount(async () => {
    await loadInstalled();
    checkForUpdates();
    try { launcherVersion = await GetLauncherVersion(); } catch { /* fallback: dev */ }

    const unsubStart = EventsOn('extension:reload:start', () => {
      reloading = true;
    });
    const unsubComplete = EventsOn('extension:reload:complete', async () => {
      reloading = false;
      await loadInstalled();
      try {
        updates = await fetchUpdates();
      } catch { /* ignore */ }
    });

    return () => {
      if (unsubStart) unsubStart();
      if (unsubComplete) unsubComplete();
    };
  });

  function setTab(tab: string) {
    activeTab = tab;
    if (tab === 'gallery') {
      loadGallery();
    }
  }

  function trustBadge(trust: string | undefined): { cls: string; label: string } {
    switch (trust) {
      case 'official':  return { cls: 'badge-official',   label: 'Official'   };
      case 'verified':  return { cls: 'badge-verified',   label: 'Verified'   };
      case 'community': return { cls: 'badge-community',  label: 'Community'  };
      default:          return { cls: 'badge-local',      label: 'Local'      };
    }
  }

  function extGradient(name: string): string {
    const g = [
      'linear-gradient(135deg, #3b82f6, #1d4ed8)',
      'linear-gradient(135deg, #8b5cf6, #6d28d9)',
      'linear-gradient(135deg, #06b6d4, #0284c7)',
      'linear-gradient(135deg, #10b981, #047857)',
      'linear-gradient(135deg, #f59e0b, #b45309)',
      'linear-gradient(135deg, #ec4899, #be185d)',
    ];
    let h = 0;
    for (let i = 0; i < name.length; i++) h = name.charCodeAt(i) + ((h << 5) - h);
    return g[Math.abs(h) % g.length];
  }

  function statusColor(status: string): string {
    if (!status) return 'rgba(255,255,255,0.2)';
    const s = status.toLowerCase();
    if (s === 'running') return '#22c55e';
    if (s === 'error')   return '#ef4444';
    return 'rgba(255,255,255,0.2)';
  }
</script>

<div class="page page-enter">
  <header class="page-header">
    <h1>Extensions</h1>
    <div class="header-actions">
      <div class="tabs">
        <button class="tab-btn {activeTab === 'installed' ? 'active' : ''}" on:click={() => setTab('installed')}>Installed</button>
        <button class="tab-btn {activeTab === 'gallery' ? 'active' : ''}" on:click={() => setTab('gallery')}>Gallery</button>
      </div>
      {#if reloading}
        <div class="reload-indicator">
          <span class="spinner"></span>
          <span>Reloading extensions...</span>
        </div>
      {/if}
      <button class="btn btn-secondary" on:click={handleReload} disabled={reloading || installedExtensions.length === 0}>
        {reloading ? 'Reloading...' : 'Reload'}
      </button>
      <button class="btn btn-secondary" on:click={checkForUpdates} disabled={checkingUpdates || reloading || installedExtensions.length === 0}>
        {checkingUpdates ? 'Checking...' : 'Check for Updates'}
      </button>
      <button class="btn btn-secondary" on:click={handleLocalInstall} disabled={isInstalling || reloading}>
        {isInstalling ? 'Installing...' : 'Install from .aex'}
      </button>
    </div>
  </header>

  {#if activeTab === 'installed'}
    {#if installedExtensions.length === 0}
      <EmptyState
        icon="puzzle"
        title="No extensions installed"
        description="Install a .aex extension or visit the Gallery to add new capabilities to Aether."
        actionLabel="Browse Gallery"
        on:action={() => setTab('gallery')}
      />
    {:else}
      <div class="grid">
        {#each installedExtensions as ext}
          {@const badge = trustBadge(ext.trust)}
          {@const grad  = extGradient(ext.name)}
          {@const dot   = statusColor(ext.status)}

          <div class="card ext-card">
            <div class="card-accent" style="background: {grad};"></div>
            <div class="card-body">
              <div class="ext-header">
                <div class="ext-icon" style={!ext.iconUrl ? `background: ${grad};` : ''}>
                  {#if ext.iconUrl}
                    <img src={ext.iconUrl} alt={ext.name} class="ext-icon-img" />
                  {:else}
                    <span class="ext-icon-letter">{ext.name.charAt(0).toUpperCase()}</span>
                  {/if}
                </div>
                <div class="ext-info">
                  <div class="ext-title-row">
                    <h3 class="ext-title">{ext.name} <span class="ext-version">v{ext.version}</span></h3>
                    <div class="ext-badges">
                      {#if ext.reloading}
                        <span class="badge badge-reloading" title="Reloading...">Reloading</span>
                      {/if}
                      {#if updateFor(ext)}
                        <span class="badge badge-update" title="Update available">Update</span>
                      {/if}
                      <span class="badge {badge.cls}">{badge.label}</span>
                    </div>
                  </div>
                  <div class="ext-meta">
                    <span class="ext-author">by {ext.author}</span>
                    {#if isDevMode}
                      <span class="dev-id" title="Extension ID">id: {ext.id}</span>
                    {/if}
                  </div>
                </div>
              </div>

              <p class="ext-desc">{ext.description}</p>
              
                <div class="ext-footer">
                  <div class="ext-status-wrap">
                    <div class="ext-status-dot" style="background: {dot}; box-shadow: 0 0 6px {dot};"></div>
                    <span class="ext-status-text">{ext.status || 'Active'}</span>
                    {#if isDevMode && ext.status === 'Running'}
                      <div class="dev-stats">
                        <span title="Memory Usage">{ext.memory || '0MB'}</span>
                        <span class="dot-separator">•</span>
                        <span title="CPU Usage">{ext.cpu || '0%'} cpu</span>
                      </div>
                    {/if}
                  </div>
                  <div class="ext-footer-actions">
                    {#if updateFor(ext)}
                      <button class="btn btn-primary" on:click={() => handleUpdate(ext)} disabled={!!updatingId || reloading || ext.reloading}>
                        {updatingId === ext.id ? 'Updating...' : ext.reloading ? 'Reloading...' : `Update to v${updateFor(ext).newVersion}`}
                      </button>
                    {/if}
                    <button class="btn btn-secondary" on:click={() => handleUninstall(ext)} disabled={reloading}>Uninstall</button>
                  </div>
                </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}

  {#if activeTab === 'gallery'}
    <input class='gallery-search' type='search' placeholder='Search extensions...' aria-label='Search extensions' bind:value={gallerySearch} />
    {#if galleryLoading}
      <div class="loading-state">
        <div class="spinner"></div>
        <p>Loading Extension Gallery...</p>
      </div>
    {:else if galleryError}
      <EmptyState
        icon="wifi-off"
        title="Gallery Unavailable"
        description={galleryError}
        actionLabel="Try Again"
        on:action={loadGallery}
      />
    {:else if filteredGalleryExtensions.length === 0}
      <EmptyState
        icon="search"
        title="No Extensions Found"
        description="The gallery is currently empty."
      />
    {:else}
      <div class="grid">
        {#each filteredGalleryExtensions as ext}
          {@const grad = extGradient(ext.name)}
          
          <div class="card ext-card">
            <div class="card-accent" style="background: {grad};"></div>
            <div class="card-body">
              <div class="ext-header">
                <div class="ext-icon" style={!ext.iconUrl ? `background: ${grad};` : ''}>
                  {#if ext.iconUrl}
                    <img src={ext.iconUrl} alt={ext.name} class="ext-icon-img" />
                  {:else}
                    <span class="ext-icon-letter">{ext.name.charAt(0).toUpperCase()}</span>
                  {/if}
                </div>
                <div class="ext-info">
                  <div class="ext-title-row">
                    <h3 class="ext-title">{ext.name} <span class="ext-version">v{ext.version}</span></h3>
                    <span class="badge {trustBadge(ext.trust).cls}">{trustBadge(ext.trust).label}</span>
                  </div>
                  <div class="ext-meta">
                    <span class="ext-author">by {ext.author}</span>
                  </div>
                </div>
              </div>

              <p class="ext-desc">{ext.description}</p>
              
              <div class="ext-footer">
                {#if requiresNewerLauncher(ext)}
                  <div class="compat-warning">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
                    Requires Aether {requiresNewerLauncher(ext)}+
                  </div>
                  <button class="btn btn-secondary" disabled title="Update Aether to install this extension">Incompatible</button>
                {:else if installedFor(ext)}
                  {#if hasUpdate(ext)}
                    <button class="btn btn-primary" on:click={() => handleRemoteInstall(ext.url, ext.id)} disabled={isInstalling}>
                      {installingId === ext.id ? "Updating..." : "Update to v" + ext.version}
                    </button>
                  {:else}
                    <button class="btn btn-secondary" disabled>Installed</button>
                  {/if}
                {:else}
                  <button class="btn btn-primary" on:click={() => handleRemoteInstall(ext.url, ext.id)} disabled={isInstalling}>
                    {installingId === ext.id ? 'Installing...' : 'Install'}
                  </button>
                {/if}
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<ConfirmDialog bind:this={confirmDialog} on:confirm={handleUninstallConfirm} />

<style>
  .page {
    padding: var(--spacing-xl);
    height: 100%;
    box-sizing: border-box;
    overflow-y: auto;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--spacing-xl);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
    gap: var(--spacing-lg);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .tabs {
    display: flex;
    background: rgba(255,255,255,0.05);
    padding: 4px;
    border-radius: 8px;
  }
  
  .tab-btn {
    background: transparent;
    border: none;
    color: rgba(255,255,255,0.5);
    padding: 6px 14px;
    border-radius: 6px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .tab-btn:hover {
    color: rgba(255,255,255,0.8);
  }

  .tab-btn.active {
    background: rgba(255,255,255,0.1);
    color: white;
    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  }

  .gallery-search {
    display: block;
    width: min(360px, 100%);
    box-sizing: border-box;
    margin: 0 0 var(--spacing-lg) auto;
    padding: 10px 14px;
    border: 1px solid rgba(255,255,255,0.1);
    border-radius: 8px;
    outline: none;
    background: rgba(255,255,255,0.05);
    color: rgba(255,255,255,0.9);
    font: inherit;
  }

  .gallery-search:focus {
    border-color: rgba(59,130,246,0.7);
    box-shadow: 0 0 0 3px rgba(59,130,246,0.15);
  }

  .gallery-search::placeholder {
    color: rgba(255,255,255,0.4);
  }
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 0;
    color: rgba(255,255,255,0.5);
    gap: 16px;
  }

  .spinner {
    width: 24px;
    height: 24px;
    border: 2px solid rgba(255,255,255,0.1);
    border-top-color: #3b82f6;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .reload-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    background: rgba(59, 130, 246, 0.15);
    border: 1px solid rgba(59, 130, 246, 0.35);
    border-radius: 8px;
    color: #60a5fa;
    font-size: 12px;
    font-weight: 500;
  }

  .ext-card {
    position: relative;
    padding: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .card-accent {
    height: 4px;
    width: 100%;
  }

  .card-body {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    flex: 1;
  }

  .ext-header {
    display: flex;
    gap: 16px;
    align-items: center;
  }

  .ext-icon {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    overflow: hidden;
    flex-shrink: 0;
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
  }

  .ext-icon-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .ext-icon-letter {
    font-size: 24px;
    font-weight: 700;
    color: rgba(255,255,255,0.9);
  }

  .ext-info {
    flex: 1;
    min-width: 0;
  }

  .ext-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .ext-title {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .ext-version {
    font-size: 12px;
    font-weight: 500;
    color: rgba(255,255,255,0.3);
    margin-left: 4px;
  }

  .ext-author {
    font-size: 13px;
    color: rgba(255,255,255,0.4);
  }

  .ext-desc {
    margin: 0;
    font-size: 13px;
    color: rgba(255,255,255,0.6);
    line-height: 1.5;
    flex: 1;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .ext-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 16px;
    border-top: 1px solid rgba(255,255,255,0.05);
    margin-top: auto;
  }

  .compat-warning {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: #f59e0b;
    background: rgba(245, 158, 11, 0.1);
    border: 1px solid rgba(245, 158, 11, 0.25);
    border-radius: 6px;
    padding: 4px 8px;
    max-width: 180px;
  }

  .ext-status-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .ext-status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .ext-status-text {
    font-size: 12px;
    font-weight: 500;
    color: rgba(255,255,255,0.5);
  }

  .badge {
    padding: 2px 8px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .badge-official  { background: rgba(59, 130, 246, 0.15);  color: #60a5fa;  border: 1px solid rgba(59, 130, 246, 0.35);  }
  .badge-verified  { background: rgba(16, 185, 129, 0.15);  color: #34d399;  border: 1px solid rgba(16, 185, 129, 0.35);  }
  .badge-community { background: rgba(168, 85, 247, 0.12);  color: #c084fc;  border: 1px solid rgba(168, 85, 247, 0.3);   }
  .badge-local     { background: rgba(245, 158, 11, 0.12);  color: #fbbf24;  border: 1px solid rgba(245, 158, 11, 0.3);   }
  .badge-update    { background: rgba(34, 197, 94, 0.15);   color: #4ade80;  border: 1px solid rgba(34, 197, 94, 0.35);   }
  .badge-reloading { background: rgba(59, 130, 246, 0.15);  color: #60a5fa;  border: 1px solid rgba(59, 130, 246, 0.35);  }

  .ext-badges {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .ext-footer-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  /* Dev Mode Styles */
  .dev-id {
    font-family: monospace;
    font-size: 11px;
    background: rgba(0,0,0,0.3);
    padding: 2px 6px;
    border-radius: 4px;
    color: #a78bfa;
    margin-left: 8px;
  }

  .dev-stats {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-left: auto;
    font-size: 11px;
    font-family: monospace;
    color: #94a3b8;
    background: rgba(0,0,0,0.2);
    padding: 2px 8px;
    border-radius: 4px;
    border: 1px solid rgba(255,255,255,0.05);
  }

  .dot-separator {
    color: rgba(255,255,255,0.2);
  }
</style>
