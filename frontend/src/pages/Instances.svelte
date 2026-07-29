<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte';
  import { GetInstances, GetAvailableVersions, CreateInstance, GetModLoaders, SelectAndImportInstance } from '../../wailsjs/go/main/App.js';
  import Dropdown from '../components/Dropdown.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import { toast } from '../stores/toast';

  const dispatch = createEventDispatcher();

  let instances: any[] = [];
  let showModal = false;
  let newInstance = { name: "", version: "", loader: "Vanilla" };
  let availableVersions: string[] = [];
  let availableLoaders: any[] = [];
  let isCreating = false;
  let includeSnapshots = false;

  async function loadInstances() {
    const res = await GetInstances();
    instances = res || [];
  }

  async function loadVersions() {
    availableVersions = await GetAvailableVersions(includeSnapshots);
    if (availableVersions.length > 0 && (!newInstance.version || !availableVersions.includes(newInstance.version))) {
      newInstance.version = availableVersions[0];
    }
  }

  onMount(async () => {
    try {
      await loadInstances();
    } catch (err) {
      console.error(err);
      toast.error("Failed to load instances: " + err);
    }

    try {
      await loadVersions();
    } catch (err) {
      console.error(err);
      toast.error("Failed to load Minecraft versions from Mojang. Check your network: " + err);
    }
    
    try {
      const loaders = await GetModLoaders();
      availableLoaders = [
        { label: 'Vanilla', value: 'Vanilla' },
        ...(loaders || []).map(l => ({ label: l.name, value: l.id }))
      ];
    } catch (err) {
      console.error(err);
      availableLoaders = [{ label: 'Vanilla', value: 'Vanilla' }];
    }
  });

  async function handleCreate() {
    if (!newInstance.name || !newInstance.version) return;
    isCreating = true;
    try {
      const created = await CreateInstance(newInstance.name, newInstance.version, newInstance.loader);
      showModal = false;
      newInstance.name = "";
      newInstance.loader = "Vanilla";
      // Navigate straight to home with the new instance selected
      dispatch('navigate', `home:instance:${created.id}`);
      toast.success("Instance created successfully!");
    } catch (err: any) {
      console.error(err);
      toast.error("Failed to create instance: " + err);
    } finally {
      isCreating = false;
    }
  }

  async function handleImport() {
    try {
      if (await SelectAndImportInstance()) {
        await loadInstances();
        toast.success('Instance imported successfully!');
      }
    } catch (err: any) {
      toast.error('Failed to import instance: ' + err);
    }
  }
</script>

<div class="page">
  <header class="page-header">
    <h1>Instances</h1>
    <div class="actions">
      <button class="btn btn-secondary" on:click={handleImport}>Import</button>
      <button class="btn btn-primary" on:click={() => showModal = true}>Create New</button>
    </div>
  </header>

  {#if instances.length === 0}
    <EmptyState
      icon="instances"
      title="No instances yet"
      description="Create your first Minecraft instance to get started."
      actionLabel="+ Create Instance"
      on:action={() => (showModal = true)}
    />
  {:else}
    <div class="grid">
      {#each instances as instance}
        <div class="card instance-card">
          <div class="card-content">
            <div class="instance-title">{instance.name}</div>
            <div class="instance-meta">
              {instance.version} • {instance.loader}
            </div>
            <div class="instance-last-played">
              Last played: {instance.lastPlayed}
            </div>
          </div>
          <div class="card-actions">
            <button class="btn btn-primary" on:click={() => dispatch('navigate', `home:instance:${instance.id}`)}>Play</button>
            <button class="btn btn-secondary" on:click={() => dispatch('navigate', `instance-details:${instance.id}`)}>Settings</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if showModal}
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="modal-backdrop" on:click={() => showModal = false}>
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="modal" on:click|stopPropagation>
        <h2>Create Instance</h2>
        
        <div class="form-group">
          <!-- svelte-ignore a11y-label-has-associated-control -->
          <label>Instance Name</label>
          <input type="text" bind:value={newInstance.name} placeholder="e.g. My Survival World" />
        </div>

        <div class="form-group">
          <!-- svelte-ignore a11y-label-has-associated-control -->
          <div class="version-header">
            <label>Version</label>
            <label class="switch-wrapper">
              <span class="switch-label">Snapshots</span>
              <div class="switch">
                <input type="checkbox" bind:checked={includeSnapshots} on:change={loadVersions} />
                <span class="slider"></span>
              </div>
            </label>
          </div>
          <Dropdown 
            options={availableVersions.map(v => ({ label: v, value: v }))} 
            bind:value={newInstance.version} 
            direction="up"
          />
        </div>

        <div class="form-group">
          <!-- svelte-ignore a11y-label-has-associated-control -->
          <label>Mod Loader</label>
          <Dropdown 
            options={availableLoaders} 
            bind:value={newInstance.loader} 
            direction="up"
          />
          <p class="loader-hint">
            💡 Need Fabric, Forge, or NeoForge? Install mod loader extensions from the Extensions Gallery.
          </p>
        </div>

        <div class="modal-actions">
          <button class="btn btn-secondary" on:click={() => showModal = false}>Cancel</button>
          <button class="btn btn-primary" on:click={handleCreate} disabled={isCreating || !newInstance.name}>
            {isCreating ? 'Creating...' : 'Create'}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

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
  
  .actions {
    display: flex;
    gap: var(--spacing-md);
  }


  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
    gap: var(--spacing-lg);
  }

  .instance-card {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
  }

  .card-content {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .instance-title {
    font-size: 18px;
    font-weight: 600;
  }

  .instance-meta {
    font-size: 14px;
    color: var(--text-meta);
  }

  .instance-last-played {
    font-size: 12px;
    color: var(--text-secondary);
    margin-top: var(--spacing-sm);
  }

  .card-actions {
    display: flex;
    gap: var(--spacing-sm);
    margin-top: auto;
  }
  
  .card-actions .btn {
    flex: 1;
  }

  /* Modal Styles */
  .modal-backdrop {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0,0,0,0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .modal {
    background: rgba(25, 25, 25, 0.7);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    padding: var(--spacing-xl);
    border-radius: var(--border-radius);
    width: 400px;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-lg);
    box-shadow: 0 10px 40px rgba(0,0,0,0.6);
    border: 1px solid rgba(255,255,255,0.1);
  }

  .modal h2 {
    margin: 0;
    font-size: 20px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .form-group label {
    font-size: 14px;
    color: var(--text-secondary);
  }

  .form-group input {
    background: rgba(0,0,0,0.2);
    border: 1px solid rgba(255,255,255,0.1);
    color: white;
    padding: 10px 12px;
    border-radius: var(--border-radius);
    font-family: inherit;
    font-size: 14px;
    outline: none;
    transition: border-color 0.2s;
  }

  .form-group input:focus {
    border-color: var(--accent);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--spacing-md);
    margin-top: var(--spacing-sm);
  }

  .version-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .switch-wrapper {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }

  .switch-label {
    font-size: 12px;
    color: var(--text-secondary);
  }

  .switch {
    position: relative;
    display: inline-block;
    width: 32px;
    height: 18px;
  }

  .switch input {
    opacity: 0;
    width: 0;
    height: 0;
  }

  .slider {
    position: absolute;
    cursor: pointer;
    top: 0; left: 0; right: 0; bottom: 0;
    background-color: rgba(255,255,255,0.1);
    transition: .3s;
    border-radius: 18px;
  }

  .slider:before {
    position: absolute;
    content: "";
    height: 14px;
    width: 14px;
    left: 2px;
    bottom: 2px;
    background-color: var(--text-meta);
    transition: .3s;
    border-radius: 50%;
  }

  input:checked + .slider {
    background-color: var(--accent);
  }

  input:checked + .slider:before {
    transform: translateX(14px);
    background-color: white;
  }

  .loader-hint {
    margin: 8px 0 0 0;
    font-size: 12px;
    color: rgba(255, 255, 255, 0.5);
    line-height: 1.4;
  }
</style>
