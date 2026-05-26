import dagre from '@dagrejs/dagre';
import type { Node, Edge } from '@xyflow/react';

// Node dimensions used for Dagre placement.
// Must match the rendered sizes in the node components.
const DIMS: Record<string, { w: number; h: number }> = {
  hostNode:    { w: 240, h: 110 },
  serviceNode: { w: 190, h: 90 },
  processNode: { w: 210, h: 70 },
  portNode:    { w: 150, h: 56 },
};

const DEFAULT_DIM = { w: 180, h: 80 };

/**
 * Runs Dagre on the supplied node/edge arrays and returns new nodes with
 * `position` set. Edges are returned unchanged (React Flow handles routing).
 *
 * Direction is top-to-bottom: Host → Services → Processes → Ports
 */
export function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({
    rankdir: 'TB',
    nodesep: 40,
    ranksep: 70,
    marginx: 20,
    marginy: 20,
  });

  for (const node of nodes) {
    const dim = DIMS[node.type ?? ''] ?? DEFAULT_DIM;
    g.setNode(node.id, { width: dim.w, height: dim.h });
  }

  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }

  dagre.layout(g);

  return nodes.map((node) => {
    const { x, y, width, height } = g.node(node.id);
    return {
      ...node,
      position: {
        x: x - width / 2,
        y: y - height / 2,
      },
    };
  });
}
