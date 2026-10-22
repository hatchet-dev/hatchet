import { KEYS_PAGE_SIZE, useConcurrencyKeys } from '../hooks/use-concurrency';
import { formatElapsed, neverQueues } from './concurrency-format';
import { PaginationFooter } from './pagination-footer';
import { Input } from '@/components/v1/ui/input';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { Tabs, TabsList, TabsTrigger } from '@/components/v1/ui/tabs';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { V1ConcurrencyKey, V1ConcurrencyStrategy } from '@/lib/api';
import { cn } from '@/lib/utils';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';
import { RiInformationLine, RiSearchLine } from 'react-icons/ri';
import { useDebounce } from 'use-debounce';

// Above this many slots per key the cells get too thin to read, so a bar is shown instead.
const MAX_SLOT_CELLS = 20;

// trailing 1fr spacer keeps the slot and count columns next to the key on wide screens
const grid =
  'grid grid-cols-[minmax(200px,420px)_170px_80px_100px_minmax(0,1fr)] items-center gap-x-3 px-4';

export type KeysPanelProps = {
  tenantId: string;
  strategy: V1ConcurrencyStrategy;
  search: string;
  onSearchChange: (search: string) => void;
  onlyQueued: boolean;
  onOnlyQueuedChange: (onlyQueued: boolean) => void;
  page: number;
  onPageChange: (page: number) => void;
  now: number;
};

export function KeysPanel({
  tenantId,
  strategy,
  search,
  onSearchChange,
  onlyQueued,
  onOnlyQueuedChange,
  page,
  onPageChange,
  now,
}: KeysPanelProps) {
  const [debouncedSearch] = useDebounce(search, 300);

  const { data, isLoading, error } = useConcurrencyKeys({
    tenantId,
    strategy,
    search: debouncedSearch,
    onlyQueued,
    page,
  });

  const keys = data?.rows ?? [];
  const numPages = data?.pagination?.num_pages ?? 0;
  const filtered = !!debouncedSearch || onlyQueued;

  let emptyText = 'No runs are running or queued on this strategy right now.';

  if (onlyQueued && neverQueues(strategy.limitStrategy)) {
    emptyText =
      'Nothing is ever queued on this strategy: its limit strategy cancels runs instead of queueing them.';
  } else if (onlyQueued) {
    emptyText = 'No runs are queued on this strategy right now.';
  } else if (debouncedSearch) {
    emptyText = 'No keys match.';
  }

  return (
    <section
      aria-label="Keys for selected strategy"
      className="flex min-w-0 flex-col overflow-hidden rounded-lg ring-1 ring-border/50"
    >
      <div className="flex flex-wrap items-start gap-3 border-b border-border/50 px-4 py-3">
        <div className="flex min-w-0 flex-1 basis-[420px] flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="font-semibold">Keys</span>
            <span className="text-muted-foreground">in</span>
            <span className="font-medium">{strategy.name}</span>
            <span className="font-mono text-xs text-indigo-900 dark:text-indigo-300">
              {strategy.expression}
            </span>
          </div>
          {strategy.usedBy && strategy.usedBy.length > 0 && (
            <div className="flex flex-wrap items-center gap-1 text-xs text-muted-foreground">
              Shared by
              {strategy.usedBy.map((u) => (
                <Link
                  key={`${u.workflowId}:${u.taskName}`}
                  to={appRoutes.tenantWorkflowRoute.to}
                  params={{ tenant: tenantId, workflow: u.workflowId }}
                  className="rounded border border-border/60 bg-muted px-1.5 font-mono text-[11px] text-foreground/80 hover:underline"
                  title={`${u.workflowName} › ${u.taskName}`}
                >
                  {u.taskName}
                </Link>
              ))}
            </div>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative">
            <RiSearchLine className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              aria-label="Search keys"
              placeholder="Find a key"
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
              className="h-8 w-48 pl-7 font-mono text-xs"
            />
          </div>
          <Tabs
            value={onlyQueued ? 'queued' : 'all'}
            onValueChange={(v) => onOnlyQueuedChange(v === 'queued')}
          >
            <TabsList aria-label="Key filter" className="h-8">
              <TabsTrigger value="all" className="h-6 px-2.5 text-xs">
                All keys
              </TabsTrigger>
              <TabsTrigger value="queued" className="h-6 px-2.5 text-xs">
                Has queued
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </div>

      <div className="overflow-x-auto">
        <div className="min-w-[560px]">
          <div
            className={cn(
              grid,
              'h-8 border-b border-border/50 bg-muted/30 text-xs font-medium text-muted-foreground',
            )}
          >
            <span>Key</span>
            <span>Slots in use</span>
            <span className="text-right">Queued</span>
            <span className="text-right">Oldest queued</span>
          </div>

          {isLoading &&
            [0, 1, 2, 3].map((i) => (
              <div
                key={i}
                className={cn(grid, 'h-9 border-b border-border/30')}
              >
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-24" />
              </div>
            ))}

          {error && (
            <div className="px-4 py-8 text-center text-sm text-destructive">
              Could not load keys for this strategy.
            </div>
          )}

          {!isLoading && !error && keys.length === 0 && (
            <div className="px-4 py-8 text-center text-sm text-muted-foreground">
              {emptyText}
            </div>
          )}

          {keys.map((k) => (
            <KeyRow key={k.key} k={k} now={now} />
          ))}
        </div>
      </div>

      <PaginationFooter
        summary={
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  aria-label="About this list"
                  className="flex items-center hover:text-foreground"
                >
                  <RiInformationLine className="size-4" />
                </button>
              </TooltipTrigger>
              <TooltipContent className="font-mono text-xs">
                {!filtered
                  ? `showing ${page * KEYS_PAGE_SIZE + keys.length} of ${data?.total ?? 0} keys`
                  : numPages === 0
                    ? 'no matching keys'
                    : `page ${page + 1} of ${numPages}`}
                {' · sorted by queued runs, then oldest queued'}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        }
        page={page}
        numPages={numPages}
        onPageChange={onPageChange}
      />
    </section>
  );
}

type KeyRowProps = { k: V1ConcurrencyKey; now: number };

function KeyRow({ k, now }: KeyRowProps) {
  return (
    <div
      className={cn(
        grid,
        'min-h-9 border-b border-border/30 hover:bg-muted/40',
      )}
    >
      <span className="truncate font-mono text-xs font-medium" title={k.key}>
        {k.key}
      </span>
      <SlotUsage used={k.runningCount} max={k.maxRuns} />
      <span
        className={cn(
          'text-right font-mono text-xs tabular-nums',
          k.queuedCount > 0
            ? 'text-amber-700 dark:text-amber-400'
            : 'text-muted-foreground/60',
          k.queuedCount > 5 && 'font-semibold',
        )}
      >
        {k.queuedCount}
      </span>
      <span className="text-right font-mono text-xs tabular-nums">
        {formatElapsed(k.oldestQueuedAt, now)}
      </span>
    </div>
  );
}

type SlotUsageProps = { used: number; max: number };

function SlotUsage({ used, max }: SlotUsageProps) {
  const filled = Math.min(used, max);

  return (
    <span className="flex items-center gap-2">
      {max <= MAX_SLOT_CELLS ? (
        <span
          className="grid w-20 gap-0.5"
          style={{ gridTemplateColumns: `repeat(${max}, minmax(0, 1fr))` }}
          aria-hidden
        >
          {Array.from({ length: max }, (_, i) => (
            <span
              key={i}
              className={cn(
                'h-2.5 rounded-sm',
                i < filled ? 'bg-blue-600 dark:bg-blue-500' : 'bg-muted',
              )}
            />
          ))}
        </span>
      ) : (
        <span
          className="h-2.5 w-20 overflow-hidden rounded-sm bg-muted"
          aria-hidden
        >
          <span
            className="block h-full bg-blue-600 dark:bg-blue-500"
            style={{ width: `${(filled / max) * 100}%` }}
          />
        </span>
      )}
      <span className="font-mono text-xs text-foreground/80">
        {used}/{max}
      </span>
    </span>
  );
}
