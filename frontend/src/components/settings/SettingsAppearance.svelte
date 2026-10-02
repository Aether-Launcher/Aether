<script lang="ts">
  export let themes: any[] = [];
  export let installingTheme = false;
  export let themeBusyId = '';
  export let onInstall: () => void;
  export let onApply: (id: string) => void;
  export let onDisable: () => void;
  export let onRemove: (id: string) => void;
</script>

<div class="settings-card card">
  <h2>Appearance</h2>

  <div class="form-group vertical">
    <div class="field-label">
      <div class="label-title">Themes</div>
      <div class="label-desc">
        Install a <code>.theme</code> package to restyle Aether. Themes are a CSS overwrite plus optional
        logo/background images — they can't touch the launcher's app icon or its name.
      </div>
    </div>

    <div class="theme-list">
      {#if themes.length === 0}
        <p class="theme-empty">No themes installed yet.</p>
      {/if}
      {#each themes as t (t.id)}
        <div class="theme-item">
          <div class="theme-item-info">
            {#if t.iconUrl}
              <img src={t.iconUrl} alt="" class="theme-icon" />
            {/if}
            <div>
              <div class="theme-name">
                {t.name}
                <span class="theme-version">v{t.version}</span>
              </div>
              {#if t.description}<div class="theme-desc">{t.description}</div>{/if}
            </div>
          </div>
          <div class="theme-item-actions">
            {#if t.active}
              <span class="badge badge-verified">Active</span>
              <button class="btn btn-secondary btn-sm" disabled={themeBusyId === t.id} on:click={onDisable}>
                Disable
              </button>
            {:else}
              <button class="btn btn-secondary btn-sm" disabled={!!themeBusyId} on:click={() => onApply(t.id)}>
                Apply
              </button>
            {/if}
            <button class="btn btn-danger btn-sm" disabled={themeBusyId === t.id} on:click={() => onRemove(t.id)}>
              Remove
            </button>
          </div>
        </div>
      {/each}
    </div>

    <button class="btn btn-secondary" disabled={installingTheme} on:click={onInstall}>
      {installingTheme ? 'Installing…' : 'Install Theme (.theme)'}
    </button>
  </div>
</div>

<style>
  .settings-card {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-lg);
  }

  .settings-card h2 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
    color: var(--text-primary);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    padding-bottom: var(--spacing-md);
  }

  .form-group {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--spacing-md);
  }

  .form-group.vertical {
    flex-direction: column;
    align-items: stretch;
  }

  .label-title {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-primary);
  }

  .label-desc {
    font-size: 12px;
    color: var(--text-meta);
    line-height: 1.4;
  }

  .theme-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
    width: 100%;
  }

  .theme-empty {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0;
  }

  .theme-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 14px;
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: var(--border-radius);
  }

  .theme-item-info {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .theme-icon {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .theme-name {
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary);
    display: flex;
    align-items: baseline;
    gap: 6px;
  }

  .theme-version {
    font-weight: 400;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .theme-desc {
    font-size: 12px;
    color: var(--text-meta);
    max-width: 42ch;
  }

  .theme-item-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .btn-sm {
    padding: 4px 10px;
    font-size: 12px;
  }
</style>
