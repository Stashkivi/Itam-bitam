import { useCallback, useEffect, useMemo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { HostNode }    from './nodes/HostNode';
import { ServiceNode } from './nodes/ServiceNode';
import { ProcessNode } from './nodes/ProcessNode';
import { PortNode }    from './nodes/PortNode';
import { DependencyEdge } from './edges/DependencyEdge';

import { applyDagreLayout } from '@/utils/layout';
import { useGraphStore }    from '@/store/graphStore';
import type { DependencyMap } from '@/types';

const NODE_TYPES = {
  hostNode:    HostNode,
  serviceNode: ServiceNode,
  processNode: ProcessNode,
  portNode:    PortNode,
};

const EDGE_TYPES = { dependency: DependencyEdge };

interface Props {
  map: DependencyMap | undefined;
  isLoading: boolean;
}

export function DependencyGraph({ map, isLoading }: Props) {
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const { setNodes: storeSetNodes, setEdges: storeSetEdges } = useGraphStore();

  // ── Build React Flow nodes + edges from the dependency map ──────────────────
  const { rawNodes, rawEdges } = useMemo(() => {
    if (!map) return { rawNodes: [], rawEdges: [] };

    const rn: Node[] = [];
    const re: Edge[] = [];

    const hostId = `host:${map.host.id}`;

    // ── Host node ────────────────────────────────────────────────────────────
    rn.push({
      id:   hostId,
      type: 'hostNode',
      position: { x: 0, y: 0 },
      data: { host: map.host },
    });

    // ── Service nodes ────────────────────────────────────────────────────────
    for (const svc of map.services) {
      const svcId = `service:${svc.name}`;
      rn.push({
        id:   svcId,
        type: 'serviceNode',
        position: { x: 0, y: 0 },
        data: { service: svc, hostId: map.host.id },
      });
      re.push({
        id:     `${hostId}->${svcId}`,
        source: hostId,
        target: svcId,
        type:   'dependency',
        data:   { kind: 'host-service' },
      });
    }

    // ── Process nodes ────────────────────────────────────────────────────────
    for (const proc of map.processes) {
      const procId = `process:${proc.pid}:${proc.name}`;

      // Determine parent service by matching open ports
      const parentService = map.services.find((s) =>
        s.pid === proc.pid ||
        map.open_ports.some((p) => p.pid === proc.pid && s.name.toLowerCase().includes(proc.name.toLowerCase())),
      );
      const parentId = parentService ? `service:${parentService.name}` : hostId;

      rn.push({
        id:   procId,
        type: 'processNode',
        position: { x: 0, y: 0 },
        data: { process: proc, hostId: map.host.id, collapsed: true },
      });
      re.push({
        id:     `${parentId}->${procId}`,
        source: parentId,
        target: procId,
        type:   'dependency',
        data:   { kind: 'service-process' },
      });
    }

    // ── Port nodes ───────────────────────────────────────────────────────────
    const approvedPortNums = new Set(map.open_ports.map((p) => p.port));

    for (const port of map.open_ports) {
      const portId = `port:${port.port}/${port.protocol}`;
      const parentProcId = `process:${port.pid}:${port.process_name ?? ''}`;
      const parentExists = rn.some((n) => n.id === parentProcId);
      const parentId = parentExists ? parentProcId : hostId;

      rn.push({
        id:   portId,
        type: 'portNode',
        position: { x: 0, y: 0 },
        data: {
          port,
          hostId: map.host.id,
          isApproved: approvedPortNums.has(port.port),
        },
      });
      re.push({
        id:     `${parentId}->${portId}`,
        source: parentId,
        target: portId,
        type:   'dependency',
        data:   { kind: 'process-port' },
      });
    }

    return { rawNodes: rn, rawEdges: re };
  }, [map]);

  // ── Run Dagre layout and commit to React Flow state ─────────────────────────
  useEffect(() => {
    if (!rawNodes.length) return;
    const laid = applyDagreLayout(rawNodes, rawEdges);
    setNodes(laid);
    setEdges(rawEdges);
    storeSetNodes(laid);
    storeSetEdges(rawEdges);
  }, [rawNodes, rawEdges, setNodes, setEdges, storeSetNodes, storeSetEdges]);

  const onInit = useCallback(() => {/* fitView handled by ReactFlow prop */}, []);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center text-muted text-sm">
        Loading dependency graph…
      </div>
    );
  }

  if (!map) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 text-muted">
        <span className="text-4xl">🖧</span>
        <p className="text-sm">Select a host to view its dependency map</p>
      </div>
    );
  }

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      nodeTypes={NODE_TYPES}
      edgeTypes={EDGE_TYPES}
      onInit={onInit}
      fitView
      fitViewOptions={{ padding: 0.15 }}
      minZoom={0.2}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
    >
      <Background
        variant={BackgroundVariant.Dots}
        color="#1e293b"
        gap={24}
        size={1.5}
      />
      <Controls
        className="!bg-card !border-border !rounded-lg"
        showInteractive={false}
      />
      <MiniMap
        className="!bg-card !border-border"
        nodeColor={(n) => {
          switch (n.type) {
            case 'hostNode':    return '#3b82f6';
            case 'serviceNode': return '#10b981';
            case 'processNode': return '#6366f1';
            case 'portNode':    return '#f59e0b';
            default:            return '#475569';
          }
        }}
        maskColor="rgba(15,23,42,0.6)"
      />
    </ReactFlow>
  );
}
