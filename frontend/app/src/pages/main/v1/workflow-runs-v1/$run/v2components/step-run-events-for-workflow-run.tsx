import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { Loading } from '@/components/v1/ui/loading';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  V1DurableEventLogEntry,
  V1DurableEventLogKind,
  V1DurableWaitConditionKind,
  V1TaskEvent,
  V1WaitItem,
  queries,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { eventTypeToSeverity, mapEventTypeToTitle } from './event-utils';

export type EventWithMetadata = V1TaskEvent & {
  metadata: {
    id: string;
  };
};

const BULK_SPAWN_THRESHOLD_MS = 1000;

export function StepRunEvents({
  taskRunId,
  workflowRunId,
  isDurable,
  taskDisplayName,
}: {
  taskRunId?: string;
  workflowRunId?: string;
  isDurable?: boolean;
  taskDisplayName?: string;
  fallbackTaskDisplayName?: string;
  onClick?: (stepRunId: string) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const eventsQuery = useQuery({
    ...queries.v1TaskEvents.list(
      tenantId,
      { limit: 50, offset: 0 },
      taskRunId,
      workflowRunId,
    ),
    refetchInterval,
  });

  const durableLogQuery = useQuery({
    ...queries.v1DurableTasks.eventLog(taskRunId!),
    refetchInterval: 2000,
    enabled: !!isDurable && !!taskRunId,
  });

  const logs = useMemo(() => {
    const taskLines = toTaskEventLogLines(eventsQuery.data?.rows ?? [], taskDisplayName);
    const durableLines = isDurable
      ? toDurableEventLogLines(durableLogQuery.data ?? [], taskDisplayName)
      : [];
    return mergeByTimestamp(taskLines, durableLines);
  }, [eventsQuery.data, durableLogQuery.data, isDurable, taskDisplayName]);

  if (eventsQuery.isLoading) {
    return <Loading />;
  }

  return (
    <div className="flex h-full min-h-[25rem] flex-1 flex-col">
      <LogViewer logs={logs} emptyMessage="No events found." showTaskName={!!taskDisplayName} />
    </div>
  );
}

function mergeByTimestamp(a: LogLine[], b: LogLine[]): LogLine[] {
  const merged = [...a, ...b];
  merged.sort((x, y) => {
    if (!x.timestamp) return -1;
    if (!y.timestamp) return 1;
    return x.timestamp < y.timestamp ? -1 : x.timestamp > y.timestamp ? 1 : 0;
  });
  return merged;
}

function toTaskEventLogLines(events: V1TaskEvent[], taskDisplayName?: string): LogLine[] {
  return events.map((event) => {
    const severity = eventTypeToSeverity(event.eventType);
    let level: string;
    switch (severity) {
      case 'CRITICAL':
        level = 'error';
        break;
      case 'WARNING':
        level = 'warn';
        break;
      default:
        level = 'info';
        break;
    }

    let line = mapEventTypeToTitle(event.eventType);
    if (event.message) {
      line += `: ${event.message}`;
    }

    return {
      timestamp: event.timestamp,
      level,
      line,
      taskDisplayName: event.taskDisplayName ?? taskDisplayName,
    };
  });
}

// ----- durable event log helpers -----

function formatDurationMs(ms: number): string {
  if (ms === 0) return '0s';
  if (ms % 3_600_000 === 0) return `${ms / 3_600_000}h`;
  if (ms % 60_000 === 0) return `${ms / 60_000}m`;
  if (ms % 1_000 === 0) return `${ms / 1_000}s`;
  return `${ms}ms`;
}

function describeCondition(c: {
  kind?: V1DurableWaitConditionKind;
  sleepDurationMs?: number | null;
  eventKey?: string | null;
  workflowName?: string | null;
}): string {
  switch (c.kind) {
    case V1DurableWaitConditionKind.SLEEP:
      return c.sleepDurationMs != null
        ? `sleep(${formatDurationMs(c.sleepDurationMs)})`
        : 'sleep';
    case V1DurableWaitConditionKind.USER_EVENT:
      return c.eventKey ? `event(${c.eventKey})` : 'event';
    case V1DurableWaitConditionKind.CHILD_WORKFLOW:
      return c.workflowName ? `run(${c.workflowName})` : 'run';
    default:
      return String(c.kind ?? 'unknown').toLowerCase();
  }
}

function describeWaitItem(item: V1WaitItem): string {
  if (item.or && item.or.length > 0) {
    return `any of: ${item.or.map(describeCondition).join(', ')}`;
  }
  if (item.kind) {
    return describeCondition(item);
  }
  return 'waiting';
}

function toReadableMessage(items: V1WaitItem[]): string {
  if (items.length === 0) return 'waiting';
  const parts = items.map(describeWaitItem);
  return parts.length === 1 ? parts[0] : parts.join(' and ');
}

function kindLabel(kind: V1DurableEventLogKind): string {
  switch (kind) {
    case V1DurableEventLogKind.RUN:
      return 'run';
    case V1DurableEventLogKind.WAIT_FOR:
      return 'wait';
    default:
      return String(kind).toLowerCase();
  }
}

function capitalizeFirst(s: string): string {
  return s ? s.charAt(0).toUpperCase() + s.slice(1) : s;
}

function completionMessage(message: string): string {
  return `${capitalizeFirst(message)} completed`;
}

function withContextPrefix(
  entry: V1DurableEventLogEntry,
  message: string,
): string {
  const prefix = `[${kindLabel(entry.kind)}${entry.branchId > 1 ? ` b${entry.branchId}` : ''}]`;
  return `${prefix} ${message}`;
}

function entryMessage(entry: V1DurableEventLogEntry): string {
  if (entry.waitData && entry.waitData.length > 0) {
    return toReadableMessage(entry.waitData);
  }
  return entry.userMessage ?? kindLabel(entry.kind);
}

interface RunGroup {
  entries: V1DurableEventLogEntry[];
}

function groupConsecutiveRuns(
  entries: V1DurableEventLogEntry[],
): (V1DurableEventLogEntry | RunGroup)[] {
  const result: (V1DurableEventLogEntry | RunGroup)[] = [];
  let currentGroup: RunGroup | null = null;

  for (const entry of entries) {
    if (entry.kind === V1DurableEventLogKind.RUN) {
      if (currentGroup) {
        const last = currentGroup.entries[currentGroup.entries.length - 1];
        const delta =
          new Date(entry.insertedAt).getTime() -
          new Date(last.insertedAt).getTime();
        if (delta <= BULK_SPAWN_THRESHOLD_MS) {
          currentGroup.entries.push(entry);
          continue;
        }
        result.push(currentGroup);
      }
      currentGroup = { entries: [entry] };
    } else {
      if (currentGroup) {
        result.push(currentGroup);
        currentGroup = null;
      }
      result.push(entry);
    }
  }

  if (currentGroup) {
    result.push(currentGroup);
  }

  return result;
}

function runGroupLabel(entries: V1DurableEventLogEntry[]): string {
  if (entries.length === 1) {
    return entryMessage(entries[0]);
  }

  const names = entries.map((e) => {
    const items = e.waitData;
    if (
      items?.length === 1 &&
      !items[0].or &&
      items[0].kind === V1DurableWaitConditionKind.CHILD_WORKFLOW
    ) {
      return items[0].workflowName ?? null;
    }
    return null;
  });

  const allSame = names.every((n) => n === names[0]);
  if (allSame && names[0] !== null) {
    return `${entries.length}x run(${names[0]})`;
  }

  return `run(${names.map((n) => n ?? 'unknown').join(', ')})`;
}

function toDurableEventLogLines(entries: V1DurableEventLogEntry[], taskDisplayName?: string): LogLine[] {
  const lines: LogLine[] = [];
  const visible = entries.filter((e) => e.kind !== V1DurableEventLogKind.MEMO);
  const grouped = groupConsecutiveRuns(visible);

  for (const item of grouped) {
    if ('entries' in item) {
      const { entries: groupEntries } = item;
      const first = groupEntries[0];
      const label = runGroupLabel(groupEntries);

      lines.push({
        timestamp: first.insertedAt,
        level: 'debug',
        line: withContextPrefix(first, capitalizeFirst(label)),
        taskDisplayName,
      });

      const satisfiedEntries = groupEntries.filter(
        (e) => e.isSatisfied && e.satisfiedAt,
      );
      if (satisfiedEntries.length > 0) {
        const lastSatisfiedAt = satisfiedEntries.reduce(
          (latest, e) => (e.satisfiedAt! > latest ? e.satisfiedAt! : latest),
          satisfiedEntries[0].satisfiedAt!,
        );
        lines.push({
          timestamp: lastSatisfiedAt,
          level: 'info',
          line: withContextPrefix(first, completionMessage(label)),
          taskDisplayName,
        });
      }
    } else {
      const message = entryMessage(item);

      lines.push({
        timestamp: item.insertedAt,
        level: item.kind === V1DurableEventLogKind.WAIT_FOR ? 'warn' : 'debug',
        line: withContextPrefix(item, capitalizeFirst(message)),
        taskDisplayName,
      });

      if (item.isSatisfied && item.satisfiedAt) {
        lines.push({
          timestamp: item.satisfiedAt,
          level: 'info',
          line: withContextPrefix(item, completionMessage(message)),
          taskDisplayName,
        });
      }
    }
  }

  return lines;
}
