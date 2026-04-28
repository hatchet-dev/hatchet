import { eventTypeToSeverity, mapEventTypeToTitle } from './event-utils';
import { TabOption } from './step-run-detail/step-run-detail';
import {
  LogLine,
  V1LogLineLevelIncludingEvictionNotice,
} from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { Loading } from '@/components/v1/ui/loading';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { useSidePanel } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import {
  V1DurableEventLogEntry,
  V1DurableEventLogKind,
  V1DurableWaitConditionKind,
  V1LogLineLevel,
  V1TaskEvent,
  V1WaitItem,
  queries,
} from '@/lib/api';
import { emptyGolangUUID } from '@/lib/utils';
import { appRoutes } from '@/router';
import { useQueries, useQuery } from '@tanstack/react-query';
import { useCallback, useMemo, useRef } from 'react';

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
  durableTaskIds,
}: {
  taskRunId?: string;
  workflowRunId?: string;
  isDurable?: boolean;
  durableTaskIds?: string[];
  taskDisplayName?: string;
  fallbackTaskDisplayName?: string;
  onClick?: (stepRunId: string) => void;
}) {
  const isDag =
    (!taskRunId && !!workflowRunId) ||
    (!!taskRunId && !!workflowRunId && taskRunId !== workflowRunId);

  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();
  const { open } = useSidePanel();
  const executingRef = useRef(false);

  const eventsQuery = useQuery({
    ...queries.v1TaskEvents.list(
      tenantId,
      { limit: 50, offset: 0 },
      taskRunId,
      workflowRunId,
    ),
    refetchInterval,
  });

  // fixme: this is an n+1 query, would be better to have a bulk getter
  const durableLogsQueries = useQueries({
    queries: [...(durableTaskIds ?? []), ...(taskRunId ? [taskRunId] : [])].map(
      (id) => ({
        ...queries.v1DurableTasks.eventLog(id),
        refetchInterval: 5000,
      }),
    ),
  });

  const logs = useMemo(() => {
    const taskLines = toTaskEventLogLines(
      eventsQuery.data?.rows ?? [],
      isDag,
      tenantId,
    );
    const durableEventLogLines = durableLogsQueries.flatMap((q) =>
      toDurableEventLogLines(q.data ?? []),
    );
    return mergeByTimestamp(taskLines, durableEventLogLines);
  }, [eventsQuery.data, isDag, tenantId, durableLogsQueries]);

  const handleTaskRunExpand = useCallback(
    (taskRunId: string) => {
      // hack to prevent click handler from firing multiple times,
      // causing index offset issues
      if (executingRef.current) {
        return;
      }

      executingRef.current = true;

      open({
        type: 'task-run-details',
        content: {
          taskRunId,
          defaultOpenTab: TabOption.Activity,
          showViewTaskRunButton: true,
        },
      });

      setTimeout(() => {
        executingRef.current = false;
      }, 100);
    },
    [open],
  );

  if (eventsQuery.isLoading) {
    return <Loading />;
  }

  return (
    <div className="flex flex-1 min-h-0 flex-col">
      <LogViewer
        logs={logs}
        emptyMessage="No events found."
        showTaskName={isDag}
        onViewRun={(taskExternalId) => {
          if (taskExternalId) {
            handleTaskRunExpand(taskExternalId);
          }
        }}
      />
    </div>
  );
}

function mergeByTimestamp(a: LogLine[], b: LogLine[]): LogLine[] {
  const merged = [...a, ...b];
  merged.sort((x, y) => {
    if (!x.timestamp) {
      return -1;
    }
    if (!y.timestamp) {
      return 1;
    }
    return x.timestamp < y.timestamp ? -1 : x.timestamp > y.timestamp ? 1 : 0;
  });
  return merged;
}

function toTaskEventLogLines(
  events: V1TaskEvent[],
  isDag: boolean,
  tenantId: string,
): LogLine[] {
  return events.map((event) => {
    const severity = eventTypeToSeverity(event.eventType);
    let level: V1LogLineLevelIncludingEvictionNotice;

    if (event.eventType === 'DURABLE_EVICTED') {
      level = 'EVICTION_NOTICE';
    } else if (event.eventType === 'DURABLE_RESTORING') {
      level = 'RESTORE_NOTICE';
    } else {
      switch (severity) {
        case 'CRITICAL':
          level = V1LogLineLevel.ERROR;
          break;
        case 'WARNING':
          level = V1LogLineLevel.WARN;
          break;
        case 'INFO':
          level = V1LogLineLevel.INFO;
          break;
        default:
          level = V1LogLineLevel.DEBUG;
          break;
      }
    }

    let line = mapEventTypeToTitle(event.eventType);
    if (event.message) {
      line += `: ${event.message}`;
    }

    const linkTo =
      event.workerId && event.workerId !== emptyGolangUUID
        ? {
            destination: appRoutes.tenantWorkerRoute.to,
            params: { tenant: tenantId, worker: event.workerId },
            hoverText: 'View worker',
          }
        : undefined;

    return {
      timestamp: event.timestamp,
      level,
      line,
      taskDisplayName: event.taskDisplayName,
      taskExternalId: event.taskId,
      error: event.errorMessage,
      linkTo,
    };
  });
}

// ----- durable event log helpers -----

const DURABLE_DISPLAY_LIMIT = 3;

function formatDurationMs(ms: number): string {
  if (ms === 0) {
    return '0s';
  }
  if (ms % 3_600_000 === 0) {
    return `${ms / 3_600_000}h`;
  }
  if (ms % 60_000 === 0) {
    return `${ms / 60_000}m`;
  }
  if (ms % 1_000 === 0) {
    return `${ms / 1_000}s`;
  }
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
        ? `sleeping for ${formatDurationMs(c.sleepDurationMs)}`
        : 'sleeping';
    case V1DurableWaitConditionKind.USER_EVENT:
      return c.eventKey
        ? `waiting for event ${c.eventKey}`
        : 'waiting for event';
    case V1DurableWaitConditionKind.CHILD_WORKFLOW:
      return c.workflowName
        ? `waiting for child ${c.workflowName} to complete`
        : 'waiting for child to complete';
    default:
      return String(c.kind ?? 'unknown').toLowerCase();
  }
}

function describeConditionCompletion(c: {
  kind?: V1DurableWaitConditionKind;
  eventKey?: string | null;
  workflowName?: string | null;
}): string {
  switch (c.kind) {
    case V1DurableWaitConditionKind.SLEEP:
      return 'sleep completed';
    case V1DurableWaitConditionKind.USER_EVENT:
      return c.eventKey ? `received event ${c.eventKey}` : 'event received';
    case V1DurableWaitConditionKind.CHILD_WORKFLOW:
      return c.workflowName
        ? `child ${c.workflowName} completed`
        : 'child completed';
    default:
      return 'completed';
  }
}

function describeWaitItem(item: V1WaitItem): string {
  if (item.or && item.or.length > 0) {
    const shown = item.or
      .slice(0, DURABLE_DISPLAY_LIMIT)
      .map(describeCondition);
    const extra = item.or.length - DURABLE_DISPLAY_LIMIT;
    if (extra > 0) {
      shown.push(`${extra} more`);
    }
    return `any of: ${shown.join(', ')}`;
  }
  if (item.kind) {
    return describeCondition(item);
  }
  return 'waiting';
}

function toReadableMessage(items: V1WaitItem[]): string {
  if (items.length === 0) {
    return 'waiting';
  }
  if (items.length === 1) {
    return describeWaitItem(items[0]);
  }
  const shown = items.slice(0, DURABLE_DISPLAY_LIMIT).map(describeWaitItem);
  const extra = items.length - DURABLE_DISPLAY_LIMIT;
  if (extra > 0) {
    return `${shown.join(', ')}, and ${extra} more`;
  }
  return `${shown.slice(0, -1).join(', ')} and ${shown[shown.length - 1]}`;
}

function toReadableCompletion(items: V1WaitItem[]): string {
  if (items.length === 0) {
    return 'completed';
  }
  if (items.length === 1) {
    const item = items[0];
    if (item.or && item.or.length > 0) {
      return 'condition satisfied';
    }
    return describeConditionCompletion(item);
  }
  return 'all conditions satisfied';
}

function withContextPrefix(
  entry: V1DurableEventLogEntry,
  message: string,
): string {
  const kind =
    entry.kind === V1DurableEventLogKind.WAIT_FOR
      ? 'durable wait'
      : 'durable run';
  const branch = entry.branchId > 1 ? ` b${entry.branchId}` : '';
  return `[${kind}${branch}] ${message}`;
}

function entryMessage(entry: V1DurableEventLogEntry): string {
  const userMessage = entry.userMessage?.trim();

  if (entry.kind === V1DurableEventLogKind.RUN) {
    const item = entry.waitData?.[0];
    if (
      entry.waitData?.length === 1 &&
      !item?.or &&
      item?.kind === V1DurableWaitConditionKind.CHILD_WORKFLOW
    ) {
      return item.workflowName
        ? `spawned child ${item.workflowName}`
        : 'spawned child';
    }
    return userMessage || 'spawned child';
  }
  if (userMessage) {
    return userMessage;
  }
  if (entry.waitData && entry.waitData.length > 0) {
    return toReadableMessage(entry.waitData);
  }
  return 'waiting';
}

function entryCompletionMessage(entry: V1DurableEventLogEntry): string {
  if (entry.kind === V1DurableEventLogKind.RUN) {
    const item = entry.waitData?.[0];
    if (
      entry.waitData?.length === 1 &&
      !item?.or &&
      item?.kind === V1DurableWaitConditionKind.CHILD_WORKFLOW
    ) {
      return item.workflowName
        ? `child ${item.workflowName} completed`
        : 'child completed';
    }
    return 'completed';
  }
  if (entry.waitData && entry.waitData.length > 0) {
    return toReadableCompletion(entry.waitData);
  }
  return 'completed';
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

function childNamesFromEntries(
  entries: V1DurableEventLogEntry[],
): (string | null)[] {
  return entries.map((e) => {
    const item = e.waitData?.[0];
    if (
      e.waitData?.length === 1 &&
      !item?.or &&
      item?.kind === V1DurableWaitConditionKind.CHILD_WORKFLOW
    ) {
      return item.workflowName ?? null;
    }
    return null;
  });
}

function runGroupLabel(entries: V1DurableEventLogEntry[]): string {
  if (entries.length === 1) {
    return entryMessage(entries[0]);
  }

  const names = childNamesFromEntries(entries);
  const allSame = names.every((n) => n === names[0]);
  if (allSame && names[0] !== null) {
    return `spawned ${entries.length}x ${names[0]} children`;
  }

  const shown = names
    .slice(0, DURABLE_DISPLAY_LIMIT)
    .map((n) => n ?? 'unknown');
  const extra = names.length - DURABLE_DISPLAY_LIMIT;
  if (extra > 0) {
    return `spawned children: ${shown.join(', ')}, and ${extra} more`;
  }
  return `spawned children: ${shown.slice(0, -1).join(', ')} and ${shown[shown.length - 1]}`;
}

function runGroupCompletionLabel(entries: V1DurableEventLogEntry[]): string {
  if (entries.length === 1) {
    return entryCompletionMessage(entries[0]);
  }

  const names = childNamesFromEntries(entries);
  const allSame = names.every((n) => n === names[0]);
  if (allSame && names[0] !== null) {
    return `${entries.length}x ${names[0]} children completed`;
  }
  return 'children completed';
}

function toDurableEventLogLines(entries: V1DurableEventLogEntry[]): LogLine[] {
  const lines: LogLine[] = [];
  const visible = entries.filter((e) => e.kind !== V1DurableEventLogKind.MEMO);
  const grouped = groupConsecutiveRuns(visible);

  for (const item of grouped) {
    if ('entries' in item) {
      const { entries: groupEntries } = item;
      const first = groupEntries[0];

      lines.push({
        timestamp: first.insertedAt,
        level: 'EVICTION_NOTICE',
        line: withContextPrefix(first, runGroupLabel(groupEntries)),
        taskDisplayName: first.taskDisplayName,
        taskExternalId: first.taskExternalId,
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
          level: V1LogLineLevel.INFO,
          line: withContextPrefix(first, runGroupCompletionLabel(groupEntries)),
          taskDisplayName: first.taskDisplayName,
          taskExternalId: first.taskExternalId,
        });
      }
    } else {
      lines.push({
        timestamp: item.insertedAt,
        level: 'EVICTION_NOTICE',
        line: withContextPrefix(item, entryMessage(item)),
        taskDisplayName: item.taskDisplayName,
        taskExternalId: item.taskExternalId,
      });

      if (item.isSatisfied && item.satisfiedAt) {
        lines.push({
          timestamp: item.satisfiedAt,
          level: V1LogLineLevel.INFO,
          line: withContextPrefix(item, entryCompletionMessage(item)),
          taskDisplayName: item.taskDisplayName,
          taskExternalId: item.taskExternalId,
        });
      }
    }
  }

  return lines;
}
