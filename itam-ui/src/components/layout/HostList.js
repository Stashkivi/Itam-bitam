import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useGraphStore } from '@/store/graphStore';
import { useHosts } from '@/api/hosts';
const OS_ICON = { linux: '🐧', windows: '🪟' };
function seenAgo(ts) {
    if (!ts)
        return 'never';
    const diffMs = Date.now() - new Date(ts).getTime();
    const mins = Math.floor(diffMs / 60_000);
    if (mins < 1)
        return 'just now';
    if (mins < 60)
        return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24)
        return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
}
function HostRow({ host, selected }) {
    const selectHost = useGraphStore((s) => s.selectHost);
    const anomalies = useGraphStore((s) => s.anomaliesByNode[`host:${host.id}`] ?? []);
    const hasCritical = anomalies.some((a) => a.severity === 'CRITICAL');
    const hasHigh = anomalies.some((a) => a.severity === 'HIGH');
    return (_jsx("button", { onClick: () => selectHost(selected ? null : host.id), className: `
        w-full text-left rounded-lg px-3 py-2.5 transition-colors
        ${selected
            ? 'bg-blue-600/20 border border-blue-500/50'
            : 'hover:bg-card border border-transparent'}
      `, children: _jsxs("div", { className: "flex items-center gap-2", children: [_jsx("span", { className: "text-base leading-none flex-shrink-0", children: OS_ICON[host.os_family] ?? '💻' }), _jsxs("div", { className: "flex-1 min-w-0", children: [_jsx("p", { className: "truncate text-sm font-medium text-slate-200", children: host.hostname }), _jsx("p", { className: "text-xs text-muted", children: seenAgo(host.last_seen_at) })] }), (hasCritical || hasHigh) && (_jsx("span", { className: `h-2 w-2 rounded-full flex-shrink-0 ${hasCritical ? 'bg-red-500 animate-pulse' : 'bg-orange-400'}` }))] }) }));
}
export function HostList() {
    const { data: hosts = [], isLoading, isError } = useHosts();
    const selectedHostId = useGraphStore((s) => s.selectedHostId);
    return (_jsxs("nav", { className: "flex h-full w-56 flex-col border-r border-border bg-card/30", children: [_jsx("header", { className: "border-b border-border px-4 py-3", children: _jsx("h2", { className: "text-xs font-semibold uppercase tracking-widest text-muted", children: "Hosts" }) }), _jsxs("div", { className: "flex-1 overflow-y-auto px-2 py-2 space-y-0.5", children: [isLoading && (_jsx("p", { className: "text-xs text-muted text-center py-6", children: "Loading hosts\u2026" })), isError && (_jsx("p", { className: "text-xs text-red-400 text-center py-6", children: "Failed to load hosts" })), hosts.map((h) => (_jsx(HostRow, { host: h, selected: h.id === selectedHostId }, h.id))), !isLoading && !hosts.length && (_jsx("p", { className: "text-xs text-muted text-center py-6", children: "No enrolled hosts" }))] })] }));
}
