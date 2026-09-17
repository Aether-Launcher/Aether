<script lang="ts">
  import { onMount } from 'svelte';
  import { GetMinecraftNews, GetAetherReleaseNotes } from '../../wailsjs/go/main/App';
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

  let activeTab: 'minecraft' | 'aether' = 'minecraft';
  let mcNews: any[] = [];
  let aetherReleases: any[] = [];
  let loading = true;

  async function loadFeed() {
    loading = true;
    try {
      const [mc, aether] = await Promise.all([
        GetMinecraftNews(),
        GetAetherReleaseNotes(),
      ]);
      mcNews = mc || [];
      aetherReleases = aether || [];
    } catch (e) {
      console.error('Failed to load news feeds:', e);
    } finally {
      loading = false;
    }
  }

  function formatDate(iso: string): string {
    if (!iso) return '';
    try {
      const d = new Date(iso);
      return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
    } catch {
      return '';
    }
  }

  function openExternal(url: string) {
    if (url) {
      BrowserOpenURL(url);
    }
  }

  onMount(() => {
    loadFeed();
  });
</script>

<div class="widget-card card">
  <div class="widget-header">
    <div class="widget-title">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/>
        <path d="M18 14h-8"/>
        <path d="M15 18h-5"/>
        <path d="M10 6h8v4h-8V6Z"/>
      </svg>
      <span>News & Updates</span>
    </div>

    <div class="tab-pills">
      <button
        class="tab-btn"
        class:active={activeTab === 'minecraft'}
        on:click={() => activeTab = 'minecraft'}
      >
        Minecraft
      </button>
      <button
        class="tab-btn"
        class:active={activeTab === 'aether'}
        on:click={() => activeTab = 'aether'}
      >
        Aether
      </button>
    </div>
  </div>

  {#if loading}
    <div class="loading-state">
      <span>Loading feed...</span>
    </div>
  {:else if activeTab === 'minecraft'}
    {#if mcNews.length === 0}
      <div class="empty-compact">
        <span class="empty-text">No news articles found</span>
      </div>
    {:else}
      <div class="news-list">
        {#each mcNews as item}
          <button type="button" class="news-item" on:click={() => openExternal(item.readMoreUrl)}>
            {#if item.image}
              <div class="news-img-wrap">
                <img src={item.image} alt="" class="news-img" loading="lazy" />
              </div>
            {/if}
            <div class="news-content">
              <div class="news-meta">
                {#if item.tag}
                  <span class="news-tag">{item.tag}</span>
                {/if}
                <span class="news-date">{formatDate(item.date)}</span>
              </div>
              <h4 class="news-title">{item.title}</h4>
              <p class="news-desc">{item.text}</p>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  {:else if activeTab === 'aether'}
    {#if aetherReleases.length === 0}
      <div class="empty-compact">
        <span class="empty-text">No launcher releases found</span>
      </div>
    {:else}
      <div class="news-list">
        {#each aetherReleases as rel}
          <button type="button" class="news-item" on:click={() => openExternal(rel.htmlUrl)}>
            <div class="news-content">
              <div class="news-meta">
                <span class="news-tag release-tag">{rel.tagName}</span>
                <span class="news-date">{formatDate(rel.publishedAt)}</span>
              </div>
              <h4 class="news-title">{rel.name}</h4>
              <p class="news-desc">{rel.body ? rel.body.replace(/[#*`_]/g, '').slice(0, 140) + '...' : 'Release notes available on GitHub.'}</p>
            </div>
          </button>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .widget-card {
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    background: var(--panel-bg);
    border: 1px solid rgba(255, 255, 255, 0.05);
    border-radius: var(--border-radius);
  }

  .widget-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .widget-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.8px;
  }

  .tab-pills {
    display: flex;
    gap: 2px;
    background: rgba(255, 255, 255, 0.04);
    padding: 2px;
    border-radius: 6px;
  }

  .tab-btn {
    background: none;
    border: none;
    font-size: 11px;
    font-weight: 500;
    color: var(--text-secondary);
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .tab-btn.active {
    background: rgba(255, 255, 255, 0.1);
    color: #ffffff;
    font-weight: 600;
  }

  .loading-state {
    padding: 14px;
    text-align: center;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .empty-compact {
    padding: 10px;
    text-align: center;
    background: rgba(255, 255, 255, 0.015);
    border-radius: 6px;
  }

  .empty-text {
    font-size: 11px;
    color: var(--text-secondary);
  }

  .news-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 220px;
    overflow-y: auto;
    padding-right: 2px;
  }

  .news-item {
    display: flex;
    gap: 10px;
    padding: 7px 8px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.035);
    border-radius: 6px;
    cursor: pointer;
    text-align: left;
    width: 100%;
    font: inherit;
    color: inherit;
    box-sizing: border-box;
    transition: background 0.15s ease, border-color 0.15s ease;
  }

  .news-item:hover {
    background: rgba(255, 255, 255, 0.04);
    border-color: rgba(255, 255, 255, 0.07);
  }

  .news-img-wrap {
    width: 60px;
    height: 44px;
    border-radius: 4px;
    overflow: hidden;
    flex-shrink: 0;
    background: #000;
  }

  .news-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .news-content {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .news-meta {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .news-tag {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--accent-color, #3b82f6);
    background: rgba(59, 130, 246, 0.1);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .release-tag {
    color: #a855f7;
    background: rgba(168, 85, 247, 0.1);
  }

  .news-date {
    font-size: 10px;
    color: var(--text-secondary);
  }

  .news-title {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .news-desc {
    font-size: 10px;
    color: var(--text-secondary);
    margin: 0;
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
</style>
