<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { flip } from 'svelte/animate';
  import { toast } from '../stores/toast';
</script>

<div class="toast-container">
  {#each $toast as t (t.id)}
    <div 
      class="toast toast-{t.type}" 
      in:fly={{ y: 20, duration: 300 }} 
      out:fade={{ duration: 200 }}
      animate:flip={{ duration: 300 }}
    >
      <div class="toast-icon">
        {#if t.type === 'success'}
          <svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>
        {:else if t.type === 'error'}
          <svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="15" y1="9" x2="9" y2="15"></line><line x1="9" y1="9" x2="15" y2="15"></line></svg>
        {:else if t.type === 'info'}
          <svg viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>
        {/if}
      </div>
      <div class="toast-content">
        {t.message}
      </div>
      <button class="toast-close" on:click={() => toast.dismiss(t.id)}>
        <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg>
      </button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    bottom: var(--spacing-lg, 24px);
    right: var(--spacing-lg, 24px);
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm, 12px);
    z-index: 9999;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    background: var(--bg-surface, #1e1e1e);
    border: 1px solid var(--border-color, #333);
    border-radius: var(--radius-md, 8px);
    padding: var(--spacing-md, 16px);
    min-width: 300px;
    max-width: 450px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5);
    pointer-events: auto;
    position: relative;
    overflow: hidden;
  }

  .toast::before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 4px;
  }

  .toast-success::before { background-color: var(--color-success, #2ecc71); }
  .toast-error::before { background-color: var(--color-danger, #e74c3c); }
  .toast-info::before { background-color: var(--color-info, #3498db); }

  .toast-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: var(--spacing-md, 16px);
  }

  .toast-success .toast-icon { color: var(--color-success, #2ecc71); }
  .toast-error .toast-icon { color: var(--color-danger, #e74c3c); }
  .toast-info .toast-icon { color: var(--color-info, #3498db); }

  .toast-content {
    flex: 1;
    color: var(--text-primary, #ffffff);
    font-size: 0.95rem;
    line-height: 1.4;
    margin-right: var(--spacing-md, 16px);
    word-break: break-word;
  }

  .toast-close {
    background: none;
    border: none;
    color: var(--text-secondary, #a0a0a0);
    cursor: pointer;
    padding: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: all 0.2s;
  }

  .toast-close:hover {
    color: var(--text-primary, #ffffff);
    background: rgba(255, 255, 255, 0.1);
  }
</style>
