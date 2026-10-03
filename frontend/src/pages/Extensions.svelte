<script lang="ts">
  import { onMount } from 'svelte';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';
  import { GetExtensions, SelectAndInstallExtension, DownloadAndInstallExtension, GetSettings, UninstallExtension, SetExtensionEnabled, GetExtensionUpdates, UpdateExtension, ReloadExtensions, GetLauncherVersion } from '../../wailsjs/go/main/App.js';
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
  const GALLERY_INDEX_URL = 'https://raw.githubusercontent.com/Aether-Launcher/Aether-Extensions/main/index.json';

  let isDevMode = false;
  let confirmDialog: any;
  let pendingUninstall: any = null;
  let openActionMenuId = '';
  let detailsExtension: any = null;
  let togglingExtensionId = '';

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
      const match = String(version || '').trim().replace(/^v/i, '').match(/^(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([a-zA-Z0-9.]+))?/);
      if (!match) return null;
      return {
        major: Number(match[1]),
        minor: Number(match[2] || 0),
        patch: Number(match[3] || 0),
        prerelease: match[4] || null,
      };
    };
    const a = parse(left), b = parse(right);
    if (!a || !b) return 0;
    if (a.major !== b.major) return a.major > b.major ? 1 : -1;
    if (a.minor !== b.minor) return a.minor > b.minor ? 1 : -1;
    if (a.patch !== b.patch) return a.patch > b.patch ? 1 : -1;
    if (!a.prerelease && b.prerelease) return 1;
    if (a.prerelease && !b.prerelease) return -1;
    if (a.prerelease && b.prerelease) {
      return a.prerelease.localeCompare(b.prerelease, undefined, { numeric: true, sensitivity: 'base' });
    }
    return 0;
  }

  $: installedMap = (() => {
    const map = new Map<string, any>();
    for (const ext of installedExtensions || []) {
      if (ext && ext.id) {
        map.set(ext.id, ext);
        map.set(ext.id.toLowerCase(), ext);
      }
    }
    return map;
  })();

  function installedFor(ext: any): any {
    if (!ext || !ext.id) return null;
    return installedMap.get(ext.id) || installedMap.get(ext.id.toLowerCase()) || null;
  }
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
        galleryExtensions = [...galleryExtensions];
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
    openActionMenuId = '';
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
      galleryExtensions = [...galleryExtensions];
      toast.success('Extension uninstalled.');
    } catch (e: any) {
      toast.error('Failed to uninstall extension: ' + e);
    }
  }

  async function handleToggleExtension(ext: any) {
    if (togglingExtensionId) return;
    togglingExtensionId = ext.id;
    openActionMenuId = '';
    try {
      await SetExtensionEnabled(ext.id, !!ext.disabled);
      await loadInstalled();
      toast.success(`${ext.disabled ? 'Enabled' : 'Disabled'} ${ext.name}.`);
    } catch (e: any) {
      toast.error(`Could not ${ext.disabled ? 'enable' : 'disable'} ${ext.name}: ${e}`);
    } finally {
      togglingExtensionId = '';
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
      galleryExtensions = [...galleryExtensions];
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
      'linear-gradient(135deg, #5268e0, #293da9)',
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
    <div class="page-heading">
      <h1>Extensions</h1>
      <div class="tabs" role="tablist" aria-label="Extension views">
        <button class="tab-btn {activeTab === 'installed' ? 'active' : ''}" role="tab" aria-selected={activeTab === 'installed'} on:click={() => setTab('installed')}>Installed</button>
        <button class="tab-btn {activeTab === 'gallery' ? 'active' : ''}" role="tab" aria-selected={activeTab === 'gallery'} on:click={() => setTab('gallery')}>Gallery</button>
      </div>
    </div>
    <div class="header-actions">
      {#if reloading}
        <div class="reload-indicator">
          <span class="spinner"></span>
          <span>Reloading</span>
        </div>
      {/if}
      <button class="icon-action" on:click={handleReload} disabled={reloading || installedExtensions.length === 0} title="Reload extensions" aria-label="Reload extensions">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 7v5h-5M4 17v-5h5"/><path d="M5.6 9a7 7 0 0 1 11.7-2L20 12M4 12l2.7 5a7 7 0 0 0 11.7-2"/></svg>
      </button>
      <button class="icon-action" on:click={checkForUpdates} disabled={checkingUpdates || reloading || installedExtensions.length === 0} title="Check for updates" aria-label="Check for updates">
        <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 11a8 8 0 0 0-14.9-3M4 4v4h4M4 13a8 8 0 0 0 14.9 3M20 20v-4h-4"/></svg>
      </button>
      <button class="btn btn-primary install-button" on:click={handleLocalInstall} disabled={isInstalling || reloading}>
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
      <div class="grid installed-grid">
        {#each installedExtensions as ext}
          {@const badge = trustBadge(ext.trust)}
          {@const grad  = extGradient(ext.name)}
          {@const dot   = statusColor(ext.status)}

          <div class="card ext-card installed-card">
            <div class="ext-card-menu-wrap">
              <button class="ext-card-menu-trigger" aria-label={`Actions for ${ext.name}`} title="Extension actions" aria-haspopup="menu" aria-expanded={openActionMenuId === ext.id} on:click={() => (openActionMenuId = openActionMenuId === ext.id ? '' : ext.id)}>
                <svg viewBox="0 0 4 16" aria-hidden="true"><circle cx="2" cy="2" r="1.5"/><circle cx="2" cy="8" r="1.5"/><circle cx="2" cy="14" r="1.5"/></svg>
              </button>
              {#if openActionMenuId === ext.id}
                <div class="ext-card-menu" role="menu" aria-label={`${ext.name} actions`}>
                  <button role="menuitem" on:click={() => { detailsExtension = ext; openActionMenuId = ''; }}>View details</button>
                  <button role="menuitem" on:click={() => handleToggleExtension(ext)} disabled={togglingExtensionId === ext.id || reloading}>
                    {togglingExtensionId === ext.id ? 'Please wait...' : ext.disabled ? 'Enable' : 'Disable'}
                  </button>
                  <button class="danger-action" role="menuitem" on:click={() => handleUninstall(ext)} disabled={reloading}>Uninstall</button>
                </div>
              {/if}
            </div>
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
                  </div>
                </div>
              </div>

              <p class="ext-desc">{ext.description}</p>
              
                <div class="ext-footer">
                  <div class="ext-status-wrap">
                    <div class="ext-status-dot" style="background: {dot};"></div>
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
                {:else if installedMap.has(ext.id) || installedMap.has(ext.id.toLowerCase())}
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

{#if detailsExtension}
  <div class="details-backdrop">
    <section class="extension-details" role="dialog" aria-modal="true" aria-labelledby="extension-details-title">
      <button class="details-close" aria-label="Close details" on:click={() => (detailsExtension = null)}>×</button>
      <div class="details-heading">
        <div class="ext-icon" style={!detailsExtension.iconUrl ? `background: ${extGradient(detailsExtension.name)};` : ''}>
          {#if detailsExtension.iconUrl}
            <img src={detailsExtension.iconUrl} alt="" class="ext-icon-img" />
          {:else}
            <span class="ext-icon-letter">{detailsExtension.name.charAt(0).toUpperCase()}</span>
          {/if}
        </div>
        <div>
          <h2 id="extension-details-title">{detailsExtension.name}</h2>
          <p>{detailsExtension.description}</p>
        </div>
      </div>
      <dl class="details-list">
        <div><dt>Version</dt><dd>{detailsExtension.version}</dd></div>
        <div><dt>Author</dt><dd>{detailsExtension.author || 'Not provided'}</dd></div>
        <div><dt>Status</dt><dd>{detailsExtension.status || 'Installed'}</dd></div>
        <div><dt>Extension ID</dt><dd class="details-id">{detailsExtension.id}</dd></div>
      </dl>
    </section>
  </div>
{/if}

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
    gap: 18px;
    margin-bottom: var(--spacing-xl);
  }

  .page-heading { display: flex; align-items: center; gap: 22px; min-width: 0; }
  .page-heading h1 { margin: 0; white-space: nowrap; }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--spacing-lg);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .icon-action {
    width: 36px;
    height: 36px;
    display: grid;
    place-items: center;
    padding: 8px;
    color: var(--text-secondary);
    background: rgba(255,255,255,0.035);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 8px;
    cursor: pointer;
    transition: color var(--transition-fast), background var(--transition-fast), border-color var(--transition-fast);
  }

  .icon-action:hover:not(:disabled) { color: var(--text-primary); background: rgba(255,255,255,0.08); border-color: rgba(255,255,255,0.13); }
  .icon-action:disabled { opacity: 0.4; cursor: not-allowed; }
  .icon-action svg { width: 17px; height: 17px; fill: none; stroke: currentColor; stroke-width: 1.7; stroke-linecap: round; stroke-linejoin: round; }
  .install-button { white-space: nowrap; }

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
    border: 1px solid rgba(255,255,255,0.06);
  }

  .tab-btn:focus-visible, .icon-action:focus-visible, .ext-card-menu-trigger:focus-visible {
    outline: 2px solid var(--accent-color);
    outline-offset: 2px;
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
    border-color: rgba(59,82,212,0.65);
    box-shadow: 0 0 0 2px rgba(59,82,212,0.14);
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
    border-top-color: #3b52d4;
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
    background: rgba(59, 82, 212, 0.12);
    border: 1px solid rgba(112, 130, 235, 0.2);
    border-radius: 8px;
    color: #a0adff;
    font-size: 12px;
    font-weight: 500;
  }

  .ext-card {
    position: relative;
    padding: 0;
    overflow: hidden;
    border-radius: var(--card-radius, 7px);
    display: flex;
    flex-direction: column;
    transition: border-color 0.15s ease, transform 0.15s ease;
  }

  .ext-card-menu-wrap { position: absolute; top: 8px; right: 8px; z-index: 2; }

  .ext-card-menu-trigger {
    width: 30px;
    height: 30px;
    display: grid;
    place-items: center;
    padding: 0 0 3px;
    color: var(--text-secondary);
    background: rgba(24,24,28,0.96);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: 7px;
    font-size: 20px;
    line-height: 1;
    cursor: pointer;
  }

  .ext-card-menu-trigger:hover { color: var(--text-primary); background: rgba(255,255,255,0.09); }

  .ext-card-menu-trigger svg {
    display: block;
    width: 8px;
    height: 16px;
    fill: currentColor;
  }

  .ext-card-menu {
    position: absolute;
    top: 34px;
    right: 0;
    min-width: 148px;
    padding: 5px;
    background: #202025;
    border: 1px solid rgba(255,255,255,0.1);
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0,0,0,0.38);
  }

  .ext-card-menu button {
    width: 100%;
    padding: 8px 9px;
    color: var(--text-primary);
    text-align: left;
    background: transparent;
    border: 0;
    border-radius: 5px;
    font: inherit;
    font-size: 12px;
    cursor: pointer;
  }

  .ext-card-menu button:hover:not(:disabled) { background: rgba(255,255,255,0.08); }
  .ext-card-menu button:disabled { opacity: 0.45; cursor: not-allowed; }
  .ext-card-menu .danger-action { color: #f28b88; }

  .installed-grid {
    gap: 12px;
  }

  @media (min-width: 1280px) {
    .grid { grid-template-columns: repeat(3, minmax(280px, 1fr)); }
  }

  .installed-card .card-body {
    padding: 12px 14px;
    gap: 10px;
  }

  .installed-card .ext-header {
    gap: 12px;
  }

  .installed-card .ext-icon {
    width: 36px;
    height: 36px;
    border-radius: 8px;
  }

  .installed-card .ext-icon-letter {
    font-size: 16px;
  }

  .installed-card .ext-title {
    font-size: 14px;
  }

  .installed-card .ext-desc {
    font-size: 12px;
    -webkit-line-clamp: 2;
  }

  .installed-card .ext-title-row { padding-right: 34px; }

  .installed-card .ext-footer {
    padding-top: 8px;
  }

  .installed-card .btn {
    padding: 5px 12px;
    font-size: 12px;
  }

  .card-body {
    padding: 18px;
    display: flex;
    flex-direction: column;
    gap: 14px;
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
    border: 1px solid rgba(255,255,255,0.08);
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
    color: rgba(255,255,255,0.65);
    margin-left: 4px;
  }

  .ext-author {
    font-size: 13px;
    color: rgba(255,255,255,0.65);
  }

  .ext-desc {
    margin: 0;
    width: 100%;
    box-sizing: border-box;
    font-size: 13px;
    color: rgba(255,255,255,0.72);
    line-height: 1.5;
    flex: 1;
    display: -webkit-box;
    -webkit-line-clamp: 2;
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
    color: rgba(255,255,255,0.65);
  }

  .badge {
    padding: 2px 8px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .badge-official  { background: rgba(59, 82, 212, 0.12); color: #9aaaff; border: 1px solid rgba(112, 130, 235, 0.2); }
  .badge-verified  { background: rgba(16, 185, 129, 0.10); color: #77d9b0; border: 1px solid rgba(16, 185, 129, 0.18); }
  .badge-community { background: rgba(168, 85, 247, 0.09); color: #d0a4f1; border: 1px solid rgba(168, 85, 247, 0.16); }
  .badge-local     { background: rgba(245, 158, 11, 0.09); color: #e7c27a; border: 1px solid rgba(245, 158, 11, 0.17); }
  .badge-update    { background: rgba(34, 197, 94, 0.10); color: #82d99c; border: 1px solid rgba(34, 197, 94, 0.18); }
  .badge-reloading { background: rgba(59, 82, 212, 0.12); color: #9aaaff; border: 1px solid rgba(112, 130, 235, 0.2); }

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

  .details-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: grid;
    place-items: center;
    padding: 24px;
    background: rgba(0,0,0,0.66);
    backdrop-filter: blur(6px);
  }

  .extension-details {
    position: relative;
    width: min(520px, 100%);
    max-height: min(620px, 90vh);
    overflow: auto;
    padding: 26px;
    color: var(--text-primary);
    background: var(--panel-bg, #18181c);
    border: 1px solid rgba(255,255,255,0.09);
    border-radius: 10px;
    box-shadow: 0 18px 48px rgba(0,0,0,0.48);
  }

  .details-close {
    position: absolute;
    top: 12px;
    right: 12px;
    width: 30px;
    height: 30px;
    color: var(--text-secondary);
    background: transparent;
    border: 0;
    border-radius: 6px;
    font-size: 22px;
    cursor: pointer;
  }

  .details-heading { display: flex; align-items: flex-start; gap: 16px; padding-right: 28px; }
  .details-heading h2 { margin: 2px 0 6px; font-size: 20px; }
  .details-heading p { color: var(--text-secondary); line-height: 1.5; }
  .details-list { margin: 24px 0 0; border-top: 1px solid rgba(255,255,255,0.07); }
  .details-list > div { display: grid; grid-template-columns: 120px minmax(0, 1fr); gap: 14px; padding: 11px 0; border-bottom: 1px solid rgba(255,255,255,0.06); }
  .details-list dt { color: var(--text-secondary); }
  .details-list dd { margin: 0; overflow-wrap: anywhere; }
  .details-id { font-family: ui-monospace, SFMono-Regular, Consolas, monospace; font-size: 12px; }

  @media (max-width: 900px) {
    .page-header { align-items: flex-start; flex-direction: column; }
    .header-actions { width: 100%; justify-content: flex-end; }
    .page-heading { width: 100%; justify-content: space-between; }
  }
</style>
