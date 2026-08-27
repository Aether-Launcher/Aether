<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
  import { GetExtensionSidebarPages, GetConnectivityStatus } from '../../wailsjs/go/main/App';
  import { themeAssets } from '../stores/theme';
  import AccountManager from './AccountManager.svelte';
  import Icon from './Icon.svelte';
  import UpdateBanner from './UpdateBanner.svelte';

  export let activePage: string = 'home';

  const dispatch = createEventDispatcher();

  const topNav = [
    { id: 'home',       label: 'Home',       icon: 'home'       },
    { id: 'instances',  label: 'Instances',  icon: 'instances'  },
    { id: 'extensions', label: 'Extensions', icon: 'extensions' },
  ];

  const bottomNav = [
    { id: 'settings', label: 'Settings', icon: 'settings' },
  ];

  type ExtensionTab = { id: string; label: string; url: string; icon?: string; extensionId: string };
  let extensionTabs: ExtensionTab[] = [];

  let connectivity: any = null;
  let checkingConnectivity = false;
  let connTimer: any = null;

  async function refreshConnectivity() {
    checkingConnectivity = true;
    try {
      connectivity = await GetConnectivityStatus();
    } catch (e) {
      connectivity = { overall: 'unknown' };
    } finally {
      checkingConnectivity = false;
    }
  }

  onMount(async () => {
    refreshConnectivity();
    connTimer = setInterval(refreshConnectivity, 60000);

    // Fetch extension UI tabs registered during backend startup
    try {
      const cachedTabs = await GetExtensionSidebarPages();
      if (cachedTabs) {
        for (const tab of cachedTabs) {
          const t = tab as ExtensionTab;
          if (!extensionTabs.find((e) => e.id === t.id)) {
            extensionTabs = [...extensionTabs, t];
            dispatch('registerExtensionRoute', t);
          }
        }
      }
    } catch (e) {
      console.error('Failed to load cached extension tabs', e);
    }

    // Listen for extension UI tabs registered dynamically at runtime
    EventsOn('extension:sidebar:add', (payload: unknown) => {
      const tab = payload as ExtensionTab;
      if (!extensionTabs.find((t) => t.id === tab.id)) {
        extensionTabs = [...extensionTabs, tab];
        dispatch('registerExtensionRoute', tab);
      }
    });

    // Extensions were reloaded — clear tabs and refetch so removed/updated
    // extensions don't leave stale sidebar entries.
    EventsOn('extension:sidebar:reset', async () => {
      extensionTabs = [];
      try {
        const tabs = await GetExtensionSidebarPages();
        if (tabs) {
          for (const tab of tabs) {
            const t = tab as ExtensionTab;
            if (!extensionTabs.find((e) => e.id === t.id)) {
              extensionTabs = [...extensionTabs, t];
              dispatch('registerExtensionRoute', t);
            }
          }
        }
      } catch (e) {
        console.error('Failed to reload extension tabs', e);
      }
    });
  });

  onDestroy(() => {
    EventsOff('extension:sidebar:add');
    EventsOff('extension:sidebar:reset');
    if (connTimer) clearInterval(connTimer);
  });

  function navigate(pageId: string) {
    dispatch('navigate', pageId);
  }

  /** Returns the first letter of a label, uppercased — used as icon fallback */
  function monogram(label: string): string {
    return label.charAt(0).toUpperCase();
  }
</script>

<aside class="sidebar">
  <div class="logo">
    <img src={$themeAssets['sidebar-logo'] || '/logo.png'} alt="Logo" class="sidebar-logo" />
    Aether
  </div>

  <nav class="top-nav">
    {#each topNav as item}
      <button
        class="nav-item {activePage === item.id ? 'active' : ''}"
        on:click={() => navigate(item.id)}
        title={item.label}
      >
        <span class="nav-icon">
          <Icon name={item.icon} size={16} />
        </span>
        <span class="nav-label">{item.label}</span>
      </button>
    {/each}

    {#if extensionTabs.length > 0}
      <div class="nav-divider"></div>
      <div class="nav-section-title">Extensions</div>
      {#each extensionTabs as tab}
        <button
          class="nav-item extension {activePage === tab.id ? 'active' : ''}"
          on:click={() => navigate(tab.id)}
          title={tab.label}
        >
          <span class="nav-icon ext-icon-wrap">
            {#if tab.icon}
              <!-- Extension-supplied icon name -->
              <Icon name={tab.icon} size={14} />
            {:else}
              <!-- Monogram fallback -->
              <span class="monogram">{monogram(tab.label)}</span>
            {/if}
          </span>
          <span class="nav-label">{tab.label}</span>
        </button>
      {/each}
    {/if}
  </nav>

  <nav class="bottom-nav">
    {#each bottomNav as item}
      <button
        class="nav-item {activePage === item.id ? 'active' : ''}"
        on:click={() => navigate(item.id)}
        title={item.label}
      >
        <span class="nav-icon">
          <Icon name={item.icon} size={16} />
        </span>
        <span class="nav-label">{item.label}</span>
      </button>
    {/each}
  </nav>

  <button class="conn-indicator" on:click={refreshConnectivity} title="Connection status — click to refresh">
    <span class="conn-dot" class:online={connectivity?.overall === 'online'} class:degraded={connectivity?.overall === 'degraded'} class:offline={connectivity?.overall === 'offline'}></span>
    <span class="conn-label">
      {#if checkingConnectivity || !connectivity}
        Checking…
      {:else if connectivity.overall === 'online'}
        Online
      {:else if connectivity.overall === 'degraded'}
        Degraded
      {:else if connectivity.overall === 'offline'}
        Offline
      {:else}
        Unknown
      {/if}
    </span>
  </button>

  <UpdateBanner />

  <AccountManager />
</aside>

<style>
  .sidebar {
    width: 220px;
    min-width: 220px;
    height: 100%;
    background-color: var(--sidebar-bg);
    display: flex;
    flex-direction: column;
    padding: 24px 12px;
    box-sizing: border-box;
    border-right: 1px solid rgba(255, 255, 255, 0.05);
    /* Fix #11: sidebar must be independently contained so Settings/account
       card never gets clipped when the main panel grows taller (macOS) */
    overflow: hidden;
    flex-shrink: 0;
  }

  .logo {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 22px;
    font-weight: 700;
    margin-bottom: 36px;
    padding: 0 12px;
    letter-spacing: -0.5px;
  }

  .sidebar-logo {
    width: 24px;
    height: 24px;
    object-fit: contain;
  }

  .top-nav {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .bottom-nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-top: auto;
  }

  .conn-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 12px;
    margin: 8px 0;
    background: none;
    border: none;
    border-radius: var(--border-radius);
    color: var(--text-secondary);
    font-size: 12px;
    font-family: inherit;
    cursor: pointer;
    text-align: left;
    transition: background-color var(--transition-fast);
  }

  .conn-indicator:hover {
    background-color: rgba(255, 255, 255, 0.05);
  }

  .conn-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #6b7280;
    flex-shrink: 0;
  }

  .conn-dot.online { background: #22c55e; }
  .conn-dot.degraded { background: #f59e0b; }
  .conn-dot.offline { background: #ef4444; }

  .conn-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Nav item — icon + label layout */
  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    background: transparent;
    border: none;
    text-align: left;
    padding: 9px 12px;
    border-radius: var(--border-radius);
    color: var(--text-secondary);
    font-family: inherit;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: background-color var(--transition-fast), color var(--transition-fast);
    width: 100%;
  }

  .nav-item:hover {
    background-color: rgba(255, 255, 255, 0.05);
    color: var(--text-primary);
  }

  .nav-item.active {
    background-color: rgba(255, 255, 255, 0.1);
    color: var(--text-primary);
    font-weight: 600;
  }

  /* Icon cell */
  .nav-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    flex-shrink: 0;
  }

  /* Monogram fallback for extension tabs */
  .ext-icon-wrap {
    border-radius: 5px;
    background: rgba(255, 255, 255, 0.08);
  }

  .monogram {
    font-size: 11px;
    font-weight: 700;
    line-height: 1;
  }

  .nav-label {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .nav-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.05);
    margin: 8px 12px;
  }

  .nav-section-title {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
    padding: 4px 12px;
    font-weight: 600;
    margin-top: 2px;
  }
</style>
