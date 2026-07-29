<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { SendExtensionMessage } from '../../wailsjs/go/main/App.js';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';

  export let url: string = '';
  export let extID: string = '';

  let iframeEl: HTMLIFrameElement;

  // Listen for messages FROM the iframe → forward to Go sandbox
  function onWindowMessage(event: MessageEvent) {
    if (!iframeEl || event.source !== iframeEl.contentWindow) return;
    if (extID && event.data && typeof event.data === 'object') {
      SendExtensionMessage(extID, event.data);
    }
  }

  // Listen for responses FROM Go sandbox → forward back to iframe
  let unsubscribe: (() => void) | null = null;

  onMount(() => {
    window.addEventListener('message', onWindowMessage);

    if (extID) {
      unsubscribe = EventsOn(`extension:message:${extID}`, (payload: any) => {
        if (iframeEl?.contentWindow) {
          iframeEl.contentWindow.postMessage(payload, '*');
        }
      });
    }
  });

  onDestroy(() => {
    window.removeEventListener('message', onWindowMessage);
    if (unsubscribe) unsubscribe();
  });
</script>

<div class="extension-view">
  {#if url}
    <iframe
      bind:this={iframeEl}
      src={url}
      aria-label="Extension View"
      sandbox="allow-scripts allow-same-origin allow-downloads"
    ></iframe>
  {:else}
    <div class="placeholder">
      <h2>Extension Loading...</h2>
    </div>
  {/if}
</div>

<style>
  .extension-view {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    background: var(--bg-color);
  }

  iframe {
    flex-grow: 1;
    width: 100%;
    height: 100%;
    border: none;
    background: transparent;
  }

  .placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
  }
</style>
