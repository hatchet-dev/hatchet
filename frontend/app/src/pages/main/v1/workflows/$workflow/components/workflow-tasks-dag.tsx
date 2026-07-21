import stepRunNode, {
  NodeData,
} from '../../../workflow-runs-v1/$run/v2components/step-run-node';
import { useTheme } from '@/components/hooks/use-theme';
import { WorkflowVersionTask } from '@/lib/api';
import dagre from 'dagre';
import { useMemo } from 'react';
import ReactFlow, {
  Position,
  MarkerType,
  Node,
  Edge,
  BezierEdge,
} from 'reactflow';
import 'reactflow/dist/style.css';

const connectionLineStyleDark = { stroke: '#fff' };
const connectionLineStyleLight = { stroke: '#000' };

const nodeTypes = {
  stepNode: stepRunNode,
};

const edgeTypes = {
  smoothstep: BezierEdge,
};

const nodeWidth = 230;
const nodeHeight = 70;

function getLayoutedElements(nodes: Node[], edges: Edge[], direction = 'LR') {
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
}

const WorkflowTasksDag = ({
  tasks,
  selectedTaskId,
  onSelectTask,
}: {
  tasks: WorkflowVersionTask[];
  selectedTaskId?: string;
  onSelectTask: (readableId: string) => void;
}) => {
  const { theme } = useTheme();

  const edges: Edge[] = useMemo(
    () =>
      tasks.flatMap((task) =>
        task.parents
          .filter((parentId) => tasks.some((t) => t.readableId === parentId))
          .map((parentId) => ({
            id: `${parentId}-${task.readableId}`,
            source: parentId,
            target: task.readableId,
            style:
              theme === 'dark'
                ? connectionLineStyleDark
                : connectionLineStyleLight,
            markerEnd: {
              type: MarkerType.ArrowClosed,
            },
            type: 'smoothstep',
          })),
      ),
    [tasks, theme],
  );

  const nodes: Node[] = useMemo(
    () =>
      tasks.map((task) => {
        const hasParent = task.parents.length > 0;
        const hasChild = tasks.some((t) => t.parents.includes(task.readableId));

        const data: NodeData = {
          taskRun: undefined,
          graphVariant:
            hasParent && hasChild
              ? 'default'
              : hasChild
                ? 'output_only'
                : hasParent
                  ? 'input_only'
                  : 'none',
          onClick: () => onSelectTask(task.readableId),
          childWorkflowsCount: 0,
          taskName: task.readableId,
        };

        return {
          id: task.readableId,
          type: 'stepNode',
          position: { x: 0, y: 0 },
          data,
          selected: task.readableId === selectedTaskId,
          selectable: true,
        };
      }),
    [tasks, selectedTaskId, onSelectTask],
  );

  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(nodes, edges),
    [nodes, edges],
  );

  return (
    <div className="h-[300px] w-full">
      <ReactFlow
        nodes={layoutedNodes}
        edges={layoutedEdges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        proOptions={{
          hideAttribution: true,
        }}
        onNodeClick={(_, node) => onSelectTask(node.id)}
        maxZoom={1}
        connectionLineStyle={
          theme === 'dark' ? connectionLineStyleDark : connectionLineStyleLight
        }
        snapToGrid={true}
      />
    </div>
  );
};

export default WorkflowTasksDag;
