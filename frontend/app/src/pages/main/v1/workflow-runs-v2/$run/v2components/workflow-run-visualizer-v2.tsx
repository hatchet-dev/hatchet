import { useEffect, useMemo } from 'react';
import ReactFlow, {
  useNodesState,
  useEdgesState,
  Position,
  MarkerType,
  Node,
  Edge,
} from 'reactflow';
import 'reactflow/dist/style.css';
import StepRunNode, { StepRunNodeProps } from './step-run-node';
import { StepRun, StepRunStatus, WorkflowRunShape } from '@/lib/api';
import dagre from 'dagre';
import invariant from 'tiny-invariant';
import { useTheme } from '@/components/v1/theme-provider';

const connectionLineStyleDark = { stroke: '#fff' };
const connectionLineStyleLight = { stroke: '#000' };

const nodeTypes = {
  stepNode: StepRunNode,
};

const WorkflowRunVisualizer = ({
  shape,
  selectedStepRunId,
  setSelectedStepRunId,
}: {
  shape: WorkflowRunShape;
  selectedStepRunId?: string;
  setSelectedStepRunId: (stepRunId: string) => void;
}) => {
  const { theme } = useTheme();

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));

  useEffect(() => {
    const stepEdges =
      shape.jobRuns
        ?.map((jobRun) => {
          invariant(jobRun.stepRuns, 'has stepRuns');
          return jobRun.stepRuns
            .map((stepRun: StepRun) => {
              const step = jobRun.job?.steps.find((step) => {
                return step.metadata.id === stepRun.stepId;
              });

              invariant(step, 'has step');

              return (
                step.parents
                  ?.map((parent) => {
                    return {
                      id: `${parent}-${step.metadata.id}`,
                      source: parent,
                      target: step.metadata.id,
                      animated: stepRun.status === StepRunStatus.RUNNING,
                      markerEnd: {
                        type: MarkerType.ArrowClosed,
                      },
                    };
                  })
                  .flat() || []
              );
            })
            .flat();
        })
        .flat() || [];

    const stepNodes =
      shape.jobRuns
        ?.map((jobRun) => {
          invariant(jobRun.stepRuns, 'has stepRuns');

          return jobRun.stepRuns.map((stepRun) => {
            // find the step in the shape
            const step = jobRun.job?.steps.find((step) => {
              return step.metadata.id === stepRun.stepId;
            });

            invariant(step, 'has step');

            const hasChild = stepEdges.some((edge) => {
              return edge?.source === step.metadata.id;
            });
            const hasParent = step?.parents?.length && step.parents.length > 0;

            const data: StepRunNodeProps = {
              stepRun: stepRun,
              onClick: () => {
                setSelectedStepRunId(stepRun?.metadata.id);
              },
              step: step,
              graphVariant:
                hasParent && hasChild
                  ? 'default'
                  : hasChild
                    ? 'output_only'
                    : 'input_only',
            };

            return {
              id: step.metadata.id,
              selectable: false,
              type: 'stepNode',
              position: { x: 0, y: 0 }, // positioning gets set by dagre later
              data,
            };
          });
        })
        .flat() || [];

    setNodes(stepNodes);
    setEdges(stepEdges);
  }, [shape, setNodes, setEdges, setSelectedStepRunId, selectedStepRunId]);

  const nodeWidth = 230;
  const nodeHeight = 70;

  const getLayoutedElements = (
    nodes: Node[],
    edges: Edge[],
    direction = 'LR',
  ) => {
    const isHorizontal = direction === 'LR';
    dagreGraph.setGraph({ rankdir: direction });

    nodes.forEach((node) => {
      dagreGraph.setNode(node.id, { width: nodeWidth, height: nodeHeight });
    });

    edges.forEach((edge) => {
      dagreGraph.setEdge(edge.source, edge.target);
    });

    dagre.layout(dagreGraph);

    nodes.forEach((node) => {
      const nodeWithPosition = dagreGraph.node(node.id);
      node.targetPosition = isHorizontal ? Position.Left : Position.Top;
      node.sourcePosition = isHorizontal ? Position.Right : Position.Bottom;

      // We are shifting the dagre node position (anchor=center center) to the top left
      // so it matches the React Flow node anchor point (top left).
      node.position = {
        x: nodeWithPosition.x - nodeWidth / 2,
        y: nodeWithPosition.y - nodeHeight / 2,
      };

      return node;
    });

    return { nodes, edges };
  };

  const dagrLayout = getLayoutedElements(nodes, edges);

  const dagrNodes = dagrLayout.nodes;
  const dagrEdges = dagrLayout.edges;

  const connectionLineStyle = useMemo(() => {
    return theme === 'dark'
      ? connectionLineStyleDark
      : connectionLineStyleLight;
  }, [theme]);

  return (
    <div className="w-full h-[300px]">
      <ReactFlow
        nodes={dagrNodes}
        edges={dagrEdges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        connectionLineStyle={connectionLineStyle}
        snapToGrid={true}
        fitView
        proOptions={{
          hideAttribution: true,
        }}
        className="border-1 border-gray-800 rounded-lg"
        maxZoom={1}
      />
    </div>
  );
};

export default WorkflowRunVisualizer;
