import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { Loading } from '@/components/v1/ui/loading';
import {
  V1DurableEventLogEntry,
  V1DurableEventLogKind,
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

function entryMessage(entry: V1DurableEventLogEntry): string {
  return (
    entry.humanReadableMessage ?? entry.userMessage ?? kindLabel(entry.kind)
  );
}

function capitalizeFirst(value: string): string {
  if (!value) {
    return value;
  }

  return value.charAt(0).toUpperCase() + value.slice(1);
}

function completionMessage(
  entry: V1DurableEventLogEntry,
  message: string,
): string {
  if (entry.kind === V1DurableEventLogKind.WAIT_FOR) {
    const stripped = message.replace(/^waiting for\s+/i, '').trim();

    if (stripped.length > 0) {
      return `${capitalizeFirst(stripped)} completed`;
    }
  }

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
        line: withContextPrefix(entry, completionMessage(entry, message)),
      });
    }
  }

  return lines;
}
