import { jsx as _jsx } from "react/jsx-runtime";
import { memo } from 'react';
import { BaseEdge, getStraightPath, } from '@xyflow/react';
// Edge stroke color is passed via data.kind from the builder
const KIND_COLOR = {
    'host-service': '#475569', // slate-600
    'service-process': '#334155', // slate-700
    'process-port': '#1e3a5f', // custom blue-dark
};
export const DependencyEdge = memo(function DependencyEdge({ sourceX, sourceY, targetX, targetY, data, }) {
    const kind = data?.kind ?? 'host-service';
    const color = KIND_COLOR[kind] ?? KIND_COLOR['host-service'];
    const [edgePath] = getStraightPath({ sourceX, sourceY, targetX, targetY });
    return (_jsx(BaseEdge, { path: edgePath, style: { stroke: color, strokeWidth: 1.5, opacity: 0.7 } }));
});
