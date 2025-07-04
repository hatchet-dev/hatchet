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

import { ChartContainer, ChartTooltipContent } from '@/components/v1/ui/chart';
import { V1TaskStatus, V1TaskTiming, queries } from '@/lib/api';
import { Link } from 'react-router-dom';
import { Button } from '@/components/v1/ui/button';
import { BsArrowUpLeftCircle } from 'react-icons/bs';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  TooltipProvider,
  Tooltip as BaseTooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useQuery } from '@tanstack/react-query';

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
          className="w-[150px] sm:w-[200px] font-mono text-xs sm:text-xs cursor-pointer"
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

  const { tenantId } = useCurrentTenantId();

  // Use v1 style queries instead of _next hooks
  const taskTimingsQuery = useQuery({
    ...queries.v1WorkflowRuns.listTaskTimings(workflowRunId, depth),
    refetchInterval: 1000,
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

    if (!autoExpandedInitially && taskData) {
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
        }, 0);
      }
    }

    const visibleTasks = new Set<string>();

    rootTasks.forEach((id) => {
      visibleTasks.add(id);
    });

    expandedTasks.forEach((expandedId) => {
      const children = taskParentMap.get(expandedId) || [];
      children.forEach((childId) => {
        visibleTasks.add(childId);
      });
    });

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

    console.log('🔍 Waterfall Debug - globalMinTime:', new Date(globalMinTime).toISOString());

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
        const queuedDuration = Math.max(0, (startedAt - queuedAt) / 1000);
        const ranDuration = Math.max(0, (finishedAt - startedAt) / 1000);
        const totalDuration = offset + queuedDuration + ranDuration;

        console.log(`🔍 Task: ${task.taskDisplayName}`, {
          depth: task.depth,
          queuedAt: task.queuedAt,
          startedAt: task.startedAt,
          finishedAt: task.finishedAt,
          calculatedTimes: {
            queuedAtMs: queuedAt,
            startedAtMs: startedAt,
            finishedAtMs: finishedAt,
          },
          durations: {
            offset: offset,
            queuedDuration: queuedDuration,
            ranDuration: ranDuration,
            total: totalDuration,
          },
          finishTimeFromStart: (finishedAt - globalMinTime) / 1000,
        });

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
      .filter((task): task is NonNullable<typeof task> => task !== null)
      .sort((a, b) => {
        if (a.depth !== b.depth) {
          return a.depth - b.depth;
        }

        return a.taskId - b.taskId;
      });

    console.log('🔍 Final waterfall data for chart:', data.map(d => ({
      taskDisplayName: d.taskDisplayName,
      depth: d.depth,
      offset: d.offset,
      queuedDuration: d.queuedDuration,
      ranDuration: d.ranDuration,
      totalFromStart: d.offset + d.queuedDuration + d.ranDuration,
    })));

    return { data, taskPathMap: new Map() };
  }, [taskData, expandedTasks, autoExpandedInitially, taskRelationships]);

  const handleBarClick = useCallback(
    (data: any) => {
      if (data?.id) {
        if (handleTaskSelect) {
          handleTaskSelect(data.id, data.workflowRunId);
        }

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
                const color = StatusColors[entry.status];
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
    return <g transform={`translate(${x},${y})`}></g>;
  }

  const indentation = task.depth * 12;

  return (
    <g transform={`translate(${x},${y})`}>
      <foreignObject x={-180} y={-10} width={180} height={20}>
        <div
          className={`group flex flex-row items-center pl-[${indentation}px] size-full`}
        >
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

          <div
            className="cursor-pointer flex flex-row justify-between w-full grow text-left text-xs overflow-auto text-ellipsis whitespace-nowrap gap-2 items-center"
            onClick={() => handleBarClick(task)}
          >
            <span
              className={`text-xs ${task.id === selectedTaskId ? 'underline' : ''}`}
              onClick={() => handleBarClick(task)}
            >
              {task.taskDisplayName}
            </span>
          </div>
          {workflowRunId === task.workflowRunId &&
          task.taskExternalId === workflowRunId &&
          task.parentId ? (
            <Link
              to={`/tenants/${tenantId}/runs/${task.parentId}`}
              onClick={(e) => e.stopPropagation()}
            >
              <TooltipProvider>
                <BaseTooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="link"
                      size="icon"
                      className="group-hover:opacity-100 opacity-0 transition-opacity duration-200"
                    >
                      <BsArrowUpLeftCircle className="w-4 h-4 transform" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Zoom out to parent task</TooltipContent>
                </BaseTooltip>
              </TooltipProvider>
            </Link>
          ) : null}
          {task.hasChildren ? (
            <Link
              to={`/tenants/${tenantId}/runs/${task.workflowRunId || task.id}`}
            >
              <TooltipProvider>
                <BaseTooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="link"
                      size="icon"
                      className="group-hover:opacity-100 opacity-0 transition-opacity duration-200"
                    >
                      <ArrowDownFromLine className="w-4 h-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Drill into child task</TooltipContent>
                </BaseTooltip>
              </TooltipProvider>
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
