import type { SpanGroupInfo } from './timeline/trace-timeline-utils';
import { formatDuration, formatTimestamp } from './utils/format-utils';
import { isQueuedOnly, statusLabel } from './utils/span-tree-utils';
import { useLiveClock } from './utils/use-live-clock';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import type { ParsedTraceQuery } from '@/components/v1/cloud/observability/trace-search';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { Card } from '@/components/v1/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { useSidePanel } from '@/hooks/use-side-panel';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { cn } from '@/lib/utils';
import { Filter, Minus, PanelRight, Plus, X } from 'lucide-react';
import { useCallback, useMemo } from 'react';

function FilterWithBadgeIcon({
  className,
  variant,
}: {
  className?: string;
  variant: 'plus' | 'minus';
}) {
  const Badge = variant === 'plus' ? Plus : Minus;
  return (
    <span className={cn('relative inline-flex size-3.5', className)}>
      <Filter className="size-full" />
      <Badge
        className="pointer-events-none absolute -bottom-0.5 -right-0.5 size-2.5"
        strokeWidth={2.5}
        aria-hidden
      />
    </span>
  );
}

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

function isFilterActive(
  activeFilters: ParsedTraceQuery | undefined,
  key: string,
  value: string,
): boolean {
  if (!activeFilters) {
    return false;
  }
  if (key.toLowerCase() === 'status') {
    return activeFilters.status === value.toLowerCase();
  }
  return activeFilters.attributes.some(([k, v]) => k === key && v === value);
}

function AttrTable({
  entries,
  title,
  activeFilters,
  onAddFilter,
  onRemoveFilter,
}: {
  entries: [string, string][];
  title: string;
  activeFilters?: ParsedTraceQuery;
  onAddFilter?: (key: string, value: string) => void;
  onRemoveFilter?: (key: string, value: string) => void;
}) {
  if (entries.length === 0) {
    return null;
  }

  return (
    <div>
      <h4 className="mb-1.5 text-xs font-medium uppercase tracking-wider text-muted-foreground">
        {title}
      </h4>
      <div className="overflow-hidden rounded-md border border-border bg-background">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Key</TableHead>
              <TableHead>Value</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {entries.map(([key, value]) => {
              const active = isFilterActive(activeFilters, key, value);
              return (
                <TableRow key={key} className="group">
                  <TableCell className="whitespace-nowrap font-mono text-muted-foreground">
                    {key}
                  </TableCell>
                  <TableCell className="break-all font-mono">
                    <span className="flex items-center justify-between gap-2">
                      <span>{value}</span>
                      {(onAddFilter || onRemoveFilter) && (
                        <Button
                          size="icon"
                          variant="ghost"
                          className={cn(
                            'size-6 shrink-0 transition-opacity',
                            active
                              ? 'opacity-100'
                              : 'opacity-0 group-hover:opacity-100',
                          )}
                          hoverText={
                            active
                              ? `Remove filter ${key}:${value}`
                              : `Filter by ${key}:${value}`
                          }
                          onClick={() =>
                            active
                              ? onRemoveFilter?.(key, value)
                              : onAddFilter?.(key, value)
                          }
                        >
                          {active ? (
                            <FilterWithBadgeIcon variant="minus" />
                          ) : (
                            <FilterWithBadgeIcon variant="plus" />
                          )}
                        </Button>
                      )}
                    </span>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

export function SpanDetail({
  span,
  onClose,
  activeFilters,
  onAddFilter,
  onRemoveFilter,
  onSpanSelect,
}: {
  span: OtelSpanTree;
  onClose: () => void;
  activeFilters?: ParsedTraceQuery;
  onAddFilter?: (key: string, value: string) => void;
  onRemoveFilter?: (key: string, value: string) => void;
  onSpanSelect?: (span: OtelSpanTree) => void;
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
  const { open } = useSidePanel();

  const handleOpenTaskRun = useCallback(() => {
    if (!taskRunId) {
      return;
    }
    open({
      type: 'task-run-details',
      content: {
        taskRunId,
        showViewTaskRunButton: true,
      },
    });
  }, [taskRunId, open]);

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
    <Card className="flex min-h-0 flex-1 flex-col">
      <div className="shrink-0 space-y-4 border-b border-border p-4">
        <div className="flex items-start justify-between gap-4">
          <div className="min-w-0">
            <h3 className="truncate font-mono text-sm font-semibold text-foreground">
              {span.spanName}
            </h3>
            <p className="mt-1 font-mono text-xs text-muted-foreground">
              {span.spanId}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-1">
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
            <Button
              type="button"
              size="icon"
              variant="ghost"
              className="size-8 shrink-0 text-muted-foreground"
              onClick={onClose}
              hoverText="Close"
              aria-label="Close panel"
            >
              <X className="size-4" />
            </Button>
          </div>
        </div>

        <div className={cn('grid gap-4', q ? 'grid-cols-4' : 'grid-cols-3')}>
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

      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        <div className="flex flex-col gap-4">
          {span.statusCode === OtelStatusCode.ERROR && span.statusMessage && (
            <Alert variant="destructive">
              <AlertTitle>Error Message</AlertTitle>
              <AlertDescription>
                <pre className="whitespace-pre-wrap break-words font-mono text-xs">
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
                    className="transition-colors hover:bg-destructive/10"
                  >
                    <AlertTitle>{err.spanName}</AlertTitle>
                    <AlertDescription>
                      <pre className="whitespace-pre-wrap break-words font-mono text-xs">
                        {err.message}
                      </pre>
                    </AlertDescription>
                  </Alert>
                </button>
              ))}
            </div>
          )}

          {(user.length > 0 || hatchet.length > 0) && (
            <div className="flex flex-col gap-3">
              <AttrTable
                entries={user}
                title="Attributes"
                activeFilters={activeFilters}
                onAddFilter={onAddFilter}
                onRemoveFilter={onRemoveFilter}
              />
              <AttrTable
                entries={hatchet}
                title="Hatchet Attributes"
                activeFilters={activeFilters}
                onAddFilter={onAddFilter}
                onRemoveFilter={onRemoveFilter}
              />
            </div>
          )}
        </div>
      </div>
    </Card>
  );
}

export function GroupDetail({
  group,
  onClose,
}: {
  group: SpanGroupInfo;
  onClose: () => void;
}) {
  const timeRangeMs = group.latestEndMs - group.earliestStartMs;
  const durations = group.spans.map((s) => s.durationNs / 1_000_000);
  const avgMs = durations.reduce((sum, d) => sum + d, 0) / durations.length;
  const minMs = Math.min(...durations);
  const maxMs = Math.max(...durations);

  return (
    <Card className="flex min-h-0 flex-1 flex-col">
      <div className="shrink-0 space-y-4 p-4">
        <div className="flex items-start justify-between gap-4">
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
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="size-8 shrink-0 text-muted-foreground"
            onClick={onClose}
            hoverText="Close"
            aria-label="Close panel"
          >
            <X className="size-4" />
          </Button>
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
    </Card>
  );
}
