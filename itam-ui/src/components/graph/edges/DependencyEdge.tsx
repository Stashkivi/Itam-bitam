import { memo } from 'react';
import {
  BaseEdge,
  getStraightPath,
  type EdgeProps,
} from '@xyflow/react';

// Edge stroke color is passed via data.kind from the builder
const KIND_COLOR: Record<string, string> = {
  'host-service': '#475569',    // slate-600
  'service-process': '#334155', // slate-700
  'process-port': '#1e3a5f',    // custom blue-dark
};

export const DependencyEdge = memo(function DependencyEdge({
  sourceX, sourceY, targetX, targetY, data,
}: EdgeProps) {
  const kind  = (data as { kind?: string } | undefined)?.kind ?? 'host-service';
  const color = KIND_COLOR[kind] ?? KIND_COLOR['host-service'];

  const [edgePath] = getStraightPath({ sourceX, sourceY, targetX, targetY });

  return (
    <BaseEdge
      path={edgePath}
      style={{ stroke: color, strokeWidth: 1.5, opacity: 0.7 }}
    />
  );
});
