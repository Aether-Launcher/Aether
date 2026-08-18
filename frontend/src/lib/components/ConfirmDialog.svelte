<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let title = 'Confirm Action';
  export let message = 'Are you sure you want to proceed?';
  export let confirmLabel = 'OK';
  export let cancelLabel = 'Cancel';
  export let danger = false;
  export let show = false;

  function confirm() {
    dispatch('confirm', true);
    show = false;
  }

  function cancel() {
    dispatch('confirm', false);
    show = false;
  }

  export function open(titleText: string, messageText: string, isDanger = false, confirmText = 'OK') {
    title = titleText;
    message = messageText;
    danger = isDanger;
    confirmLabel = confirmText;
    show = true;
  }
</script>

{#if show}
  <div
    class="overlay"
    role="presentation"
    on:click|self={cancel}
    on:keydown={(e) => e.key === 'Escape' && cancel()}
  >
    <div
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="dialog-title"
      on:click={(e) => e.stopPropagation()}
      on:keydown={(e) => e.stopPropagation()}
    >
      <h3 class="dialog-title" id="dialog-title">{title}</h3>
      <p class="dialog-message">{message}</p>
      <div class="dialog-actions">
        <button class="btn btn-secondary" on:click={cancel}>{cancelLabel}</button>
        <button
          class="btn"
          class:btn-danger={danger}
          class:btn-primary={!danger}
          on:click={confirm}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 1500;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    animation: fade-in 120ms ease;
  }

  .dialog {
    background: rgba(28, 28, 28, 0.85);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 28px 32px 24px;
    width: 380px;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
    animation: scale-in 150ms cubic-bezier(0.16, 1, 0.3, 1);
  }

  .dialog-title {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 12px;
  }

  .dialog-message {
    overflow-wrap: anywhere;
    font-size: 14px;
    line-height: 1.5;
    color: var(--text-meta);
    margin: 0 0 28px;
  }

  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }

  @keyframes fade-in {
    0% { opacity: 0; }
    100% { opacity: 1; }
  }

  @keyframes scale-in {
    0% { opacity: 0; transform: scale(0.95) translateY(4px); }
    100% { opacity: 1; transform: scale(1) translateY(0); }
  }
</style>
