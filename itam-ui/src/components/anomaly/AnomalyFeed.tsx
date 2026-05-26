import { useGraphStore } from '@/store/graphStore';
import { useHostAnomalies } from '@/api/hosts';
import { SEVERITY_TEXT, SEVERITY_BG, SEVERITY_BORDER } from '@/utils/colors';
import type { Anomaly, Severity } from '@/types';

function SeverityBadge({ severity }: { severity: Severity }) {
  return (
    <span
      className={`
        inline-flex items-center rounded px-1.5 py-0.5
        text-xs font-bold uppercase tracking-wide
        ${SEVERITY_TEXT[severity]} ${SEVERITY_BG[severity]}
      `}
    >
      {severity}
    </span>
  );
}

function AnomalyRow({ anomaly }: { anomaly: Anomaly }) {
  const ts = new Date(anomaly.detected_at);

  return (
    <div className={`rounded-lg border p-3 ${SEVERITY_BORDER[anomaly.severity]} bg-card/60`}>
      <div className="flex items-start justify-between gap-2 mb-1">
        <SeverityBadge severity={anomaly.severity} />
        <span className="text-xs text-muted tabular-nums">
          {ts.toLocaleTimeString()}
        </span>
      </div>
      <p className="text-xs font-mono text-slate-300 font-medium">{anomaly.rule_id}</p>
      <p className="text-xs text-muted mt-0.5">
        {anomaly.entity_type}:{' '}
        <span className="text-slate-300 font-mono">{anomaly.entity_key}</span>
      </p>
      {anomaly.details && (
        <details className="mt-1.5">
          <summary className="text-xs text-muted cursor-pointer select-none hover:text-slate-400">
            details
          </summary>
          <pre className="mt-1 text-xs text-slate-400 bg-surface rounded p-2 overflow-x-auto">
            {JSON.stringify(anomaly.details, null, 2)}
          </pre>
        </details>
      )}
    </div>
  );
}

export function AnomalyFeed() {
  const selectedHostId = useGraphStore((s) => s.selectedHostId);
  const liveToasts     = useGraphStore((s) => s.toasts);

  const { data: serverAnomalies = [], isLoading } = useHostAnomalies(selectedHostId);

  // Merge live WS events with server-fetched list, dedup by id
  const merged = [
    ...liveToasts.filter((a) => a.host_uuid === selectedHostId),
    ...serverAnomalies,
  ].reduce<Anomaly[]>((acc, a) => {
    if (!acc.find((x) => x.id === a.id)) acc.push(a);
    return acc;
  }, []);

  return (
    <aside className="flex h-full w-72 flex-col border-l border-border bg-card/30">
      <header className="flex items-center justify-between border-b border-border px-4 py-3">
        <h2 className="text-sm font-semibold text-slate-200">Anomaly Feed</h2>
        {merged.length > 0 && (
          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs font-bold text-white">
            {merged.length > 99 ? '99+' : merged.length}
          </span>
        )}
      </header>

      <div className="flex-1 overflow-y-auto p-3 space-y-2">
        {isLoading && (
          <p className="text-xs text-muted text-center py-4">Loading…</p>
        )}
        {!isLoading && !selectedHostId && (
          <p className="text-xs text-muted text-center py-4">
            Select a host to see its anomalies
          </p>
        )}
        {!isLoading && selectedHostId && merged.length === 0 && (
          <div className="text-center py-6">
            <p className="text-2xl mb-1">✓</p>
            <p className="text-xs text-muted">No active anomalies</p>
          </div>
        )}
        {merged
          .sort((a, b) => new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime())
          .map((a) => (
            <AnomalyRow key={`${a.id}-${a.detected_at}`} anomaly={a} />
          ))}
      </div>
    </aside>
  );
}
