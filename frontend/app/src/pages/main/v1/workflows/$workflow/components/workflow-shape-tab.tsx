import { EmptyState, FieldLabel } from './settings-primitives';
import WorkflowTasksDag from './workflow-tasks-dag';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/v1/ui/tabs';
import {
  WorkflowVersion,
  WorkflowVersionTask,
  WorkflowVersionTaskDesiredWorkerLabel,
  WorkflowVersionTaskRateLimit,
} from '@/lib/api';
import { useMemo, useState } from 'react';

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

export default function WorkflowShapeTab({
  workflow,
}: {
  workflow: WorkflowVersion;
}) {
  const tasks = useMemo(
    () => topologicalSort(workflow.tasks ?? []),
    [workflow.tasks],
  );
  const [activeSubTab, setActiveSubTab] = useState(tasks[0]?.readableId ?? '');

  if (tasks.length === 0) {
    return (
      <div className="px-1">
        <EmptyState message="This workflow has no tasks." />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 px-1">
      <div className="relative flex h-[300px] w-full shrink-0 overflow-auto rounded-md border border-border bg-slate-100 dark:bg-slate-900">
        <WorkflowTasksDag
          tasks={tasks}
          selectedTaskId={activeSubTab}
          onSelectTask={setActiveSubTab}
        />
      </div>

      <Tabs value={activeSubTab} onValueChange={setActiveSubTab}>
        <TabsList layout="underlined">
          {tasks.map((task) => (
            <TabsTrigger
              key={task.readableId}
              variant="underlined"
              value={task.readableId}
              className="font-mono"
            >
              {task.readableId}
            </TabsTrigger>
          ))}
        </TabsList>

        {tasks.map((task) => (
          <TabsContent key={task.readableId} value={task.readableId}>
            <TaskDetail task={task} />
          </TabsContent>
        ))}
      </Tabs>
    </div>
  );
}

function TaskDetail({ task }: { task: WorkflowVersionTask }) {
  const rateLimits = (task.rateLimits ?? [])
    .map(formatRateLimit)
    .filter(Boolean);
  const workerLabels = (task.desiredWorkerLabels ?? []).map(formatWorkerLabel);

  return (
    <div className="pt-3.5">
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
        <DetailField label="Rate Limits">
          <ChipList items={rateLimits} />
        </DetailField>
        <DetailField label="Worker Labels">
          <ChipList items={workerLabels} />
        </DetailField>
      </div>
    </div>
  );
}

function ChipList({ items }: { items: string[] }) {
  if (items.length === 0) {
    return <div className="text-foreground">—</div>;
  }

  return (
    <div className="flex max-w-[240px] flex-wrap gap-1">
      {items.map((item, i) => (
        <span
          key={i}
          className="rounded bg-white/[0.06] px-1.5 py-0.5 font-mono text-[11px] text-foreground"
        >
          {item}
        </span>
      ))}
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
