import dagre from 'dagre';
import { Edge, Node, Position } from 'reactflow';

export const NODE_WIDTH = 230;
export const NODE_HEIGHT = 70;

/**
 * Applies dagre left-to-right layout to the given ReactFlow nodes and edges.
 * Returns new node/edge arrays with computed positions.
 */
export function getLayoutedElements(
  nodes: Node[],
  edges: Edge[],
  direction = 'LR',
): { nodes: Node[]; edges: Edge[] } {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));

  const isHorizontal = direction === 'LR';
  dagreGraph.setGraph({ rankdir: direction });

  nodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    node.targetPosition = isHorizontal ? Position.Left : Position.Top;
    node.sourcePosition = isHorizontal ? Position.Right : Position.Bottom;

    node.position = {
      x: nodeWithPosition.x - NODE_WIDTH / 2,
      y: nodeWithPosition.y - NODE_HEIGHT / 2,
    };

    return { ...node };
  });

  return { nodes: layoutedNodes, edges };
}
