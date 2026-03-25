import { Button } from '@/components/v1/ui/button';
import { ChartContainer, ChartTooltipContent } from '@/components/v1/ui/chart';
import { Skeleton } from '@/components/v1/ui/skeleton';
import {
  TooltipProvider,
  Tooltip as BaseTooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { V1TaskStatus, V1TaskTiming, queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { CirclePlus, CircleMinus, Loader } from 'lucide-react';
import { useState, useMemo, useCallback } from 'react';
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

// Helper function to check if a task is a descendant of another task
function isDescendantOf(
  task: ProcessedTaskData,
  ancestorId: string,
  allTasks: ProcessedTaskData[],
): boolean {
  let currentParentId = task.parentId;

  while (currentParentId) {
    if (currentParentId === ancestorId) {
      return true;
    }

    // Find the parent task to continue traversing up
    const parentTaskId = currentParentId;
    const parentTask = allTasks.find((t) => t.id === parentTaskId);
    if (!parentTask) {
      break;
    }
    currentParentId = parentTask.parentId;
  }

  return false;
}

// Helper function to sort tasks in preorder traversal
function sortTasksPreorder(
  tasks: ProcessedTaskData[],
  taskParentMap: Map<string, string[]>,
  rootTasks: string[],
): ProcessedTaskData[] {
  const result: ProcessedTaskData[] = [];
  const taskMap = new Map<string, ProcessedTaskData>();

  // Create a map for quick lookup
  tasks.forEach((task) => {
    taskMap.set(task.id, task);
  });

  // Recursive function to add tasks in preorder
  function addTasksPreorder(taskId: string) {
    const task = taskMap.get(taskId);
    if (task) {
      result.push(task);

      // Add children in order
      const children = taskParentMap.get(taskId) || [];
      // Sort children by their taskId for consistent ordering
      const sortedChildren = children
        .map((childId) => ({ childId, task: taskMap.get(childId) }))
        .filter(({ task }) => task !== undefined)
        .sort((a, b) => (a.task!.taskId || 0) - (b.task!.taskId || 0))
        .map(({ childId }) => childId);

      sortedChildren.forEach((childId) => {
        addTasksPreorder(childId);
      });
    }
  }

  // Start with root tasks, sorted by taskId
  const sortedRootTasks = rootTasks
    .map((rootId) => ({ rootId, task: taskMap.get(rootId) }))
    .filter(({ task }) => task !== undefined)
    .sort((a, b) => (a.task!.taskId || 0) - (b.task!.taskId || 0))
    .map(({ rootId }) => rootId);

  sortedRootTasks.forEach((rootId) => {
    addTasksPreorder(rootId);
  });

  return result;
}

// Status color configuration compatible with v1
const StatusColors: Record<V1TaskStatus, string> = {
  [V1TaskStatus.COMPLETED]: '#10b981', // green-500
  [V1TaskStatus.FAILED]: '#ef4444', // red-500
  [V1TaskStatus.CANCELLED]: '#ef4444', // red-500
  [V1TaskStatus.RUNNING]: '#f59e0b', // amber-500
  [V1TaskStatus.QUEUED]: '#6b7280', // gray-500
};

interface ProcessedTaskData {
  id: string;
  taskExternalId: string;
  workflowRunId?: string;
  taskDisplayName: string;
  parentId?: string;
  hasChildren: boolean;
  depth: number;
  isExpanded: boolean;
  offset: number;
  queuedDuration: number | null;
  ranDuration: number | null;
  status: V1TaskStatus;
  taskId: number;
  attempt: number;
  isShowMoreEntry?: boolean;
  totalChildren?: number;
}

interface ProcessedData {
  data: ProcessedTaskData[];
  taskPathMap: Map<string, string[]>;
}

interface WaterfallProps {
  workflowRunId: string;
  selectedTaskId?: string;
  handleTaskSelect?: (taskId: string, childWorkflowRunId?: string) => void;
}

const CustomTooltip = (props: {
  active?: boolean;
  payload?: Array<{
    dataKey: string;
    name: string;
    value: number;
    payload?: V1TaskTiming;
    [key: string]: unknown;
  }>;
  [key: string]: unknown;
}) => {
  const { active, payload } = props;

  if (active && payload?.length) {
    const filteredPayload = payload.filter(
      (entry) => entry.dataKey !== 'offset' && entry.name !== 'Offset',
    );

    if (filteredPayload.length > 0) {
      filteredPayload.forEach((entry, i) => {
        if (filteredPayload[i].payload && entry.dataKey === 'ranDuration') {
          const taskStatus = filteredPayload[0]?.payload?.status;

          if (taskStatus && StatusColors[taskStatus]) {
            filteredPayload[i].color = StatusColors[taskStatus];
          }
        }
      });

      const modifiedProps = { ...props, payload: filteredPayload };

      return (
        <ChartTooltipContent
          {...modifiedProps}
          className="w-[150px] cursor-pointer font-mono text-xs sm:w-[200px] sm:text-xs"
        />
      );
    }
  }

  return null;
};

export function Waterfall({
  workflowRunId,
  selectedTaskId,
  handleTaskSelect,
}: WaterfallProps) {
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set());
  const [autoExpandedInitially, setAutoExpandedInitially] = useState(false);
  const [depth, setDepth] = useState(2);
  const [showAllChildren, setShowAllChildren] = useState<Set<string>>(
    new Set(),
  );
  const { refetchInterval } = useRefetchInterval();

  // Use v1 style queries instead of _next hooks
  const taskTimingsQuery = useQuery({
    ...queries.v1WorkflowRuns.listTaskTimings(workflowRunId, depth),
    refetchInterval,
    enabled: !!workflowRunId,
  });

  const taskData = taskTimingsQuery.data;
  const isLoading = taskTimingsQuery.isLoading;
  const isError = taskTimingsQuery.isError;

  // Process and memoize task relationships to allow collapsing all descendants
  const taskRelationships = useMemo(() => {
    if (!taskData?.rows || taskData.rows.length === 0) {
      return {
        taskMap: new Map<string, V1TaskTiming>(),
        taskParentMap: new Map<string, string[]>(),
        taskDescendantsMap: new Map<string, Set<string>>(),
        taskHasChildrenMap: new Map<string, boolean>(),
        taskDepthMap: new Map<string, number>(),
        rootTasks: [],
      };
    }

    const taskMap = new Map<string, V1TaskTiming>();
    const taskParentMap = new Map<string, string[]>();
    const taskHasChildrenMap = new Map<string, boolean>();
    const taskDepthMap = new Map<string, number>();
    const rootTasks: string[] = [];

    // First pass: build basic maps
    taskData.rows.forEach((task) => {
      if (task.metadata?.id) {
        taskMap.set(task.metadata.id, task);

        if (task.parentTaskExternalId) {
          const parentExists = taskData.rows.some(
            (t) => t.metadata?.id === task.parentTaskExternalId,
          );

          if (parentExists) {
            if (!taskParentMap.has(task.parentTaskExternalId)) {
              taskParentMap.set(task.parentTaskExternalId, []);
            }
            const children = taskParentMap.get(task.parentTaskExternalId);
            if (children) {
              children.push(task.metadata.id);
            }
          } else {
            rootTasks.push(task.metadata.id);
          }
        } else {
          rootTasks.push(task.metadata.id);
        }

        const hasChildren = taskData.rows.some(
          (t) => t.parentTaskExternalId === task.metadata?.id,
        );
        taskHasChildrenMap.set(task.metadata.id, hasChildren);
        taskDepthMap.set(task.metadata.id, task.depth);
      }
    });

    // Build descendant map to support recursive collapsing
    const taskDescendantsMap = new Map<string, Set<string>>();

    const getDescendants = (taskId: string): Set<string> => {
      if (taskDescendantsMap.has(taskId)) {
        const result = taskDescendantsMap.get(taskId);
        if (result !== undefined) {
          return result;
        }
      }
      const descendants = new Set<string>();
      const children = taskParentMap.get(taskId) || [];

      children.forEach((childId) => {
        descendants.add(childId);
        const childDescendants = getDescendants(childId);
        childDescendants.forEach((descendantId) => {
          descendants.add(descendantId);
        });
      });

      taskDescendantsMap.set(taskId, descendants);
      return descendants;
    };

    taskMap.forEach((_, taskId) => {
      getDescendants(taskId);
    });

    return {
      taskMap,
      taskParentMap,
      taskDescendantsMap,
      taskHasChildrenMap,
      taskDepthMap,
      rootTasks,
    };
  }, [taskData]);

  const closeTask = useCallback(
    (taskId: string) => {
      const newExpandedTasks = new Set(expandedTasks);
      newExpandedTasks.delete(taskId);

      const descendants =
        taskRelationships.taskDescendantsMap.get(taskId) || new Set<string>();
      descendants.forEach((descendantId) => {
        newExpandedTasks.delete(descendantId);
      });

      const processDescendants = (parentId: string) => {
        const children = taskRelationships.taskParentMap.get(parentId) || [];
        children.forEach((childId) => {
          newExpandedTasks.delete(childId);
          processDescendants(childId);
        });
      };

      processDescendants(taskId);
      setExpandedTasks(newExpandedTasks);
    },
    [expandedTasks, taskRelationships],
  );

  const openTask = useCallback(
    (taskId: string, taskDepth: number) => {
      const newExpandedTasks = new Set(expandedTasks);
      newExpandedTasks.add(taskId);

      if (taskDepth + 1 >= depth) {
        setDepth(depth + 1);
      }

      setExpandedTasks(newExpandedTasks);
    },
    [expandedTasks, setDepth, depth],
  );

  const toggleTask = useCallback(
    (taskId: string, hasChildren: boolean, taskDepth: number) => {
      if (!hasChildren) {
        return;
      }

      if (expandedTasks.has(taskId)) {
        closeTask(taskId);
      } else {
        openTask(taskId, taskDepth);
      }
    },
    [expandedTasks, closeTask, openTask],
  );

  const toggleShowAllChildren = useCallback(
    (taskId: string) => {
      const newShowAllChildren = new Set(showAllChildren);
      if (newShowAllChildren.has(taskId)) {
        newShowAllChildren.delete(taskId);
      } else {
        newShowAllChildren.add(taskId);
      }
      setShowAllChildren(newShowAllChildren);
    },
    [showAllChildren],
  );

  // Transform and filter data based on expanded state
  const processedData = useMemo<ProcessedData>(() => {
    if (!taskData?.rows || taskData.rows.length === 0) {
      return { data: [], taskPathMap: new Map() };
    }

    const {
      taskMap,
      taskParentMap,
      taskHasChildrenMap = new Map<string, boolean>(),
      taskDepthMap = new Map<string, number>(),
      rootTasks = [],
    } = taskRelationships;

    // Auto-expand root tasks with children on initial load
    if (!autoExpandedInitially && taskData && rootTasks.length > 0) {
      const rootTasksWithChildren = rootTasks.filter((id) =>
        taskHasChildrenMap.get(id),
      );

      if (rootTasksWithChildren.length > 0) {
        setTimeout(() => {
          const newExpandedTasks = new Set(expandedTasks);
          rootTasksWithChildren.forEach((id) => {
            newExpandedTasks.add(id);
          });
          setExpandedTasks(newExpandedTasks);
          setAutoExpandedInitially(true);
        }, 100);
      } else {
        setAutoExpandedInitially(true);
      }
    }

    const visibleTasks = new Set<string>();
    const showMoreEntries = new Map<string, number>(); // parentId -> totalChildren

    // Add root tasks first
    rootTasks.forEach((id) => {
      visibleTasks.add(id);
    });

    // Use a queue to process tasks level by level to ensure proper visibility
    const taskQueue = [...rootTasks];
    const processed = new Set<string>();

    while (taskQueue.length > 0) {
      const currentTaskId = taskQueue.shift()!;

      if (processed.has(currentTaskId)) {
        continue;
      }
      processed.add(currentTaskId);

      // Only process children if this task is expanded
      if (expandedTasks.has(currentTaskId)) {
        const children = taskParentMap.get(currentTaskId) || [];
        const shouldShowAll = showAllChildren.has(currentTaskId);

        if (shouldShowAll || children.length <= 20) {
          // Show all children
          children.forEach((childId) => {
            if (!visibleTasks.has(childId)) {
              visibleTasks.add(childId);
              taskQueue.push(childId);
            }
          });
        } else {
          // Show first 20 children and track the total for "show more"
          children.slice(0, 20).forEach((childId) => {
            if (!visibleTasks.has(childId)) {
              visibleTasks.add(childId);
              taskQueue.push(childId);
            }
          });
          showMoreEntries.set(currentTaskId, children.length);
        }
      }
    }

    if (visibleTasks.size === 0) {
      return { data: [], taskPathMap: new Map() };
    }

    const globalMinTime = [...visibleTasks].reduce((acc, id) => {
      const task = taskMap.get(id);

      if (task) {
        const minTime = task.queuedAt
          ? new Date(task.queuedAt).getTime()
          : task.startedAt
            ? new Date(task.startedAt).getTime()
            : new Date(task.taskInsertedAt).getTime();

        if (minTime < acc) {
          return minTime;
        }
      }

      return acc;
    }, Number.MAX_SAFE_INTEGER);

    const data = [...visibleTasks]
      .map((id) => {
        const task = taskMap.get(id);
        if (!task) {
          return null;
        }

        const queuedAt = task.queuedAt
          ? new Date(task.queuedAt).getTime()
          : task.startedAt
            ? new Date(task.startedAt).getTime()
            : new Date(task.taskInsertedAt).getTime();

        const startedAt = task.startedAt
          ? new Date(task.startedAt).getTime()
          : queuedAt;

        const now = new Date().getTime();
        const finishedAt =
          task.status === V1TaskStatus.RUNNING
            ? now
            : task.finishedAt
              ? new Date(task.finishedAt).getTime()
              : startedAt;

        const offset = Math.max(0, (queuedAt - globalMinTime) / 1000);
        const startedOffset = Math.max(0, (startedAt - globalMinTime) / 1000);
        const finishedOffset = Math.max(0, (finishedAt - globalMinTime) / 1000);

        const queuedDuration = Math.max(0, startedOffset - offset);
        const ranDuration = Math.max(0, finishedOffset - startedOffset);

        return {
          id: task.metadata.id,
          taskExternalId: task.taskExternalId,
          taskDisplayName: task.taskDisplayName,
          parentId: task.parentTaskExternalId,
          hasChildren: taskHasChildrenMap.get(task.metadata.id) || false,
          depth: taskDepthMap.get(task.metadata.id) || 0,
          isExpanded: expandedTasks.has(task.metadata.id),
          workflowRunId: task.workflowRunId,
          offset: offset,
          queuedDuration: queuedDuration,
          ranDuration: ranDuration,
          status: task.status,
          taskId: task.taskId,
          attempt: task.attempt || 1,
        };
      })
      .filter((task) => task !== null);

    // Sort tasks in preorder traversal first, then insert show more entries
    const sortedData = sortTasksPreorder(data, taskParentMap, rootTasks);

    // Insert "show more" entries at the correct positions
    const finalData: Array<{ entry: ProcessedTaskData; index: number }> = [];
    showMoreEntries.forEach((totalChildren, parentId) => {
      const parentTask = taskMap.get(parentId);
      if (parentTask) {
        const parentDepth = taskDepthMap.get(parentId) || 0;
        const showMoreEntry: ProcessedTaskData = {
          id: `${parentId}-show-more`,
          taskExternalId: `${parentId}-show-more`,
          taskDisplayName: `Show ${totalChildren - 20} more...`,
          parentId: parentId,
          hasChildren: false,
          depth: parentDepth + 1,
          isExpanded: false,
          offset: 0,
          queuedDuration: 0,
          ranDuration: 0,
          status: V1TaskStatus.COMPLETED,
          taskId: 0,
          attempt: 1,
          isShowMoreEntry: true,
          totalChildren: totalChildren,
        };

        // Find the position where this show more entry should be inserted
        // It should go after the last visible child of the parent AND their descendants
        let insertIndex = -1;

        // Find the last direct child of the parent
        let lastChildIndex = -1;
        for (let i = sortedData.length - 1; i >= 0; i--) {
          if (sortedData[i].parentId === parentId) {
            lastChildIndex = i;
            break;
          }
        }

        if (lastChildIndex !== -1) {
          // Find the end of all descendants of the last child
          const lastChildId = sortedData[lastChildIndex].id;
          insertIndex = lastChildIndex + 1;

          // Look for any descendants of the last child that come after it
          for (let i = lastChildIndex + 1; i < sortedData.length; i++) {
            const currentTask = sortedData[i];
            // Check if this task is a descendant of the last child
            if (isDescendantOf(currentTask, lastChildId, sortedData)) {
              insertIndex = i + 1;
            } else {
              // If we hit a task that's not a descendant, stop looking
              break;
            }
          }
        }

        if (insertIndex !== -1) {
          finalData.push({ entry: showMoreEntry, index: insertIndex });
        }
      }
    });

    // Build the final sorted data with show more entries inserted
    const result = [...sortedData];
    finalData
      .sort((a, b) => b.index - a.index) // Sort by index descending to insert from end
      .forEach(({ entry, index }) => {
        result.splice(index, 0, entry);
      });

    return { data: result, taskPathMap: new Map() };
  }, [
    taskData,
    expandedTasks,
    autoExpandedInitially,
    taskRelationships,
    showAllChildren,
  ]);

  const handleBarClick = useCallback(
    (data: any) => {
      if (data?.id) {
        // Handle "show more" entry clicks
        if (data.isShowMoreEntry) {
          const parentId = data.parentId;
          if (parentId) {
            toggleShowAllChildren(parentId);
          }
          return;
        }

        // Don't open the sheet if clicking on the current task run
        if (data.id !== workflowRunId && handleTaskSelect) {
          handleTaskSelect(data.id, data.workflowRunId);
        }

        // Remove the automatic expand behavior when clicking on a row
        // Users should use the expand/collapse buttons instead
      }
    },
    [handleTaskSelect, workflowRunId, toggleShowAllChildren],
  );

  const renderTick = useCallback(
    (props: { x: number; y: number; payload: { value: string } }) => {
      const { x, y, payload } = props;

      return (
        <Tick
          x={x}
          y={y}
          payload={payload}
          workflowRunId={workflowRunId}
          selectedTaskId={selectedTaskId}
          handleBarClick={handleBarClick}
          toggleTask={toggleTask}
          processedData={processedData}
        />
      );
    },
    [workflowRunId, selectedTaskId, handleBarClick, toggleTask, processedData],
  );

  if (
    !isLoading &&
    (isError || !processedData.data || processedData.data.length === 0)
  ) {
    return null;
  }

  if (isLoading) {
    return <Skeleton className="h-[100px] w-full" />;
  }

  const BAR_SIZE = 14;
  const ROW_GAP = 20;
  const ROW_HEIGHT = BAR_SIZE + ROW_GAP;
  const PADDING = ROW_GAP;
  const chartHeight = processedData.data.length * ROW_HEIGHT + PADDING;

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
      className="w-full overflow-visible"
      style={{ height: chartHeight }}
    >
      <div style={{ width: '100%', height: '100%' }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart
            data={processedData.data}
            layout="vertical"
            margin={{
              top: 0,
              left: 0,
            }}
            barSize={BAR_SIZE}
            barGap={ROW_GAP}
            onClick={(data) =>
              data && handleBarClick(data.activePayload?.[0]?.payload)
            }
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
              dataKey="id"
              type="category"
              width={180}
              axisLine={false}
              tickLine={false}
              tickMargin={8}
              style={{ userSelect: 'none' }}
              tick={renderTick}
            />
            <Tooltip content={<CustomTooltip />} />
            <Bar
              dataKey="offset"
              name="Selected"
              stackId="a"
              fill="transparent"
              maxBarSize={BAR_SIZE + ROW_GAP}
              className="cursor-pointer"
            >
              {processedData.data.map((entry) => (
                <Cell
                  key={`selected-${entry.id}`}
                  fill={
                    entry.id === selectedTaskId
                      ? 'rgb(99 102 241 / 0.1)'
                      : 'transparent'
                  }
                  width={10000}
                />
              ))}
            </Bar>
            <Bar
              dataKey="offset"
              name="Offset"
              stackId="a"
              fill="transparent"
              maxBarSize={BAR_SIZE}
            />
            <Bar
              dataKey="queuedDuration"
              name="Queue time"
              stackId="a"
              fill={chartConfig.queued.color}
              maxBarSize={BAR_SIZE}
            />
            <Bar
              dataKey="ranDuration"
              name="Running time"
              stackId="a"
              fill={chartConfig.runFor.color}
              maxBarSize={BAR_SIZE}
            >
              {processedData.data.map((entry) => {
                // For "show more" entries, use a different color
                if (entry.isShowMoreEntry) {
                  return (
                    <Cell
                      key={`cell-${entry.id}`}
                      fill="rgb(99 102 241 / 0.3)"
                    />
                  );
                }
                const color = StatusColors[entry.status];
                return <Cell key={`cell-${entry.id}`} fill={color} />;
              })}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </ChartContainer>
  );
}

const Tick = ({
  x,
  y,
  payload,
  workflowRunId,
  selectedTaskId,
  handleBarClick,
  toggleTask,
  processedData,
}: {
  x: number;
  y: number;
  payload: { value: string };
  workflowRunId: string;
  selectedTaskId?: string;
  handleBarClick: (task: ProcessedTaskData) => void;
  toggleTask: (taskId: string, hasChildren: boolean, taskDepth: number) => void;
  processedData: ProcessedData;
}) => {
  const task = processedData.data.find((t) => t.id === payload.value);
  if (!task) {
    return <g transform={`translate(${x},${y})`}></g>;
  }

  return (
    <g transform={`translate(${x},${y})`}>
      <foreignObject x={-160} y={-10} width={180} height={20}>
        <div
          className={`group flex size-full flex-row items-center`}
          style={{ paddingLeft: `${task.depth * 12}px` }}
        >
          <div
            className={`${task.id === workflowRunId ? 'cursor-default' : 'cursor-pointer'} flex w-full min-w-0 grow flex-row items-center justify-between gap-2 text-left text-xs`}
            onClick={() => handleBarClick(task)}
          >
            <span
              className={`text-xs ${task.id === selectedTaskId ? 'underline' : ''} ${task.isShowMoreEntry ? 'cursor-pointer text-indigo-600 hover:underline dark:text-indigo-400' : ''} truncate`}
              style={{ maxWidth: `${180 - task.depth * 12}px` }}
              title={task.taskDisplayName}
              onClick={() => handleBarClick(task)}
            >
              {task.taskDisplayName}
            </span>
          </div>
          {task.hasChildren && !task.isShowMoreEntry ? (
            <TooltipProvider>
              <BaseTooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="link"
                    size="icon"
                    className="opacity-0 transition-opacity duration-200 group-hover:opacity-100"
                    onClick={(e) => {
                      e.stopPropagation();
                      if (task.hasChildren) {
                        toggleTask(task.id, task.hasChildren, task.depth);
                      }
                    }}
                  >
                    {task.isExpanded ? (
                      <CircleMinus className="size-3" />
                    ) : (
                      <CirclePlus className="size-3" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>
                  {task.isExpanded ? 'Collapse children' : 'Expand children'}
                </TooltipContent>
              </BaseTooltip>
            </TooltipProvider>
          ) : null}
          {task.queuedDuration === null && !task.isShowMoreEntry && (
            <TooltipProvider>
              <BaseTooltip>
                <TooltipTrigger>
                  <Loader className="h-4 animate-[spin_3s_linear_infinite]" />
                </TooltipTrigger>
                <TooltipContent>This task has not started</TooltipContent>
              </BaseTooltip>
            </TooltipProvider>
          )}
        </div>
      </foreignObject>
    </g>
  );
};
