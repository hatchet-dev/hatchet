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
import {
  ArrowDownFromLine,
  ChevronDown,
  ChevronRight,
  Loader,
} from 'lucide-react';

import { ChartContainer, ChartTooltipContent } from '@/components/ui/chart';
import { V1TaskStatus, V1TaskTiming } from '@/lib/api';
import { RunStatusConfigs } from '../runs/runs-badge';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { Button } from '../ui/button';
import { RunId } from '../runs/run-id';
import { BsArrowUpLeftCircle } from 'react-icons/bs';
import { Skeleton } from '../ui/skeleton';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';
import {
  TooltipProvider,
  Tooltip as BaseTooltip,
  TooltipContent,
  TooltipTrigger,
} from '../ui/tooltip';

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
  taskId: number; // Added for tie-breaking
  attempt: number;
}

interface ProcessedData {
  data: ProcessedTaskData[];
  taskPathMap: Map<string, string[]>;
}

// Waterfall component to render bars for queued, started, and finished durations
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
    // Filter out any offset entries from the tooltip
    const filteredPayload = payload.filter(
      (entry) => entry.dataKey !== 'offset' && entry.name !== 'Offset',
    );

    // Only render if we have visible entries
    if (filteredPayload.length > 0) {
      filteredPayload.forEach((entry, i) => {
        if (filteredPayload[i].payload && entry.dataKey === 'ranDuration') {
          const taskStatus = filteredPayload[0]?.payload?.status;

          if (taskStatus && RunStatusConfigs[taskStatus]) {
            filteredPayload[i].color =
              RunStatusConfigs[taskStatus].primaryOKLCH;
          }
        }
      });

      // Create modified props with filtered payload
      const modifiedProps = { ...props, payload: filteredPayload };

      return (
        <ChartTooltipContent
          {...modifiedProps}
          className="w-[150px] sm:w-[200px] font-mono text-xs sm:text-xs cursor-pointer"
        />
      );
    }
  }

  return null;
};

// Add this helper function before the Waterfall component
const inferTaskState = (
  tasks: V1TaskTiming[],
): {
  status: V1TaskStatus;
  startedAt: string | undefined;
  finishedAt: string | undefined;
  queuedAt: string | undefined;
} => {
  if (tasks.length === 0) {
    return {
      status: V1TaskStatus.QUEUED,
      startedAt: undefined,
      finishedAt: undefined,
      queuedAt: undefined,
    };
  }

  // Get all valid timestamps
  const startTimes = tasks
    .filter((t) => Boolean(t.startedAt))
    .map((t) => new Date(t.startedAt!).getTime());
  const finishedTimes = tasks
    .filter((t) => t.finishedAt)
    .map((t) => new Date(t.finishedAt!).getTime());
  const queueTimes = tasks
    .filter((t) => t.queuedAt)
    .map((t) => new Date(t.queuedAt!).getTime());

  // Infer status based on child tasks
  let status: V1TaskStatus = V1TaskStatus.QUEUED;
  const statusCounts = new Map<V1TaskStatus, number>();

  tasks.forEach((task) => {
    const count = statusCounts.get(task.status) || 0;
    statusCounts.set(task.status, count + 1);
  });

  // If any task failed, the group failed
  if (statusCounts.get(V1TaskStatus.FAILED)) {
    status = V1TaskStatus.FAILED;
  }
  // If any task is running, the group is running
  else if (statusCounts.get(V1TaskStatus.RUNNING)) {
    status = V1TaskStatus.RUNNING;
  }
  // If all tasks are completed, the group is completed
  else if (statusCounts.get(V1TaskStatus.COMPLETED) === tasks.length) {
    status = V1TaskStatus.COMPLETED;
  }
  // If any task is cancelled, the group is cancelled
  else if (statusCounts.get(V1TaskStatus.CANCELLED)) {
    status = V1TaskStatus.CANCELLED;
  }

  // Calculate timing information
  const earliestStart =
    startTimes.length > 0 ? new Date(Math.min(...startTimes)) : undefined;
  const earliestQueue =
    queueTimes.length > 0 ? new Date(Math.min(...queueTimes)) : undefined;

  // Handle finishedAt time based on status
  let latestFinished: Date | undefined;
  if (
    status === V1TaskStatus.COMPLETED ||
    status === V1TaskStatus.FAILED ||
    status === V1TaskStatus.CANCELLED
  ) {
    // Only use finished times if all tasks have finished
    if (finishedTimes.length === tasks.length) {
      latestFinished = new Date(Math.max(...finishedTimes));
    }
  } else if (status === V1TaskStatus.RUNNING) {
    // For running tasks, use the latest finished time of completed tasks
    // or the current time if no tasks have finished yet
    if (finishedTimes.length > 0) {
      latestFinished = new Date(Math.max(...finishedTimes));
    } else {
      latestFinished = new Date(); // Current time for running tasks
    }
  }

  return {
    status,
    startedAt: earliestStart?.toISOString(),
    finishedAt: latestFinished?.toISOString(),
    queuedAt: earliestQueue?.toISOString(),
  };
};

export function Waterfall({
  workflowRunId,
  selectedTaskId,
  handleTaskSelect,
}: WaterfallProps) {
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set());
  const [autoExpandedInitially, setAutoExpandedInitially] = useState(false);

  const {
    timings: { data: taskData, isLoading, error: isError, depth, setDepth },
  } = useRunDetail();
  const { tenantId } = useCurrentTenantId();

  // Process and memoize task relationships to allow collapsing all descendants
  const taskRelationships = useMemo(() => {
    if (!taskData?.rows || taskData.rows.length === 0) {
      return {
        taskMap: new Map<string, V1TaskTiming>(),
        taskParentMap: new Map<string, string[]>(),
        taskDescendantsMap: new Map<string, Set<string>>(),
      };
    }

    // Create maps for lookup
    const taskMap = new Map<string, V1TaskTiming>();
    const taskParentMap = new Map<string, string[]>();
    const taskHasChildrenMap = new Map<string, boolean>();
    const taskDepthMap = new Map<string, number>();
    const rootTasks: string[] = [];

    // First pass: build basic maps
    taskData.rows.forEach((task) => {
      if (task.metadata?.id) {
        taskMap.set(task.metadata.id, task);

        // Record parent-child relationship
        if (task.parentTaskExternalId) {
          // Check if parent exists in our dataset
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
            // If parent doesn't exist in our dataset, treat this as a root task
            rootTasks.push(task.metadata.id);
          }
        } else {
          rootTasks.push(task.metadata.id);
        }

        // Store if task has children
        const hasChildren = taskData.rows.some(
          (t) => t.parentTaskExternalId === task.metadata?.id,
        );
        taskHasChildrenMap.set(task.metadata.id, hasChildren);
        taskDepthMap.set(task.metadata.id, task.depth);
      }
    });

    // Build descendant map to support recursive collapsing
    const taskDescendantsMap = new Map<string, Set<string>>();

    // Function to get all descendants of a task
    const getDescendants = (taskId: string): Set<string> => {
      // If we've already calculated this, return the cached result
      if (taskDescendantsMap.has(taskId)) {
        return taskDescendantsMap.get(taskId)!;
      }

      const descendants = new Set<string>();
      const children = taskParentMap.get(taskId) || [];

      // Add direct children
      children.forEach((childId) => {
        descendants.add(childId);

        // Recursively add their descendants
        const childDescendants = getDescendants(childId);
        childDescendants.forEach((descendantId) => {
          descendants.add(descendantId);
        });
      });

      // Cache the result
      taskDescendantsMap.set(taskId, descendants);
      return descendants;
    };

    // Calculate descendants for all tasks
    taskMap.forEach((_, taskId) => {
      getDescendants(taskId);
    });

    // Group tasks by workflowRunId at each level
    const groupTasksByWorkflowRun = (
      parentId: string | undefined,
      depth: number,
    ) => {
      const children = parentId ? taskParentMap.get(parentId) || [] : rootTasks;

      // Group children by workflowRunId
      const groups = new Map<string, string[]>();
      children.forEach((childId: string) => {
        const task = taskMap.get(childId);
        if (task?.workflowRunId) {
          if (!groups.has(task.workflowRunId)) {
            groups.set(task.workflowRunId, []);
          }
          groups.get(task.workflowRunId)?.push(childId);
        }
      });

      // Create phantom parents for groups with multiple tasks
      groups.forEach((taskIds, workflowRunId) => {
        if (taskIds.length > 1) {
          const tasks = taskIds.map((id) => taskMap.get(id)!);

          // Infer state from child tasks
          const inferredState = inferTaskState(tasks);

          // Create phantom parent
          const phantomParent: V1TaskTiming = {
            depth: depth,
            startedAt: inferredState.startedAt,
            finishedAt: inferredState.finishedAt,
            queuedAt: inferredState.queuedAt,
            metadata: {
              id: workflowRunId,
              createdAt: inferredState.startedAt || new Date().toISOString(),
              updatedAt: inferredState.finishedAt || new Date().toISOString(),
            },
            taskDisplayName: `dag`, // TODO: change to workflow run name
            taskExternalId: workflowRunId,
            taskId: -1,
            status: inferredState.status,
            workflowRunId,
            taskInsertedAt: inferredState.startedAt || new Date().toISOString(),
            tenantId: tasks[0].tenantId,
            parentTaskExternalId: tasks[0].parentTaskExternalId,
          };

          // Add phantom parent to task map and update parent-child relationships
          taskMap.set(phantomParent.metadata.id, phantomParent);
          taskHasChildrenMap.set(phantomParent.metadata.id, true);
          taskDepthMap.set(phantomParent.metadata.id, depth);

          // Update children to point to phantom parent
          taskIds.forEach((childId) => {
            const child = taskMap.get(childId);
            child.parentTaskExternalId = phantomParent.metadata.id;
            child.depth = depth + 1;
            taskDepthMap.set(childId, depth + 1);
          });

          // Update parent's children list
          if (parentId) {
            const parentChildren = taskParentMap.get(parentId) || [];
            const newParentChildren = parentChildren.filter(
              (id) => !taskIds.includes(id),
            );
            newParentChildren.push(phantomParent.metadata.id);
            taskParentMap.set(parentId, newParentChildren);
          } else {
            // For root level, remove the original tasks and add the phantom parent
            rootTasks.length = 0;
            rootTasks.push(phantomParent.metadata.id);
          }

          // Add phantom parent's children
          taskParentMap.set(phantomParent.metadata.id, taskIds);

          // Recursively process children
          taskIds.forEach((childId) => {
            groupTasksByWorkflowRun(childId, depth + 2);
          });
        } else {
          // Process single task's children
          groupTasksByWorkflowRun(taskIds[0], depth + 1);
        }
      });
    };

    // Start grouping from root level
    groupTasksByWorkflowRun(undefined, 0);

    return {
      taskMap,
      taskParentMap,
      taskDescendantsMap,
      taskHasChildrenMap,
      taskDepthMap,
      rootTasks,
    };
  }, [taskData]); // Only recompute when taskData changes

  const closeTask = useCallback(
    (taskId: string) => {
      const newExpandedTasks = new Set(expandedTasks);
      newExpandedTasks.delete(taskId);

      // Get all descendants and remove them from expanded set
      const descendants =
        taskRelationships.taskDescendantsMap.get(taskId) || new Set<string>();
      descendants.forEach((descendantId) => {
        newExpandedTasks.delete(descendantId);
      });

      // Also remove any descendants that might be in the expanded set
      // This handles the case where some descendants might not be in the taskDescendantsMap
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

      // If expanding requires a deeper query, update the depth
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

    // Find the global minimum time (queuedAt or startedAt) among visible tasks
    const globalMinTime = [...visibleTasks].reduce((acc, id) => {
      const task = taskMap.get(id);

      if (task) {
        // Use queuedAt if available, otherwise use startedAt
        const minTime = task.queuedAt
          ? new Date(task.queuedAt).getTime()
          : task.startedAt
            ? new Date(task.startedAt).getTime()
            : null;

        if (minTime !== null && minTime < acc) {
          return minTime;
        }
      }

      return acc;
    }, Number.MAX_SAFE_INTEGER);

    // Create the processed data for rendering
    const data = [...visibleTasks]
      .map((id) => {
        const task = taskMap.get(id);
        if (!task) {
          return null;
        }

        // Handle missing queuedAt by defaulting to startedAt (no queue time)
        const queuedAt = task.queuedAt
          ? new Date(task.queuedAt).getTime()
          : task.startedAt
            ? new Date(task.startedAt).getTime()
            : new Date(task.taskInsertedAt).getTime();

        const startedAt = task.startedAt
          ? new Date(task.startedAt).getTime()
          : null;

        // For running tasks, always use current time as finishedAt
        const now = new Date().getTime();
        const finishedAt =
          task.status === V1TaskStatus.RUNNING
            ? now
            : task.finishedAt
              ? new Date(task.finishedAt).getTime()
              : startedAt;

        return {
          id: task.metadata.id,
          taskExternalId: task.taskExternalId,
          taskDisplayName: task.taskDisplayName,
          parentId: task.parentTaskExternalId,
          hasChildren: taskHasChildrenMap.get(task.metadata.id) || false,
          depth: taskDepthMap.get(task.metadata.id) || 0,
          isExpanded: expandedTasks.has(task.metadata.id),
          workflowRunId: task.workflowRunId,
          // Chart data
          offset: (queuedAt - globalMinTime) / 1000, // in seconds
          // If queuedAt equals startedAt (due to our fallback logic), then queuedDuration will be 0
          queuedDuration: startedAt
            ? task.queuedAt
              ? (startedAt - queuedAt) / 1000
              : 0
            : null, // in seconds
          ranDuration:
            startedAt && finishedAt ? (finishedAt - startedAt) / 1000 : null, // in seconds
          status: task.status,
          taskId: task.taskId, // Add taskId for tie-breaking in sorting
          attempt: task.attempt || 1,
        };
      })
      .filter((task): task is NonNullable<typeof task> => task !== null)
      // Sort by task path for consistent ordering, break ties with taskId
      .sort((a, b) => {
        // First sort by depth
        if (a.depth !== b.depth) {
          return a.depth - b.depth;
        }

        // If depths are equal, check if one is a phantom parent
        const aIsPhantom = a.id.startsWith('phantom-');
        const bIsPhantom = b.id.startsWith('phantom-');
        if (aIsPhantom !== bIsPhantom) {
          return aIsPhantom ? -1 : 1; // Phantom parents come first
        }

        // If both are phantom parents or both are regular tasks, sort by taskId
        return a.taskId - b.taskId;
      });

    return { data, taskPathMap: new Map() };
  }, [taskData, expandedTasks, autoExpandedInitially, taskRelationships]); // Only recompute when dependencies change

  // Handler for bar click events
  const handleBarClick = useCallback(
    (data: any) => {
      if (data?.id) {
        // Handle task selection for sidebar
        if (handleTaskSelect) {
          handleTaskSelect(data.id, data.workflowRunId);
        }

        // Handle expansion if the task has children
        if (data.hasChildren) {
          openTask(data.id, data.depth);
        }
      }
    },
    [handleTaskSelect, openTask],
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
          tenantId={tenantId}
        />
      );
    },
    [
      workflowRunId,
      selectedTaskId,
      handleBarClick,
      toggleTask,
      processedData,
      tenantId,
    ],
  );

  // Handle loading or error states
  if (
    !isLoading &&
    (isError || !processedData.data || processedData.data.length === 0)
  ) {
    return null;
  }

  if (isLoading) {
    return <Skeleton className="h-[100px] w-full" />;
  }

  // Compute dynamic chart height
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
            {/* Background bar for selected state */}
            <Bar
              dataKey="offset"
              name="Selected"
              stackId="a"
              fill="transparent"
              maxBarSize={BAR_SIZE + ROW_GAP}
              className="cursor-pointer"
            >
              {processedData.data.map((entry, index) => (
                <Cell
                  key={`selected-${index}`}
                  fill={
                    entry.id === selectedTaskId
                      ? 'rgb(99 102 241 / 0.1)'
                      : 'transparent'
                  }
                  width={10000} // Use a large fixed width to ensure full coverage
                />
              ))}
            </Bar>
            {/* Transparent offset bar for spacing */}
            <Bar
              dataKey="offset"
              name="Offset"
              stackId="a"
              fill="transparent"
              maxBarSize={BAR_SIZE}
              className="cursor-pointer"
            />
            <Bar
              dataKey="queuedDuration"
              name="Queue time"
              stackId="a"
              fill={chartConfig.queued.color}
              maxBarSize={BAR_SIZE}
              className="cursor-pointer"
            />
            <Bar
              dataKey="ranDuration"
              name="Running time"
              stackId="a"
              fill={chartConfig.runFor.color}
              maxBarSize={BAR_SIZE}
              className="cursor-pointer"
            >
              {processedData.data.map((entry, index) => {
                const color = RunStatusConfigs[entry.status].primaryOKLCH;

                return <Cell key={`cell-${index}`} fill={color} />;
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
  tenantId,
}: {
  x: number;
  y: number;
  payload: { value: string };
  workflowRunId: string;
  selectedTaskId?: string;
  handleBarClick: (task: ProcessedTaskData) => void;
  toggleTask: (taskId: string, hasChildren: boolean, taskDepth: number) => void;
  processedData: ProcessedData;
  tenantId: string;
}) => {
  const task = processedData.data.find(
    (t) => t.taskDisplayName === payload.value,
  );
  if (!task) {
    // Return empty element instead of null
    return <g transform={`translate(${x},${y})`}></g>;
  }

  const indentation = task.depth * 12; // 12px indentation per level

  return (
    <g transform={`translate(${x},${y})`}>
      <foreignObject
        x={-180} // Start position (right aligned)
        y={-10} // Vertically center
        width={180}
        height={20}
      >
        <div
          className={`group flex flex-row items-center pl-[${indentation}px] size-full`}
        >
          {/* Expand/collapse button */}
          <div
            data-haschildren={task.hasChildren}
            className="data-[haschildren=true]:cursor-info flex flex-row items-center justify-between size-[20px] h-[20px] mr-[4px]"
            onClick={() =>
              task.hasChildren &&
              toggleTask(task.id, task.hasChildren, task.depth)
            }
          >
            {task.hasChildren ? (
              task.isExpanded ? (
                <ChevronDown size={14} />
              ) : (
                <ChevronRight size={14} />
              )
            ) : null}
          </div>

          {/* Task label */}
          <div
            className="cursor-pointer flex flex-row justify-between w-full grow text-left text-xs overflow-auto text-ellipsis whitespace-nowrap gap-2 items-center"
            onClick={() => handleBarClick(task)}
          >
            <RunId
              displayName={task.taskDisplayName}
              id={task.id}
              onClick={() => handleBarClick(task)}
              className={task.id === selectedTaskId ? 'underline' : ''}
              attempt={task.attempt}
            />
          </div>
          {workflowRunId === task.workflowRunId &&
          task.taskExternalId === workflowRunId &&
          task.parentId ? (
            <Link
              to={ROUTES.runs.detail(tenantId, task.parentId)}
              onClick={(e) => e.stopPropagation()}
            >
              <Button
                tooltip="Zoom out to parent task"
                variant="link"
                size="icon"
                className="group-hover:opacity-100 opacity-0 transition-opacity duration-200"
              >
                <BsArrowUpLeftCircle className="w-4 h-4 transform" />
              </Button>
            </Link>
          ) : null}
          {task.hasChildren ? (
            <Link
              to={ROUTES.runs.detail(tenantId, task.workflowRunId || task.id)}
            >
              <Button
                tooltip="Drill into child task"
                variant="link"
                size="icon"
                className="group-hover:opacity-100 opacity-0 transition-opacity duration-200"
              >
                {' '}
                <ArrowDownFromLine className="w-4 h-4" />
              </Button>
            </Link>
          ) : null}
          {task.queuedDuration === null && (
            <TooltipProvider>
              <BaseTooltip>
                <TooltipTrigger>
                  <Loader className="animate-[spin_3s_linear_infinite] h-4" />
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
