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
    'border-red-200 bg-red-50 text-red-800 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelNewest]:
    'border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelQueuedExceptNewest]:
    'border-violet-200 bg-violet-50 text-violet-800 dark:border-violet-500/30 dark:bg-violet-500/10 dark:text-violet-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelQueuedExceptOldest]:
    'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyNone]:
    'border-border bg-muted text-muted-foreground',
};

const limitStrategyExplanation: Record<V1ConcurrencyLimitStrategy, string> = {
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyGroupRoundRobin]:
    'Up to {max} runs per key at once. Extra runs are queued for their key, and when a slot frees up the next queued run is picked round-robin across keys, so no single key can monopolize capacity.',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelInProgress]:
    'Up to {max} runs per key. When the key is full, a running run for the same key is cancelled to free a slot, so the most recent run always takes priority. Nothing is queued.',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelNewest]:
    'Up to {max} runs per key. When the key is full, the new run is cancelled immediately and running runs continue uninterrupted. Nothing is queued.',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelQueuedExceptNewest]:
    'Up to {max} runs per key. When the key is full, the most recent {max} queued runs are kept and older queued runs are cancelled, which guarantees a trailing run.',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelQueuedExceptOldest]:
    'Up to {max} runs per key. When the key is full, the oldest {max} queued runs are kept and newer runs are cancelled.',
  [V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyNone]:
    'Up to {max} runs per key.',
};

export function explainStrategy(s: V1ConcurrencyStrategy): string {
  const max = s.maxRunsExpression ? 'N' : String(s.maxRuns);
  let text = limitStrategyExplanation[s.limitStrategy]
    .split('{max}')
    .join(max)
    .replace(/\b1 (queued )?runs\b/g, '1 $1run');

  if (s.maxRunsExpression) {
    text += ` N is computed per key by the max runs expression ${s.maxRunsExpression}.`;
  }

  return text;
}

export function neverQueues(limitStrategy: V1ConcurrencyLimitStrategy) {
  return (
    limitStrategy ===
      V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelInProgress ||
    limitStrategy ===
      V1ConcurrencyLimitStrategy.V1ConcurrencyLimitStrategyCancelNewest
  );
}

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
