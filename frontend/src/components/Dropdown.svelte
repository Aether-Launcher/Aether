<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from 'svelte';
  
  export let options: { label: string, value: string, disabled?: boolean }[] = [];
  export let value: string = "";
  export let disabled = false;
  export let direction: 'up' | 'down' = 'down';

  let isOpen = false;
  let dropdownRef: HTMLElement;
  // Unique ID for this dropdown instance used for mutual exclusion
  const uid = Math.random().toString(36).slice(2);

  const dispatch = createEventDispatcher();

  $: selectedLabel = options.find(o => o.value === value)?.label || "Select...";

  function selectOption(opt: any) {
    if (disabled || opt.disabled) return;
    value = opt.value;
    isOpen = false;
    dispatch('change', value);
  }

  function toggle() {
    if (disabled) return;
    if (!isOpen) {
      open();
    } else {
      isOpen = false;
    }
  }

  function open() {
    isOpen = true;
    // Tell every other dropdown instance to close
    window.dispatchEvent(new CustomEvent('aether:dropdown-opened', { detail: uid }));
  }

  // Close when another dropdown opens
  function onOtherDropdownOpened(e: Event) {
    const evt = e as CustomEvent<string>;
    if (evt.detail !== uid) {
      isOpen = false;
    }
  }

  // Click outside to close
  function handleOutsideClick(e: MouseEvent) {
    if (dropdownRef && !dropdownRef.contains(e.target as Node)) {
      isOpen = false;
    }
  }

  onMount(() => {
    window.addEventListener('aether:dropdown-opened', onOtherDropdownOpened);
  });

  onDestroy(() => {
    window.removeEventListener('aether:dropdown-opened', onOtherDropdownOpened);
  });
</script>

<svelte:window on:click={handleOutsideClick} />

<div class="custom-dropdown" bind:this={dropdownRef} class:disabled>
  <div
    class="dropdown-selected"
    role="button"
    tabindex="0"
    on:click={toggle}
    on:keydown={(e) => (e.key === 'Enter' || e.key === ' ') && toggle()}
    class:open={isOpen}
  >
    <span>{selectedLabel}</span>
    <svg class="chevron" class:open={isOpen} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="6 9 12 15 18 9"></polyline>
    </svg>
  </div>
  
  {#if isOpen}
    <div class="dropdown-options" class:drop-up={direction === 'up'}>
      {#each options as opt}
        <div
          class="dropdown-option"
          class:selected={opt.value === value}
          class:opt-disabled={opt.disabled}
          role="option"
          aria-selected={opt.value === value}
          tabindex="0"
          on:click={() => selectOption(opt)}
          on:keydown={(e) => (e.key === 'Enter' || e.key === ' ') && selectOption(opt)}
        >
          {opt.label}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .custom-dropdown {
    position: relative;
    width: 100%;
    font-size: 14px;
    user-select: none;
  }
  
  .disabled {
    opacity: 0.5;
    pointer-events: none;
  }

  .dropdown-selected {
    background: rgba(0,0,0,0.2);
    border: 1px solid rgba(255,255,255,0.1);
    color: white;
    padding: 10px 12px;
    border-radius: var(--border-radius);
    display: flex;
    justify-content: space-between;
    align-items: center;
    cursor: pointer;
    transition: all 0.2s;
  }

  .dropdown-selected:hover, .dropdown-selected.open {
    border-color: rgba(255,255,255,0.3);
    background: rgba(255,255,255,0.05);
  }

  .chevron {
    width: 16px;
    height: 16px;
    transition: transform 0.2s;
    color: var(--text-secondary);
  }
  
  .chevron.open {
    transform: rotate(180deg);
  }

  .dropdown-options {
    position: absolute;
    top: calc(100% + 4px);
    left: 0;
    right: 0;
    background: rgba(22, 22, 22, 0.92);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255,255,255,0.08);
    border-radius: var(--border-radius);
    max-height: 200px;
    overflow-y: auto;
    z-index: 2000;
    box-shadow: 0 8px 32px rgba(0,0,0,0.6);
    padding: 4px 0;
  }
  
  .dropdown-options.drop-up {
    top: auto;
    bottom: calc(100% + 4px);
  }

  .dropdown-option {
    padding: 10px 12px;
    cursor: pointer;
    color: var(--text-meta);
    transition: background 0.1s, color 0.1s;
    border-left: 2px solid transparent;
  }

  .dropdown-option:not(.opt-disabled):hover {
    background: rgba(255,255,255,0.06);
    color: white;
  }

  .dropdown-option.selected {
    color: var(--accent-color);
    background: rgba(59, 130, 246, 0.1);
    border-left-color: var(--accent-color);
    font-weight: 600;
  }

  .dropdown-option.opt-disabled {
    opacity: 0.4;
    cursor: not-allowed;
    font-style: italic;
  }
  
  /* Custom Scrollbar */
  .dropdown-options::-webkit-scrollbar {
    width: 8px;
  }
  .dropdown-options::-webkit-scrollbar-track {
    background: transparent;
  }
  .dropdown-options::-webkit-scrollbar-thumb {
    background: rgba(255,255,255,0.12);
    border-radius: 4px;
    border: 2px solid transparent;
    background-clip: content-box;
  }
  .dropdown-options::-webkit-scrollbar-thumb:hover {
    background: rgba(255,255,255,0.25);
    background-clip: content-box;
  }
</style>
