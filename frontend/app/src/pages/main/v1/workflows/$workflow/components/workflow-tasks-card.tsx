import {
  EmptyState,
  FieldLabel,
  formatLimitStrategy,
} from './settings-primitives';
import WorkflowTasksDag from './workflow-tasks-dag';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/v1/ui/accordion';
import { Card, CardContent } from '@/components/v1/ui/card';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import {
  ConcurrencyScope,
  ConcurrencySetting,
  WorkflowVersion,
  WorkflowVersionTask,
  WorkflowVersionTaskDesiredWorkerLabel,
  WorkflowVersionTaskRateLimit,
} from '@/lib/api';
import { useEffect, useMemo, useState } from 'react';

function topologicalSort(tasks: WorkflowVersionTask[]): WorkflowVersionTask[] {
  const byId = new Map(tasks.map((t) => [t.readableId, t]));
  const sorted: WorkflowVersionTask[] = [];
  const visited = new Set<string>();

  function visit(task: WorkflowVersionTask) {
    if (visited.has(task.readableId)) {
      return;
    }
    visited.add(task.readableId);

    for (const parentId of task.parents) {
      const parent = byId.get(parentId);
      if (parent) {
        visit(parent);
      }
    }

    sorted.push(task);
  }

  tasks.forEach(visit);

  return sorted;
}

function formatBackoff(task: WorkflowVersionTask): string {
  if (!task.retryBackoffFactor) {
    return '—';
  }

  return `x${task.retryBackoffFactor}${
    task.retryBackoffMaxSeconds ? ` up to ${task.retryBackoffMaxSeconds}s` : ''
  }`;
}

function formatRateLimit(rl: WorkflowVersionTaskRateLimit): string {
  const key = rl.key ?? rl.keyExpression ?? '';
  const units =
    rl.unitsExpression ?? (rl.units != null ? String(rl.units) : '');
  const base = [key, units].filter(Boolean).join(': ');
  return rl.duration ? `${base} / ${rl.duration}` : base;
}

function formatWorkerLabel(
  label: WorkflowVersionTaskDesiredWorkerLabel,
): string {
  const value =
    label.strValue != null
      ? `'${label.strValue}'`
      : label.intValue != null
        ? String(label.intValue)
        : '';
  const base = `${label.key}${value ? ` = ${value}` : ''}`;
  return label.required ? `${base} (required)` : base;
}

function formatConcurrencyRule(c: ConcurrencySetting): string {
  return `Max ${c.maxRuns} · ${formatLimitStrategy(c.limitStrategy)} · ${c.expression}`;
}

export default function WorkflowTasksCard({
  workflow,
}: {
  workflow: WorkflowVersion;
}) {
  const tasks = useMemo(
    () => topologicalSort(workflow.tasks ?? []),
    [workflow.tasks],
  );
  const [selectedTaskId, setSelectedTaskId] = useState(
    tasks[0]?.readableId ?? '',
  );

  useEffect(() => {
    if (!selectedTaskId && tasks.length > 0) {
      setSelectedTaskId(tasks[0].readableId);
    }
  }, [tasks, selectedTaskId]);

  const selectedTask = tasks.find((t) => t.readableId === selectedTaskId);
  const selectedTaskConcurrency = (workflow.v1Concurrency ?? []).filter(
    (c) =>
      c.scope === ConcurrencyScope.TASK && c.stepReadableId === selectedTaskId,
  );

  return (
    <Card className="bg-muted/30">
      <CardContent className="p-3">
        {tasks.length === 0 ? (
          <EmptyState message="This workflow has no tasks." />
        ) : (
          <div className="flex flex-col gap-3">
            <div className="text-[13px] font-semibold text-foreground">
              Workflow
            </div>
            <div className="relative flex h-[300px] w-full shrink-0 overflow-hidden rounded-md border border-border bg-slate-100 dark:bg-slate-900">
              <WorkflowTasksDag
                tasks={tasks}
                selectedTaskId={selectedTaskId}
                onSelectTask={setSelectedTaskId}
              />
            </div>

            {selectedTask && (
              <Accordion
                type="single"
                collapsible
                className="border-t border-border"
              >
                <AccordionItem
                  value={selectedTask.readableId}
                  className="border-b-0"
                >
                  <AccordionTrigger className="py-3 hover:no-underline">
                    <span className="flex items-center gap-2.5">
                      <span className="font-mono text-[13px] font-semibold text-foreground">
                        {selectedTask.readableId}
                      </span>
                      <span className="font-mono text-xs text-muted-foreground">
                        {selectedTask.action}
                      </span>
                    </span>
                  </AccordionTrigger>
                  <AccordionContent>
                    <TaskDetail
                      task={selectedTask}
                      concurrency={selectedTaskConcurrency}
                    />
                  </AccordionContent>
                </AccordionItem>
              </Accordion>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function TaskDetail({
  task,
  concurrency,
}: {
  task: WorkflowVersionTask;
  concurrency: ConcurrencySetting[];
}) {
  const rateLimits = (task.rateLimits ?? [])
    .map(formatRateLimit)
    .filter(Boolean);
  const workerLabels = (task.desiredWorkerLabels ?? []).map(formatWorkerLabel);
  const concurrencyRules = concurrency.map(formatConcurrencyRule);

  return (
    <div>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
        {task.parents.length > 0 && (
          <DetailTile label="Depends On">{task.parents.join(', ')}</DetailTile>
        )}
        <DetailTile label="Retries">{task.retries}</DetailTile>
        {task.timeout && (
          <DetailTile label="Timeout">{task.timeout}</DetailTile>
        )}
        {task.scheduleTimeout && (
          <DetailTile label="Schedule Timeout">
            {task.scheduleTimeout}
          </DetailTile>
        )}
        {!!task.retryBackoffFactor && (
          <DetailTile label="Backoff">{formatBackoff(task)}</DetailTile>
        )}
        {concurrencyRules.length > 0 && (
          <DetailTile label="Concurrency">
            <ChipList items={concurrencyRules} />
          </DetailTile>
        )}
        {rateLimits.length > 0 && (
          <DetailTile label="Rate Limits">
            <ChipList items={rateLimits} />
          </DetailTile>
        )}
        {workerLabels.length > 0 && (
          <DetailTile label="Worker Labels">
            <ChipList items={workerLabels} />
          </DetailTile>
        )}
      </div>
    </div>
  );
}

function ChipList({ items }: { items: string[] }) {
  if (items.length === 0) {
    return <div className="text-foreground">—</div>;
  }

  return (
    <TooltipProvider delayDuration={200}>
      <div className="flex max-w-full flex-col gap-1">
        {items.map((item, i) => (
          <Tooltip key={i}>
            <TooltipTrigger asChild>
              <span className="block max-w-full truncate rounded bg-white/[0.06] px-1.5 py-0.5 font-mono text-[11px] text-foreground">
                {item}
              </span>
            </TooltipTrigger>
            <TooltipContent
              className="max-w-xs break-words font-mono text-[11px]"
              side="top"
            >
              {item}
            </TooltipContent>
          </Tooltip>
        ))}
      </div>
    </TooltipProvider>
  );
}

function DetailTile({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="min-w-0 rounded-md border border-border px-2.5 py-2">
      <FieldLabel>{label}</FieldLabel>
      <div className="min-w-0 max-h-48 overflow-y-auto text-xs text-foreground">
        {children}
      </div>
    </div>
  );
}
