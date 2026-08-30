<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { GetActiveAccount, GetAccounts, SetActiveAccount, RemoveAccount } from '../../wailsjs/go/main/App';
  import type { auth } from '../../wailsjs/go/models';
  import LoginModal from './LoginModal.svelte';

  let activeAccount: auth.Account | null = null;
  let accounts: auth.Account[] = [];
  let showDropdown = false;
  let showLoginModal = false;
  let dropdownEl: HTMLElement;
  let avatarError = false;

  $: if (activeAccount?.username) {
    avatarError = false;
  }

  async function loadData() {
    try {
      activeAccount = await GetActiveAccount();
      accounts = await GetAccounts();
    } catch (e) {
      console.error("Failed to load accounts data", e);
    }
  }

  onMount(() => {
    loadData();
    window.addEventListener('click', handleClickOutside);
  });

  onDestroy(() => {
    window.removeEventListener('click', handleClickOutside);
  });

  function handleClickOutside(event: MouseEvent) {
    const target = event.target as Element;
    if (showDropdown && dropdownEl && !dropdownEl.contains(target) && !target.closest('.account-card')) {
      showDropdown = false;
    }
  }

  async function handleSwitch(id: string) {
    try {
      await SetActiveAccount(id);
      await loadData();
      showDropdown = false;
    } catch (e) {
      console.error("Failed to switch account", e);
    }
  }

  async function handleLogout(id: string, event: Event) {
    event.stopPropagation(); // Prevent triggering handleSwitch
    try {
      await RemoveAccount(id);
      await loadData();
    } catch (e) {
      console.error("Failed to remove account", e);
    }
  }

  function handleLogin() {
    loadData();
  }

  function handleImageError(e: Event) {
    const el = e.currentTarget as HTMLElement;
    if (el) el.style.display = 'none';
  }
</script>

<div class="account-manager-wrapper">
  {#if showDropdown}
    <div class="accounts-dropdown" bind:this={dropdownEl}>
      <div class="dropdown-header">
        <span>Switch Account</span>
      </div>
      
      <div class="accounts-list">
        {#if accounts.length === 0}
          <div class="no-accounts">No accounts added yet</div>
        {:else}
          {#each accounts as acc}
            <div 
              class="account-item {activeAccount && activeAccount.id === acc.id ? 'active' : ''}" 
              on:click={() => handleSwitch(acc.id)}
              on:keydown={(e) => e.key === 'Enter' && handleSwitch(acc.id)}
              role="button"
              tabindex="0"
            >
              <div class="avatar-small">
                {#if acc.type === 'microsoft'}
                  <img 
                    src={`https://mc-heads.net/avatar/${encodeURIComponent(acc.username)}/24`} 
                    alt={acc.username} 
                    class="avatar-img"
                    on:error={handleImageError}
                  />
                {/if}
                <span class="avatar-fallback">{acc.username.charAt(0).toUpperCase()}</span>
              </div>
              <div class="acc-details">
                <span class="acc-username">{acc.username}</span>
                <span class="acc-type">{acc.type === 'microsoft' ? 'Microsoft' : 'Offline'}</span>
              </div>
              {#if activeAccount && activeAccount.id === acc.id}
                <div class="active-indicator"></div>
              {/if}
              <button 
                class="logout-btn" 
                on:click={(e) => handleLogout(acc.id, e)} 
                title="Remove Account"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
              </button>
            </div>
          {/each}
        {/if}
      </div>
      
      <div class="dropdown-actions">
        <button class="add-account-btn" on:click={() => { showLoginModal = true; showDropdown = false; }}>
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          Add Account
        </button>
      </div>
    </div>
  {/if}

  <div class="account-section-header">Playing as</div>

  <button 
    class="account-card {showDropdown ? 'open' : ''}" 
    on:click={() => {
      if (activeAccount) {
        showDropdown = !showDropdown;
      } else {
        showLoginModal = true;
      }
    }}
    type="button"
  >
    <div class="account-info">
      <div class="avatar">
        {#if activeAccount}
          {#if activeAccount.type === 'microsoft' && !avatarError}
            <img 
              src={`https://mc-heads.net/avatar/${encodeURIComponent(activeAccount.username)}/36`} 
              alt={activeAccount.username} 
              class="avatar-img"
              on:error={() => { avatarError = true; }}
            />
          {:else}
            <span class="avatar-fallback">{activeAccount.username.charAt(0).toUpperCase()}</span>
          {/if}
        {:else}
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
            <circle cx="12" cy="7" r="4"></circle>
          </svg>
        {/if}
      </div>
      <div class="details">
        {#if activeAccount}
          <span class="username">{activeAccount.username}</span>
          <span class="status">{activeAccount.type === 'microsoft' ? 'Microsoft Account' : 'Offline Account'}</span>
        {:else}
          <span class="username">Guest</span>
          <span class="status">Click to login</span>
        {/if}
      </div>
    </div>

    <div class="chevron-icon {showDropdown ? 'rotated' : ''}">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <polyline points="6 9 12 15 18 9"></polyline>
      </svg>
    </div>
  </button>
</div>

<LoginModal bind:showModal={showLoginModal} on:login={handleLogin} />

<style>
  .account-manager-wrapper {
    position: relative;
    margin-top: 12px;
  }

  .account-section-header {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 6px;
    padding-left: 2px;
    letter-spacing: -0.2px;
  }

  .account-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 8px 10px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: var(--border-radius);
    cursor: pointer;
    text-align: left;
    font-family: inherit;
    transition: background var(--transition-fast), border-color var(--transition-fast);
    box-sizing: border-box;
    gap: 8px;
  }

  .account-card:hover, .account-card.open {
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.12);
  }

  .account-info {
    display: flex;
    align-items: center;
    gap: 10px;
    overflow: hidden;
    flex: 1;
    min-width: 0;
  }

  .avatar {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.12);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-primary);
    font-weight: 600;
    font-size: 14px;
    flex-shrink: 0;
    overflow: hidden;
    position: relative;
  }

  .avatar-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    image-rendering: pixelated;
    display: block;
  }

  .avatar-fallback {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
  }

  .avatar svg {
    width: 18px;
    height: 18px;
  }

  .details {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
    flex: 1;
    min-width: 0;
  }

  .username {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.2;
  }

  .status {
    font-size: 11px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    line-height: 1.2;
  }

  .chevron-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
    flex-shrink: 0;
    transition: transform var(--transition-fast), color var(--transition-fast);
  }

  .account-card:hover .chevron-icon {
    color: var(--text-primary);
  }

  .chevron-icon.rotated {
    transform: rotate(180deg);
  }

  .chevron-icon svg {
    width: 16px;
    height: 16px;
  }

  /* Dropdown Styles */
  .accounts-dropdown {
    position: absolute;
    bottom: calc(100% + 8px);
    left: 0;
    right: 0;
    background: rgba(26, 26, 26, 0.95);
    backdrop-filter: blur(20px);
    -webkit-backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: var(--border-radius);
    box-shadow: 0 -10px 25px rgba(0, 0, 0, 0.5), 0 10px 25px rgba(0, 0, 0, 0.5);
    z-index: 1000;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: slideUp var(--transition-fast) cubic-bezier(0.16, 1, 0.3, 1);
  }

  @keyframes slideUp {
    from {
      transform: translateY(8px);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }

  .dropdown-header {
    padding: 10px 12px;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  }

  .accounts-list {
    max-height: 200px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  .no-accounts {
    padding: 16px;
    text-align: center;
    color: var(--text-secondary);
    font-size: 12px;
  }

  .account-item {
    display: flex;
    align-items: center;
    padding: 8px 12px;
    gap: 12px;
    cursor: pointer;
    position: relative;
    transition: background var(--transition-fast);
  }

  .account-item:hover {
    background: rgba(255, 255, 255, 0.05);
  }

  .account-item.active {
    background: rgba(255, 255, 255, 0.02);
  }

  .avatar-small {
    width: 24px;
    height: 24px;
    border-radius: 5px;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-primary);
    font-weight: 600;
    font-size: 11px;
    flex-shrink: 0;
    overflow: hidden;
    position: relative;
  }

  .acc-details {
    display: flex;
    flex-direction: column;
    overflow: hidden;
    flex-grow: 1;
  }

  .acc-username {
    font-size: 12px;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .acc-type {
    font-size: 10px;
    color: var(--text-secondary);
  }

  .active-indicator {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #00ff66;
    margin-right: 4px;
    flex-shrink: 0;
  }

  .logout-btn {
    background: transparent;
    border: none;
    padding: 6px;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: 4px;
    opacity: 0.3;
    transition: all var(--transition-fast);
    flex-shrink: 0;
  }

  .account-item:hover .logout-btn {
    opacity: 1;
  }

  .logout-btn:hover {
    background: rgba(255, 85, 85, 0.15);
    color: #ff5555;
  }

  .logout-btn svg {
    width: 14px;
    height: 14px;
  }

  .dropdown-actions {
    padding: 8px 12px;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
    background: rgba(0, 0, 0, 0.1);
  }

  .add-account-btn {
    width: 100%;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.05);
    color: var(--text-primary);
    padding: 6px;
    border-radius: var(--border-radius);
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    transition: all var(--transition-fast);
  }

  .add-account-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.1);
  }

  .add-account-btn svg {
    width: 12px;
    height: 12px;
  }
</style>
