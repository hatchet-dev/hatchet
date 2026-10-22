import { STRATEGIES_PAGE_SIZE } from '../hooks/use-concurrency';
import {
  explainStrategy,
  formatElapsed,
  limitStrategyClassName,
  scopeClassName,
  scopeLabel,
  strategyLocation,
  strategyRef,
} from './concurrency-format';
import { PaginationFooter } from './pagination-footer';
import { RefetchIntervalDropdown } from '@/components/refetch-interval-dropdown';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { Tabs, TabsList, TabsTrigger } from '@/components/v1/ui/tabs';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { V1ConcurrencyScope, V1ConcurrencyStrategy } from '@/lib/api';
import { cn } from '@/lib/utils';

export type ScopeFilter = 'ALL' | V1ConcurrencyScope;

const scopeFilters: { value: ScopeFilter; label: string }[] = [
  { value: 'ALL', label: 'All' },
  { value: V1ConcurrencyScope.V1ConcurrencyScopeShared, label: 'Shared' },
  { value: V1ConcurrencyScope.V1ConcurrencyScopeWorkflow, label: 'Workflow' },
  { value: V1ConcurrencyScope.V1ConcurrencyScopeTask, label: 'Task' },
];

const grid =
  'grid grid-cols-[minmax(200px,1.3fr)_minmax(150px,1fr)_190px_64px_64px_80px_80px_92px] items-center gap-x-3 px-4';

export type StrategiesPanelProps = {
  // the current page only
  strategies: V1ConcurrencyStrategy[];
  loading: boolean;
  selectedRef: string | undefined;
  onSelect: (s: V1ConcurrencyStrategy) => void;
  scope: ScopeFilter;
  onScopeChange: (scope: ScopeFilter) => void;
  // zero-based
  page: number;
  numPages: number;
  // strategies matching the scope filter, across all pages
  total: number;
  onPageChange: (page: number) => void;
  isRefetching: boolean;
  onRefetch: () => void;
  now: number;
};

export function StrategiesPanel({
  strategies,
  loading,
  selectedRef,
  onSelect,
  scope,
  onScopeChange,
  page,
  numPages,
  total,
  onPageChange,
  isRefetching,
  onRefetch,
  now,
}: StrategiesPanelProps) {
  return (
    <section
      aria-label="Strategies"
      className="flex min-w-0 flex-col overflow-hidden rounded-lg ring-1 ring-border/50"
    >
      <div className="flex flex-wrap items-center gap-3 border-b border-border/50 px-4 py-2.5">
        <span className="text-sm font-semibold">Strategies</span>
        <Tabs
          value={scope}
          onValueChange={(v) => onScopeChange(v as ScopeFilter)}
        >
          <TabsList aria-label="Scope" className="h-8">
            {scopeFilters.map((f) => (
              <TabsTrigger
                key={f.value}
                value={f.value}
                className="h-6 px-2.5 text-xs"
              >
                {f.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <div className="ml-auto">
          <RefetchIntervalDropdown
            isRefetching={isRefetching}
            onRefetch={onRefetch}
          />
        </div>
      </div>

      <div className="overflow-x-auto">
        <div className="min-w-[940px]">
          <div
            className={cn(
              grid,
              'h-8 border-b border-border/50 bg-muted/30 text-xs font-medium text-muted-foreground',
            )}
          >
            <span>Strategy</span>
            <span>Key expression</span>
            <span>Limit strategy</span>
            <span className="text-right">Max runs</span>
            <span className="text-right">Keys</span>
            <span className="text-right">Running</span>
            <span className="text-right">Queued ↓</span>
            <span className="text-right">Oldest queued</span>
          </div>

          {loading &&
            [0, 1, 2].map((i) => (
              <div
                key={i}
                className={cn(grid, 'h-12 border-b border-border/30')}
              >
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-4 w-32" />
              </div>
            ))}

          {!loading && strategies.length === 0 && (
            <div className="px-4 py-8 text-center text-sm text-muted-foreground">
              No strategies match these filters.
            </div>
          )}

          <TooltipProvider>
            {strategies.map((s) => (
              <StrategyRow
                key={strategyRef(s)}
                strategy={s}
                selected={strategyRef(s) === selectedRef}
                onSelect={() => onSelect(s)}
                now={now}
              />
            ))}
          </TooltipProvider>
        </div>
      </div>

      {numPages > 1 && (
        <PaginationFooter
          summary={`showing ${page * STRATEGIES_PAGE_SIZE + 1}–${page * STRATEGIES_PAGE_SIZE + strategies.length} of ${total} strategies · sorted by queued runs, then oldest queued`}
          page={page}
          numPages={numPages}
          onPageChange={onPageChange}
        />
      )}
    </section>
  );
}

type StrategyRowProps = {
  strategy: V1ConcurrencyStrategy;
  selected: boolean;
  onSelect: () => void;
  now: number;
};

function StrategyRow({
  strategy: s,
  selected,
  onSelect,
  now,
}: StrategyRowProps) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onSelect}
      className={cn(
        grid,
        'min-h-12 w-full border-b border-l-[3px] border-b-border/30 text-left text-sm hover:bg-muted/40',
        selected ? 'border-l-foreground bg-muted/50' : 'border-l-transparent',
      )}
    >
      <span className="flex min-w-0 flex-col gap-0.5 py-1.5">
        <span className="flex min-w-0 items-center gap-1.5">
          <span className="truncate font-medium">{s.name}</span>
          <span
            className={cn(
              'shrink-0 rounded-full border px-1.5 text-[10.5px] font-medium leading-4',
              scopeClassName[s.scope],
            )}
          >
            {scopeLabel[s.scope]}
          </span>
        </span>
        <span className="truncate text-xs text-muted-foreground">
          {strategyLocation(s)}
        </span>
      </span>
      <span className="truncate font-mono text-xs text-indigo-900 dark:text-indigo-300">
        {s.expression}
      </span>
      <span className="min-w-0">
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              className={cn(
                'inline-block max-w-full cursor-help truncate rounded border px-1.5 font-mono text-[10.5px]',
                limitStrategyClassName[s.limitStrategy],
              )}
            >
              {s.limitStrategy}
            </span>
          </TooltipTrigger>
          <TooltipContent className="max-w-xs text-xs leading-relaxed">
            {explainStrategy(s)}
          </TooltipContent>
        </Tooltip>
      </span>
      <span className="text-right font-mono text-xs tabular-nums">
        {s.maxRunsExpression ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="cursor-help underline decoration-dotted underline-offset-2">
                f(input)
              </span>
            </TooltipTrigger>
            <TooltipContent className="font-mono text-xs">
              {s.maxRunsExpression}
            </TooltipContent>
          </Tooltip>
        ) : (
          s.maxRuns
        )}
      </span>
      <span className="text-right font-mono text-xs tabular-nums">
        {s.keyCount}
      </span>
      <span className="text-right font-mono text-xs tabular-nums">
        {s.runningCount}
      </span>
      <span
        className={cn(
          'text-right font-mono text-xs tabular-nums',
          s.queuedCount > 0
            ? 'text-amber-700 dark:text-amber-400'
            : 'text-muted-foreground/60',
          s.queuedCount > 20 && 'font-semibold',
        )}
      >
        {s.queuedCount}
      </span>
      <span className="text-right font-mono text-xs tabular-nums">
        {formatElapsed(s.oldestQueuedAt, now)}
      </span>
    </button>
  );
}
