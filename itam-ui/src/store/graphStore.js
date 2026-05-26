import { create } from 'zustand';
// ── Severity precedence ───────────────────────────────────────────────────────
const SEVERITY_RANK = {
    CRITICAL: 4, HIGH: 3, MEDIUM: 2, LOW: 1,
};
function higherSeverity(a, b) {
    if (!a)
        return b;
    return SEVERITY_RANK[a] >= SEVERITY_RANK[b] ? a : b;
}
export const useGraphStore = create((set, get) => ({
    nodes: [],
    edges: [],
    anomaliesByNode: {},
    selectedHostId: null,
    dependencyMap: null,
    toasts: [],
    setNodes: (nodes) => set({ nodes }),
    setEdges: (edges) => set({ edges }),
    setDependencyMap: (map) => set({ dependencyMap: map }),
    selectHost: (id) => set({ selectedHostId: id }),
    applyAnomaly: (anomaly) => {
        const nodeId = `${anomaly.entity_type}:${anomaly.entity_key}`;
        const hostNodeId = `host:${anomaly.host_uuid}`;
        set((state) => {
            // Add to per-node list
            const prev = state.anomaliesByNode[nodeId] ?? [];
            const prevHost = state.anomaliesByNode[hostNodeId] ?? [];
            // Update affected node's data.anomalySeverity in the nodes array
            const nodes = state.nodes.map((n) => {
                if (n.id === nodeId || n.id === hostNodeId) {
                    const currentSev = n.data.anomalySeverity;
                    return {
                        ...n,
                        data: {
                            ...n.data,
                            anomalySeverity: higherSeverity(currentSev, anomaly.severity),
                        },
                    };
                }
                return n;
            });
            return {
                nodes,
                toasts: [anomaly, ...state.toasts].slice(0, 10),
                anomaliesByNode: {
                    ...state.anomaliesByNode,
                    [nodeId]: [anomaly, ...prev].slice(0, 50),
                    [hostNodeId]: [anomaly, ...prevHost].slice(0, 50),
                },
            };
        });
    },
    resolveAnomaly: (hostUUID, ruleId, entityKey) => {
        const nodeId = `${entityKey}`;
        set((state) => {
            const updated = { ...state.anomaliesByNode };
            if (updated[nodeId]) {
                updated[nodeId] = updated[nodeId].filter((a) => !(a.host_uuid === hostUUID && a.rule_id === ruleId));
            }
            return { anomaliesByNode: updated };
        });
    },
    dismissToast: (id) => set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) })),
    getNodeSeverity: (nodeId) => {
        const list = get().anomaliesByNode[nodeId];
        if (!list?.length)
            return undefined;
        return list.reduce((best, a) => higherSeverity(best, a.severity), 'LOW');
    },
}));
