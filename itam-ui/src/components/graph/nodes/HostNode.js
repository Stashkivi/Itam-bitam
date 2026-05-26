import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import { SEVERITY_BORDER, SEVERITY_RING_COLOR } from '@/utils/colors';
const OS_ICON = {
    linux: '🐧',
    windows: '🪟',
};
function healthBorder(sev) {
    if (!sev)
        return 'border-slate-600';
    return SEVERITY_BORDER[sev];
}
export const HostNode = memo(function HostNode({ data }) {
    const { host, anomalySeverity } = data;
    const ringColor = anomalySeverity ? SEVERITY_RING_COLOR[anomalySeverity] : undefined;
    return (_jsxs("div", { className: `
        relative w-60 rounded-xl border-2 bg-card px-4 py-3 shadow-lg
        ${healthBorder(anomalySeverity)}
        transition-all duration-300
      `, style: ringColor ? { '--ring-color': ringColor } : undefined, children: [anomalySeverity && (_jsx("span", { className: "absolute inset-0 rounded-xl animate-pulse-ring pointer-events-none", "aria-hidden": true })), _jsxs("div", { className: "flex items-center gap-2 mb-2", children: [_jsx("span", { className: "text-xl leading-none", role: "img", "aria-label": host.os_family, children: OS_ICON[host.os_family] ?? '💻' }), _jsxs("div", { className: "flex-1 min-w-0", children: [_jsx("p", { className: "truncate font-semibold text-slate-100 text-sm leading-tight", children: host.hostname }), _jsxs("p", { className: "text-xs text-muted", children: [host.os_family, " \u00B7 ", host.arch] })] }), _jsx("span", { className: `h-2.5 w-2.5 rounded-full flex-shrink-0 ${anomalySeverity === 'CRITICAL' ? 'bg-red-500' :
                            anomalySeverity === 'HIGH' ? 'bg-orange-500' :
                                anomalySeverity === 'MEDIUM' ? 'bg-yellow-400' :
                                    anomalySeverity === 'LOW' ? 'bg-blue-400' :
                                        'bg-emerald-400'}` })] }), host.ip_primary && (_jsx("p", { className: "text-xs text-muted font-mono truncate", children: host.ip_primary })), host.last_seen_at && (_jsxs("p", { className: "text-xs text-muted mt-0.5", children: ["seen ", new Date(host.last_seen_at).toLocaleTimeString()] })), _jsx(Handle, { type: "source", position: Position.Bottom, className: "!bg-slate-500 !border-slate-700" })] }));
});
