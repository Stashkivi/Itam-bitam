import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import type { PortNodeData } from '@/types';
import { SEVERITY_BORDER, SEVERITY_RING_COLOR, portLabel, isExposedBind } from '@/utils/colors';

export const PortNode = memo(function PortNode({ data }: NodeProps) {
  const { port, isApproved, anomalySeverity } = data as unknown as PortNodeData;

  const exposed   = isExposedBind(port.bind_addr);
  const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;

  const borderClass = anomalySeverity
    ? SEVERITY_BORDER[anomalySeverity]
    : isApproved
      ? 'border-slate-700'
      : 'border-amber-500';   // amber = not in baseline but no anomaly rule yet

  return (
    <div
      className={`
        relative w-36 rounded-md border bg-card px-3 py-2 shadow-sm
        ${borderClass}
        transition-all duration-200
      `}
      style={ringColor ? ({ '--ring-color': ringColor } as React.CSSProperties) : undefined}
    >
      {anomalySeverity && (
        <span className="absolute inset-0 rounded-md animate-pulse-ring pointer-events-none" aria-hidden />
      )}

      <Handle type="target" position={Position.Top}
        className="!bg-slate-600 !border-slate-700 !w-2 !h-2" />

      <div className="flex items-center gap-2">
        {/* Bind scope indicator */}
        <span
          className={`text-sm leading-none ${exposed ? 'text-orange-400' : 'text-slate-400'}`}
          title={exposed ? `Exposed on ${port.bind_addr}` : `Local: ${port.bind_addr}`}
        >
          {exposed ? '🌐' : '🔒'}
        </span>

        <div className="flex-1 min-w-0">
          <div className="flex items-baseline gap-1">
            <span className="font-mono font-bold text-slate-100 text-sm">
              {portLabel(port.port)}
            </span>
            <span className="text-xs text-muted uppercase">{port.protocol}</span>
          </div>
          {port.port !== parseInt(portLabel(port.port), 10) && (
            <p className="text-xs text-muted font-mono">{port.port}</p>
          )}
        </div>
      </div>

      {/* Unapproved badge */}
      {!isApproved && !anomalySeverity && (
        <p className="mt-1 text-xs text-amber-400 font-medium">⚠ unknown</p>
      )}
    </div>
  );
});
