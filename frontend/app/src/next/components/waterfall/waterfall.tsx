import { useState, useMemo } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts';
import { ChevronDown, ChevronRight } from 'lucide-react';

import { ChartContainer, ChartTooltipContent } from '@/components/ui/chart';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api/queries';
import { V1TaskStatus, V1TaskTiming } from '@/lib/api';

interface ProcessedTaskData {
  id: string;
  taskDisplayName: string;
  parentId?: string;
  hasChildren: boolean;
  depth: number;
  isExpanded: boolean;
  offset: number;
  queuedDuration: number;
  ranDuration: number;
  status: V1TaskStatus;
  taskId: number; // Added for tie-breaking
}

interface ProcessedData {
  data: ProcessedTaskData[];
  taskPathMap: Map<string, string[]>;
}

// Waterfall component to render bars for queued, started, and finished durations
interface WaterfallProps {
  workflowRunId: string;
}

// Custom tooltip component that filters out offset entries
const CustomTooltip = (props: {
  active?: boolean;
  payload?: Array<{
    dataKey: string;
    name: string;
    value: number;
    [key: string]: unknown;
  }>;
  [key: string]: unknown;
}) => {
  const { active, payload } = props;

  if (active && payload && payload.length) {
    // Filter out any offset entries from the tooltip
    const filteredPayload = payload.filter(
      (entry) => entry.dataKey !== 'offset' && entry.name !== 'Offset',
    );

    // Only render if we have visible entries
    if (filteredPayload.length > 0) {
      // Create modified props with filtered payload
      const modifiedProps = { ...props, payload: filteredPayload };
      return (
        <ChartTooltipContent
          {...modifiedProps}
          className="w-[150px] sm:w-[200px] font-mono text-xs sm:text-xs"
        />
      );
    }
  }

  return null;
};

export function Waterfall({ workflowRunId }: WaterfallProps) {
  const [depth, setDepth] = useState(2);
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set());
  const [autoExpandedInitially, setAutoExpandedInitially] = useState(false);

  // Query with the current depth
  const {
    data: taskData,
    isLoading,
    isError,
  } = useQuery({
    ...queries.v1WorkflowRuns.listTaskTimings(workflowRunId, depth),
  });

  // Handle task expansion
  const toggleTaskExpansion = (
    taskId: string,
    hasChildren: boolean,
    taskDepth: number,
  ) => {
    const newExpandedTasks = new Set(expandedTasks);

    if (expandedTasks.has(taskId)) {
      // Collapse: remove this task from expanded set
      newExpandedTasks.delete(taskId);
    } else if (hasChildren) {
      // Expand: add this task to expanded set
      newExpandedTasks.add(taskId);

      // If expanding requires a deeper query, update the depth
      if (taskDepth + 1 >= depth) {
        setDepth(depth + 1);
      }
    }

    setExpandedTasks(newExpandedTasks);
  };

  // Transform and filter data based on expanded state
  const processedData = useMemo<ProcessedData>(() => {
    if (!taskData?.rows || taskData.rows.length === 0) {
      return { data: [], taskPathMap: new Map() };
    }

    // Create a map of task IDs to their data for quick lookups
    const taskMap = new Map<string, V1TaskTiming>();
    const rootTasks: string[] = [];
    const taskParentMap = new Map<string, string[]>();
    const taskDepthMap = new Map<string, number>();
    const taskPathMap = new Map<string, string[]>(); // For tracking task paths
    const taskHasChildrenMap = new Map<string, boolean>();

    // First pass: build the task map and parent-child relationships
    taskData.rows.forEach((task) => {
      if (task.metadata?.id) {
        taskMap.set(task.metadata.id, task);

        // Store if task has children (check if any rows have this task as a parent)
        const hasChildren = taskData.rows.some(
          (t) => t.parentTaskExternalId === task.metadata?.id,
        );

        taskHasChildrenMap.set(task.metadata.id, hasChildren);

        // Record task depth from API or calculate
        taskDepthMap.set(task.metadata.id, task.depth);

        // Record parent-child relationship
        if (task.depth == 0) {
          rootTasks.push(task.metadata.id);
        } else if (task.parentTaskExternalId) {
          if (!taskParentMap.has(task.parentTaskExternalId)) {
            taskParentMap.set(task.parentTaskExternalId, []);
          }
          const children = taskParentMap.get(task.parentTaskExternalId);
          if (children) {
            children.push(task.metadata.id);
          }
        }

        // Initialize path
        taskPathMap.set(task.metadata.id, [task.metadata.id]);
      }
    });

    // Second pass: build complete paths
    taskMap.forEach((task) => {
      if (
        task.metadata?.id &&
        task.parentTaskExternalId &&
        taskPathMap.has(task.parentTaskExternalId)
      ) {
        const parentPath = taskPathMap.get(task.parentTaskExternalId);
        if (parentPath) {
          taskPathMap.set(task.metadata.id, [...parentPath, task.metadata.id]);
        }
      }
    });

    // Auto-expand first set of root tasks with children
    if (!autoExpandedInitially && taskData) {
      // Find root tasks with children and expand them
      const rootTasksWithChildren = rootTasks.filter((id) =>
        taskHasChildrenMap.get(id),
      );
      if (rootTasksWithChildren.length > 0) {
        // Only expand the first time when data is available
        setTimeout(() => {
          const newExpandedTasks = new Set(expandedTasks);
          rootTasksWithChildren.forEach((id) => {
            newExpandedTasks.add(id);
          });
          setExpandedTasks(newExpandedTasks);
          setAutoExpandedInitially(true);
        }, 0);
      }
    }

    // Determine which tasks should be visible based on expanded state
    const visibleTasks = new Set<string>();

    // Always add root tasks to visible tasks
    rootTasks.forEach((id) => {
      visibleTasks.add(id);
    });

    // Add children of expanded tasks
    expandedTasks.forEach((expandedId) => {
      const children = taskParentMap.get(expandedId) || [];
      children.forEach((childId) => {
        visibleTasks.add(childId);
      });
    });

    // If there are no visible tasks, return early to avoid errors
    if (visibleTasks.size === 0) {
      return { data: [], taskPathMap: new Map() };
    }

    // Find the global minimum queuedAt time among visible tasks
    let globalMinQueuedAt = Number.MAX_SAFE_INTEGER;
    visibleTasks.forEach((id) => {
      const task = taskMap.get(id);
      if (task && task.queuedAt) {
        const queuedTime = new Date(task.queuedAt).getTime();
        if (queuedTime < globalMinQueuedAt) {
          globalMinQueuedAt = queuedTime;
        }
      }
    });

    // Create the processed data for rendering
    const data = Array.from(visibleTasks)
      .map((id) => {
        const task = taskMap.get(id);
        if (!task || !task.queuedAt || !task.startedAt || !task.finishedAt) {
          return null;
        }

        const queuedAt = new Date(task.queuedAt).getTime();
        const startedAt = new Date(task.startedAt).getTime();
        const finishedAt = new Date(task.finishedAt).getTime();

        return {
          id: task.metadata.id,
          taskDisplayName: task.taskDisplayName,
          parentId: task.parentTaskExternalId,
          hasChildren: taskHasChildrenMap.get(task.metadata.id) || false,
          depth: taskDepthMap.get(task.metadata.id) || 0,
          isExpanded: expandedTasks.has(task.metadata.id),
          // Chart data
          offset: (queuedAt - globalMinQueuedAt) / 1000, // in seconds
          queuedDuration: (startedAt - queuedAt) / 1000, // in seconds
          ranDuration: (finishedAt - startedAt) / 1000, // in seconds
          status: task.status,
          taskId: task.taskId, // Add taskId for tie-breaking in sorting
        };
      })
      .filter((task): task is NonNullable<typeof task> => task !== null)
      // Sort by task path for consistent ordering, break ties with taskId
      .sort((a, b) => {
        const pathA = taskPathMap.get(a.id) || [];
        const pathB = taskPathMap.get(b.id) || [];

        // Compare each path segment
        for (let i = 0; i < Math.min(pathA.length, pathB.length); i++) {
          if (pathA[i] !== pathB[i]) {
            const taskA = taskMap.get(pathA[i]);
            const taskB = taskMap.get(pathB[i]);

            if (taskA && taskB) {
              return taskA.taskId - taskB.taskId;
            }
            return 0;
          }
        }

        // If one path is a prefix of the other, shorter comes first
        if (pathA.length !== pathB.length) {
          return pathA.length - pathB.length;
        }

        // If paths are identical, break ties with taskId
        return a.taskId - b.taskId;
      });

    return { data, taskPathMap };
  }, [taskData, expandedTasks, depth, autoExpandedInitially]);

  // Custom tick renderer with expand/collapse buttons
  const renderTick = (props: {
    x: number;
    y: number;
    payload: { value: string };
  }) => {
    const { x, y, payload } = props;
    const task = processedData.data.find(
      (t) => t.taskDisplayName === payload.value,
    );
    if (!task) {
      // Return empty element instead of null
      return <g transform={`translate(${x},${y})`}></g>;
    }

    const label = payload.value;
    const truncatedLabel =
      label.length > 16 ? label.slice(0, 16) + '...' : label;
    const indentation = task.depth * 12; // 12px indentation per level

    return (
      <g transform={`translate(${x},${y})`}>
        <foreignObject
          x={-160} // Start position (right aligned)
          y={-10} // Vertically center
          width={160}
          height={20}
        >
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              paddingLeft: `${indentation}px`,
              height: '100%',
            }}
          >
            {/* Expand/collapse button */}
            <div
              style={{
                cursor: task.hasChildren ? 'pointer' : 'default',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: '20px',
                height: '20px',
                marginRight: '4px',
              }}
              onClick={() =>
                task.hasChildren &&
                toggleTaskExpansion(task.id, task.hasChildren, task.depth)
              }
            >
              {task.hasChildren &&
                (task.isExpanded ? (
                  <ChevronDown size={14} />
                ) : (
                  <ChevronRight size={14} />
                ))}
            </div>

            {/* Task label */}
            <div
              style={{
                whiteSpace: 'nowrap',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                fontSize: '12px',
                textAlign: 'left',
                flexGrow: 1,
              }}
            >
              {truncatedLabel}
            </div>
          </div>
        </foreignObject>
      </g>
    );
  };

  // Handle loading or error states
  if (
    isLoading ||
    isError ||
    !processedData.data ||
    processedData.data.length === 0
  ) {
    return null;
  }

  // Compute dynamic chart height
  const barSize = 14;
  const rowGap = 20;
  const chartHeight = processedData.data.length * (barSize + rowGap) + rowGap;

  const chartConfig = {
    queued: {
      label: 'Queued',
      color: `rgb(229 231 235 / 0.2)`,
    },
    runFor: {
      label: 'Ran For',
      color: 'rgb(99 102 241 / 0.8)',
    },
  };

  return (
    <ChartContainer
      config={chartConfig}
      className="w-full overflow-visible max-h-[300px] overflow-y-auto"
    >
      <div style={{ width: '100%', height: chartHeight }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={processedData.data}
            layout="vertical"
            margin={{
              top: 0,
              left: 0,
            }}
            barSize={barSize}
            barGap={rowGap}
          >
            <CartesianGrid horizontal={false} vertical={true} />
            <XAxis
              type="number"
              tickLine={false}
              axisLine={false}
              height={30}
              tickMargin={8}
              style={{ fontSize: '12px', userSelect: 'none', top: 0 }}
              tickFormatter={(v) => v.toString() + 's'}
              minTickGap={20}
              orientation="top"
            />
            <YAxis
              dataKey="taskDisplayName"
              type="category"
              width={180}
              axisLine={false}
              tickLine={false}
              tickMargin={8}
              style={{ userSelect: 'none' }}
              tick={renderTick}
            />
            <Tooltip content={<CustomTooltip />} />
            {/* Transparent offset bar for spacing */}
            <Bar
              dataKey="offset"
              name="Offset"
              stackId="a"
              fill="transparent"
              maxBarSize={barSize}
            />
            <Bar
              dataKey="queuedDuration"
              name="Queue time"
              stackId="a"
              fill={chartConfig.queued.color}
              maxBarSize={barSize}
            />
            <Bar
              dataKey="ranDuration"
              name="Running time"
              stackId="a"
              fill={chartConfig.runFor.color}
              maxBarSize={barSize}
            >
              {processedData.data.map((_, index) => {
                // TODO: modify color based on status
                // const color = RunStatusConfigs[entry.status].primary;
                return (
                  <Cell key={`cell-${index}`} fill={'rgb(99 102 241 / 0.8)'} />
                );
              })}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </ChartContainer>
  );
}
