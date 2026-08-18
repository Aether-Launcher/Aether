<script lang="ts">
  import { onMount } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import { ResolveExtensionConfirmation, WindowChrome } from '../wailsjs/go/main/App.js';
  import TitleBar from './components/TitleBar.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import CommandPalette from './components/CommandPalette.svelte';
  import ToastContainer from './components/ToastContainer.svelte';
  import Home from './pages/Home.svelte';
  import Instances from './pages/Instances.svelte';
  import Extensions from './pages/Extensions.svelte';
  import ExtensionView from './pages/ExtensionView.svelte';
  import InstanceDetails from './pages/InstanceDetails.svelte';
  import Settings from './pages/Settings.svelte';
  import ConfirmDialog from './lib/components/ConfirmDialog.svelte';

  const isMacOS = /Macintosh|Mac OS X/i.test(navigator.userAgent);
  let titleBarStyle: 'custom' | 'system' | 'pending' = 'pending';
  let activePage = 'home';
  let targetInstanceId = '';
  let activeInstanceId = '';
  let extensionRoutes: Record<string, { url: string; extensionId: string }> = {};
  let paletteOpen = false;
  let confirmationDialog: any;
  let pendingConfirmation: any = null;

  // ── Navigation ──────────────────────────────────────────────────────────────

  function navigate(page: string) {
    activePage = page;
  }

  function handleNavigate(event: CustomEvent<string>) {
    const payload = event.detail;
    if (payload.startsWith('instance-details:')) {
      targetInstanceId = payload.split(':')[1];
      navigate('instance-details');
    } else if (payload.startsWith('home:instance:')) {
      // Navigate home and pre-select the given instance ID
      activeInstanceId = payload.split(':')[2];
      navigate('home');
    } else {
      navigate(payload);
    }
  }

  function handleRegisterExtensionRoute(event: CustomEvent<{ id: string; url: string; extensionId: string }>) {
    extensionRoutes[event.detail.id] = { url: event.detail.url, extensionId: event.detail.extensionId };
  }

  // ── Command Palette ──────────────────────────────────────────────────────────

  /** Core commands — every extension can push more in later */
  const commands = [
    { id: 'go-home',       label: 'Go to Home',        category: 'Navigation', action: () => navigate('home')       },
    { id: 'go-instances',  label: 'Go to Instances',   category: 'Navigation', action: () => navigate('instances')  },
    { id: 'go-extensions', label: 'Go to Extensions',  category: 'Navigation', action: () => navigate('extensions') },
    { id: 'go-settings',   label: 'Go to Settings',    category: 'Navigation', action: () => navigate('settings')   },
    { id: 'open-gallery',  label: 'Open Gallery',       category: 'Extensions', action: () => navigate('extensions') },
    {
      id: 'create-instance',
      label: 'Create Instance',
      category: 'Instances',
      action: () => {
        navigate('instances');
        // Slight delay so the page mounts before opening the modal
        setTimeout(() => {
          window.dispatchEvent(new CustomEvent('aether:open-create-instance'));
        }, 80);
      },
    },
  ];

  function openPalette() {
    paletteOpen = true;
  }

  function onGlobalKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'P') {
      e.preventDefault();
      openPalette();
    }
  }

  function showExtensionConfirmation(request: any) {
    pendingConfirmation = request;
    const extensionName = request.extensionName || request.extensionId || 'An extension';
    const action = request.action || 'perform a sensitive action';
    const target = request.jarName
      ? `mod "${request.jarName}" in instance "${request.instanceId}"`
      : `instance "${request.instanceId}"`;
    let source = '';
    if (request.url) {
      try {
        source = ' Source: ' + new URL(request.url).hostname;
      } catch {
        source = ' Source: external download';
      }
    }
    confirmationDialog.open(
      'Allow extension action?',
      `${extensionName} wants to ${action} on ${target}. This may change files that Minecraft will load.${source}`,
      action === 'delete mod',
      'Allow'
    );
  }

  async function resolveExtensionConfirmation(event: CustomEvent<boolean>) {
    const request = pendingConfirmation;
    pendingConfirmation = null;
    if (!request) return;
    try {
      await ResolveExtensionConfirmation(request.requestId, event.detail);
    } catch (error) {
      console.error('Failed to resolve extension confirmation:', error);
    }
  }

  onMount(async () => {
    const chrome = await WindowChrome().catch(() => 'custom');
    titleBarStyle = chrome === 'system' ? 'system' : 'custom';
    window.addEventListener('keydown', onGlobalKeydown);
    const unsubscribe = EventsOn('extension:confirmation', showExtensionConfirmation);
    return () => {
      window.removeEventListener('keydown', onGlobalKeydown);
      unsubscribe();
    };
  });
</script>

<div class="app-container">
  {#if titleBarStyle === 'custom'}
    <TitleBar />
  {:else if titleBarStyle === 'system' && isMacOS}
    <div class="native-titlebar-spacer" aria-hidden="true"></div>
  {/if}
  <div class="layout">
    <Sidebar
      {activePage}
      on:navigate={handleNavigate}
      on:registerExtensionRoute={handleRegisterExtensionRoute}
    />

    <main class="content">
      {#if activePage === 'home'}
        <Home on:navigate={handleNavigate} activeInstanceId={activeInstanceId} />
      {:else if activePage === 'instances'}
        <Instances on:navigate={handleNavigate} />
      {:else if activePage === 'instance-details'}
        <InstanceDetails instanceId={targetInstanceId} on:navigate={handleNavigate} />
      {:else if activePage === 'extensions'}
        <Extensions />
      {:else if activePage === 'settings'}
        <Settings />
      {:else if extensionRoutes[activePage]}
        <ExtensionView url={extensionRoutes[activePage].url} extID={extensionRoutes[activePage].extensionId} />
      {:else}
        <div class="placeholder">
          <h2>{activePage.charAt(0).toUpperCase() + activePage.slice(1)}</h2>
          <p>This page is under construction.</p>
        </div>
      {/if}
    </main>
  </div>

  <!-- Command Palette — rendered above everything -->
  <CommandPalette bind:open={paletteOpen} {commands} on:close={() => (paletteOpen = false)} />

  <!-- Global Toasts -->
  <ToastContainer />

  <ConfirmDialog bind:this={confirmationDialog} on:confirm={resolveExtensionConfirmation} />
</div>

<style>
  :global(body) {
    margin: 0;
    overflow: hidden;
  }

  .app-container {
    display: flex;
    flex-direction: column;
    width: 100vw;
    height: 100vh;
    overflow: hidden;
    background: var(--bg-dark, #0d0d0d);
  }

  .native-titlebar-spacer {
    height: 32px;
    flex-shrink: 0;
    background: #0d0d0d;
  }

  .layout {
    display: flex;
    width: 100%;
    flex: 1;
    min-height: 0;
  }

  .content {
    flex-grow: 1;
    background-color: var(--bg-color);
    overflow: hidden;
    height: 100%;
    position: relative;
  }

  .placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
  }

  .placeholder h2 {
    color: var(--text-primary);
    margin-bottom: var(--spacing-sm);
  }
</style>
