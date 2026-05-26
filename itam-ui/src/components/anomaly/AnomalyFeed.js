import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useGraphStore } from '@/store/graphStore';
import { useHostAnomalies } from '@/api/hosts';
import { SEVERITY_TEXT, SEVERITY_BG, SEVERITY_BORDER } from '@/utils/colors';
function SeverityBadge({ severity }) {
    return (_jsx("span", { className: `
        inline-flex items-center rounded px-1.5 py-0.5
        text-xs font-bold uppercase tracking-wide
        ${SEVERITY_TEXT[severity]} ${SEVERITY_BG[severity]}
      `, children: severity }));
}
function AnomalyRow({ anomaly }) {
    const ts = new Date(anomaly.detected_at);
    return (_jsxs("div", { className: `rounded-lg border p-3 ${SEVERITY_BORDER[anomaly.severity]} bg-card/60`, children: [_jsxs("div", { className: "flex items-start justify-between gap-2 mb-1", children: [_jsx(SeverityBadge, { severity: anomaly.severity }), _jsx("span", { className: "text-xs text-muted tabular-nums", children: ts.toLocaleTimeString() })] }), _jsx("p", { className: "text-xs font-mono text-slate-300 font-medium", children: anomaly.rule_id }), _jsxs("p", { className: "text-xs text-muted mt-0.5", children: [anomaly.entity_type, ":", ' ', _jsx("span", { className: "text-slate-300 font-mono", children: anomaly.entity_key })] }), anomaly.details && (_jsxs("details", { className: "mt-1.5", children: [_jsx("summary", { className: "text-xs text-muted cursor-pointer select-none hover:text-slate-400", children: "details" }), _jsx("pre", { className: "mt-1 text-xs text-slate-400 bg-surface rounded p-2 overflow-x-auto", children: JSON.stringify(anomaly.details, null, 2) })] }))] }));
}
export function AnomalyFeed() {
    const selectedHostId = useGraphStore((s) => s.selectedHostId);
    const liveToasts = useGraphStore((s) => s.toasts);
    const { data: serverAnomalies = [], isLoading } = useHostAnomalies(selectedHostId);
    // Merge live WS events with server-fetched list, dedup by id
    const merged = [
        ...liveToasts.filter((a) => a.host_uuid === selectedHostId),
        ...serverAnomalies,
    ].reduce((acc, a) => {
        if (!acc.find((x) => x.id === a.id))
            acc.push(a);
        return acc;
    }, []);
    return (_jsxs("aside", { className: "flex h-full w-72 flex-col border-l border-border bg-card/30", children: [_jsxs("header", { className: "flex items-center justify-between border-b border-border px-4 py-3", children: [_jsx("h2", { className: "text-sm font-semibold text-slate-200", children: "Anomaly Feed" }), merged.length > 0 && (_jsx("span", { className: "flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-xs font-bold text-white", children: merged.length > 99 ? '99+' : merged.length }))] }), _jsxs("div", { className: "flex-1 overflow-y-auto p-3 space-y-2", children: [isLoading && (_jsx("p", { className: "text-xs text-muted text-center py-4", children: "Loading\u2026" })), !isLoading && !selectedHostId && (_jsx("p", { className: "text-xs text-muted text-center py-4", children: "Select a host to see its anomalies" })), !isLoading && selectedHostId && merged.length === 0 && (_jsxs("div", { className: "text-center py-6", children: [_jsx("p", { className: "text-2xl mb-1", children: "\u2713" }), _jsx("p", { className: "text-xs text-muted", children: "No active anomalies" })] })), merged
                        .sort((a, b) => new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime())
                        .map((a) => (_jsx(AnomalyRow, { anomaly: a }, `${a.id}-${a.detected_at}`)))] })] }));
}
