import {
  V1WorkflowRun,
  V1TaskEvent,
  V1TaskEventType,
  V1TaskSummary,
} from '@/lib/api';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { cn } from '@/next/lib/utils';
import { RunStatusConfigs } from '../runs-badge';
import { WorkflowRunStatus } from '@/lib/api';
import {
  CheckCircle2,
  PlayCircle,
  Clock,
  AlertCircle,
  XCircle,
  RefreshCw,
  PauseCircle,
  Timer,
  UserCog,
  Unlock,
  RotateCw,
  Send,
  AlertTriangle,
  Bell,
  Plus,
  SkipForward,
  ArrowUpRight,
  Info,
  Cpu,
  Bug,
  CpuIcon,
  ArrowUpCircle,
  ArrowDownCircle,
} from 'lucide-react';
import { VscJson } from 'react-icons/vsc';
import { Button } from '@/next/components/ui/button';
import { Time } from '@/next/components/ui/time';
import { useMemo, useEffect } from 'react';
import { RunId } from '../run-id';
import {
  FilterGroup,
  FilterText,
  FilterSelect,
} from '@/next/components/ui/filters/filters';
import { useFilters } from '@/next/hooks/utils/use-filters';
import { ROUTES } from '@/next/lib/routes';
import { FilterProvider } from '@/next/hooks/utils/use-filters';

interface RunEventLogProps {
  filters?: ActivityFilters;
  workflow: V1WorkflowRun;
  onTaskSelect?: (
    event: V1TaskEvent,
    options?: Parameters<typeof ROUTES.runs.detailWithSheet>[3],
  ) => void;
  showFilters?: {
    search?: boolean;
    eventType?: boolean;
    taskId?: boolean;
    attempt?: boolean;
  };
  showNextButton?: {
    label: string;
    onClick: () => void;
  };
  showPreviousButton?: {
    label: string;
    onClick: () => void;
  };
}

type EventConfig = {
  icon: React.ComponentType<{ className?: string }>;
  message: string | ((event: V1TaskEvent) => string);
  showWorkerButton?: boolean;
  status: WorkflowRunStatus;
  title: string;
};

const LogEventType = 'LOG_LINE' as V1TaskEventType;

interface ActivityFilters {
  search?: string;
  eventType?: V1TaskEventType[];
  taskId?: string[];
  attempt?: number;
}

const EVENT_CONFIG: Record<V1TaskEventType, EventConfig> = {
  [V1TaskEventType.FINISHED]: {
    icon: CheckCircle2,
    message: 'Task completed successfully',
    status: WorkflowRunStatus.SUCCEEDED,
    title: 'Task Finished',
  },
  [V1TaskEventType.STARTED]: {
    icon: PlayCircle,
    message: 'Task execution started',
    status: WorkflowRunStatus.RUNNING,
    title: 'Task Started',
  },
  [V1TaskEventType.ASSIGNED]: {
    icon: Cpu,
    message: (event) => `Assigned to worker ${event.workerId}`,
    showWorkerButton: true,
    status: WorkflowRunStatus.PENDING,
    title: 'Task Assigned',
  },
  [V1TaskEventType.QUEUED]: {
    icon: Clock,
    message: 'Task queued for execution',
    status: WorkflowRunStatus.PENDING,
    title: 'Task Queued',
  },
  [V1TaskEventType.FAILED]: {
    icon: AlertCircle,
    message: (event) => event.errorMessage || 'Task failed',
    status: WorkflowRunStatus.FAILED,
    title: 'Task Failed',
  },
  [V1TaskEventType.CANCELLED]: {
    icon: XCircle,
    message: 'Task cancelled',
    status: WorkflowRunStatus.CANCELLED,
    title: 'Task Cancelled',
  },
  [V1TaskEventType.RETRYING]: {
    icon: RefreshCw,
    message: 'Retrying task',
    status: WorkflowRunStatus.BACKOFF,
    title: 'Task Retrying',
  },
  [V1TaskEventType.TIMED_OUT]: {
    icon: Timer,
    message: 'Task timed out',
    status: WorkflowRunStatus.FAILED,
    title: 'Task Timed Out',
  },
  [V1TaskEventType.REASSIGNED]: {
    icon: UserCog,
    message:
      'Reassigned as the worker became inactive (did not heartbeat for more than 30 seconds)',
    showWorkerButton: true,
    status: WorkflowRunStatus.RUNNING,
    title: 'Task Reassigned',
  },
  [V1TaskEventType.SLOT_RELEASED]: {
    icon: Unlock,
    message: 'Worker slot released',
    status: WorkflowRunStatus.PENDING,
    title: 'Slot Released',
  },
  [V1TaskEventType.TIMEOUT_REFRESHED]: {
    icon: RotateCw,
    message: 'Task timeout refreshed',
    status: WorkflowRunStatus.RUNNING,
    title: 'Timeout Refreshed',
  },
  [V1TaskEventType.RETRIED_BY_USER]: {
    icon: RefreshCw,
    message: 'Task retried by user',
    status: WorkflowRunStatus.BACKOFF,
    title: 'User Retry',
  },
  [V1TaskEventType.SENT_TO_WORKER]: {
    icon: Send,
    message: 'Task sent to worker',
    status: WorkflowRunStatus.RUNNING,
    title: 'Sent to Worker',
  },
  [V1TaskEventType.RATE_LIMIT_ERROR]: {
    icon: AlertTriangle,
    message: 'Rate limit error occurred',
    status: WorkflowRunStatus.FAILED,
    title: 'Rate Limit Error',
  },
  [V1TaskEventType.ACKNOWLEDGED]: {
    icon: Bell,
    message: 'Task acknowledged by worker',
    status: WorkflowRunStatus.RUNNING,
    title: 'Task Acknowledged',
  },
  [V1TaskEventType.CREATED]: {
    icon: Plus,
    message: 'Task created',
    status: WorkflowRunStatus.PENDING,
    title: 'Task Created',
  },
  [V1TaskEventType.SKIPPED]: {
    icon: SkipForward,
    message: 'Task skipped',
    status: WorkflowRunStatus.SUCCEEDED,
    title: 'Task Skipped',
  },
  [V1TaskEventType.REQUEUED_NO_WORKER]: {
    icon: PauseCircle,
    message: 'Task requeued - no available worker',
    status: WorkflowRunStatus.BACKOFF,
    title: 'Requeued - No Worker',
  },
  [V1TaskEventType.REQUEUED_RATE_LIMIT]: {
    icon: PauseCircle,
    message: 'Task requeued - rate limit reached',
    status: WorkflowRunStatus.BACKOFF,
    title: 'Requeued - Rate Limit',
  },
  [V1TaskEventType.SCHEDULING_TIMED_OUT]: {
    icon: PauseCircle,
    message: 'Task scheduling timed out',
    status: WorkflowRunStatus.FAILED,
    title: 'Scheduling Timeout',
  },
  [LogEventType]: {
    icon: Info,
    message: (event: V1TaskEvent) => event.message || 'Log message',
    status: WorkflowRunStatus.PENDING,
    title: 'Log Message',
  },
} as const;

interface EventIconProps {
  eventType: V1TaskEventType;
  className?: string;
}

const EventIcon = ({ eventType, className }: EventIconProps) => {
  const config = EVENT_CONFIG[eventType];
  const textColor = RunStatusConfigs[config.status]?.primary || 'text-gray-500';
  return (
    <config.icon className={cn('h-2 w-2 rounded-full', textColor, className)} />
  );
};

interface EventMessageProps {
  event: V1TaskEvent;
  onTaskSelect?: RunEventLogProps['onTaskSelect'];
}

const EventMessage = ({ event, onTaskSelect }: EventMessageProps) => {
  const config = EVENT_CONFIG[event.eventType];
  const message =
    typeof config.message === 'function'
      ? config.message(event)
      : config.message;

  if (event.eventType === V1TaskEventType.FAILED) {
    let error = { message: 'Unknown error' };
    try {
      error = event.errorMessage
        ? JSON.parse(event.errorMessage)
        : { message: 'Unknown error' };
    } catch {
      error = { message: 'Unknown error' };
    }

    return (
      <div className="flex justify-between items-center gap-2">
        <span className="text-xs text-destructive">{error.message}</span>
        {onTaskSelect && (
          <Button
            variant="outline"
            size="sm"
            className="h-5 px-1 text-xs text-destructive hover:text-destructive/80 border-destructive/50"
            onClick={(e) => {
              e.stopPropagation();
              onTaskSelect(event);
            }}
          >
            <Bug className="h-3 w-3" />
          </Button>
        )}
      </div>
    );
  }

  if (event.eventType === V1TaskEventType.FINISHED) {
    return (
      <div className="flex justify-between items-center gap-2">
        <span className="text-xs">{message}</span>
        {onTaskSelect && (
          <Button
            variant="outline"
            size="sm"
            className="h-5 px-1 text-xs text-muted-foreground hover:text-muted-foreground/80 border-muted-foreground/50"
            onClick={(e) => {
              e.stopPropagation();
              onTaskSelect(event);
            }}
          >
            <VscJson className="h-3 w-3" />
          </Button>
        )}
      </div>
    );
  }

  return (
    <div className="flex justify-between items-center gap-2">
      <span className="text-xs">{message}</span>
      {config.showWorkerButton && event.workerId && onTaskSelect && (
        <Button
          variant="outline"
          size="sm"
          className="h-5 p-1 text-xs text-muted-foreground hover:text-muted-foreground/80 border-muted-foreground/50"
          onClick={(e) => {
            e.stopPropagation();
            onTaskSelect(event, { taskTab: 'worker' });
          }}
        >
          <CpuIcon className="h-3 w-3" />
        </Button>
      )}
    </div>
  );
};

export function RunEventLog(props: RunEventLogProps) {
  return (
    <FilterProvider initialFilters={props.filters}>
      <RunEventLogContent {...props} />
    </FilterProvider>
  );
}

const DEFAULT_FILTERS = {
  search: true,
  eventType: true,
  taskId: true,
  attempt: true,
};

function RunEventLogContent({
  onTaskSelect,
  filters: initialFilters,
  showFilters: initialShowFilters,
  showNextButton,
  showPreviousButton,
}: RunEventLogProps) {
  const { data, activity } = useRunDetail();
  const { filters, setFilters } = useFilters<ActivityFilters>();

  const showFilters = useMemo(() => {
    return {
      ...DEFAULT_FILTERS,
      ...initialShowFilters,
    };
  }, [initialShowFilters]);

  // Update filters when initialFilters changes, but only if they're different
  useEffect(() => {
    if (
      initialFilters &&
      JSON.stringify(initialFilters) !== JSON.stringify(filters)
    ) {
      setFilters(initialFilters);
    }
  }, [initialFilters, setFilters, filters]);

  const tasks = useMemo(() => {
    return data?.tasks.reduce(
      (acc, task) => {
        acc[task.metadata.id] = task;
        return acc;
      },
      {} as Record<string, V1TaskSummary>,
    );
  }, [data]);

  const taskOptions = useMemo(() => {
    if (!tasks) {
      return [];
    }
    return Object.entries(tasks).map(([id, task]) => ({
      label: task.displayName || `Task-${id.substring(0, 8)}`,
      value: id,
    }));
  }, [tasks]);

  const findMostRecentNonLogEventForAttempt = (
    events: V1TaskEvent[],
    targetAttempt: number,
  ): V1TaskEvent | undefined => {
    return events
      .filter(
        (event) =>
          event.attempt === targetAttempt && event.eventType !== LogEventType,
      )
      .sort(
        (a, b) =>
          new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
      )[0];
  };

  const mergedActivity = useMemo<V1TaskEvent[]>(() => {
    const events = activity?.events || [];
    const logs = activity?.logs || [];

    const logEvents: V1TaskEvent[] = logs.map((log, index) => ({
      id: index + 1,
      taskId: log.taskId,
      timestamp: log.createdAt,
      eventType: LogEventType,
      message: log.message,
      metadata: log.metadata,
      retryCount: log.retryCount,
      attempt: log.attempt,
    }));

    const allEvents = [...events, ...logEvents];
    const sortedEvents = allEvents.sort((a, b) => {
      const timeA = new Date(a.timestamp).getTime();
      const timeB = new Date(b.timestamp).getTime();

      // First sort by timestamp (newest first)
      if (timeA !== timeB) {
        return timeB - timeA;
      }

      // Then sort by event type (STARTED first)
      if (
        a.eventType === V1TaskEventType.STARTED &&
        b.eventType !== V1TaskEventType.STARTED
      ) {
        return 1;
      }
      if (
        a.eventType !== V1TaskEventType.STARTED &&
        b.eventType === V1TaskEventType.STARTED
      ) {
        return -1;
      }

      return 0;
    });

    // If there's a previous attempt, find its most recent non-log event
    if (filters.attempt) {
      const previousAttemptEvent = findMostRecentNonLogEventForAttempt(
        allEvents,
        filters.attempt - 1,
      );
      if (previousAttemptEvent) {
        // Create a copy of the event with a special ID to mark it as the previous attempt summary
        const previousAttemptSummary: V1TaskEvent = {
          ...previousAttemptEvent,
          id: -1, // Use a negative ID to mark it as special
        };
        return [...sortedEvents, previousAttemptSummary];
      }
    }

    return sortedEvents;
  }, [activity?.events, activity?.logs, filters.attempt]);

  const attemptOptions = useMemo(() => {
    const attempts = new Set<number>();
    mergedActivity.forEach((event) => {
      if (event.attempt !== undefined) {
        attempts.add(event.attempt);
      }
    });
    return Array.from(attempts)
      .sort((a, b) => a - b)
      .map((attempt) => ({
        label: `Attempt ${attempt}`,
        value: attempt,
      }));
  }, [mergedActivity]);

  const eventTypeOptions = useMemo(() => {
    return Object.entries(EVENT_CONFIG).map(([type, config]) => ({
      label: config.title,
      value: type as V1TaskEventType,
    }));
  }, []);

  const filteredActivity = useMemo(() => {
    let filtered = mergedActivity;

    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      filtered = filtered.filter((event) => {
        const config = EVENT_CONFIG[event.eventType];
        const message =
          typeof config.message === 'function'
            ? config.message(event)
            : config.message;

        return (
          message.toLowerCase().includes(searchLower) ||
          event.eventType.toLowerCase().includes(searchLower) ||
          (event.taskId && event.taskId.toLowerCase().includes(searchLower)) ||
          (event.workerId && event.workerId.toLowerCase().includes(searchLower))
        );
      });
    }

    if (filters.eventType?.length) {
      filtered = filtered.filter((event) =>
        filters.eventType?.includes(event.eventType),
      );
    }

    if (filters.taskId?.length) {
      filtered = filtered.filter(
        (event) => event.taskId && filters.taskId?.includes(event.taskId),
      );
    }

    if (filters.attempt !== undefined) {
      filtered = filtered.filter(
        (event) =>
          event.attempt !== undefined && event.attempt === filters.attempt,
      );
    }

    return filtered;
  }, [
    mergedActivity,
    filters.search,
    filters.eventType,
    filters.taskId,
    filters.attempt,
  ]);

  return (
    <div className="space-y-2">
      <FilterGroup>
        {showFilters.search && (
          <FilterText<ActivityFilters>
            name="search"
            placeholder="Search activity..."
          />
        )}
        {showFilters.eventType && (
          <FilterSelect<ActivityFilters, V1TaskEventType>
            name="eventType"
            placeholder="Event Type"
            options={eventTypeOptions}
            multi
          />
        )}
        {showFilters.taskId && taskOptions.length > 1 && (
          <FilterSelect<ActivityFilters, string>
            name="taskId"
            placeholder="Task"
            options={taskOptions}
            multi
          />
        )}
        {showFilters.attempt && attemptOptions.length > 0 && (
          <FilterSelect<ActivityFilters, number>
            name="attempt"
            placeholder="Attempt"
            options={attemptOptions}
          />
        )}
      </FilterGroup>
      <div className="space-y-2">
        {showNextButton && (
          <Button
            variant="link"
            className="p-2 text-muted-foreground"
            size="sm"
            onClick={showNextButton.onClick}
          >
            <ArrowUpCircle className="h-4 w-4" />
            {showNextButton.label}
          </Button>
        )}
        <div className="space-y-0.5 bg-background p-1 rounded-md">
          {filteredActivity?.map((event) => (
            <div
              key={event.id}
              className={cn(
                'flex flex-col gap-0.5 rounded-sm p-1 text-xs font-mono',
                'hover:bg-muted/50 cursor-pointer transition-colors',
                'group relative',
              )}
              onClick={() => onTaskSelect?.(event)}
            >
              <div className="flex flex-col gap-0.5 w-full">
                <div className="flex items-center gap-1.5">
                  <div className="flex flex-col min-w-0 w-full">
                    <div className="flex items-center gap-2 flex-wrap">
                      <EventIcon
                        eventType={event.eventType}
                        className="shrink-0"
                      />
                      <Time
                        date={event.timestamp}
                        variant="timestamp"
                        className="text-gray-500 shrink-0"
                        asChild
                      >
                        <span />
                      </Time>
                      <p className="text-gray-500 shrink-0">
                        {tasks?.[event.taskId] && (
                          <RunId
                            taskRun={tasks[event.taskId] as any}
                            attempt={event.attempt}
                          />
                        )}
                      </p>
                      {event.eventType === LogEventType ? (
                        <EventMessage
                          event={event}
                          onTaskSelect={onTaskSelect}
                        />
                      ) : (
                        <>
                          <p
                            className={cn(
                              'font-medium shrink-0',

                              RunStatusConfigs[
                                EVENT_CONFIG[event.eventType].status
                              ]?.primary || 'text-gray-500',
                              'bg-transparent',
                            )}
                          >
                            {event.eventType}
                          </p>
                          <div className="text-gray-500 break-all">
                            <EventMessage
                              event={event}
                              onTaskSelect={onTaskSelect}
                            />
                          </div>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="absolute right-2 top-1 h-4 w-4 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
                onClick={(e) => {
                  e.stopPropagation();
                  onTaskSelect?.(event);
                }}
              >
                <ArrowUpRight className="h-3 w-3" />
              </Button>
            </div>
          ))}
        </div>
        {showPreviousButton && (
          <Button
            variant="link"
            className="p-2 text-muted-foreground"
            size="sm"
            onClick={showPreviousButton.onClick}
          >
            <ArrowDownCircle className="h-4 w-4" />
            {showPreviousButton.label}
          </Button>
        )}
      </div>
    </div>
  );
}
