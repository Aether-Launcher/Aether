<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { LoginOffline, StartMicrosoftAuth } from '../../wailsjs/go/main/App';

  export let showModal = false;

  const dispatch = createEventDispatcher();

  let username = '';
  let isLoading = false;
  let isMsLoading = false;
  let errorMsg = '';

  async function handleMicrosoftLogin() {
    isMsLoading = true;
    errorMsg = '';
    try {
      const account = await StartMicrosoftAuth();
      dispatch('login', account);
      showModal = false;
    } catch (err: any) {
      const raw = err?.toString() || 'Microsoft login failed';
      if (raw.includes('No internet') || raw.includes('no such host') || raw.includes('ERR_NET') || raw.includes('ERR_NET_LISTEN')) {
        errorMsg = 'No internet connection. Please check your network and try again, or use Offline Login below.';
      } else if (raw.includes('ERR_NO_GAME')) {
        errorMsg = 'This Microsoft account does not own Minecraft Java Edition.';
      } else if (raw.includes('ERR_NO_XBOX')) {
        errorMsg = 'This Microsoft account has no Xbox profile. Create one at xbox.com first.';
      } else {
        // Strip Go prefix like "ERR_NET: ..." for cleaner display
        errorMsg = raw.replace(/^[A-Z_]+:\s*/, '');
        if (errorMsg.length > 180) errorMsg = errorMsg.slice(0, 180) + '…';
      }
    } finally {
      isMsLoading = false;
    }
  }
  
  async function handleLogin() {
    if (!username.trim()) {
      errorMsg = 'Username cannot be empty';
      return;
    }
    
    isLoading = true;
    errorMsg = '';
    
    try {
      const account = await LoginOffline(username.trim());
      dispatch('login', account);
      showModal = false;
      username = '';
    } catch (err: any) {
      errorMsg = err?.toString() || 'Offline login failed';
    } finally {
      isLoading = false;
    }
  }
</script>

{#if showModal}
  <div class="modal-backdrop" on:click={() => { if(!isMsLoading && !isLoading) showModal = false; }} on:keydown={(e) => e.key === 'Escape' && !isMsLoading && !isLoading && (showModal = false)} role="button" tabindex="0">
    <div class="modal" on:click|stopPropagation on:keydown|stopPropagation role="button" tabindex="0">
      
      <h2>Sign In</h2>
      <p class="subtitle">Log in to play Minecraft.</p>
      
      <button class="btn primary btn-microsoft" on:click={handleMicrosoftLogin} disabled={isMsLoading || isLoading}>
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 21 21">
          <rect x="1" y="1" width="9" height="9" fill="#f35325"/>
          <rect x="11" y="1" width="9" height="9" fill="#81bc06"/>
          <rect x="1" y="11" width="9" height="9" fill="#05a6f0"/>
          <rect x="11" y="11" width="9" height="9" fill="#ffba08"/>
        </svg>
        {isMsLoading ? 'Authenticating in browser...' : 'Sign in with Microsoft'}
      </button>

      <div class="divider">
        <span>or</span>
      </div>

      <div class="form-group">
        <label for="username">Offline Username</label>
        <input 
          id="username"
          type="text" 
          bind:value={username} 
          placeholder="e.g. Notch" 
          on:keydown={(e) => e.key === 'Enter' && handleLogin()}
          disabled={isMsLoading || isLoading}
        />
        {#if errorMsg}
          <span class="error">{errorMsg}</span>
        {/if}
      </div>

      <div class="modal-actions">
        <button class="btn secondary" on:click={() => showModal = false} disabled={isMsLoading || isLoading}>Cancel</button>
        <button class="btn primary" on:click={handleLogin} disabled={isMsLoading || isLoading || !username.trim()}>
          {isLoading ? 'Logging in...' : 'Offline Login'}
        </button>
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
    background: rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: rgba(30, 30, 30, 0.6);
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border: 1px solid rgba(255, 255, 255, 0.1);
    padding: 32px;
    border-radius: 16px;
    width: 400px;
    box-shadow: 0 24px 48px rgba(0, 0, 0, 0.4);
    color: var(--text-primary);
  }

  h2 {
    margin: 0 0 8px 0;
    font-size: 24px;
    font-weight: 600;
  }

  .subtitle {
    margin: 0 0 24px 0;
    color: var(--text-secondary);
    font-size: 14px;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 24px;
  }

  label {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-secondary);
  }

  input {
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: var(--text-primary);
    padding: 12px 16px;
    border-radius: var(--border-radius);
    font-size: 14px;
    font-family: inherit;
    transition: border-color var(--transition-fast);
  }

  input:focus {
    outline: none;
    border-color: var(--accent-color);
  }

  .error {
    color: #ff5555;
    font-size: 12px;
    margin-top: 4px;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .btn {
    padding: 10px 20px;
    border-radius: var(--border-radius);
    font-size: 14px;
    font-weight: 500;
    font-family: inherit;
    cursor: pointer;
    border: none;
    transition: all var(--transition-fast);
  }

  .btn.secondary {
    background: transparent;
    color: var(--text-secondary);
  }

  .btn.secondary:hover {
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-primary);
  }

  .btn.primary {
    background: var(--accent-color);
    color: white;
  }

  .btn.primary:hover {
    filter: brightness(1.1);
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-microsoft {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    background: white;
    color: #333;
    font-weight: 600;
    margin-bottom: 24px;
    box-shadow: 0 2px 8px rgba(0,0,0,0.2);
  }

  .btn-microsoft:hover:not(:disabled) {
    background: #f0f0f0;
    filter: none;
  }

  .divider {
    display: flex;
    align-items: center;
    text-align: center;
    margin-bottom: 24px;
    color: var(--text-secondary);
    font-size: 13px;
  }
  
  .divider::before,
  .divider::after {
    content: '';
    flex: 1;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }
  
  .divider span {
    padding: 0 16px;
  }
</style>
