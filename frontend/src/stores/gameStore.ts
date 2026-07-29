import { writable } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';

export interface RunningState {
  instanceId: string | null;
  state: 'Idle' | 'Starting...' | 'Running' | 'Stopped' | 'Crashed' | 'Error' | string;
  logs: string[];
}

function createGameStore() {
  const { subscribe, update, set } = writable<RunningState>({
    instanceId: null,
    state: 'Idle',
    logs: [],
  });

  // Wire up global Wails events once — these outlive any single page
  EventsOn('instance:state', (data: { id: string; state: string }) => {
    update(s => ({
      ...s,
      instanceId: data.id,
      state: data.state,
      // Clear logs when stopped/crashed so they don't bleed into the next session
      logs: data.state === 'Stopped' || data.state === 'Crashed' ? [] : s.logs,
    }));
  });

  EventsOn('instance:log', (line: string) => {
    update(s => ({ ...s, logs: [...s.logs, line].slice(-200) }));
  });

  return { subscribe, update, set };
}

export const gameStore = createGameStore();
