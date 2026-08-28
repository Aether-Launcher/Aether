<script lang="ts">
  import { WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime.js';
  import { themeAssets } from '../stores/theme';

  let isMaximized = false;

  function minimize() {
    WindowMinimise();
  }

  function toggleMaximize() {
    WindowToggleMaximise();
    isMaximized = !isMaximized;
  }

  function closeApp() {
    Quit();
  }
</script>

<div class="titlebar" style="--wails-draggable: drag">
  <div class="title">
    <img src={$themeAssets['titlebar-logo'] || '/logo.png'} alt="Logo" class="logo-img" />
    <span>Aether</span>
  </div>

  <div class="controls" style="--wails-draggable: no-drag">
    <!-- Minimize -->
    <button class="win-btn" on:click={minimize} title="Minimize">
      <svg width="10" height="10" viewBox="0 0 10 10" stroke="currentColor" stroke-width="1.2" fill="none">
        <line x1="0" y1="5" x2="10" y2="5" />
      </svg>
    </button>

    <!-- Maximize / Restore — icon toggles on click -->
    <button class="win-btn" on:click={toggleMaximize} title={isMaximized ? 'Restore' : 'Maximize'}>
      {#if isMaximized}
        <!-- Restore icon: two overlapping squares -->
        <svg width="10" height="10" viewBox="0 0 10 10" stroke="currentColor" stroke-width="1.2" fill="none">
          <rect x="0" y="2" width="7" height="7" rx="0.5" />
          <path d="M2.5 2V1.5a.5.5 0 0 1 .5-.5h6a.5.5 0 0 1 .5.5v6a.5.5 0 0 1-.5.5H8.5" />
        </svg>
      {:else}
        <!-- Maximize icon: single square -->
        <svg width="10" height="10" viewBox="0 0 10 10" stroke="currentColor" stroke-width="1.2" fill="none">
          <rect x="1" y="1" width="8" height="8" rx="0.5" />
        </svg>
      {/if}
    </button>

    <!-- Close -->
    <button class="win-btn close-btn" on:click={closeApp} title="Close">
      <svg width="10" height="10" viewBox="0 0 10 10" stroke="currentColor" stroke-width="1.2" fill="none">
        <line x1="1" y1="1" x2="9" y2="9" />
        <line x1="9" y1="1" x2="1" y2="9" />
      </svg>
    </button>
  </div>
</div>

<style>
  .titlebar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    height: 32px;
    background: #0d0d0d;
    user-select: none;
    -webkit-user-select: none;
    flex-shrink: 0;
    /* border removed */
  }

  .title {
    display: flex;
    align-items: center;
    padding-left: 12px;
    font-size: 12px;
    color: #a0a0a0;
    gap: 8px;
    font-family: inherit;
  }

  .logo-img {
    width: 14px;
    height: 14px;
    object-fit: contain;
  }

  .controls {
    display: flex;
    height: 100%;
  }

  .win-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 46px;
    height: 100%;
    background: transparent;
    border: none;
    color: #a0a0a0;
    cursor: pointer;
    transition: background 0.1s, color 0.1s;
    outline: none;
  }

  .win-btn:hover {
    background: rgba(255, 255, 255, 0.08);
    color: #ffffff;
  }

  .close-btn:hover {
    background: #e81123;
    color: #ffffff;
  }
</style>
