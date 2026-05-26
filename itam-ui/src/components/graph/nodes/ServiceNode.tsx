import { memo } from 'react';
import { Handle, Position, type NodeProps } from '@xyflow/react';
import type { ServiceNodeData } from '@/types';
import {
  STATUS_DOT, SEVERITY_BORDER, SEVERITY_RING_COLOR,
  serviceAbbrev, serviceColor,
} from '@/utils/colors';

export const ServiceNode = memo(function ServiceNode({ data }: NodeProps) {
  const { service, anomalySeverity } = data as unknown as ServiceNodeData;

  const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;

  return (
    <div
      className={`
        relative w-48 rounded-lg border bg-card px-3 py-2.5 shadow-md
        ${anomalySeverity ? SEVERITY_BORDER[anomalySeverity] : 'border-slate-600'}
        transition-all duration-300
      `}
      style={ringColor ? ({ '--ring-color': ringColor } as React.CSSProperties) : undefined}
    >
      {anomalySeverity && (
        <span className="absolute inset-0 rounded-lg animate-pulse-ring pointer-events-none" aria-hidden />
      )}

      <Handle type="target" position={Position.Top}
        className="!bg-slate-500 !border-slate-700" />

      <div className="flex items-center gap-2">
        {/* Service icon */}
        <span
          className={`
            flex h-8 w-8 flex-shrink-0 items-center justify-center
            rounded-md text-xs font-bold text-white
            ${serviceColor(service.name)}
          `}
        >
          {serviceAbbrev(service.name)}
        </span>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5">
            {/* Status dot */}
            <span
              className={`h-2 w-2 rounded-full flex-shrink-0 ${STATUS_DOT[service.status] ?? 'bg-slate-600'}`}
            />
            <p className="truncate text-sm font-medium text-slate-100">
              {service.name}
            </p>
          </div>
          <div className="flex items-center gap-1.5 mt-0.5">
            <span
              className={`text-xs ${
                service.status === 'failed'  ? 'text-red-400' :
                service.status === 'running' ? 'text-emerald-400' :
                'text-muted'
              }`}
            >
              {service.status}
            </span>
            {service.version && (
              <span className="text-xs text-muted font-mono">v{service.version}</span>
            )}
          </div>
        </div>
      </div>

      <Handle type="source" position={Position.Bottom}
        className="!bg-slate-500 !border-slate-700" />
    </div>
  );
});
