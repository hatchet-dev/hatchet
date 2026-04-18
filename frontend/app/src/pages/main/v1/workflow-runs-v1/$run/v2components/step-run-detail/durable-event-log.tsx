import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { Loading } from '@/components/v1/ui/loading';
import {
  V1DurableEventLogEntry,
  V1DurableEventLogKind,
  V1DurableWaitCondition,
  V1DurableWaitConditionKind,
  V1DurableWaitOrGroup,
  V1WaitData,
  queries,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

interface DurableEventLogProps {
  taskRunId: string;
}

export function DurableEventLog({ taskRunId }: DurableEventLogProps) {
  const logQuery = useQuery({
    ...queries.v1DurableTasks.eventLog(taskRunId),
    refetchInterval: 2000,
  });

  const logs = useMemo(
    () => toDurableEventLogLines(logQuery.data ?? []),
    [logQuery.data],
  );

  if (logQuery.isLoading) {
    return <Loading />;
  }

  return (
    <div className="flex h-full min-h-[25rem] flex-1 flex-col">
      <LogViewer logs={logs} emptyMessage="No durable event log entries yet." />
    </div>
  );
}

function kindLabel(kind: V1DurableEventLogKind): string {
  switch (kind) {
    case V1DurableEventLogKind.RUN:
      return 'run';
    case V1DurableEventLogKind.WAIT_FOR:
      return 'wait';
    case V1DurableEventLogKind.MEMO:
      return 'memo';
    default:
      return String(kind).toLowerCase();
  }
}

function formatDurationMs(ms: number): string {
  if (ms === 0) return '0s';
  if (ms % 3_600_000 === 0) return `${ms / 3_600_000}h`;
  if (ms % 60_000 === 0) return `${ms / 60_000}m`;
  if (ms % 1_000 === 0) return `${ms / 1_000}s`;
  return `${ms}ms`;
}

function describeCondition(c: V1DurableWaitCondition): string {
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
      return String(c.kind).toLowerCase();
  }
}

function describeOrGroup(group: V1DurableWaitOrGroup): string {
  if (group.conditions.length === 0) {
    return 'waiting';
  }
  const parts = group.conditions.map(describeCondition);
  return parts.length === 1 ? parts[0] : `any of: ${parts.join(', ')}`;
}

function toReadableMessage(waitData: V1WaitData): string {
  if (waitData.orGroups.length === 0) {
    return 'waiting';
  }
  const parts = waitData.orGroups.map(describeOrGroup);
  return parts.length === 1 ? parts[0] : parts.join(' and ');
}

function entryMessage(entry: V1DurableEventLogEntry): string {
  if (entry.waitData) {
    return toReadableMessage(entry.waitData);
  }
  return entry.userMessage ?? kindLabel(entry.kind);
}

function capitalizeFirst(value: string): string {
  if (!value) {
    return value;
  }

  return value.charAt(0).toUpperCase() + value.slice(1);
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

function toDurableEventLogLines(entries: V1DurableEventLogEntry[]): LogLine[] {
  const lines: LogLine[] = [];

  for (const entry of entries) {
    if (entry.kind === V1DurableEventLogKind.MEMO) {
      continue;
    }

    const message = entryMessage(entry);

    lines.push({
      timestamp: entry.insertedAt,
      level: entry.kind === V1DurableEventLogKind.WAIT_FOR ? 'warn' : 'debug',
      line: withContextPrefix(entry, capitalizeFirst(message)),
    });

    if (entry.isSatisfied && entry.satisfiedAt) {
      lines.push({
        timestamp: entry.satisfiedAt,
        level: 'info',
        line: withContextPrefix(entry, completionMessage(message)),
      });
    }
  }

  return lines;
}
