import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import ReactFlow, {
  Position,
  MarkerType,
  Node,
  Edge,
  BezierEdge,
  ReactFlowInstance,
} from 'reactflow';
import 'reactflow/dist/style.css';
import dagre from 'dagre';
import { useTheme } from '@/next/components/theme-provider';
import stepRunNode, { NodeData } from './step-run-node';
import { V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { RunDetailProvider, useRunDetail } from '@/next/hooks/use-run-detail';
import { cn } from '@/lib/utils';
import { ChevronDownIcon, ChevronUpIcon } from '@radix-ui/react-icons';

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
  patchTask?: V1TaskSummary | null;
  onTaskSelect?: (taskId: string, childWfrId?: string) => void;
  selectedTaskId?: string;
}

function WorkflowRunVisualizer(props: WorkflowRunVisualizerProps) {
  return (
    <RunDetailProvider runId={props.workflowRunId}>
      <WorkflowRunVisualizerContent
      {...props}
      />
    </RunDetailProvider>
  );
}

function WorkflowRunVisualizerContent({
  onTaskSelect,
  selectedTaskId,
  patchTask
}: WorkflowRunVisualizerProps) {
  const { theme } = useTheme();
  const { data } = useRunDetail();
  const reactFlowInstance = useRef<ReactFlowInstance | null>(null);
  const [isExpanded, setIsExpanded] = useState(false);

  const shape = useMemo(() => data?.shape || [], [data]);
  const taskRuns = useMemo(() => data?.tasks.map((t) => patchTask ? patchTask : t) || [], [data, patchTask]);

  const [containerWidth, setContainerWidth] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);



  useEffect(() => {
    const resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerWidth(entry.contentRect.width);
      }
    });

    if (containerRef.current) {
      resizeObserver.observe(containerRef.current);
    }

    return () => {
      if (containerRef.current) {
        resizeObserver.unobserve(containerRef.current);
      }
    };
  }, []);

  const setSelectedTask = useCallback(
    (task: V1TaskSummary) => {
      if (onTaskSelect) {
        onTaskSelect(task.taskExternalId, task.workflowRunExternalId);
      }
    },
    [onTaskSelect],
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
          onClick: () => task && setSelectedTask(task),
          childWorkflowsCount: task?.numSpawnedChildren || 0,
          taskName: shapeItem.taskName,
          isSelected: selectedTaskId === task?.taskExternalId,
        };

        return {
          id: shapeItem.stepId,
          type: 'stepNode',
          position: { x: 0, y: 0 },
          data,
          selectable: true,
        };
      }) || [],
    [shape, taskRuns, setSelectedTask, selectedTaskId],
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
    dagreGraph.setGraph({
      rankdir: direction,
      nodesep: 50,
      ranksep: 50,
      edgesep: 10,
      acyclicer: 'greedy',
      ranker: 'network-simplex'
    });

    // Sort nodes by ID to ensure consistent ordering
    const sortedNodes = [...nodes].sort((a, b) => a.id.localeCompare(b.id));
    sortedNodes.forEach((node) => {
      dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
    });

    // Sort edges to ensure consistent ordering
    const sortedEdges = [...edges].sort((a, b) => {
      const sourceCompare = a.source.localeCompare(b.source);
      if (sourceCompare !== 0) return sourceCompare;
      return a.target.localeCompare(b.target);
    });
    sortedEdges.forEach((edge) => {
      dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    const layoutedNodes = sortedNodes.map((node) => {
      const nodeWithPosition = dagreGraph.node(node.id);
      node.targetPosition = isHorizontal ? Position.Left : Position.Top;
      node.sourcePosition = isHorizontal ? Position.Right : Position.Bottom;

      node.position = {
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      };

      return { ...node };
    });

    return { nodes: layoutedNodes, edges: sortedEdges };
  };

  const { nodes: layoutedNodes, edges: layoutedEdges } = useMemo(
    () => getLayoutedElements(nodes, edges),
    [nodes, edges],
  );

  // Track the last centered task ID to avoid unnecessary recentering
  const lastCenteredTaskId = useRef<string | undefined>(undefined);

  const recenter = useCallback(() => {

    setTimeout(() => {

      if (!reactFlowInstance.current) {
        return;
      }
    const node = layoutedNodes?.find((n: Node) => n.data.taskRun?.taskExternalId === selectedTaskId);

    if (node) {
        const centerX = node.position.x + (nodeWidth / 2) - 20;
        const centerY = node.position.y + (nodeHeight / 2) - 20;
        reactFlowInstance.current.setCenter(centerX, centerY, {
          duration: 800
        });
        lastCenteredTaskId.current = selectedTaskId;
    }
  }, 1);
  }, [selectedTaskId, layoutedNodes]);

  useEffect(() => {
    if (selectedTaskId === lastCenteredTaskId.current || !reactFlowInstance.current) {
      return;
    }

    if (selectedTaskId) {
      recenter();
    } else {
      // Fit the view to show all nodes with proper padding
      setTimeout(() => {
        // HACK this is a hack to ensure the fitView is called after the nodes are rendered
        if (reactFlowInstance.current) {
          reactFlowInstance.current.fitView({
            duration: 800,
            padding: 0.2,
            minZoom: 0.5,
            maxZoom: 1
          });
        }
      }, 1);
      lastCenteredTaskId.current = selectedTaskId;
    }
  }, [selectedTaskId, layoutedNodes]);


  const toggleExpand = useCallback(() => {
    setIsExpanded(!isExpanded);
    setTimeout(() => {
      recenter()
    }, 100);
  }, [isExpanded, recenter]);

  return (
    <div
      ref={containerRef}
      className={cn(
        "w-full relative transition-all duration-300 ease-in-out",
        isExpanded ? "h-[500px]" : "h-[160px]"
      )}
    >
      <ReactFlow
        key={containerWidth}
        nodes={layoutedNodes}
        edges={layoutedEdges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        fitView
        proOptions={{
          hideAttribution: true,
        }}
        onInit={(instance) => {
          reactFlowInstance.current = instance;
        }}
        onNodeClick={(_, node) => {
          const task = taskRuns.find((t) => t.stepId === node.id);

          if (task) {
            console.log('task', task);
            setSelectedTask(task);
          }
        }}
        maxZoom={1}
        connectionLineStyle={
          theme === 'dark' ? connectionLineStyleDark : connectionLineStyleLight
        }
        snapToGrid={true}
      />
      <button
        onClick={toggleExpand}
        className={cn(
          "absolute inset-x-0 -bottom-2 z-20 h-4 transition-all ease-linear after:absolute after:inset-x-0 after:top-1/2 after:h-[2px] hover:after:bg-border",
          isExpanded ? "cursor-n-resize" : "cursor-s-resize"
        )}
        title="Toggle height"
      />
      <button
        onClick={toggleExpand}
        className="absolute bottom-2 right-2 z-20 p-1 rounded-sm opacity-30 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
        title={isExpanded ? "Collapse" : "Expand"}
      >
        {isExpanded ? (
          <ChevronUpIcon className="h-4 w-4" />
        ) : (
          <ChevronDownIcon className="h-4 w-4" />
        )}
        <span className="sr-only">{isExpanded ? "Collapse" : "Expand"}</span>
      </button>
    </div>
  );
}

export default WorkflowRunVisualizer;
