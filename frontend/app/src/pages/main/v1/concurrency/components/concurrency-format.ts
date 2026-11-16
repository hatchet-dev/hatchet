import {
  V1ConcurrencyLimitStrategy,
  V1ConcurrencyScope,
  V1ConcurrencyStrategy,
} from '@/lib/api';

export function strategyRef(s: Pick<V1ConcurrencyStrategy, 'scope' | 'id'>) {
  return `${s.scope}:${s.id}`;
}

export const scopeLabel: Record<V1ConcurrencyScope, string> = {
  [V1ConcurrencyScope.V1ConcurrencyScopeShared]: 'Shared',
  [V1ConcurrencyScope.V1ConcurrencyScopeWorkflow]: 'Workflow',
  [V1ConcurrencyScope.V1ConcurrencyScopeTask]: 'Task',
};

export const scopeClassName: Record<V1ConcurrencyScope, string> = {
  [V1ConcurrencyScope.V1ConcurrencyScopeShared]:
    'border-transparent bg-foreground text-background',
  [V1ConcurrencyScope.V1ConcurrencyScopeWorkflow]:
    'border-border bg-muted text-foreground/80',
  [V1ConcurrencyScope.V1ConcurrencyScopeTask]:
    'border-border bg-transparent text-foreground/80',
};

export const limitStrategyClassName: Record<
  V1ConcurrencyLimitStrategy,
  string
> = {
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyGroupRoundRobin]:
    'border-indigo-200 bg-indigo-50 text-indigo-900 dark:border-indigo-500/30 dark:bg-indigo-500/10 dark:text-indigo-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelInProgress]:
    'border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelNewest]:
    'border-fuchsia-200 bg-fuchsia-50 text-fuchsia-800 dark:border-fuchsia-500/30 dark:bg-fuchsia-500/10 dark:text-fuchsia-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelQueuedExceptNewest]:
    'border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelQueuedExceptOldest]:
    'border-cyan-200 bg-cyan-50 text-cyan-800 dark:border-cyan-500/30 dark:bg-cyan-500/10 dark:text-cyan-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyNone]:
    'border-border bg-muted text-muted-foreground',
};

export function strategyLocation(s: V1ConcurrencyStrategy): string {
  switch (s.scope) {
    case V1ConcurrencyScope.V1ConcurrencyScopeShared: {
      const usedBy = s.usedBy ?? [];

      const workflows = new Set(usedBy.map((u) => u.workflowId)).size;

      return `used by ${plural(usedBy.length, 'task')} in ${plural(workflows, 'workflow')}`;
    }
    case V1ConcurrencyScope.V1ConcurrencyScopeWorkflow:
      return `workflow ${s.workflowName ?? ''}${versionSuffix(s)}`;
    case V1ConcurrencyScope.V1ConcurrencyScopeTask:
      return s.taskName === s.workflowName
        ? `standalone task${versionSuffix(s)}`
        : `${s.workflowName ?? ''} › ${s.taskName ?? ''}${versionSuffix(s)}`;
  }
}

function versionSuffix(s: V1ConcurrencyStrategy) {
  return s.workflowVersion ? ` · ${s.workflowVersion}` : '';
}

function plural(n: number, word: string) {
  return `${n} ${word}${n === 1 ? '' : 's'}`;
}

/** Compact elapsed time: 41s, 12m 40s, 2h 14m, 7d 2h. */
export function formatElapsed(since: string | undefined, now: number) {
  if (!since) {
    return '—';
  }

  const s = Math.max(0, Math.floor((now - new Date(since).getTime()) / 1000));

  if (s < 60) {
    return `${s}s`;
  }

  const m = Math.floor(s / 60);

  if (m < 60) {
    return `${m}m ${String(s % 60).padStart(2, '0')}s`;
  }

  const h = Math.floor(m / 60);

  if (h < 24) {
    return `${h}h ${String(m % 60).padStart(2, '0')}m`;
  }

  return `${Math.floor(h / 24)}d ${h % 24}h`;
}
