import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { Button } from '@/components/v1/ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { cn } from '@/lib/utils';
import { PanelRight, X } from 'lucide-react';
import { useCallback } from 'react';

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
      return { label: 'OK', dot: 'bg-success' };
    case OtelStatusCode.ERROR:
      return { label: 'Error', dot: 'bg-danger' };
    default:
      return { label: 'Unset', dot: 'bg-muted-foreground' };
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

function AttrTable({
  entries,
  title,
}: {
  entries: [string, string][];
  title: string;
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
            {entries.map(([key, value]) => (
              <tr key={key} className="border-b border-border last:border-b-0">
                <td className="whitespace-nowrap px-3 py-1.5 font-mono text-muted-foreground">
                  {key}
                </td>
                <td className="break-all px-3 py-1.5 font-mono text-foreground">
                  {value}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export function SpanDetail({
  span,
  onClose,
}: {
  span: OtelSpanTree;
  onClose: () => void;
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

      <div className="grid grid-cols-3 gap-4">
        <div>
          <span className="text-xs text-muted-foreground">Duration</span>
          <p className="mt-0.5 font-mono text-sm font-medium text-foreground">
            {formatDuration(span.durationNs)}
          </p>
        </div>
        <div>
          <span className="text-xs text-muted-foreground">Status</span>
          <div className="mt-0.5 flex items-center gap-1.5">
            <span
              className={cn('size-2 shrink-0 rounded-full', status.dot)}
            />
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
          <AttrTable entries={user} title="Attributes" />
          <AttrTable entries={hatchet} title="Hatchet Attributes" />
        </div>
      )}
    </div>
  );
}
