import { useWsStore }       from '@/store/wsStore';
import { useGraphStore }    from '@/store/graphStore';
import { useDependencyMap } from '@/api/hosts';
import { useWebSocket }     from '@/hooks/useWebSocket';
import { getToken }         from '@/api/client';

import { HostList }              from './HostList';
import { DependencyGraph }       from '@/components/graph/DependencyGraph';
import { AnomalyFeed }           from '@/components/anomaly/AnomalyFeed';
import { AnomalyToastContainer } from '@/components/anomaly/AnomalyToast';

function WsStatusPill() {
  const status = useWsStore((s) => s.status);

  const label: Record<typeof status, string> = {
    connected:    'Live',
    connecting:   'Connecting…',
    disconnected: 'Offline',
    error:        'Error',
  };
  const dot: Record<typeof status, string> = {
    connected:    'bg-emerald-400',
    connecting:   'bg-yellow-400 animate-pulse',
    disconnected: 'bg-slate-500',
    error:        'bg-red-500',
  };

  return (
    <div className="flex items-center gap-1.5">
      <span className={`h-1.5 w-1.5 rounded-full ${dot[status]}`} />
      <span className="text-xs text-muted">{label[status]}</span>
    </div>
  );
}

export function AppShell() {
  const token          = getToken();
  const selectedHostId = useGraphStore((s) => s.selectedHostId);

  // Start WebSocket connection on mount
  useWebSocket(token);

  const { data: depMap, isLoading } = useDependencyMap(selectedHostId);

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-surface text-slate-100">
      {/* ── Top bar ──────────────────────────────────────────────────────── */}
      <header className="flex h-12 flex-shrink-0 items-center justify-between border-b border-border bg-card/60 px-6">
        <div className="flex items-center gap-3">
          <span className="text-lg font-bold tracking-tight text-slate-100">ITAM</span>
          <span className="h-4 w-px bg-border" />
          <span className="text-sm text-muted">Dependency Map</span>
        </div>
        <WsStatusPill />
      </header>

      {/* ── Main content ─────────────────────────────────────────────────── */}
      <div className="flex flex-1 overflow-hidden">
        {/* Host list sidebar */}
        <HostList />

        {/* Graph canvas — fills all remaining horizontal space */}
        <main className="relative flex-1 overflow-hidden">
          <DependencyGraph map={depMap} isLoading={isLoading} />
        </main>

        {/* Anomaly feed sidebar */}
        <AnomalyFeed />
      </div>

      {/* Toast overlay */}
      <AnomalyToastContainer />
    </div>
  );
}
