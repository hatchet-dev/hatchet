import { V1WorkflowRun, V1TaskEvent, V1TaskEventType } from '@/lib/api';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { cn } from '@/next/lib/utils';
import { RunStatusConfigs } from '../runs-badge';
import { WorkflowRunStatus } from '@/next/lib/api';
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
} from 'lucide-react';
import { Button } from '@/next/components/ui/button';
import { Time } from '@/next/components/ui/time';
import { useMemo } from 'react';

interface RunEventLogProps {
  workflow: V1WorkflowRun;
  onTaskSelect?: (taskId: string) => void;
  onWorkerSelect?: (workerId: string) => void;
}

type EventConfig = {
  icon: React.ComponentType<{ className?: string }>;
  message: string | ((event: V1TaskEvent) => string);
  showWorkerButton?: boolean;
  status: WorkflowRunStatus;
  title: string;
};

const LogEventType = 'LOG_LINE' as V1TaskEventType;

interface LogMetadata {
  taskId?: string;
  [key: string]: any;
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
    status: WorkflowRunStatus.RUNNING,
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
    message: (event) => `Reassigned to worker ${event.workerId}`,
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
  const textColor =
    RunStatusConfigs[config.status]?.textColor || 'text-gray-500';
  return <config.icon className={cn('h-3 w-3', textColor, className)} />;
};

interface EventMessageProps {
  event: V1TaskEvent;
  onTaskSelect?: (taskId: string) => void;
  onWorkerSelect?: (workerId: string) => void;
}

const EventMessage = ({
  event,
  onTaskSelect,
  onWorkerSelect,
}: EventMessageProps) => {
  const config = EVENT_CONFIG[event.eventType];
  const message =
    typeof config.message === 'function'
      ? config.message(event)
      : config.message;

  return (
    <div className="flex items-center gap-1">
      <span className="text-xs">{message}</span>
      {config.showWorkerButton && event.workerId && onWorkerSelect && (
        <Button
          variant="ghost"
          size="sm"
          className="h-4 px-1 text-xs"
          onClick={(e) => {
            e.stopPropagation();
            onWorkerSelect(event.workerId!);
          }}
        >
          View Worker
        </Button>
      )}
    </div>
  );
};

export function RunEventLog({
  workflow,
  onTaskSelect,
  onWorkerSelect,
}: RunEventLogProps) {
  const { activity } = useRunDetail(workflow.metadata.id || '');

  const mergedActivity = useMemo<V1TaskEvent[]>(() => {
    const events = activity?.events?.rows || [];
    const logs = activity?.logs?.rows || [];

    const logEvents: V1TaskEvent[] = logs.map((log, index) => ({
      id: index + 1,
      taskId: (log.metadata as LogMetadata)?.taskId || '',
      timestamp: log.createdAt,
      eventType: LogEventType,
      message: log.message,
      metadata: log.metadata,
    }));

    const allEvents = [...events, ...logEvents];
    return allEvents.sort((a, b) => {
      const timeA = new Date(a.timestamp).getTime();
      const timeB = new Date(b.timestamp).getTime();
      return timeB - timeA;
    });
  }, [activity]);

  return (
    <div className="space-y-1">
      {mergedActivity?.map((event) => (
        <div
          key={event.id}
          className={cn(
            'flex items-center gap-2 rounded border p-1.5 text-xs',
            'hover:bg-muted/50 cursor-pointer transition-colors',
            'group',
          )}
          onClick={() => onTaskSelect?.(event.taskId)}
        >
          <div className="flex-shrink-0">
            <EventIcon eventType={event.eventType} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between">
              <p className="font-medium truncate">{event.eventType}</p>
              <div className="ml-2">
                <Time date={event.timestamp} variant="timestamp" />
              </div>
            </div>
            <EventMessage
              event={event}
              onTaskSelect={onTaskSelect}
              onWorkerSelect={onWorkerSelect}
            />
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="h-4 w-4 p-0 opacity-0 group-hover:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation();
              onTaskSelect?.(event.taskId);
            }}
          >
            <ArrowUpRight className="h-3 w-3" />
          </Button>
        </div>
      ))}
    </div>
  );
}
