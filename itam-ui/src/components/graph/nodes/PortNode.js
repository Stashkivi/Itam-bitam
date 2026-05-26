import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import { SEVERITY_BORDER, SEVERITY_RING_COLOR, portLabel, isExposedBind } from '@/utils/colors';
export const PortNode = memo(function PortNode({ data }) {
    const { port, isApproved, anomalySeverity } = data;
    const exposed = isExposedBind(port.bind_addr);
    const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;
    const borderClass = anomalySeverity
        ? SEVERITY_BORDER[anomalySeverity]
        : isApproved
            ? 'border-slate-700'
            : 'border-amber-500'; // amber = not in baseline but no anomaly rule yet
    return (_jsxs("div", { className: `
        relative w-36 rounded-md border bg-card px-3 py-2 shadow-sm
        ${borderClass}
        transition-all duration-200
      `, style: ringColor ? { '--ring-color': ringColor } : undefined, children: [anomalySeverity && (_jsx("span", { className: "absolute inset-0 rounded-md animate-pulse-ring pointer-events-none", "aria-hidden": true })), _jsx(Handle, { type: "target", position: Position.Top, className: "!bg-slate-600 !border-slate-700 !w-2 !h-2" }), _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("span", { className: `text-sm leading-none ${exposed ? 'text-orange-400' : 'text-slate-400'}`, title: exposed ? `Exposed on ${port.bind_addr}` : `Local: ${port.bind_addr}`, children: exposed ? '🌐' : '🔒' }), _jsxs("div", { className: "flex-1 min-w-0", children: [_jsxs("div", { className: "flex items-baseline gap-1", children: [_jsx("span", { className: "font-mono font-bold text-slate-100 text-sm", children: portLabel(port.port) }), _jsx("span", { className: "text-xs text-muted uppercase", children: port.protocol })] }), port.port !== parseInt(portLabel(port.port), 10) && (_jsx("p", { className: "text-xs text-muted font-mono", children: port.port }))] })] }), !isApproved && !anomalySeverity && (_jsx("p", { className: "mt-1 text-xs text-amber-400 font-medium", children: "\u26A0 unknown" }))] }));
});
