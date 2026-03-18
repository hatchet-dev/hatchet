import type { ParsedTraceQuery } from './trace-search/types';
import type { SpanGroupInfo } from './trace-timeline';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { Button } from '@/components/v1/ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { cn } from '@/lib/utils';
import { PanelRight, X } from 'lucide-react';
import { useCallback } from 'react';

function FilterPlusIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      {/* funnel */}
      <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
      {/* plus badge */}
      <line x1="19" y1="15" x2="19" y2="21" />
      <line x1="16" y1="18" x2="22" y2="18" />
    </svg>
  );
}

function FilterMinusIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
    >
      {/* funnel */}
      <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3" />
      {/* minus badge */}
      <line x1="16" y1="18" x2="22" y2="18" />
    </svg>
  );
}

function formatDuration(ns: number): string {
  const ms = ns / 1_000_000;
  if (ms < 1) {
    return `${(ns / 1_000).toFixed(1)}µs`;
  }
  if (ms < 1000) {
    return `${ms.toFixed(ms < 10 ? 2 : 1)}ms`;
  }
  if (ms < 60_000) {
    return `${(ms / 1000).toFixed(2)}s`;
  }
  const m = Math.floor(ms / 60_000);
  const s = ((ms % 60_000) / 1000).toFixed(1);
  return `${m}m ${s}s`;
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
    hour12: true,
  });
}

function statusConfig(code: string) {
  switch (code) {
    case OtelStatusCode.OK:
      return { label: 'OK', dot: 'bg-green-500' };
    case OtelStatusCode.ERROR:
      return { label: 'Error', dot: 'bg-red-500' };
    default:
      return { label: 'Unset', dot: 'bg-slate-500' };
  }
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
      <div className="overflow-hidden rounded-md border border-border">
        <table className="w-full text-xs">
          <tbody>
            {entries.map(([key, value]) => {
              const active = isFilterActive(activeFilters, key, value);
              return (
                <tr
                  key={key}
                  className="group border-b border-border last:border-b-0 transition-colors hover:bg-muted/50"
                >
                  <td className="whitespace-nowrap px-3 py-1.5 font-mono text-muted-foreground">
                    {key}
                  </td>
                  <td className="break-all px-3 py-1.5 font-mono text-foreground">
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
                            <FilterMinusIcon className="size-3.5" />
                          ) : (
                            <FilterPlusIcon className="size-3.5" />
                          )}
                        </Button>
                      )}
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
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
}: {
  span: OtelSpanTree;
  onClose: () => void;
  activeFilters?: ParsedTraceQuery;
  onAddFilter?: (key: string, value: string) => void;
  onRemoveFilter?: (key: string, value: string) => void;
}) {
  const status = statusConfig(span.statusCode);
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

  return (
    <div className="flex flex-col gap-4 rounded-lg border border-border bg-background p-4">
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
          <button
            onClick={onClose}
            className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <X className="size-4" />
          </button>
        </div>
      </div>

      <div className={cn('grid gap-4', span.queuedPhase ? 'grid-cols-4' : 'grid-cols-3')}>
        {span.queuedPhase && (
          <div>
            <span className="text-xs text-muted-foreground">Queue Time</span>
            <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
              {formatDuration(span.queuedPhase.durationNs)}
            </p>
          </div>
        )}
        <div>
          <span className="text-xs text-muted-foreground">Duration</span>
          <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
            {formatDuration(span.durationNs)}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Status</span>
          <div className="mt-0.5 flex items-center gap-1.5">
            <span className={cn('size-2 shrink-0 rounded-full', status.dot)} />
            <span className="font-mono text-sm text-foreground">
              {status.label}
            </span>
          </div>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Started</span>
          <p className="mt-0.5 font-mono text-sm text-foreground">
            {formatTimestamp(span.createdAt)}
          </p>
        </div>
      </div>

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
  );
}

function formatDurationMs(ms: number): string {
  if (ms < 1) {
    return '<1ms';
  }
  if (ms < 1000) {
    return `${ms.toFixed(ms < 10 ? 2 : 1)}ms`;
  }
  if (ms < 60_000) {
    return `${(ms / 1000).toFixed(2)}s`;
  }
  const m = Math.floor(ms / 60_000);
  const s = ((ms % 60_000) / 1000).toFixed(1);
  return `${m}m ${s}s`;
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
    <div className="flex flex-col gap-4 rounded-lg border border-border bg-background p-4">
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
        <button
          onClick={onClose}
          className="shrink-0 rounded p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <X className="size-4" />
        </button>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <div>
          <span className="text-xs text-muted-foreground">Time Range</span>
          <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
            {formatDurationMs(timeRangeMs)}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Avg Duration</span>
          <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
            {formatDurationMs(avgMs)}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Min Duration</span>
          <p className="mt-0.5 font-mono text-sm text-foreground">
            {formatDurationMs(minMs)}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Max Duration</span>
          <p className="mt-0.5 font-mono text-sm text-foreground">
            {formatDurationMs(maxMs)}
          </p>
        </div>
      </div>
    </div>
  );
}
