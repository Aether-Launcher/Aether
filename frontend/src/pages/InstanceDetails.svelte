<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { GetInstances, UpdateInstance, DeleteInstance, LaunchInstance, OpenInstanceFolder, PickInstanceIcon, RemoveInstanceIcon, GetInstanceIcon } from '../../wailsjs/go/main/App.js';
  import Dropdown from '../components/Dropdown.svelte';
  import ConfirmDialog from '../lib/components/ConfirmDialog.svelte';

  export let instanceId = '';

  let loadInstanceToken = 0;

  const dispatch = createEventDispatcher();
  let instance: any = null;

  // Form fields
  let editName = '';
  let editMemory = '2048'; // Using dropdown for memory as per plan (2G, 4G, 8G)

  const memoryOptions = [
    { label: '2 GB', value: '2048' },
    { label: '4 GB', value: '4096' },
    { label: '6 GB', value: '6144' },
    { label: '8 GB', value: '8192' },
    { label: '12 GB', value: '12288' },
    { label: '16 GB', value: '16384' },
  ];

  onMount(async () => {
    await loadInstance();
  });

  // Reactive: reload instance when instanceId changes without remount
  $: if (instanceId) {
    loadInstanceToken++;
    const token = loadInstanceToken;
    loadInstance().then(() => {
      if (token === loadInstanceToken) loadInstanceToken = 0;
    });
  }

  let iconDataUrl = '';
  let pickingIcon = false;

  async function loadInstance() {
    const all = await GetInstances();
    instance = all.find((i: any) => i.id === instanceId);
    if (instance) {
      editName = instance.name;
      editMemory = instance.memory || '2048';
      try {
        iconDataUrl = await GetInstanceIcon(instance.id) || '';
      } catch {
        iconDataUrl = '';
      }
    }
  }

  async function pickIcon() {
    if (!instance || pickingIcon) return;
    pickingIcon = true;
    try {
      const url = await PickInstanceIcon(instance.id);
      if (url) iconDataUrl = url;
    } catch (e: any) {
      console.error("Failed to set instance icon:", e);
    } finally {
      pickingIcon = false;
    }
  }

  async function removeIcon() {
    if (!instance) return;
    try {
      await RemoveInstanceIcon(instance.id);
      iconDataUrl = '';
    } catch (e: any) {
      console.error("Failed to remove instance icon:", e);
    }
  }

  async function saveChanges() {
    if (!instance) return;
    instance = { ...instance, name: editName, memory: editMemory };
    
    try {
      await UpdateInstance(instance);
      // Go back to instances page
      dispatch('navigate', 'instances');
    } catch (e) {
      console.error("Failed to save instance:", e);
    }
  }

  let confirmDialog: any;
  let pendingDelete = false;

  async function deleteInstance() {
    if (!instance) return;
    confirmDialog.open(
      `Delete ${instance.name}?`,
      `Are you sure you want to delete "${instance.name}"? This will permanently remove all its files and cannot be undone.`,
      true
    );
  }

  function handleDeleteConfirm(event: CustomEvent<boolean>) {
    if (!event.detail || !instance || pendingDelete) return;
    pendingDelete = true;
    DeleteInstance(instance.id).then(() => {
      dispatch('navigate', 'instances');
    }).catch((e: any) => {
      console.error("Failed to delete instance:", e);
      pendingDelete = false;
    });
  }

  function launch() {
    if (!instance) return;
    LaunchInstance(instance.id);
    dispatch('navigate', 'home'); // switch to home to see status
  }

  async function openFolder() {
    if (!instance) return;
    try {
      await OpenInstanceFolder(instance.id);
    } catch (e: any) {
      console.error("Failed to open instance folder:", e);
    }
  }

  // Consistent gradient generator based on ID
  function generateGradient(id: string) {
    let hash = 0;
    for (let i = 0; i < id.length; i++) {
      hash = id.charCodeAt(i) + ((hash << 5) - hash);
    }
    const hue1 = hash % 360;
    const hue2 = (hash * 2) % 360;
    return `linear-gradient(135deg, hsl(${hue1}, 70%, 60%), hsl(${hue2}, 70%, 40%))`;
  }
</script>

<div class="page page-enter">
  {#if instance}
    <div class="header">
      <button class="btn btn-secondary back-btn" on:click={() => dispatch('navigate', 'instances')}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M19 12H5M12 19l-7-7 7-7"/>
        </svg>
        Back
      </button>

      <div class="header-content">
        {#if iconDataUrl}
          <img src={iconDataUrl} alt="Instance icon" class="art-square art-square-img" />
        {:else}
          <div class="art-square" style="background: {generateGradient(instance.id)}"></div>
        {/if}
        <div class="info">
          <h1>{instance.name}</h1>
          <p class="meta">
            <span class="badge badge-version">{instance.version}</span>
            {#if instance.loader === 'Fabric'}
              <span class="badge badge-community">Fabric</span>
            {:else}
              <span class="badge badge-official">Vanilla</span>
            {/if}
            <span class="last-played">{instance.lastPlayed ? `Last played ${instance.lastPlayed.split('T')[0]}` : 'Never played'}</span>
          </p>
        </div>
      </div>
    </div>

    <div class="settings-card card">
      <h2>Instance Settings</h2>
      
      <div class="form-group">
        <label for="name">Name</label>
        <input id="name" type="text" bind:value={editName} class="input" />
      </div>

      <div class="form-group">
        <!-- svelte-ignore a11y-label-has-associated-control -->
        <label>Memory Allocation</label>
        <Dropdown options={memoryOptions} bind:value={editMemory} />
      </div>

      <div class="form-group">
        <!-- svelte-ignore a11y-label-has-associated-control -->
        <label>Icon</label>
        <div class="icon-row">
          {#if iconDataUrl}
            <img src={iconDataUrl} alt="Instance icon" class="icon-preview" />
          {:else}
            <div class="icon-preview icon-fallback" style="background: {generateGradient(instance.id)}">
              <span>{instance.name.charAt(0).toUpperCase()}</span>
            </div>
          {/if}
          <div class="icon-actions">
            <button class="btn btn-secondary" on:click={pickIcon} disabled={pickingIcon}>
              {pickingIcon ? 'Choosing…' : 'Choose from disk'}
            </button>
            {#if iconDataUrl}
              <button class="btn btn-secondary" on:click={removeIcon}>Remove</button>
            {/if}
          </div>
        </div>
        <div class="label-desc">PNG or JPEG, up to 5 MB. Modpack installs use the pack's icon automatically.</div>
      </div>

      <div class="actions">
        <button class="btn btn-danger" on:click={deleteInstance}>Delete Instance</button>
        <div class="right-actions">
          <button class="btn btn-secondary" on:click={openFolder}>Open Folder</button>
          <button class="btn btn-secondary" on:click={launch}>Play</button>
          <button class="btn btn-primary" on:click={saveChanges}>Save Changes</button>
        </div>
      </div>
    </div>
  {:else}
    <div class="loading">Loading instance...</div>
  {/if}
</div>

<ConfirmDialog bind:this={confirmDialog} on:confirm={handleDeleteConfirm} />

<style>
  .page {
    padding: var(--spacing-xl);
    /* Fix #7: explicit height + overflow-y:auto makes this an independent
       scroll context on macOS WebKit, preventing content/dropdown clipping */
    height: 100%;
    box-sizing: border-box;
    overflow-y: auto;
    overflow-x: hidden;
    /* Extra bottom padding so action buttons don't get clipped by the scroll container */
    padding-bottom: calc(var(--spacing-xl) * 2);
  }

  .back-btn {
    margin-bottom: var(--spacing-lg);
    gap: var(--spacing-sm);
  }

  .header-content {
    display: flex;
    align-items: center;
    gap: var(--spacing-lg);
    margin-bottom: var(--spacing-xl);
  }

  .art-square {
    width: 96px;
    height: 96px;
    border-radius: 16px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  .art-square-img {
    object-fit: cover;
  }

  .info h1 {
    font-size: 32px;
    margin: 0 0 8px 0;
    color: var(--text-primary);
  }

  .meta {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-meta);
    font-size: 13px;
  }

  .last-played {
    color: var(--text-secondary);
  }

  .settings-card {
    max-width: 600px;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-lg);
    /* overflow: visible so the Dropdown list (position: absolute)
       can escape the card boundary on macOS WebKit (fix for #7) */
    overflow: visible;
  }

  .settings-card h2 {
    font-size: 18px;
    font-weight: 600;
    margin: 0;
    color: var(--text-primary);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding-bottom: var(--spacing-md);
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  label {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-meta);
  }

  .input {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: var(--text-primary);
    padding: 10px 12px;
    border-radius: var(--border-radius);
    font-size: 14px;
    font-family: inherit;
    outline: none;
    transition: border-color var(--transition-fast);
  }

  .input:focus {
    border-color: var(--accent-color);
  }

  .icon-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .icon-preview {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .icon-fallback {
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: 800;
    color: rgba(255, 255, 255, 0.9);
  }

  .icon-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .label-desc {
    font-size: 12px;
    color: var(--text-meta);
    line-height: 1.4;
  }

  .actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: var(--spacing-md);
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    padding-top: var(--spacing-lg);
  }

  .right-actions {
    display: flex;
    gap: var(--spacing-sm);
  }
</style>
