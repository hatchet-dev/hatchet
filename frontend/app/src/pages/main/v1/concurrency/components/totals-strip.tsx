import { formatElapsed } from './concurrency-format';
import { StatCell } from '@/components/v1/molecules/stat-cell';
import {
  V1ConcurrencyScope,
  V1ConcurrencyStrategy,
  V1ConcurrencyTotals,
} from '@/lib/api';

export type TotalsStripProps = {
  strategies: V1ConcurrencyStrategy[];
  // fetched separately: a run in a chain of strategies holds a slot in each of them, so summing
  // the rows would count it once per strategy
  totals: V1ConcurrencyTotals | undefined;
  totalsLoading: boolean;
  totalsError: boolean;
  loading: boolean;
  now: number;
  onOpenKey: (strategy: V1ConcurrencyStrategy, key: string) => void;
};

export function TotalsStrip({
  strategies,
  totals,
  totalsLoading,
  totalsError,
  loading,
  now,
  onOpenKey,
}: TotalsStripProps) {
  const countScope = (scope: V1ConcurrencyScope) =>
    strategies.filter((s) => s.scope === scope).length;

  const keys = totals?.keyCount ?? 0;
  const running = totals?.runningCount ?? 0;
  const queued = totals?.queuedCount ?? 0;
  const queuedKeys = totals?.queuedKeyCount ?? 0;

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
        loading={totalsLoading}
        error={totalsError}
        hint="with a slot in use or a queued run"
      >
        {keys}
      </StatCell>
      <StatCell
        label="Running runs"
        loading={totalsLoading}
        error={totalsError}
        hint="past every strategy of their chain"
      >
        {running}
      </StatCell>
      <StatCell
        label="Queued runs"
        loading={totalsLoading}
        error={totalsError}
        valueClassName={
          queued > 0 ? 'text-amber-700 dark:text-amber-400' : undefined
        }
        hint={
          totalsError
            ? undefined
            : `across ${queuedKeys} ${queuedKeys === 1 ? 'key' : 'keys'}`
        }
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
