<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';
  import { GetLogs, ClearLogs } from '../../wailsjs/go/main/App.js';
  import { toast } from '../stores/toast.js';
  import Icon from './Icon.svelte';

  type LogEntry = {
    timestamp: string;
    level: string;
    target: string;
    message: string;
  };

  export let show = false;

  let entries: LogEntry[] = [];
  let filterLevel = 'ALL';
  let searchQuery = '';
  let autoScroll = true;
  let logsContainer: HTMLDivElement;

  $: filteredEntries = entries.filter((e) => {
    const matchesLevel = filterLevel === 'ALL' || e.level === filterLevel;
    const matchesQuery =
      !searchQuery ||
      e.message.toLowerCase().includes(searchQuery.toLowerCase()) ||
      e.target.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesLevel && matchesQuery;
  });

  async function scrollToBottom() {
    if (autoScroll && logsContainer) {
      await tick();
      logsContainer.scrollTop = logsContainer.scrollHeight;
    }
  }

  function handleLogEvent(payload: unknown) {
    const entry = payload as LogEntry;
    if (entry && entry.message) {
      entries = [...entries, entry];
      scrollToBottom();
    }
  }

  async function loadLogs() {
    try {
      const logs = (await GetLogs()) || [];
      entries = logs;
      scrollToBottom();
    } catch (e) {
      console.error('Failed to load logs:', e);
    }
  }

  async function handleClear() {
    try {
      await ClearLogs();
      entries = [];
      toast.info('Terminal logs cleared.');
    } catch (e) {
      console.error(e);
    }
  }

  function copyLogs() {
    const text = entries
      .map((e) => `[${e.timestamp}] [${e.level}] [${e.target}] ${e.message}`)
      .join('\n');
    navigator.clipboard.writeText(text);
    toast.success('Logs copied to clipboard!');
  }

  function toggleOpen() {
    show = !show;
    if (show) {
      loadLogs();
    }
  }

  function handleOpenEvent() {
    show = true;
    loadLogs();
  }

  function handleToggleEvent() {
    toggleOpen();
  }

  onMount(() => {
    EventsOn('log:event', handleLogEvent);
    window.addEventListener('aether:open-terminal-logs', handleOpenEvent);
    window.addEventListener('aether:toggle-terminal-logs', handleToggleEvent);
  });

  onDestroy(() => {
    EventsOff('log:event');
    window.removeEventListener('aether:open-terminal-logs', handleOpenEvent);
    window.removeEventListener('aether:toggle-terminal-logs', handleToggleEvent);
  });
</script>

{#if show}
  <div
    class="modal-backdrop"
    role="presentation"
    on:click|self={() => (show = false)}
    on:keydown={(e) => e.key === 'Escape' && (show = false)}
  >
    <div
      class="terminal-card"
      role="dialog"
      aria-modal="true"
      aria-labelledby="terminal-title"
      on:click={(e) => e.stopPropagation()}
      on:keydown={(e) => e.stopPropagation()}
    >
      <div class="terminal-header">
        <div class="header-left">
          <span class="terminal-icon">
            <Icon name="terminal" size={14} />
          </span>
          <h3 id="terminal-title">Developer Logs</h3>
          <span class="entry-count">{entries.length} events</span>
        </div>

        <div class="header-right">
          <button
            class="action-btn"
            class:active={autoScroll}
            on:click={() => (autoScroll = !autoScroll)}
            title="Auto-scroll to latest"
          >
            {autoScroll ? 'Auto-scroll: On' : 'Auto-scroll: Off'}
          </button>
          <button class="action-btn" on:click={copyLogs} title="Copy all logs">
            Copy
          </button>
          <button class="action-btn danger" on:click={handleClear} title="Clear log buffer">
            Clear
          </button>
          <button class="close-btn" on:click={() => (show = false)} title="Close">&times;</button>
        </div>
      </div>

      <div class="terminal-toolbar">
        <input
          type="text"
          class="search-input"
          placeholder="Filter logs..."
          bind:value={searchQuery}
        />

        <div class="level-filters">
          {#each ['ALL', 'INFO', 'WARN', 'ERROR', 'SANDBOX', 'DEBUG'] as lvl}
            <button
              class="lvl-btn {lvl.toLowerCase()}"
              class:selected={filterLevel === lvl}
              on:click={() => (filterLevel = lvl)}
            >
              {lvl}
            </button>
          {/each}
        </div>
      </div>

      <div class="terminal-body" bind:this={logsContainer}>
        {#if filteredEntries.length === 0}
          <div class="empty-logs">
            No log events recorded matching current filters.
          </div>
        {:else}
          {#each filteredEntries as entry}
            <div class="log-row">
              <span class="timestamp">{entry.timestamp}</span>
              <span class="level-badge {entry.level.toLowerCase()}">{entry.level}</span>
              <span class="target-tag">[{entry.target}]</span>
              <span class="message-text">{entry.message}</span>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background: rgba(0, 0, 0, 0.7);
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
    z-index: 99999;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    box-sizing: border-box;
    animation: fadeIn 0.15s ease-out;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .terminal-card {
    width: 100%;
    max-width: 920px;
    height: 600px;
    max-height: 85vh;
    background: var(--sidebar-bg, #141414);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: var(--border-radius, 8px);
    box-shadow: 0 24px 48px rgba(0, 0, 0, 0.7);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    color: var(--text-primary, #ffffff);
    font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  }

  .terminal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    background: var(--sidebar-bg, #141414);
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .terminal-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--accent-color, #3b82f6);
    background: rgba(59, 130, 246, 0.12);
    border: 1px solid rgba(59, 130, 246, 0.25);
    padding: 4px 6px;
    border-radius: 4px;
    font-size: 13px;
  }

  .header-left h3 {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary, #fff);
    font-family: inherit;
  }

  .entry-count {
    font-size: 11px;
    color: var(--text-secondary, #888);
    background: rgba(255, 255, 255, 0.06);
    padding: 2px 8px;
    border-radius: 10px;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .action-btn {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: var(--text-meta, #c2c2c2);
    font-size: 12px;
    font-weight: 500;
    padding: 4px 10px;
    border-radius: var(--border-radius, 6px);
    cursor: pointer;
    transition: all var(--transition-fast, 150ms ease);
    font-family: inherit;
  }

  .action-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-primary, #fff);
  }

  .action-btn.active {
    border-color: var(--accent-color, #3b82f6);
    color: var(--accent-color, #3b82f6);
    background: rgba(59, 130, 246, 0.12);
  }

  .action-btn.danger:hover {
    background: rgba(239, 68, 68, 0.15);
    border-color: var(--danger-color, #ef4444);
    color: var(--danger-color, #ef4444);
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-secondary, #888);
    font-size: 18px;
    line-height: 1;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
    transition: color 0.15s, background 0.15s;
  }

  .close-btn:hover {
    color: var(--text-primary, #fff);
    background: rgba(255, 255, 255, 0.1);
  }

  .terminal-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 16px;
    background: var(--panel-bg, #1c1c1c);
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    gap: 12px;
  }

  .search-input {
    flex: 1;
    max-width: 280px;
    background: rgba(0, 0, 0, 0.35);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: var(--border-radius, 6px);
    padding: 6px 10px;
    font-size: 12px;
    color: var(--text-primary, #fff);
    font-family: inherit;
    outline: none;
    transition: border-color var(--transition-fast, 150ms ease);
  }

  .search-input:focus {
    border-color: var(--accent-color, #3b82f6);
  }

  .search-input::placeholder {
    color: var(--text-disabled, #555);
  }

  .level-filters {
    display: flex;
    gap: 4px;
  }

  .lvl-btn {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-secondary, #888);
    font-size: 11px;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
    cursor: pointer;
    font-family: inherit;
    transition: all var(--transition-fast, 150ms ease);
  }

  .lvl-btn:hover {
    color: var(--text-primary, #fff);
    background: rgba(255, 255, 255, 0.05);
  }

  .lvl-btn.selected {
    background: rgba(59, 130, 246, 0.15);
    color: var(--accent-color, #3b82f6);
    border-color: rgba(59, 130, 246, 0.35);
  }

  .terminal-body {
    flex: 1;
    overflow-y: auto;
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    line-height: 1.5;
    background: var(--bg-color, #0d0d0d);
  }

  .empty-logs {
    color: var(--text-disabled, #555);
    font-style: italic;
    padding: 40px 0;
    text-align: center;
  }

  .log-row {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    word-break: break-all;
  }

  .timestamp {
    color: var(--text-disabled, #555);
    font-size: 11px;
    flex-shrink: 0;
  }

  .level-badge {
    font-size: 10px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 3px;
    flex-shrink: 0;
    text-transform: uppercase;
  }

  .level-badge.info { background: rgba(59, 130, 246, 0.15); color: var(--accent-color, #3b82f6); }
  .level-badge.warn { background: rgba(245, 158, 11, 0.15); color: #f59e0b; }
  .level-badge.error { background: rgba(239, 68, 68, 0.15); color: var(--danger-color, #ef4444); }
  .level-badge.sandbox { background: rgba(168, 85, 247, 0.15); color: #c084fc; }
  .level-badge.debug { background: rgba(255, 255, 255, 0.08); color: var(--text-secondary, #888); }

  .target-tag {
    color: var(--text-secondary, #888);
    flex-shrink: 0;
    font-weight: 600;
  }

  .message-text {
    color: var(--text-meta, #c2c2c2);
  }
</style>
