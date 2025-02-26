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
import StepNode from './step-node';
import { WorkflowVersion } from '@/lib/api';
import dagre from 'dagre';
import { useTheme } from '@/components/v1/theme-provider';

// import ColorSelectorNode from './ColorSelectorNode';

const initBgColorDark = '#050c1c';
const initBgColorLight = '#ffffff';

const connectionLineStyleDark = { stroke: '#fff' };
const connectionLineStyleLight = { stroke: '#000' };

const nodeTypes = {
  stepNode: StepNode,
};

const WorkflowVisualizer = ({ workflow }: { workflow: WorkflowVersion }) => {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));

  useEffect(() => {
    const stepEdges =
      workflow.jobs
        ?.map((job) => {
          return job.steps
            .map((step) => {
              return (
                step.parents
                  ?.map((parent) => {
                    return {
                      id: `${parent}-${step.metadata.id}`,
                      source: parent,
                      target: step.metadata.id,
                      //   animated: true,
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
      workflow.jobs
        ?.map((job) => {
          return job.steps.map((step) => {
            const hasChild = stepEdges.some(
              (edge) => edge.source === step.metadata.id,
            );
            const hasParent = step.parents?.length && step.parents.length > 0;

            return {
              id: step.metadata.id,
              selectable: false,
              type: 'stepNode',
              position: { x: 0, y: 0 }, // positioning gets set by dagre later
              data: {
                step: step,
                variant:
                  hasParent && hasChild
                    ? 'default'
                    : hasChild
                      ? 'output_only'
                      : 'input_only',
              },
            };
          });
        })
        .flat() || [];

    setNodes(stepNodes);
    setEdges(stepEdges);
  }, [workflow, setNodes, setEdges]);

  const nodeWidth = 172;
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

  const { theme } = useTheme();

  const bgColor = useMemo(() => {
    return theme === 'dark' ? initBgColorDark : initBgColorLight;
  }, [theme]);

  const connectionLineStyle = useMemo(() => {
    return theme === 'dark'
      ? connectionLineStyleDark
      : connectionLineStyleLight;
  }, [theme]);

  return (
    <ReactFlow
      nodes={dagrNodes}
      edges={dagrEdges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      style={{ background: bgColor }}
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
  );
};

export default WorkflowVisualizer;
