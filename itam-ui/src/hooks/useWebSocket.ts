import { useEffect, useRef, useCallback } from 'react';
import { useGraphStore } from '@/store/graphStore';
import { useWsStore } from '@/store/wsStore';
import type { WsMessage } from '@/types';

const MAX_RECONNECT_DELAY_MS = 30_000;
const BASE_DELAY_MS = 1_000;

export function useWebSocket(token: string | null) {
  const wsRef      = useRef<WebSocket | null>(null);
  const timerRef   = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);

  const { setStatus, ping, resetAttempts } = useWsStore();
  const applyAnomaly = useGraphStore((s) => s.applyAnomaly);

  const connect = useCallback(() => {
    if (!token) return;

    setStatus('connecting');

    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${protocol}://${window.location.host}/v1/ws`;

    const ws = new WebSocket(url, ['Bearer', token]);
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus('connected');
      resetAttempts();
      attemptRef.current = 0;
    };

    ws.onmessage = (event: MessageEvent) => {
      try {
        const msg: WsMessage = JSON.parse(event.data as string);
        if (msg.event === 'ping') {
          ping();
          ws.send(JSON.stringify({ event: 'pong' }));
          return;
        }
        if (msg.event === 'anomaly.detected' && msg.payload) {
          applyAnomaly(msg.payload);
        }
      } catch {
        // Malformed frame — ignore
      }
    };

    ws.onclose = () => {
      setStatus('disconnected');
      wsRef.current = null;
      scheduleReconnect();
    };

    ws.onerror = () => {
      // onclose always fires after onerror — reconnect is handled there
      setStatus('error');
    };
  }, [token, setStatus, ping, resetAttempts, applyAnomaly]);

  const scheduleReconnect = useCallback(() => {
    const delay = Math.min(
      BASE_DELAY_MS * 2 ** attemptRef.current,
      MAX_RECONNECT_DELAY_MS,
    );
    // Add ±20% jitter to spread reconnection storms
    const jitter = delay * 0.2 * (Math.random() * 2 - 1);
    attemptRef.current += 1;

    timerRef.current = setTimeout(connect, delay + jitter);
  }, [connect]);

  useEffect(() => {
    connect();
    return () => {
      timerRef.current && clearTimeout(timerRef.current);
      wsRef.current?.close();
    };
  }, [connect]);
}
