import { writable, get } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';

export type InstallStatus = 'idle' | 'installing' | 'complete' | 'error';

export interface InstallState {
  status: InstallStatus;
  progress: number;
  statusText: string;
  error: string;
}

export const EMPTY_INSTALL_STATE: InstallState = {
  status: 'idle',
  progress: 0,
  statusText: '',
  error: '',
};

const { subscribe, update } = writable<Record<string, InstallState>>({});

export const installStore = {
  subscribe,

  start(id: string) {
    update(all => ({
      ...all,
      [id]: {
        status: 'installing',
        progress: 0,
        statusText: 'Installing...',
        error: '',
      },
    }));
  },

  fail(id: string, message: string) {
    update(all => ({
      ...all,
      [id]: {
        ...(all[id] || EMPTY_INSTALL_STATE),
        status: 'error',
        statusText: 'Error',
        error: message || 'Installation failed.',
      },
    }));
  },
};

export function getInstallState(id: string): InstallState {
  return get(installStore)[id] || EMPTY_INSTALL_STATE;
}

// Keep install state global so navigating away from Home does not lose the
// progress bar or make a remount start a second install pipeline.
EventsOn('instance:progress', (data: { id?: string; progress?: number; status?: string }) => {
  const id = data?.id;
  if (!id) return;

  const progress = typeof data.progress === 'number' ? data.progress : 0;
  const complete = progress >= 100;

  update(all => {
    const prev = all[id] || EMPTY_INSTALL_STATE;
    return {
      ...all,
      [id]: {
        ...prev,
        status: complete ? 'complete' : 'installing',
        progress,
        statusText: complete ? '' : (data.status || prev.statusText),
        error: complete ? '' : prev.error,
      },
    };
  });
});

EventsOn('instance:error', (data: { id?: string; message?: string }) => {
  const id = data?.id;
  if (!id) return;
  installStore.fail(id, data.message || 'Installation failed.');
});
