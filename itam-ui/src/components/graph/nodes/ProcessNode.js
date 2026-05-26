import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { memo, useCallback } from 'react';
import { Handle, Position, useReactFlow } from '@xyflow/react';
import { SEVERITY_BORDER, SEVERITY_RING_COLOR } from '@/utils/colors';
function cpuColor(pct) {
    if (pct > 80)
        return 'text-red-400';
    if (pct > 50)
        return 'text-yellow-400';
    return 'text-emerald-400';
}
export const ProcessNode = memo(function ProcessNode({ id, data }) {
    const { process, collapsed, anomalySeverity } = data;
    const { updateNodeData } = useReactFlow();
    const toggle = useCallback(() => {
        updateNodeData(id, { collapsed: !collapsed });
    }, [id, collapsed, updateNodeData]);
    const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;
    return (_jsxs("div", { className: `
        relative w-52 rounded-lg border bg-card shadow-sm cursor-pointer
        ${anomalySeverity ? SEVERITY_BORDER[anomalySeverity] : 'border-slate-700'}
        hover:border-slate-500 transition-all duration-200
      `, style: ringColor ? { '--ring-color': ringColor } : undefined, onClick: toggle, children: [anomalySeverity && (_jsx("span", { className: "absolute inset-0 rounded-lg animate-pulse-ring pointer-events-none", "aria-hidden": true })), _jsx(Handle, { type: "target", position: Position.Top, className: "!bg-slate-600 !border-slate-700 !w-2 !h-2" }), _jsxs("div", { className: "flex items-center gap-2 px-3 py-2", children: [_jsx("span", { className: "text-xs font-mono text-muted w-12 flex-shrink-0", children: process.pid }), _jsx("p", { className: "flex-1 truncate text-xs font-medium text-slate-200", children: process.name }), _jsxs("span", { className: `text-xs font-mono ${cpuColor(process.cpu_pct)}`, children: [process.cpu_pct.toFixed(1), "%"] }), _jsx("span", { className: "text-muted text-xs ml-1", children: collapsed ? '›' : '‹' })] }), !collapsed && (_jsxs("div", { className: "border-t border-slate-700 px-3 py-2 space-y-1", children: [_jsxs("div", { className: "flex justify-between text-xs", children: [_jsx("span", { className: "text-muted", children: "user" }), _jsx("span", { className: "text-slate-300 font-mono", children: process.username || '—' })] }), _jsxs("div", { className: "flex justify-between text-xs", children: [_jsx("span", { className: "text-muted", children: "RSS" }), _jsxs("span", { className: "text-slate-300", children: [process.mem_rss_mb.toFixed(1), " MB"] })] }), process.exe && (_jsx("p", { className: "text-xs text-muted font-mono truncate", title: process.exe, children: process.exe }))] })), _jsx(Handle, { type: "source", position: Position.Bottom, className: "!bg-slate-600 !border-slate-700 !w-2 !h-2" })] }));
});
