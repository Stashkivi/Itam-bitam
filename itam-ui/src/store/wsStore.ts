import { create } from 'zustand';

export type WsStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

interface WsState {
  status: WsStatus;
  lastPingAt: Date | null;
  reconnectAttempt: number;
  setStatus: (status: WsStatus) => void;
  ping: () => void;
  incrementAttempt: () => void;
  resetAttempts: () => void;
}

export const useWsStore = create<WsState>((set) => ({
  status:           'disconnected',
  lastPingAt:       null,
  reconnectAttempt: 0,

  setStatus:       (status) => set({ status }),
  ping:            () => set({ lastPingAt: new Date() }),
  incrementAttempt:() => set((s) => ({ reconnectAttempt: s.reconnectAttempt + 1 })),
  resetAttempts:   () => set({ reconnectAttempt: 0 }),
}));
