import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import type { HostNodeData, Severity } from '@/types';
import { SEVERITY_BORDER, SEVERITY_RING_COLOR } from '@/utils/colors';

const OS_ICON: Record<string, string> = {
  linux:   '🐧',
  windows: '🪟',
};

function healthBorder(sev: Severity | undefined): string {
  if (!sev) return 'border-slate-600';
  return SEVERITY_BORDER[sev];
}

export const HostNode = memo(function HostNode({ data }: NodeProps) {
  const { host, anomalySeverity } = data as unknown as HostNodeData;

  const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;

  return (
    <div
      className={`
        relative w-60 rounded-xl border-2 bg-card px-4 py-3 shadow-lg
        ${healthBorder(anomalySeverity)}
        transition-all duration-300
      `}
      style={ringColor ? ({ '--ring-color': ringColor } as React.CSSProperties) : undefined}
    >
      {/* Pulsing ring overlay — only rendered when anomaly is active */}
      {anomalySeverity && (
        <span
          className="absolute inset-0 rounded-xl animate-pulse-ring pointer-events-none"
          aria-hidden
        />
      )}

      {/* Header row */}
      <div className="flex items-center gap-2 mb-2">
        <span className="text-xl leading-none" role="img" aria-label={host.os_family}>
          {OS_ICON[host.os_family] ?? '💻'}
        </span>
        <div className="flex-1 min-w-0">
          <p className="truncate font-semibold text-slate-100 text-sm leading-tight">
            {host.hostname}
          </p>
          <p className="text-xs text-muted">{host.os_family} · {host.arch}</p>
        </div>
        {/* Health indicator dot */}
        <span
          className={`h-2.5 w-2.5 rounded-full flex-shrink-0 ${
            anomalySeverity === 'CRITICAL' ? 'bg-red-500' :
            anomalySeverity === 'HIGH'     ? 'bg-orange-500' :
            anomalySeverity === 'MEDIUM'   ? 'bg-yellow-400' :
            anomalySeverity === 'LOW'      ? 'bg-blue-400' :
                                             'bg-emerald-400'
          }`}
        />
      </div>

      {/* IP */}
      {host.ip_primary && (
        <p className="text-xs text-muted font-mono truncate">{host.ip_primary}</p>
      )}

      {/* Last seen */}
      {host.last_seen_at && (
        <p className="text-xs text-muted mt-0.5">
          seen {new Date(host.last_seen_at).toLocaleTimeString()}
        </p>
      )}

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-slate-500 !border-slate-700"
      />
    </div>
  );
});
