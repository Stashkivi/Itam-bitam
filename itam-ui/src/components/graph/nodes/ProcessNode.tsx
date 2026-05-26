import { memo, useCallback } from 'react';
import { Handle, Position, type NodeProps, useReactFlow } from '@xyflow/react';
import type { ProcessNodeData } from '@/types';
import { SEVERITY_BORDER, SEVERITY_RING_COLOR } from '@/utils/colors';

function cpuColor(pct: number): string {
  if (pct > 80) return 'text-red-400';
  if (pct > 50) return 'text-yellow-400';
  return 'text-emerald-400';
}

export const ProcessNode = memo(function ProcessNode({ id, data }: NodeProps) {
  const { process, collapsed, anomalySeverity } = data as unknown as ProcessNodeData;
  const { updateNodeData } = useReactFlow();

  const toggle = useCallback(() => {
    updateNodeData(id, { collapsed: !collapsed });
  }, [id, collapsed, updateNodeData]);

  const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;

  return (
    <div
      className={`
        relative w-52 rounded-lg border bg-card shadow-sm cursor-pointer
        ${anomalySeverity ? SEVERITY_BORDER[anomalySeverity] : 'border-slate-700'}
        hover:border-slate-500 transition-all duration-200
      `}
      style={ringColor ? ({ '--ring-color': ringColor } as React.CSSProperties) : undefined}
      onClick={toggle}
    >
      {anomalySeverity && (
        <span className="absolute inset-0 rounded-lg animate-pulse-ring pointer-events-none" aria-hidden />
      )}

      <Handle type="target" position={Position.Top}
        className="!bg-slate-600 !border-slate-700 !w-2 !h-2" />

      {/* Always-visible summary row */}
      <div className="flex items-center gap-2 px-3 py-2">
        <span className="text-xs font-mono text-muted w-12 flex-shrink-0">
          {process.pid}
        </span>
        <p className="flex-1 truncate text-xs font-medium text-slate-200">
          {process.name}
        </p>
        <span className={`text-xs font-mono ${cpuColor(process.cpu_pct)}`}>
          {process.cpu_pct.toFixed(1)}%
        </span>
        {/* Expand indicator */}
        <span className="text-muted text-xs ml-1">{collapsed ? '›' : '‹'}</span>
      </div>

      {/* Expanded details */}
      {!collapsed && (
        <div className="border-t border-slate-700 px-3 py-2 space-y-1">
          <div className="flex justify-between text-xs">
            <span className="text-muted">user</span>
            <span className="text-slate-300 font-mono">{process.username || '—'}</span>
          </div>
          <div className="flex justify-between text-xs">
            <span className="text-muted">RSS</span>
            <span className="text-slate-300">{process.mem_rss_mb.toFixed(1)} MB</span>
          </div>
          {process.exe && (
            <p className="text-xs text-muted font-mono truncate" title={process.exe}>
              {process.exe}
            </p>
          )}
        </div>
      )}

      <Handle type="source" position={Position.Bottom}
        className="!bg-slate-600 !border-slate-700 !w-2 !h-2" />
    </div>
  );
});
