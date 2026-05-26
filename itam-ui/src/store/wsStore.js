import { create } from 'zustand';
export const useWsStore = create((set) => ({
    status: 'disconnected',
    lastPingAt: null,
    reconnectAttempt: 0,
    setStatus: (status) => set({ status }),
    ping: () => set({ lastPingAt: new Date() }),
    incrementAttempt: () => set((s) => ({ reconnectAttempt: s.reconnectAttempt + 1 })),
    resetAttempts: () => set({ reconnectAttempt: 0 }),
}));
