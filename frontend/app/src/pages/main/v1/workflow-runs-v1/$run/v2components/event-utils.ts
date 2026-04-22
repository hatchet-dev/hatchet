import { V1TaskEventType, StepRunEventSeverity } from '@/lib/api';

export type EventSeverity = StepRunEventSeverity | 'EVICTION';

export function eventTypeToSeverity(
  eventType: V1TaskEventType | undefined,
): EventSeverity {
  switch (eventType) {
    case V1TaskEventType.FAILED:
    case V1TaskEventType.RATE_LIMIT_ERROR:
    case V1TaskEventType.SCHEDULING_TIMED_OUT:
    case V1TaskEventType.TIMED_OUT:
    case V1TaskEventType.CANCELLED:
      return StepRunEventSeverity.CRITICAL;
    case V1TaskEventType.REASSIGNED:
    case V1TaskEventType.REQUEUED_NO_WORKER:
    case V1TaskEventType.REQUEUED_RATE_LIMIT:
    case V1TaskEventType.RETRIED_BY_USER:
    case V1TaskEventType.RETRYING:
    case V1TaskEventType.DURABLE_RESTORING:
      return StepRunEventSeverity.WARNING;
    case V1TaskEventType.DURABLE_EVICTED:
      return 'EVICTION';
    default:
      return StepRunEventSeverity.INFO;
  }
}

export function mapEventTypeToTitle(
  eventType: V1TaskEventType | undefined,
): string {
  switch (eventType) {
    case V1TaskEventType.ASSIGNED:
      return 'Assigned to worker';
    case V1TaskEventType.STARTED:
      return 'Started';
    case V1TaskEventType.FINISHED:
      return 'Completed';
    case V1TaskEventType.FAILED:
      return 'Failed';
    case V1TaskEventType.CANCELLED:
      return 'Cancelled';
    case V1TaskEventType.RETRYING:
      return 'Retrying';
    case V1TaskEventType.REQUEUED_NO_WORKER:
      return 'Requeuing (no worker available)';
    case V1TaskEventType.REQUEUED_RATE_LIMIT:
      return 'Requeuing (rate limit)';
    case V1TaskEventType.SCHEDULING_TIMED_OUT:
      return 'Scheduling timed out';
    case V1TaskEventType.TIMEOUT_REFRESHED:
      return 'Timeout refreshed';
    case V1TaskEventType.REASSIGNED:
      return 'Reassigned';
    case V1TaskEventType.TIMED_OUT:
      return 'Execution timed out';
    case V1TaskEventType.SLOT_RELEASED:
      return 'Slot released';
    case V1TaskEventType.RETRIED_BY_USER:
      return 'Replayed by user';
    case V1TaskEventType.ACKNOWLEDGED:
      return 'Acknowledged by worker';
    case V1TaskEventType.CREATED:
      return 'Created';
    case V1TaskEventType.RATE_LIMIT_ERROR:
      return 'Rate limit error';
    case V1TaskEventType.SENT_TO_WORKER:
      return 'Sent to worker';
    case V1TaskEventType.QUEUED:
      return 'Queued';
    case V1TaskEventType.SKIPPED:
      return 'Skipped';
    case V1TaskEventType.COULD_NOT_SEND_TO_WORKER:
      return 'Could not send to worker';
    case V1TaskEventType.DURABLE_EVICTED:
      return 'Durable task evicted';
    case V1TaskEventType.DURABLE_RESTORING:
      return 'Durable task restoring';
    case V1TaskEventType.WORKFLOW_PAUSED:
      return 'Workflow paused';
    case V1TaskEventType.WORKFLOW_UNPAUSED:
      return 'Workflow unpaused';
    case undefined:
      return 'Unknown';
    default:
      const exhaustiveCheck: never = eventType;
      throw new Error(`Unhandled case: ${exhaustiveCheck}`);
  }
}

export const EVENT_SEVERITY_COLORS: Record<EventSeverity, string> = {
  INFO: 'bg-green-500',
  CRITICAL: 'bg-red-500',
  WARNING: 'bg-yellow-500',
  EVICTION: 'bg-indigo-500',
};

export const RUN_STATUS_VARIANTS: Record<EventSeverity, string> = {
  INFO: 'border-transparent rounded-full bg-green-500',
  CRITICAL: 'border-transparent rounded-full bg-red-500',
  WARNING: 'border-transparent rounded-full bg-yellow-500',
  EVICTION: 'border-transparent rounded-full bg-indigo-500',
};
