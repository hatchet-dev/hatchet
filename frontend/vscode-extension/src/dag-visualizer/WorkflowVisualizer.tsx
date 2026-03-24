import { useMemo } from 'react';
import ReactFlow, { BezierEdge, Edge, MarkerType, Node } from 'reactflow';
// Note: consumers must import 'reactflow/dist/style.css' in their entry point.
import { getLayoutedElements } from './layout';
import StepNode from './StepNode';
import { DagShape, NodeDisplayData } from './types';

const connectionLineStyleDark = { stroke: '#fff' };
const connectionLineStyleLight = { stroke: '#000' };

const nodeTypes = {
  stepNode: StepNode,
};

const edgeTypes = {
  smoothstep: BezierEdge,
};

export interface WorkflowVisualizerProps {
  shape: DagShape;
  theme?: 'dark' | 'light';
  onNodeClick?: (stepId: string) => void;
  className?: string;
}

export function WorkflowVisualizer({
  shape,
  theme = 'dark',
  onNodeClick,
  className,
}: WorkflowVisualizerProps) {
  const connectionLineStyle =
    theme === 'dark' ? connectionLineStyleDark : connectionLineStyleLight;

  const edges: Edge[] = useMemo(
    () =>
      shape.flatMap((shapeItem) =>
        shapeItem.childrenStepIds
          .filter((childId) => shape.some((t) => t.stepId === childId))
          .map((childId) => ({
            id: `${shapeItem.stepId}-${childId}`,
            source: shapeItem.stepId,
            target: childId,
            animated: shapeItem.status === 'running',
            style: connectionLineStyle,
            markerEnd: {
              type: MarkerType.ArrowClosed,
            },
            type: 'smoothstep',
          })),
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [shape, theme],
  );

  const nodes: Node[] = useMemo(
    () =>
      shape.map((shapeItem) => {
        const hasParent = shape.some((s) =>
          s.childrenStepIds.includes(shapeItem.stepId),
        );
        const hasChildren = shapeItem.childrenStepIds.length > 0;

        const graphVariant: NodeDisplayData['graphVariant'] =
          hasParent && hasChildren
            ? 'default'
            : hasParent
              ? 'input_only'
              : hasChildren
                ? 'output_only'
                : 'output_only'; // isolated node: show as root

        const data: NodeDisplayData = {
          taskName: shapeItem.taskName,
          graphVariant,
          status: shapeItem.status,
          durationMs: shapeItem.durationMs,
          isSkipped: shapeItem.isSkipped,
          onClick: () => onNodeClick?.(shapeItem.stepId),
        };

        return {
          id: shapeItem.stepId,
          type: 'stepNode',
          position: { x: 0, y: 0 },
          data,
          selectable: true,
        };
      }),
    [shape, onNodeClick],
  );

  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(nodes, edges),
    [nodes, edges],
  );

  return (
    <div className={className ?? 'h-[300px] w-full'}>
      <ReactFlow
        nodes={layoutedNodes}
        edges={layoutedEdges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        proOptions={{ hideAttribution: true }}
        onNodeClick={(_, node) => onNodeClick?.(node.id)}
        maxZoom={1}
        connectionLineStyle={connectionLineStyle}
        snapToGrid={true}
      />
    </div>
  );
}
