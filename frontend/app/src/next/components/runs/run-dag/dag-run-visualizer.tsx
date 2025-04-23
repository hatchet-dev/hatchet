import { useCallback, useMemo } from 'react';
import ReactFlow, {
  Position,
  MarkerType,
  Node,
  Edge,
  BezierEdge,
} from 'reactflow';
import 'reactflow/dist/style.css';
import dagre from 'dagre';
import { useTheme } from '@/next/components/theme-provider';
import stepRunNode, { NodeData } from './step-run-node';
import { V1TaskStatus } from '@/lib/api';
import { RunDetailProvider, useRunDetail } from '@/next/hooks/use-run-detail';
import { useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
const connectionLineStyleDark = { stroke: '#fff' };
const connectionLineStyleLight = { stroke: '#000' };

const nodeTypes = {
  stepNode: stepRunNode,
};

const edgeTypes = {
  smoothstep: BezierEdge,
};

interface WorkflowRunVisualizerProps {
  workflowRunId: string;
  onTaskSelect?: (taskId: string) => void;
}

export function WorkflowRunVisualizer({
  workflowRunId,
  onTaskSelect,
}: WorkflowRunVisualizerProps) {
  return (
    <RunDetailProvider runId={workflowRunId}>
      <WorkflowRunVisualizerContent
        onTaskSelect={onTaskSelect}
        workflowRunId={workflowRunId}
      />
    </RunDetailProvider>
  );
}

function WorkflowRunVisualizerContent({
  workflowRunId,
  onTaskSelect,
}: WorkflowRunVisualizerProps) {
  const { theme } = useTheme();
  const navigate = useNavigate();
  const { data, isLoading, error } = useRunDetail();

  const shape = useMemo(() => data?.shape || [], [data]);
  const taskRuns = useMemo(() => data?.tasks || [], [data]);

  const setSelectedTaskRunId = useCallback(
    (taskRunId: string) => {
      if (onTaskSelect) {
        onTaskSelect(taskRunId);
      } else {
        navigate(ROUTES.runs.taskDetail(workflowRunId, taskRunId));
      }
    },
    [navigate, workflowRunId, onTaskSelect],
  );

  const edges: Edge[] = useMemo(
    () =>
      (
        shape.flatMap((shapeItem) =>
          shapeItem.childrenStepIds.map((childId) => {
            const child = shape.find((t) => t.stepId === childId);
            const childTaskRun = taskRuns.find((t) => t.stepId === childId);

            if (!child) {
              return null;
            }

            return {
              id: `${shapeItem.stepId}-${childId}`,
              source: shapeItem.stepId,
              target: childId,
              animated: childTaskRun?.status === V1TaskStatus.RUNNING,
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
      shape.map((shapeItem) => {
        const hasParent = shape.some((s) =>
          s.childrenStepIds.includes(shapeItem.stepId),
        );
        const hasChild = shape.some((s) => s.stepId === shapeItem.stepId);

        const task = taskRuns.find((t) => t.stepId === shapeItem.stepId);

        const data: NodeData = {
          taskRun: task,
          graphVariant:
            hasParent && hasChild
              ? 'default'
              : hasChild
                ? 'output_only'
                : 'input_only',
          onClick: () => task && setSelectedTaskRunId(task.metadata.id),
          childWorkflowsCount: task?.numSpawnedChildren || 0,
          taskName: shapeItem.taskName,
        };

        return {
          id: shapeItem.stepId,
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

  if (
    isLoading ||
    error ||
    layoutedNodes.length === 0 ||
    layoutedEdges.length === 0
  ) {
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
          const task = taskRuns.find((t) => t.stepId === node.id);

          if (task) {
            setSelectedTaskRunId(task.metadata.id);
          }
        }}
        maxZoom={1}
        connectionLineStyle={
          theme === 'dark' ? connectionLineStyleDark : connectionLineStyleLight
        }
        snapToGrid={true}
      />
    </div>
  );
}

export default WorkflowRunVisualizer;
