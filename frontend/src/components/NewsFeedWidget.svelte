<script lang="ts">
  import { onMount } from 'svelte';
  import { GetMinecraftNews, GetAetherReleaseNotes } from '../../wailsjs/go/main/App';
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';

  let activeTab: 'minecraft' | 'aether' = 'minecraft';
  let mcNews: Array<{
    title: string;
    tag: string;
    date: string;
    text: string;
    image: string;
    readMoreUrl: string;
  }> = [];
  let aetherReleases: Array<{
    tagName: string;
    name: string;
    body: string;
    publishedAt: string;
    htmlUrl: string;
  }> = [];
  let loading = true;

  async function loadData() {
    loading = true;
    try {
      const [news, releases] = await Promise.all([
        GetMinecraftNews().catch(() => []),
        GetAetherReleaseNotes().catch(() => []),
      ]);
      mcNews = news || [];
      aetherReleases = releases || [];
    } catch (e) {
      console.error('Failed to load news feed:', e);
    } finally {
      loading = false;
    }
  }

  function formatDate(d: string): string {
    if (!d) return '';
    try {
      return new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
    } catch {
      return d;
    }
  }

  function openExternal(url: string) {
    if (!url) return;
    try {
      BrowserOpenURL(url);
    } catch {
      window.open(url, '_blank');
    }
  }

  onMount(() => {
    loadData();
  });
</script>

<div class="widget-card card">
  <div class="widget-header">
    <div class="tabs-wrap">
      <button
        class="feed-tab {activeTab === 'minecraft' ? 'active' : ''}"
        on:click={() => (activeTab = 'minecraft')}
      >
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
        </svg>
        <span>Minecraft Notes</span>
      </button>
      <button
        class="feed-tab {activeTab === 'aether' ? 'active' : ''}"
        on:click={() => (activeTab = 'aether')}
      >
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
        <span>Aether Updates</span>
      </button>
    </div>
  </div>

  {#if loading}
    <div class="loading-state">
      <span>Loading news feed...</span>
    </div>
  {:else if activeTab === 'minecraft'}
    {#if mcNews.length === 0}
      <div class="empty-state">
        <p class="empty-text">No news articles found.</p>
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
      <div class="empty-state">
        <p class="empty-text">No launcher releases found.</p>
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
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .widget-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .tabs-wrap {
    display: flex;
    background: rgba(255, 255, 255, 0.04);
    padding: 3px;
    border-radius: 6px;
    gap: 2px;
  }

  .feed-tab {
    display: flex;
    align-items: center;
    gap: 6px;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    font-size: 11px;
    font-weight: 600;
    padding: 4px 10px;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .feed-tab:hover {
    color: var(--text-primary);
  }

  .feed-tab.active {
    background: rgba(255, 255, 255, 0.08);
    color: #ffffff;
  }

  .loading-state {
    padding: 24px;
    text-align: center;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .empty-state {
    padding: 20px;
    text-align: center;
  }

  .empty-text {
    font-size: 12px;
    color: var(--text-secondary);
    margin: 0;
  }

  .news-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 260px;
    overflow-y: auto;
    padding-right: 4px;
  }

  .news-item {
    display: flex;
    gap: 10px;
    padding: 8px;
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.04);
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
    background: rgba(255, 255, 255, 0.05);
    border-color: rgba(255, 255, 255, 0.08);
  }

  .news-img-wrap {
    width: 68px;
    height: 52px;
    border-radius: 4px;
    overflow: hidden;
    flex-shrink: 0;
    background: #000;
  }

  .news-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .news-content {
    flex: 1;
    min-width: 0;
  }

  .news-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 2px;
  }

  .news-tag {
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    background: rgba(59, 130, 246, 0.15);
    color: #60a5fa;
    border: 1px solid rgba(59, 130, 246, 0.25);
    padding: 1px 5px;
    border-radius: 3px;
  }

  .release-tag {
    background: rgba(16, 185, 129, 0.15);
    color: #34d399;
    border-color: rgba(16, 185, 129, 0.25);
  }

  .news-date {
    font-size: 10px;
    color: var(--text-secondary);
  }

  .news-title {
    margin: 0 0 3px 0;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .news-desc {
    margin: 0;
    font-size: 11px;
    color: var(--text-secondary);
    line-height: 1.35;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
</style>
