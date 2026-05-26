import { useEffect } from 'react';
import { useGraphStore } from '@/store/graphStore';
import { SEVERITY_BG, SEVERITY_TEXT } from '@/utils/colors';
import type { Anomaly } from '@/types';

const AUTO_DISMISS_MS = 6_000;

function Toast({ anomaly, onDismiss }: { anomaly: Anomaly; onDismiss: () => void }) {
  useEffect(() => {
    if (anomaly.severity === 'LOW' || anomaly.severity === 'MEDIUM') {
      const t = setTimeout(onDismiss, AUTO_DISMISS_MS);
      return () => clearTimeout(t);
    }
  }, [anomaly.severity, onDismiss]);

  return (
    <div
      className={`
        flex items-start gap-3 rounded-lg border border-slate-700 shadow-xl
        bg-card px-4 py-3 animate-slide-in w-80
        ${SEVERITY_BG[anomaly.severity]}
      `}
      role="alert"
    >
      {/* Severity stripe */}
      <div
        className={`mt-0.5 h-4 w-1 flex-shrink-0 rounded-full ${
          anomaly.severity === 'CRITICAL' ? 'bg-red-500' :
          anomaly.severity === 'HIGH'     ? 'bg-orange-500' :
          anomaly.severity === 'MEDIUM'   ? 'bg-yellow-400' :
                                            'bg-blue-400'
        }`}
      />

      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <span className={`text-xs font-bold uppercase tracking-wide ${SEVERITY_TEXT[anomaly.severity]}`}>
            {anomaly.severity}
          </span>
          <span className="text-xs text-muted">{anomaly.rule_id}</span>
        </div>
        <p className="mt-0.5 text-sm font-medium text-slate-100 truncate">
          {anomaly.hostname}
        </p>
        <p className="text-xs text-muted truncate">
          {anomaly.entity_type}: <span className="font-mono">{anomaly.entity_key}</span>
        </p>
      </div>

      <button
        onClick={onDismiss}
        className="text-muted hover:text-slate-300 text-lg leading-none flex-shrink-0"
        aria-label="Dismiss"
      >
        ×
      </button>
    </div>
  );
}

export function AnomalyToastContainer() {
  const toasts       = useGraphStore((s) => s.toasts);
  const dismissToast = useGraphStore((s) => s.dismissToast);

  if (!toasts.length) return null;

  return (
    <div className="fixed bottom-6 right-6 z-50 flex flex-col gap-2">
      {toasts.map((a) => (
        <Toast
          key={a.id}
          anomaly={a}
          onDismiss={() => dismissToast(a.id)}
        />
      ))}
    </div>
  );
}
