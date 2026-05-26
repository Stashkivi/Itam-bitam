import { useGraphStore } from '@/store/graphStore';
import { useHosts }      from '@/api/hosts';
import type { Host }     from '@/types';

const OS_ICON: Record<string, string> = { linux: '🐧', windows: '🪟' };

function seenAgo(ts: string | null): string {
  if (!ts) return 'never';
  const diffMs = Date.now() - new Date(ts).getTime();
  const mins   = Math.floor(diffMs / 60_000);
  if (mins < 1)   return 'just now';
  if (mins < 60)  return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24)   return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

function HostRow({ host, selected }: { host: Host; selected: boolean }) {
  const selectHost = useGraphStore((s) => s.selectHost);
  const anomalies  = useGraphStore((s) => s.anomaliesByNode[`host:${host.id}`] ?? []);

  const hasCritical = anomalies.some((a) => a.severity === 'CRITICAL');
  const hasHigh     = anomalies.some((a) => a.severity === 'HIGH');

  return (
    <button
      onClick={() => selectHost(selected ? null : host.id)}
      className={`
        w-full text-left rounded-lg px-3 py-2.5 transition-colors
        ${selected
          ? 'bg-blue-600/20 border border-blue-500/50'
          : 'hover:bg-card border border-transparent'}
      `}
    >
      <div className="flex items-center gap-2">
        <span className="text-base leading-none flex-shrink-0">
          {OS_ICON[host.os_family] ?? '💻'}
        </span>
        <div className="flex-1 min-w-0">
          <p className="truncate text-sm font-medium text-slate-200">{host.hostname}</p>
          <p className="text-xs text-muted">{seenAgo(host.last_seen_at)}</p>
        </div>
        {/* Anomaly indicator */}
        {(hasCritical || hasHigh) && (
          <span
            className={`h-2 w-2 rounded-full flex-shrink-0 ${
              hasCritical ? 'bg-red-500 animate-pulse' : 'bg-orange-400'
            }`}
          />
        )}
      </div>
    </button>
  );
}

export function HostList() {
  const { data: hosts = [], isLoading, isError } = useHosts();
  const selectedHostId = useGraphStore((s) => s.selectedHostId);

  return (
    <nav className="flex h-full w-56 flex-col border-r border-border bg-card/30">
      <header className="border-b border-border px-4 py-3">
        <h2 className="text-xs font-semibold uppercase tracking-widest text-muted">
          Hosts
        </h2>
      </header>
      <div className="flex-1 overflow-y-auto px-2 py-2 space-y-0.5">
        {isLoading && (
          <p className="text-xs text-muted text-center py-6">Loading hosts…</p>
        )}
        {isError && (
          <p className="text-xs text-red-400 text-center py-6">Failed to load hosts</p>
        )}
        {hosts.map((h) => (
          <HostRow key={h.id} host={h} selected={h.id === selectedHostId} />
        ))}
        {!isLoading && !hosts.length && (
          <p className="text-xs text-muted text-center py-6">No enrolled hosts</p>
        )}
      </div>
    </nav>
  );
}
