import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { MarkdownRenderer } from '@/components/v1/ui/markdown';
import {
  ConcurrencyLimitStrategy,
  ConcurrencyScope,
  ConcurrencySetting,
  WorkflowVersion,
  WorkflowVersionTask,
} from '@/lib/api';
import { formatCron } from '@/lib/cron';
import { cn } from '@/lib/utils';
import { useLayoutEffect, useMemo, useRef, useState } from 'react';

function formatLimitStrategy(strategy: ConcurrencyLimitStrategy): string {
  switch (strategy) {
    case ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS:
      return 'Cancel In Progress';
    case ConcurrencyLimitStrategy.DROP_NEWEST:
      return 'Drop Newest';
    case ConcurrencyLimitStrategy.QUEUE_NEWEST:
      return 'Queue Newest';
    case ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN:
      return 'Group Round Robin';
    default: {
      const exhaustiveCheck: never = strategy;
      return exhaustiveCheck;
    }
  }
}

function formatScope(scope: ConcurrencyScope): string {
  switch (scope) {
    case ConcurrencyScope.WORKFLOW:
      return 'Workflow';
    case ConcurrencyScope.TASK:
      return 'Task';
    default: {
      const exhaustiveCheck: never = scope;
      return exhaustiveCheck;
    }
  }
}

function formatBackoff(task: WorkflowVersionTask): string {
  if (!task.retryBackoffFactor) {
    return '—';
  }

  return `x${task.retryBackoffFactor}${
    task.retryBackoffMaxSeconds ? ` up to ${task.retryBackoffMaxSeconds}s` : ''
  }`;
}

function formatRateLimits(task: WorkflowVersionTask): string {
  if (!task.rateLimits || task.rateLimits.length === 0) {
    return '—';
  }

  return task.rateLimits
    .map((rl) => rl.key ?? rl.keyExpression ?? '')
    .filter(Boolean)
    .join(', ');
}

export default function WorkflowGeneralSettings({
  workflow,
  onDeleteClick,
}: {
  workflow: WorkflowVersion;
  onDeleteClick: () => void;
}) {
  const hasEvents =
    !!workflow.triggers?.events && workflow.triggers.events.length > 0;
  const hasCrons =
    !!workflow.triggers?.crons && workflow.triggers.crons.length > 0;
  const hasTriggers = hasEvents || hasCrons;
  const hasConcurrency =
    !!workflow.v1Concurrency && workflow.v1Concurrency.length > 0;
  const hasExecution =
    !!workflow.idempotency ||
    !!workflow.sticky ||
    !!workflow.defaultPriority ||
    !!workflow.scheduleTimeout;
  const tasks = workflow.tasks ?? [];

  return (
    <div className="px-1">
      {workflow.description && (
        <Section title="Description" className="mb-6">
          <MarkdownRenderer content={workflow.description} />
        </Section>
      )}

      <div className="flex flex-wrap items-start gap-5">
        <div className="flex w-[340px] max-w-full shrink-0 flex-col gap-6">
          {hasTriggers && (
            <Section title="Triggers">
              <TriggerSettings workflow={workflow} />
            </Section>
          )}

          {hasConcurrency && (
            <Section title="Concurrency">
              <ConcurrencySettings concurrency={workflow.v1Concurrency ?? []} />
            </Section>
          )}

          {hasExecution && (
            <Section title="Execution">
              <ExecutionSettings workflow={workflow} />
            </Section>
          )}

          <DangerZone onDeleteClick={onDeleteClick} />
        </div>

        <div className="min-w-[min(100%,420px)] flex-1">
          {tasks.length > 0 ? (
            <TasksGraph tasks={tasks} />
          ) : (
            <Section title="Tasks">
              <p className="text-sm italic text-muted-foreground">
                This workflow has no tasks.
              </p>
            </Section>
          )}
        </div>
      </div>
    </div>
  );
}

function Section({
  title,
  titleRight,
  className,
  children,
}: {
  title: string;
  titleRight?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div className={className}>
      <div className="mb-3 flex items-center justify-between gap-2 border-b border-border pb-2">
        <h3 className="text-[15px] font-semibold text-foreground">{title}</h3>
        {titleRight}
      </div>
      {children}
    </div>
  );
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return <div className="mb-1 text-xs text-muted-foreground">{children}</div>;
}

function ExpressionBlock({ code }: { code: string }) {
  return (
    <CodeHighlighter
      language="text"
      className="whitespace-pre-wrap break-words text-xs leading-relaxed"
      code={code}
      copy={false}
      maxHeight="8rem"
    />
  );
}

function TriggerSettings({ workflow }: { workflow: WorkflowVersion }) {
  return (
    <div className="space-y-4">
      {workflow.triggers?.events && workflow.triggers.events.length > 0 && (
        <div>
          <FieldLabel>Events</FieldLabel>
          <div className="flex flex-wrap gap-1.5">
            {workflow.triggers.events.map((event) => (
              <Badge
                key={event.event_key}
                variant="secondary"
                className="font-mono text-xs"
              >
                {event.event_key}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {workflow.triggers?.crons && workflow.triggers.crons.length > 0 && (
        <div>
          <FieldLabel>Cron Schedules</FieldLabel>
          <div className="space-y-2">
            {workflow.triggers.crons.map((cronTrigger) => (
              <div key={cronTrigger.cron}>
                <Badge variant="secondary" className="font-mono text-xs">
                  {cronTrigger.cron}
                </Badge>
                {cronTrigger.cron && (
                  <p className="mt-1 text-xs text-muted-foreground">
                    Runs {formatCron(cronTrigger.cron)}
                  </p>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function KeyValueRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right text-foreground">{children}</span>
    </div>
  );
}

function ConcurrencySettings({
  concurrency,
}: {
  concurrency: ConcurrencySetting[];
}) {
  return (
    <div className="space-y-5">
      {concurrency.map((c, i) => (
        <div key={`${c.scope}-${c.stepReadableId ?? 'workflow'}-${i}`}>
          {concurrency.length > 1 && c.stepReadableId && (
            <div className="mb-2 font-mono text-xs font-semibold text-foreground">
              {c.stepReadableId}
            </div>
          )}
          <div className="space-y-2">
            <KeyValueRow label="Scope">{formatScope(c.scope)}</KeyValueRow>
            <KeyValueRow label="Max">{c.maxRuns}</KeyValueRow>
            <KeyValueRow label="Strategy">
              <Badge variant="secondary" className="font-normal">
                {formatLimitStrategy(c.limitStrategy)}
              </Badge>
            </KeyValueRow>
            <div>
              <FieldLabel>Expression</FieldLabel>
              <ExpressionBlock code={c.expression} />
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function ExecutionSettings({ workflow }: { workflow: WorkflowVersion }) {
  return (
    <div className="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
      {workflow.idempotency && (
        <div className="col-span-2">
          <FieldLabel>Idempotency Expression</FieldLabel>
          <ExpressionBlock code={workflow.idempotency.expression} />
        </div>
      )}
      {workflow.idempotency && (
        <div>
          <FieldLabel>Idempotency TTL</FieldLabel>
          <div className="text-foreground">
            {workflow.idempotency.ttlMs.toLocaleString()} ms
          </div>
        </div>
      )}
      {workflow.sticky && (
        <div>
          <FieldLabel>Sticky Strategy</FieldLabel>
          <div className="font-mono text-foreground">{workflow.sticky}</div>
        </div>
      )}
      {workflow.defaultPriority != null && (
        <div>
          <FieldLabel>Default Priority</FieldLabel>
          <div className="text-foreground">{workflow.defaultPriority}</div>
        </div>
      )}
      {workflow.scheduleTimeout && (
        <div>
          <FieldLabel>Schedule Timeout</FieldLabel>
          <div className="font-mono text-foreground">
            {workflow.scheduleTimeout}
          </div>
        </div>
      )}
    </div>
  );
}

function DangerZone({ onDeleteClick }: { onDeleteClick: () => void }) {
  return (
    <div className="rounded-[10px] border border-destructive/40 bg-destructive/[0.04] p-4">
      <div className="text-sm font-semibold text-foreground">Danger Zone</div>
      <p className="mt-1 text-xs text-muted-foreground">
        Permanently delete this workflow and all its data. This action cannot be
        undone.
      </p>
      <Button
        variant="destructive"
        size="sm"
        className="mt-3"
        onClick={onDeleteClick}
      >
        Delete Workflow
      </Button>
    </div>
  );
}

function computeColumns(tasks: WorkflowVersionTask[]): WorkflowVersionTask[][] {
  const byId = new Map(tasks.map((t) => [t.readableId, t]));
  const columns: WorkflowVersionTask[][] = [];
  const processed = new Set<string>();

  const columnIndexOf = (id: string) =>
    columns.findIndex((col) => col.some((t) => t.readableId === id));

  const process = (task: WorkflowVersionTask) => {
    if (processed.has(task.readableId)) {
      return;
    }

    const parents = task.parents.filter((p) => byId.has(p));

    if (parents.length === 0) {
      (columns[0] ??= []).push(task);
      processed.add(task.readableId);
      return;
    }

    const parentColumns = parents.map(columnIndexOf);

    if (parentColumns.some((c) => c === -1)) {
      return;
    }

    const index = Math.max(...parentColumns) + 1;
    (columns[index] ??= []).push(task);
    processed.add(task.readableId);
  };

  let iterations = tasks.length + 1;
  while (processed.size < tasks.length && iterations-- > 0) {
    tasks.forEach(process);
  }

  const leftover = tasks.filter((t) => !processed.has(t.readableId));
  if (leftover.length > 0) {
    columns.push(leftover);
  }

  return columns;
}

function TasksGraph({ tasks }: { tasks: WorkflowVersionTask[] }) {
  const columns = useMemo(() => computeColumns(tasks), [tasks]);
  const [dense, setDense] = useState(false);
  const [selectedId, setSelectedId] = useState<string>(
    () =>
      tasks.find((t) => t.parents.length === 0)?.readableId ??
      tasks[0]?.readableId ??
      '',
  );

  const containerRef = useRef<HTMLDivElement>(null);
  const nodeRefs = useRef(new Map<string, HTMLDivElement>());
  const [edges, setEdges] = useState<{ id: string; d: string }[]>([]);

  useLayoutEffect(() => {
    const compute = () => {
      const container = containerRef.current;
      if (!container) {
        return;
      }

      const containerRect = container.getBoundingClientRect();
      const paths: { id: string; d: string }[] = [];

      for (const task of tasks) {
        const childEl = nodeRefs.current.get(task.readableId);
        if (!childEl) {
          continue;
        }

        const childRect = childEl.getBoundingClientRect();

        for (const parentId of task.parents) {
          const parentEl = nodeRefs.current.get(parentId);
          if (!parentEl) {
            continue;
          }

          const parentRect = parentEl.getBoundingClientRect();
          const x1 = parentRect.right - containerRect.left;
          const y1 = parentRect.top - containerRect.top + parentRect.height / 2;
          const x2 = childRect.left - containerRect.left;
          const y2 = childRect.top - containerRect.top + childRect.height / 2;
          const midX = (x1 + x2) / 2;

          paths.push({
            id: `${parentId}->${task.readableId}`,
            d: `M ${x1} ${y1} C ${midX} ${y1}, ${midX} ${y2}, ${x2} ${y2}`,
          });
        }
      }

      setEdges(paths);
    };

    compute();

    const observer = new ResizeObserver(compute);
    if (containerRef.current) {
      observer.observe(containerRef.current);
    }
    window.addEventListener('resize', compute);

    return () => {
      observer.disconnect();
      window.removeEventListener('resize', compute);
    };
  }, [tasks, columns, dense]);

  const selected = tasks.find((t) => t.readableId === selectedId) ?? tasks[0];

  return (
    <Section
      title="Tasks"
      titleRight={
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <button
            type="button"
            onClick={() => setDense((d) => !d)}
            className="hover:text-foreground"
          >
            {dense ? 'Hide details' : 'Show details'}
          </button>
          <span>
            {tasks.length} task{tasks.length === 1 ? '' : 's'} · click a node
            for details
          </span>
        </div>
      }
    >
      <div ref={containerRef} className="relative flex min-h-[300px] w-full">
        <svg className="pointer-events-none absolute inset-0 h-full w-full">
          {edges.map((edge) => (
            <path
              key={edge.id}
              d={edge.d}
              className="text-foreground/25"
              stroke="currentColor"
              strokeWidth={1.5}
              fill="none"
            />
          ))}
        </svg>
        <div className="relative flex w-full flex-row gap-x-10">
          {columns.map((column, colIndex) => (
            <div
              key={colIndex}
              className="flex flex-1 flex-col justify-center gap-6"
            >
              {column.map((task) => (
                <TaskNode
                  key={task.readableId}
                  assignRef={(el) => {
                    if (el) {
                      nodeRefs.current.set(task.readableId, el);
                    } else {
                      nodeRefs.current.delete(task.readableId);
                    }
                  }}
                  task={task}
                  dense={dense}
                  selected={task.readableId === selected?.readableId}
                  onClick={() => setSelectedId(task.readableId)}
                />
              ))}
            </div>
          ))}
        </div>
      </div>

      {selected && <TaskDetailBar task={selected} />}
    </Section>
  );
}

function TaskNode({
  assignRef,
  task,
  dense,
  selected,
  onClick,
}: {
  assignRef: (el: HTMLDivElement | null) => void;
  task: WorkflowVersionTask;
  dense: boolean;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <div
      ref={assignRef}
      onClick={onClick}
      className={cn(
        'cursor-pointer rounded-lg border px-3 py-2.5 transition-colors',
        selected
          ? 'border-brand bg-brand/10'
          : 'border-border bg-white/[0.04] hover:border-muted-foreground/40',
      )}
    >
      <div className="truncate font-mono text-[13px] font-semibold text-foreground">
        {task.readableId}
      </div>
      <div className="mt-0.5 truncate font-mono text-[11px] text-muted-foreground">
        {task.action}
      </div>
      {dense && (
        <div className="mt-1.5 flex flex-wrap gap-1.5">
          <NodeChip>R {task.retries}</NodeChip>
          {task.timeout && <NodeChip>{task.timeout}</NodeChip>}
        </div>
      )}
    </div>
  );
}

function NodeChip({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex rounded bg-white/[0.06] px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
      {children}
    </span>
  );
}

function TaskDetailBar({ task }: { task: WorkflowVersionTask }) {
  return (
    <div className="mt-4 border-t border-border pt-3.5">
      <div className="mb-2.5 flex items-center gap-2.5">
        <span className="font-mono text-[13px] font-semibold text-foreground">
          {task.readableId}
        </span>
        <span className="font-mono text-xs text-muted-foreground">
          {task.action}
        </span>
      </div>
      <div className="flex flex-wrap gap-x-6 gap-y-3 text-xs">
        <DetailField label="Depends On">
          {task.parents.length > 0 ? task.parents.join(', ') : '—'}
        </DetailField>
        <DetailField label="Retries">{task.retries}</DetailField>
        <DetailField label="Timeout">{task.timeout || '—'}</DetailField>
        <DetailField label="Schedule Timeout">
          {task.scheduleTimeout || '—'}
        </DetailField>
        <DetailField label="Backoff">{formatBackoff(task)}</DetailField>
        <DetailField label="Rate Limits">{formatRateLimits(task)}</DetailField>
      </div>
    </div>
  );
}

function DetailField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <FieldLabel>{label}</FieldLabel>
      <div className="text-foreground">{children}</div>
    </div>
  );
}
