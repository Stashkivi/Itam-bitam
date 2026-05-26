import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { useEffect } from 'react';
import { useGraphStore } from '@/store/graphStore';
import { SEVERITY_BG, SEVERITY_TEXT } from '@/utils/colors';
const AUTO_DISMISS_MS = 6_000;
function Toast({ anomaly, onDismiss }) {
    useEffect(() => {
        if (anomaly.severity === 'LOW' || anomaly.severity === 'MEDIUM') {
            const t = setTimeout(onDismiss, AUTO_DISMISS_MS);
            return () => clearTimeout(t);
        }
    }, [anomaly.severity, onDismiss]);
    return (_jsxs("div", { className: `
        flex items-start gap-3 rounded-lg border border-slate-700 shadow-xl
        bg-card px-4 py-3 animate-slide-in w-80
        ${SEVERITY_BG[anomaly.severity]}
      `, role: "alert", children: [_jsx("div", { className: `mt-0.5 h-4 w-1 flex-shrink-0 rounded-full ${anomaly.severity === 'CRITICAL' ? 'bg-red-500' :
                    anomaly.severity === 'HIGH' ? 'bg-orange-500' :
                        anomaly.severity === 'MEDIUM' ? 'bg-yellow-400' :
                            'bg-blue-400'}` }), _jsxs("div", { className: "flex-1 min-w-0", children: [_jsxs("div", { className: "flex items-center justify-between gap-2", children: [_jsx("span", { className: `text-xs font-bold uppercase tracking-wide ${SEVERITY_TEXT[anomaly.severity]}`, children: anomaly.severity }), _jsx("span", { className: "text-xs text-muted", children: anomaly.rule_id })] }), _jsx("p", { className: "mt-0.5 text-sm font-medium text-slate-100 truncate", children: anomaly.hostname }), _jsxs("p", { className: "text-xs text-muted truncate", children: [anomaly.entity_type, ": ", _jsx("span", { className: "font-mono", children: anomaly.entity_key })] })] }), _jsx("button", { onClick: onDismiss, className: "text-muted hover:text-slate-300 text-lg leading-none flex-shrink-0", "aria-label": "Dismiss", children: "\u00D7" })] }));
}
export function AnomalyToastContainer() {
    const toasts = useGraphStore((s) => s.toasts);
    const dismissToast = useGraphStore((s) => s.dismissToast);
    if (!toasts.length)
        return null;
    return (_jsx("div", { className: "fixed bottom-6 right-6 z-50 flex flex-col gap-2", children: toasts.map((a) => (_jsx(Toast, { anomaly: a, onDismiss: () => dismissToast(a.id) }, a.id))) }));
}
