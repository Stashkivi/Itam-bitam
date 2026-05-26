import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useWsStore } from '@/store/wsStore';
import { useGraphStore } from '@/store/graphStore';
import { useDependencyMap } from '@/api/hosts';
import { useWebSocket } from '@/hooks/useWebSocket';
import { getToken } from '@/api/client';
import { HostList } from './HostList';
import { DependencyGraph } from '@/components/graph/DependencyGraph';
import { AnomalyFeed } from '@/components/anomaly/AnomalyFeed';
import { AnomalyToastContainer } from '@/components/anomaly/AnomalyToast';
function WsStatusPill() {
    const status = useWsStore((s) => s.status);
    const label = {
        connected: 'Live',
        connecting: 'Connecting…',
        disconnected: 'Offline',
        error: 'Error',
    };
    const dot = {
        connected: 'bg-emerald-400',
        connecting: 'bg-yellow-400 animate-pulse',
        disconnected: 'bg-slate-500',
        error: 'bg-red-500',
    };
    return (_jsxs("div", { className: "flex items-center gap-1.5", children: [_jsx("span", { className: `h-1.5 w-1.5 rounded-full ${dot[status]}` }), _jsx("span", { className: "text-xs text-muted", children: label[status] })] }));
}
export function AppShell() {
    const token = getToken();
    const selectedHostId = useGraphStore((s) => s.selectedHostId);
    // Start WebSocket connection on mount
    useWebSocket(token);
    const { data: depMap, isLoading } = useDependencyMap(selectedHostId);
    return (_jsxs("div", { className: "flex h-screen flex-col overflow-hidden bg-surface text-slate-100", children: [_jsxs("header", { className: "flex h-12 flex-shrink-0 items-center justify-between border-b border-border bg-card/60 px-6", children: [_jsxs("div", { className: "flex items-center gap-3", children: [_jsx("span", { className: "text-lg font-bold tracking-tight text-slate-100", children: "ITAM" }), _jsx("span", { className: "h-4 w-px bg-border" }), _jsx("span", { className: "text-sm text-muted", children: "Dependency Map" })] }), _jsx(WsStatusPill, {})] }), _jsxs("div", { className: "flex flex-1 overflow-hidden", children: [_jsx(HostList, {}), _jsx("main", { className: "relative flex-1 overflow-hidden", children: _jsx(DependencyGraph, { map: depMap, isLoading: isLoading }) }), _jsx(AnomalyFeed, {})] }), _jsx(AnomalyToastContainer, {})] }));
}
