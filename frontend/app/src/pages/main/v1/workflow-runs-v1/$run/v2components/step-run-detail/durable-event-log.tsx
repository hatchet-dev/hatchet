import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Loading } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import {
  V1DurableEventLogEntry,
  V1DurableEventLogKind,
  V1DurableWaitConditionKind,
  queries,
} from '@/lib/api';
import { cn } from '@/lib/utils';
import { useQuery } from '@tanstack/react-query';
import {
  CheckCircle2Icon,
  CircleIcon,
  ClockIcon,
  PlayIcon,
  ZapIcon,
} from 'lucide-react';

interface DurableEventLogProps {
  taskRunId: string;
}

export function DurableEventLog({ taskRunId }: DurableEventLogProps) {
  const logQuery = useQuery({
    ...queries.v1DurableTasks.eventLog(taskRunId),
    refetchInterval: 2000,
  });

  if (logQuery.isLoading) {
    return <Loading />;
  }

  const entries = logQuery.data ?? [];

  if (entries.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground">
        No durable event log entries yet.
      </div>
    );
  }

  return (
    <div className="py-4">
      <ol className="relative">
        {entries.map((entry, i) => (
          <DurableEventLogRow
            key={`${entry.branchId}-${entry.nodeId}`}
            entry={entry}
            isLast={i === entries.length - 1}
          />
        ))}
      </ol>
    </div>
  );
}

function DurableEventLogRow({
  entry,
  isLast,
}: {
  entry: V1DurableEventLogEntry;
  isLast: boolean;
}) {
  return (
    <li className="relative flex gap-4">
      {/* Vertical connector line */}
      {!isLast && (
        <div className="absolute left-[11px] top-6 bottom-0 w-px bg-border" />
      )}

      {/* Icon */}
      <div className="relative z-10 mt-1 flex-shrink-0">
        <EntryIcon entry={entry} />
      </div>

      {/* Content */}
      <div className="flex min-w-0 flex-1 flex-col gap-1 pb-6">
        <div className="flex flex-wrap items-center gap-2">
          <EntryKindBadge entry={entry} />
          {entry.branchId > 1 && (
            <span className="text-xs text-muted-foreground">
              branch {entry.branchId}
            </span>
          )}
          <span className="text-xs text-muted-foreground">
            <RelativeDate date={entry.insertedAt} />
          </span>
        </div>

        <p className="text-sm font-medium leading-snug">
          {entry.humanReadableMessage ?? kindLabel(entry.kind)}
        </p>

        {entry.userMessage &&
          entry.userMessage !== entry.humanReadableMessage && (
            <p className="text-xs text-muted-foreground italic">
              &ldquo;{entry.userMessage}&rdquo;
            </p>
          )}
        {entry.humanReadableMessage &&
          entry.userMessage !== entry.humanReadableMessage && (
            <p className="text-xs text-muted-foreground italic">
              &ldquo;{entry.humanReadableMessage}&rdquo;
            </p>
          )}
      </div>
    </li>
  );
}

function EntryIcon({ entry }: { entry: V1DurableEventLogEntry }) {
  const base = 'flex size-6 items-center justify-center rounded-full';

  if (entry.isSatisfied) {
    return (
      <div
        className={cn(
          base,
          'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-400',
        )}
      >
        <CheckCircle2Icon className="size-4" />
      </div>
    );
  }

  switch (entry.kind) {
    case V1DurableEventLogKind.RUN:
      return (
        <div
          className={cn(
            base,
            'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400',
          )}
        >
          <PlayIcon className="size-3.5" />
        </div>
      );
    case V1DurableEventLogKind.WAIT_FOR:
      return (
        <div
          className={cn(
            base,
            'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/40 dark:text-yellow-400',
          )}
        >
          <ClockIcon className="size-3.5" />
        </div>
      );
    case V1DurableEventLogKind.MEMO:
      return (
        <div
          className={cn(
            base,
            'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-400',
          )}
        >
          <ZapIcon className="size-3.5" />
        </div>
      );
    default:
      return (
        <div className={cn(base, 'bg-muted text-muted-foreground')}>
          <CircleIcon className="size-3.5" />
        </div>
      );
  }
}

function EntryKindBadge({ entry }: { entry: V1DurableEventLogEntry }) {
  switch (entry.kind) {
    case V1DurableEventLogKind.RUN:
      return (
        <Badge variant={entry.isSatisfied ? 'successful' : 'inProgress'}>
          Run
        </Badge>
      );
    case V1DurableEventLogKind.WAIT_FOR:
      return (
        <Badge variant={entry.isSatisfied ? 'successful' : 'queued'}>
          Wait
        </Badge>
      );
    case V1DurableEventLogKind.MEMO:
      return (
        <Badge variant={entry.isSatisfied ? 'successful' : 'evicted'}>
          Memo
        </Badge>
      );
    default:
      return <Badge variant="outline">{entry.kind}</Badge>;
  }
}

function kindLabel(kind: V1DurableEventLogKind): string {
  switch (kind) {
    case V1DurableEventLogKind.RUN:
      return 'Task started';
    case V1DurableEventLogKind.WAIT_FOR:
      return 'Waiting';
    case V1DurableEventLogKind.MEMO:
      return 'Memo';
    default:
      return kind;
  }
}
