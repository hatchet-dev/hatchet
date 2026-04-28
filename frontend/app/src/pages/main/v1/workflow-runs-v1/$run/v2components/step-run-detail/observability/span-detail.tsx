import type { SpanGroupInfo } from './timeline/trace-timeline-utils';
import { formatDuration, formatTimestamp } from './utils/format-utils';
import { isQueuedOnly, statusLabel } from './utils/span-tree-utils';
import { useLiveClock } from './utils/use-live-clock';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { PanelRight } from 'lucide-react';
import { useCallback, useMemo } from 'react';

function statusBadgeVariant(code: string): 'successful' | 'failed' | 'queued' {
  if (code === OtelStatusCode.ERROR) {
    return 'failed';
  }
  if (code === OtelStatusCode.OK) {
    return 'successful';
  }
  return 'queued';
}

interface ChildError {
  span: OtelSpanTree;
  spanName: string;
  message: string;
}

function collectChildErrors(node: OtelSpanTree): ChildError[] {
  const errors: ChildError[] = [];
  const stack = [...node.children];
  while (stack.length > 0) {
    const child = stack.pop()!;
    if (child.statusCode === OtelStatusCode.ERROR && child.statusMessage) {
      const taskName =
        child.spanAttributes?.['hatchet.step_name'] ??
        child.spanAttributes?.['hatchet.task_name'];
      errors.push({
        span: child,
        spanName: taskName ?? child.spanName,
        message: child.statusMessage,
      });
    }
    stack.push(...child.children);
  }
  errors.sort(
    (a, b) =>
      new Date(a.span.createdAt).getTime() -
      new Date(b.span.createdAt).getTime(),
  );
  return errors;
}

const HATCHET_ATTR_PREFIX = 'hatchet.';

function partitionAttributes(attrs: Record<string, string> | undefined) {
  const hatchet: [string, string][] = [];
  const user: [string, string][] = [];

  if (!attrs) {
    return { hatchet, user };
  }

  for (const [key, value] of Object.entries(attrs)) {
    if (key.startsWith(HATCHET_ATTR_PREFIX)) {
      hatchet.push([key, value]);
    } else {
      user.push([key, value]);
    }
  }

  hatchet.sort((a, b) => a[0].localeCompare(b[0]));
  user.sort((a, b) => a[0].localeCompare(b[0]));

  return { hatchet, user };
}

const attrColumns = [
  {
    columnLabel: 'Key',
    cellRenderer: ([key]: [string, string]) => (
      <span className="whitespace-nowrap text-xs text-muted-foreground">
        {key}
      </span>
    ),
  },
  {
    columnLabel: 'Value',
    cellRenderer: ([, value]: [string, string]) => (
      <span className="break-all text-xs">{value}</span>
    ),
  },
];

export function SpanDetail({
  span,
  onSpanSelect,
  onOpenTaskRun,
}: {
  span: OtelSpanTree;
  onClose?: () => void;
  onSpanSelect?: (span: OtelSpanTree) => void;
  onOpenTaskRun?: (taskRunId: string) => void;
}) {
  const isLive = !!span.inProgress || isQueuedOnly(span);
  const now = useLiveClock(isLive);

  const queuedOnlySpan = isQueuedOnly(span);
  const startMs = new Date(span.createdAt).getTime();

  const durationNs = span.inProgress
    ? Math.max(0, now - startMs) * 1_000_000
    : span.durationNs;

  const q = span.queuedPhase;
  let queueNs = q ? q.durationNs : 0;
  if (q && queuedOnlySpan) {
    const qStartMs = new Date(q.createdAt).getTime();
    queueNs = Math.max(0, now - qStartMs) * 1_000_000;
  }

  const status = queuedOnlySpan
    ? { label: 'Queued', variant: 'queued' as const }
    : span.inProgress
      ? { label: 'In Progress', variant: 'inProgress' as const }
      : {
          label: statusLabel(span.statusCode),
          variant: statusBadgeVariant(span.statusCode),
        };
  const { hatchet, user } = partitionAttributes(span.spanAttributes);
  const taskRunId = span.spanAttributes?.['hatchet.step_run_id'];

  const handleOpenTaskRun = useCallback(() => {
    if (!taskRunId) {
      return;
    }
    onOpenTaskRun?.(taskRunId);
  }, [taskRunId, onOpenTaskRun]);

  const childErrors = useMemo(() => {
    if (
      span.statusCode !== OtelStatusCode.ERROR ||
      span.children.length === 0
    ) {
      return [];
    }
    return collectChildErrors(span);
  }, [span]);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h3 className="truncate font-mono text-sm font-semibold text-foreground">
              {span.spanName}
            </h3>
            <p className="mt-1 font-mono text-xs text-muted-foreground">
              {span.spanId}
            </p>
          </div>
          {taskRunId && (
            <Button
              size="sm"
              variant="outline"
              onClick={handleOpenTaskRun}
              leftIcon={<PanelRight className="size-4" />}
            >
              View Task Run
            </Button>
          )}
        </div>

        <div className="flex flex-wrap gap-x-6 gap-y-3">
          {q && (
            <div>
              <span className="text-xs text-muted-foreground">Queue Time</span>
              <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
                {formatDuration(queueNs, { unit: 'ns', precise: true })}
              </p>
            </div>
          )}
          <div>
            <span className="text-xs text-muted-foreground">Duration</span>
            <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
              {queuedOnlySpan
                ? '–'
                : formatDuration(durationNs, { unit: 'ns', precise: true })}
            </p>
          </div>
          <div>
            <span className="text-xs text-muted-foreground">Status</span>
            <div className="mt-0.5">
              <Badge variant={status.variant}>{status.label}</Badge>
            </div>
          </div>
          <div>
            <span className="text-xs text-muted-foreground">Started</span>
            <p className="mt-0.5 font-mono text-sm text-foreground">
              {formatTimestamp(span.createdAt)}
            </p>
          </div>
        </div>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto">
        {span.statusCode === OtelStatusCode.ERROR && span.statusMessage && (
          <Alert variant="destructive">
            <AlertTitle>Error Message</AlertTitle>
            <AlertDescription>
              <pre className="whitespace-pre-wrap break-words text-xs">
                {span.statusMessage}
              </pre>
            </AlertDescription>
          </Alert>
        )}

        {childErrors.length > 0 && (
          <div className="flex flex-col gap-2">
            {childErrors.map((err, i) => (
              <button
                key={i}
                type="button"
                className="w-full text-left"
                onClick={() => onSpanSelect?.(err.span)}
              >
                <Alert
                  variant="destructive"
                  className="transition-colors hover:bg-red-100 dark:hover:bg-red-950/60"
                >
                  <AlertTitle>{err.spanName}</AlertTitle>
                  <AlertDescription>
                    <pre className="whitespace-pre-wrap break-words text-xs">
                      {err.message}
                    </pre>
                  </AlertDescription>
                </Alert>
              </button>
            ))}
          </div>
        )}

        {user.length > 0 && (
          <div>
            <h4 className="mb-1.5 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Attributes
            </h4>
            <SimpleTable
              columns={attrColumns}
              data={user}
              rowKey={([key]) => key}
            />
          </div>
        )}
        {hatchet.length > 0 && (
          <div>
            <h4 className="mb-1.5 text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Hatchet Attributes
            </h4>
            <SimpleTable
              columns={attrColumns}
              data={hatchet}
              rowKey={([key]) => key}
            />
          </div>
        )}
      </div>
    </div>
  );
}

export function GroupDetail({
  group,
}: {
  group: SpanGroupInfo;
  onClose?: () => void;
}) {
  const timeRangeMs = group.latestEndMs - group.earliestStartMs;
  const durations = group.spans.map((s) => s.durationNs / 1_000_000);
  const avgMs = durations.reduce((sum, d) => sum + d, 0) / durations.length;
  const minMs = Math.min(...durations);
  const maxMs = Math.max(...durations);

  return (
    <div className="space-y-4">
      <div className="min-w-0">
        <h3 className="truncate font-mono text-sm font-semibold text-foreground">
          {group.groupName}
        </h3>
        <p className="mt-1 font-mono text-xs text-muted-foreground">
          {group.totalCount.toLocaleString()} spans
          {group.errorCount > 0 && (
            <span className="text-red-500">
              {' '}
              · {group.errorCount.toLocaleString()} errors
            </span>
          )}
        </p>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div>
          <span className="text-xs text-muted-foreground">Time Range</span>
          <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
            {formatDuration(timeRangeMs, { precise: true })}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Avg Duration</span>
          <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
            {formatDuration(avgMs, { precise: true })}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Min Duration</span>
          <p className="mt-0.5 font-mono text-sm text-foreground">
            {formatDuration(minMs, { precise: true })}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Max Duration</span>
          <p className="mt-0.5 font-mono text-sm text-foreground">
            {formatDuration(maxMs, { precise: true })}
          </p>
        </div>
      </div>
    </div>
  );
}
