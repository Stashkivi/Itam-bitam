import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import { STATUS_DOT, SEVERITY_BORDER, SEVERITY_RING_COLOR, serviceAbbrev, serviceColor, } from '@/utils/colors';
export const ServiceNode = memo(function ServiceNode({ data }) {
    const { service, anomalySeverity } = data;
    const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;
    return (_jsxs("div", { className: `
        relative w-48 rounded-lg border bg-card px-3 py-2.5 shadow-md
        ${anomalySeverity ? SEVERITY_BORDER[anomalySeverity] : 'border-slate-600'}
        transition-all duration-300
      `, style: ringColor ? { '--ring-color': ringColor } : undefined, children: [anomalySeverity && (_jsx("span", { className: "absolute inset-0 rounded-lg animate-pulse-ring pointer-events-none", "aria-hidden": true })), _jsx(Handle, { type: "target", position: Position.Top, className: "!bg-slate-500 !border-slate-700" }), _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("span", { className: `
            flex h-8 w-8 flex-shrink-0 items-center justify-center
            rounded-md text-xs font-bold text-white
            ${serviceColor(service.name)}
          `, children: serviceAbbrev(service.name) }), _jsxs("div", { className: "flex-1 min-w-0", children: [_jsxs("div", { className: "flex items-center gap-1.5", children: [_jsx("span", { className: `h-2 w-2 rounded-full flex-shrink-0 ${STATUS_DOT[service.status] ?? 'bg-slate-600'}` }), _jsx("p", { className: "truncate text-sm font-medium text-slate-100", children: service.name })] }), _jsxs("div", { className: "flex items-center gap-1.5 mt-0.5", children: [_jsx("span", { className: `text-xs ${service.status === 'failed' ? 'text-red-400' :
                                            service.status === 'running' ? 'text-emerald-400' :
                                                'text-muted'}`, children: service.status }), service.version && (_jsxs("span", { className: "text-xs text-muted font-mono", children: ["v", service.version] }))] })] })] }), _jsx(Handle, { type: "source", position: Position.Bottom, className: "!bg-slate-500 !border-slate-700" })] }));
});
