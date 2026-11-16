import { formatElapsed } from './concurrency-format';
import { StatCell } from '@/components/v1/molecules/stat-cell';
import { V1ConcurrencyScope, V1ConcurrencyStrategy } from '@/lib/api';

export type TotalsStripProps = {
  strategies: V1ConcurrencyStrategy[];
  loading: boolean;
  now: number;
  onOpenKey: (strategy: V1ConcurrencyStrategy, key: string) => void;
};

export function TotalsStrip({
  strategies,
  loading,
  now,
  onOpenKey,
}: TotalsStripProps) {
  const countScope = (scope: V1ConcurrencyScope) =>
    strategies.filter((s) => s.scope === scope).length;

  const keys = sum(strategies, (s) => s.keyCount);
  const running = sum(strategies, (s) => s.runningCount);
  const queued = sum(strategies, (s) => s.queuedCount);
  const queuedKeys = sum(strategies, (s) => s.queuedKeyCount);

  const longest = strategies
    .filter((s) => s.oldestQueuedAt)
    .sort(
      (a, b) =>
        new Date(a.oldestQueuedAt!).getTime() -
        new Date(b.oldestQueuedAt!).getTime(),
    )[0];

  return (
    <section
      aria-label="Totals"
      className="grid grid-cols-2 divide-x divide-y divide-border/50 rounded-lg ring-1 ring-border/50 md:grid-cols-3 lg:grid-cols-5 lg:divide-y-0"
    >
      <StatCell
        label="Strategies"
        loading={loading}
        hint={`${countScope(V1ConcurrencyScope.V1ConcurrencyScopeShared)} shared · ${countScope(V1ConcurrencyScope.V1ConcurrencyScopeWorkflow)} workflow · ${countScope(V1ConcurrencyScope.V1ConcurrencyScopeTask)} task`}
      >
        {strategies.length}
      </StatCell>
      <StatCell
        label="Keys in use"
        loading={loading}
        hint="with running or queued runs"
      >
        {keys}
      </StatCell>
      <StatCell
        label="Runs holding a slot"
        loading={loading}
        hint="across all keys"
      >
        {running}
      </StatCell>
      <StatCell
        label="Queued runs"
        loading={loading}
        valueClassName={
          queued > 0 ? 'text-amber-700 dark:text-amber-400' : undefined
        }
        hint={`across ${queuedKeys} ${queuedKeys === 1 ? 'key' : 'keys'}`}
      >
        {queued}
      </StatCell>
      <StatCell
        label="Longest queued"
        loading={loading}
        hint={
          longest?.oldestQueuedKey ? (
            <button
              type="button"
              className="max-w-full truncate font-mono text-primary hover:underline"
              onClick={() => onOpenKey(longest, longest.oldestQueuedKey!)}
            >
              {longest.oldestQueuedKey}
            </button>
          ) : (
            'nothing is queued'
          )
        }
      >
        {formatElapsed(longest?.oldestQueuedAt, now)}
      </StatCell>
    </section>
  );
}

function sum<T>(rows: T[], f: (row: T) => number) {
  return rows.reduce((acc, row) => acc + f(row), 0);
}
