import { useMemo } from 'react';
import ReactFlow, {
  Position,
  MarkerType,
  Node,
  Edge,
  BezierEdge,
} from 'reactflow';
import 'reactflow/dist/style.css';
import dagre from 'dagre';
import { useTheme } from '@/components/theme-provider';
import stepRunNode, { NodeData } from './step-run-node';
import { V1TaskStatus } from '@/lib/api';
import { useWorkflowDetails } from '../../hooks/workflow-details';

const connectionLineStyleDark = { stroke: '#fff' };
const connectionLineStyleLight = { stroke: '#000' };

const nodeTypes = {
  stepNode: stepRunNode,
};

const edgeTypes = {
  smoothstep: BezierEdge,
};

const createNodeId = (taskId: string) => taskId;

const WorkflowRunVisualizer = ({
  setSelectedTaskRunId,
}: {
  setSelectedTaskRunId: (id: string) => void;
}) => {
  const { theme } = useTheme();
  const { shape, taskRuns, isLoading, isError } = useWorkflowDetails();

  const edges: Edge[] = useMemo(
    () =>
      (
        shape.flatMap((task) =>
          task.childrenExternalIds.map((childId) => {
            const child = taskRuns.find((t) => t.metadata.id === childId);

            if (!child) {
              return null;
            }

            return {
              id: `${task.taskExternalId}-${childId}`,
              source: task.taskExternalId,
              target: childId,
              animated: child.status === V1TaskStatus.RUNNING,
              style:
                theme === 'dark'
                  ? connectionLineStyleDark
                  : connectionLineStyleLight,
              markerEnd: {
                type: MarkerType.ArrowClosed,
              },
              type: 'smoothstep',
            };
          }),
        ) || []
      ).filter((x) => Boolean(x)) as Edge[],
    [shape, theme, taskRuns],
  );

  const nodes: Node[] = useMemo(
    () =>
      taskRuns.map((task) => {
        const hasParent = shape.some((s) =>
          s.childrenExternalIds.includes(task.metadata.id),
        );
        const hasChild = shape.some(
          (s) => s.taskExternalId === task.metadata.id,
        );

        // TODO: get the actual number of children
        const childWorkflowsCount = 0;

        const data: NodeData = {
          task,
          graphVariant:
            hasParent && hasChild
              ? 'default'
              : hasChild
                ? 'output_only'
                : 'input_only',
          onClick: () => setSelectedTaskRunId(task.metadata.id),
          childWorkflowsCount,
          taskName:
            shape.find((i) => i.taskExternalId === task.metadata.id)
              ?.taskName || '',
        };

        return {
          id: createNodeId(task.metadata.id),
          type: 'stepNode',
          position: { x: 0, y: 0 },
          data,
          selectable: true,
        };
      }) || [],
    [shape, taskRuns, setSelectedTaskRunId],
  );

  const nodeWidth = 230;
  const nodeHeight = 70;

  const getLayoutedElements = (
    nodes: Node[],
    edges: Edge[],
    direction = 'LR',
  ) => {
    const dagreGraph = new dagre.graphlib.Graph();
    dagreGraph.setDefaultEdgeLabel(() => ({}));

    const isHorizontal = direction === 'LR';
    dagreGraph.setGraph({ rankdir: direction });

    nodes.forEach((node) => {
      dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
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
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      };

      return { ...node };
    });

    return { nodes: layoutedNodes, edges };
  };

  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(nodes, edges),
    [nodes, edges],
  );

  if (isLoading || isError) {
    return null;
  }

  return (
    <div className="w-full h-[300px]">
      <ReactFlow
        nodes={layoutedNodes}
        edges={layoutedEdges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        proOptions={{
          hideAttribution: true,
        }}
        onNodeClick={(_, node) => {
          setSelectedTaskRunId(node.id);
        }}
        maxZoom={1}
        connectionLineStyle={
          theme === 'dark' ? connectionLineStyleDark : connectionLineStyleLight
        }
        snapToGrid={true}
      />
    </div>
  );
};

export default WorkflowRunVisualizer;
