import { V1WorkflowRun, V1TaskEvent, V1TaskEventType } from '@/lib/api';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { cn } from '@/next/lib/utils';
import {
  CheckCircle2,
  PlayCircle,
  User,
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
} from 'lucide-react';
import { Button } from '@/next/components/ui/button';
import { Time } from '@/next/components/ui/time';

interface RunEventLogProps {
  workflow: V1WorkflowRun;
  onTaskSelect?: (taskId: string) => void;
  onWorkerSelect?: (workerId: string) => void;
}

type EventConfig = {
  icon: React.ComponentType<{ className?: string }>;
  message: string | ((event: V1TaskEvent) => string);
  showWorkerButton?: boolean;
};

const EVENT_CONFIG: Record<V1TaskEventType, EventConfig> = {
  [V1TaskEventType.FINISHED]: {
    icon: CheckCircle2,
    message: 'Task completed successfully',
  },
  [V1TaskEventType.STARTED]: {
    icon: PlayCircle,
    message: 'Task execution started',
  },
  [V1TaskEventType.ASSIGNED]: {
    icon: User,
    message: (event) => `Assigned to worker ${event.workerId}`,
    showWorkerButton: true,
  },
  [V1TaskEventType.QUEUED]: {
    icon: Clock,
    message: 'Task queued for execution',
  },
  [V1TaskEventType.FAILED]: {
    icon: AlertCircle,
    message: (event) => event.errorMessage || 'Task failed',
  },
  [V1TaskEventType.CANCELLED]: {
    icon: XCircle,
    message: 'Task cancelled',
  },
  [V1TaskEventType.RETRYING]: {
    icon: RefreshCw,
    message: 'Retrying task',
  },
  [V1TaskEventType.TIMED_OUT]: {
    icon: Timer,
    message: 'Task timed out',
  },
  [V1TaskEventType.REASSIGNED]: {
    icon: UserCog,
    message: (event) => `Reassigned to worker ${event.workerId}`,
    showWorkerButton: true,
  },
  [V1TaskEventType.SLOT_RELEASED]: {
    icon: Unlock,
    message: 'Worker slot released',
  },
  [V1TaskEventType.TIMEOUT_REFRESHED]: {
    icon: RotateCw,
    message: 'Task timeout refreshed',
  },
  [V1TaskEventType.RETRIED_BY_USER]: {
    icon: RefreshCw,
    message: 'Task retried by user',
  },
  [V1TaskEventType.SENT_TO_WORKER]: {
    icon: Send,
    message: 'Task sent to worker',
  },
  [V1TaskEventType.RATE_LIMIT_ERROR]: {
    icon: AlertTriangle,
    message: 'Rate limit error occurred',
  },
  [V1TaskEventType.ACKNOWLEDGED]: {
    icon: Bell,
    message: 'Task acknowledged by worker',
  },
  [V1TaskEventType.CREATED]: {
    icon: Plus,
    message: 'Task created',
  },
  [V1TaskEventType.SKIPPED]: {
    icon: SkipForward,
    message: 'Task skipped',
  },
  [V1TaskEventType.REQUEUED_NO_WORKER]: {
    icon: PauseCircle,
    message: 'Task requeued - no available worker',
  },
  [V1TaskEventType.REQUEUED_RATE_LIMIT]: {
    icon: PauseCircle,
    message: 'Task requeued - rate limit reached',
  },
  [V1TaskEventType.SCHEDULING_TIMED_OUT]: {
    icon: PauseCircle,
    message: 'Task scheduling timed out',
  },
} as const;

interface EventIconProps {
  eventType: V1TaskEventType;
  className?: string;
}

const EventIcon = ({ eventType, className }: EventIconProps) => {
  const Icon = EVENT_CONFIG[eventType].icon;
  return <Icon className={cn('h-3 w-3', className)} />;
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
  const { taskEvents } = useRunDetail(workflow.metadata.id || '');

  return (
    <div className="space-y-1">
      {taskEvents?.data?.rows?.map((event) => (
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
