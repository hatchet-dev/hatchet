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

const BULK_SPAWN_THRESHOLD_MS = 1000;

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
  const parts: string[] = [];

  for (const c of waitData.conditions ?? []) {
    parts.push(describeCondition(c));
  }
  for (const g of waitData.orGroups ?? []) {
    parts.push(describeOrGroup(g));
  }

  if (parts.length === 0) {
    return 'waiting';
  }
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
    const standalone = e.waitData?.conditions;
    if (standalone?.length === 1 && standalone[0].kind === V1DurableWaitConditionKind.CHILD_WORKFLOW) {
      return standalone[0].workflowName ?? null;
    }
    // legacy: single-condition OR group (already normalized server-side, but handle defensively)
    const orGroups = e.waitData?.orGroups;
    if (orGroups?.length === 1 && orGroups[0].conditions.length === 1) {
      const c = orGroups[0].conditions[0];
      if (c.kind === V1DurableWaitConditionKind.CHILD_WORKFLOW) {
        return c.workflowName ?? null;
      }
    }
    return null;
  });

  const allSame = names.every((n) => n === names[0]);
  if (allSame && names[0] !== null) {
    return `${entries.length}x run(${names[0]})`;
  }

  return `run(${names.map((n) => n ?? 'unknown').join(', ')})`;
}

function toDurableEventLogLines(entries: V1DurableEventLogEntry[]): LogLine[] {
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
      });

      const satisfiedEntries = groupEntries.filter((e) => e.isSatisfied && e.satisfiedAt);
      if (satisfiedEntries.length > 0) {
        const lastSatisfiedAt = satisfiedEntries.reduce((latest, e) =>
          e.satisfiedAt! > latest ? e.satisfiedAt! : latest,
          satisfiedEntries[0].satisfiedAt!,
        );
        lines.push({
          timestamp: lastSatisfiedAt,
          level: 'info',
          line: withContextPrefix(first, completionMessage(label)),
        });
      }
    } else {
      const message = entryMessage(item);

      lines.push({
        timestamp: item.insertedAt,
        level: item.kind === V1DurableEventLogKind.WAIT_FOR ? 'warn' : 'debug',
        line: withContextPrefix(item, capitalizeFirst(message)),
      });

      if (item.isSatisfied && item.satisfiedAt) {
        lines.push({
          timestamp: item.satisfiedAt,
          level: 'info',
          line: withContextPrefix(item, completionMessage(message)),
        });
      }
    }
  }

  return lines;
}
