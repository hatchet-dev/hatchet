import { useRunsContext } from '../hooks/runs-provider';
import { useRunFilterActions } from '../hooks/use-run-filter-actions';
import {
  additionalMetadataKey,
  createdAtKey,
  flattenDAGsKey,
  idempotencyKeyKey,
  statusKey,
  TaskRunColumn,
  workflowKey,
} from './v1/task-runs-columns';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  V1AdditionalMetadataOperator,
  V1RunningFilter,
  V1TaskStatus,
} from '@/lib/api';
import { cn } from '@/lib/utils';
import {
  isTimeWindowOutsideRetention,
  type RunsTimeWindowPreset,
} from '@/lib/utils/retention';
import { Cross2Icon } from '@radix-ui/react-icons';
import { Lock, Search } from 'lucide-react';
import { startTransition, useMemo, useState } from 'react';

const RUN_STATUS_META: Record<V1TaskStatus, { label: string; dot: string }> = {
  [V1TaskStatus.COMPLETED]: { label: 'Succeeded', dot: 'bg-green-500' },
  [V1TaskStatus.RUNNING]: { label: 'Running', dot: 'bg-yellow-500' },
  [V1TaskStatus.QUEUED]: { label: 'Queued', dot: 'bg-slate-500' },
  [V1TaskStatus.FAILED]: { label: 'Failed', dot: 'bg-red-500' },
  [V1TaskStatus.CANCELLED]: { label: 'Cancelled', dot: 'bg-orange-500' },
};

const STATUS_ORDER = [
  V1TaskStatus.COMPLETED,
  V1TaskStatus.RUNNING,
  V1TaskStatus.QUEUED,
  V1TaskStatus.FAILED,
  V1TaskStatus.CANCELLED,
];

const RUNNING_SUB_LABELS: Record<V1RunningFilter, string> = {
  [V1RunningFilter.ALL]: 'All',
  [V1RunningFilter.ON_WORKER]: 'On worker',
  [V1RunningFilter.EVICTED]: 'Evicted',
};

const TIME_PRESETS: RunsTimeWindowPreset[] = [
  '15m',
  '1h',
  '4h',
  '6h',
  '1d',
  '7d',
];

function formatRangePoint(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function FilterBlock({
  title,
  aside,
  children,
}: {
  title: string;
  aside?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="border-b px-3.5 py-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <h3 className="text-xs font-semibold">{title}</h3>
        {aside}
      </div>
      {children}
    </section>
  );
}

function Segmented<T extends string>({
  options,
  value,
  onChange,
  label,
  disabled,
}: {
  options: { value: T; label: string }[];
  value: T;
  onChange: (value: T) => void;
  label: string;
  disabled?: boolean;
}) {
  return (
    <div
      role="group"
      aria-label={label}
      className="flex w-full gap-0.5 rounded-md border bg-muted p-0.5"
    >
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          disabled={disabled}
          aria-pressed={value === option.value}
          onClick={() => onChange(option.value)}
          className={cn(
            'h-6 flex-1 rounded px-2 text-xs text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-60',
            value === option.value && 'bg-background text-foreground shadow-sm',
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

function RemovableRow({
  children,
  onRemove,
  removeLabel,
}: {
  children: React.ReactNode;
  onRemove: () => void;
  removeLabel: string;
}) {
  return (
    <div className="flex items-center justify-between gap-2 rounded-md bg-muted/50 px-2 py-1 font-mono text-xs">
      <div className="min-w-0 truncate">{children}</div>
      <Button
        variant="icon"
        size="xs"
        aria-label={removeLabel}
        onClick={onRemove}
      >
        <Cross2Icon className="size-3" />
      </Button>
    </div>
  );
}

function TimeRangeBlock() {
  const { filters } = useRunsContext();
  const {
    timeWindow,
    isCustomTimeRange,
    apiFilters,
    retentionPeriod,
    retentionGate,
  } = filters;
  const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;

  return (
    <FilterBlock
      title="Time range"
      aside={
        <span className="font-mono text-[11px] text-muted-foreground">
          {timeZone}
        </span>
      }
    >
      <div className="flex flex-wrap gap-1">
        {TIME_PRESETS.map((preset) => {
          const active = !isCustomTimeRange && timeWindow === preset;
          const locked =
            !!retentionPeriod &&
            isTimeWindowOutsideRetention(preset, retentionPeriod);

          return (
            <Button
              key={preset}
              size="sm"
              className="h-6 px-2 text-xs"
              variant={active ? 'default' : 'outline'}
              aria-pressed={active}
              onClick={() => filters.setTimeWindow(preset)}
            >
              {locked && <Lock className="size-3" />}
              {preset}
            </Button>
          );
        })}
        <Button
          size="sm"
          className="h-6 px-2 text-xs"
          variant={isCustomTimeRange ? 'default' : 'outline'}
          aria-pressed={isCustomTimeRange}
          onClick={() =>
            filters.setCustomTimeRange({
              start: apiFilters.since,
              end: apiFilters.until || new Date().toISOString(),
            })
          }
        >
          Custom
        </Button>
      </div>

      {isCustomTimeRange && (
        <div className="mt-2 space-y-2">
          <DateTimePicker
            label="After"
            date={apiFilters.since ? new Date(apiFilters.since) : undefined}
            setDate={(date) =>
              apiFilters.until &&
              filters.setCustomTimeRange({
                start:
                  date?.toISOString() ||
                  new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                end: apiFilters.until,
              })
            }
            triggerClassName="w-full min-w-0"
            retentionPeriod={retentionPeriod}
            onBlockedDate={retentionGate.blockSince}
          />
          <DateTimePicker
            label="Before"
            date={apiFilters.until ? new Date(apiFilters.until) : undefined}
            setDate={(date) =>
              filters.setCustomTimeRange({
                start: apiFilters.since,
                end: date?.toISOString() || new Date().toISOString(),
              })
            }
            triggerClassName="w-full min-w-0"
            retentionPeriod={retentionPeriod}
            onBlockedDate={retentionGate.blockSince}
          />
        </div>
      )}

      <div className="mt-2 font-mono text-[11px] text-muted-foreground">
        {formatRangePoint(apiFilters.since)} →{' '}
        {apiFilters.until ? formatRangePoint(apiFilters.until) : 'now'}
      </div>
    </FilterBlock>
  );
}

function StatusBlock() {
  const { runStatusCounts } = useRunsContext();
  const {
    selectedStatuses,
    toggleStatus,
    isRunningSubChecked,
    toggleRunningSub,
  } = useRunFilterActions();

  const countFor = (status: V1TaskStatus) =>
    runStatusCounts.find((m) => m.status === status)?.count;
  const maxCount = Math.max(1, ...runStatusCounts.map((m) => m.count));
  const runningDetail = runStatusCounts.find(
    (m) => m.status === V1TaskStatus.RUNNING,
  )?.runningDetailCount;

  const runningSubs = [
    {
      value: V1RunningFilter.ON_WORKER,
      other: V1RunningFilter.EVICTED,
      count: runningDetail?.onWorker,
    },
    {
      value: V1RunningFilter.EVICTED,
      other: V1RunningFilter.ON_WORKER,
      count: runningDetail?.evicted,
    },
  ];

  return (
    <FilterBlock title="Status">
      <div className="space-y-0.5">
        {STATUS_ORDER.map((status) => {
          const meta = RUN_STATUS_META[status];
          const count = countFor(status);
          const id = `runs-sidebar-status-${status}`;

          return (
            <div key={status}>
              <div className="flex h-7 items-center gap-2 rounded px-1 hover:bg-muted/50">
                <Checkbox
                  id={id}
                  checked={selectedStatuses.includes(status)}
                  onCheckedChange={() => toggleStatus(status)}
                />
                <span
                  className={cn('size-2 shrink-0 rounded-full', meta.dot)}
                  aria-hidden="true"
                />
                <Label
                  htmlFor={id}
                  className="flex-1 cursor-pointer text-[13px] font-normal"
                >
                  {meta.label}
                </Label>
                <span className="font-mono text-[11px] text-muted-foreground">
                  {count?.toLocaleString('en-US') ?? '–'}
                </span>
              </div>
              <div className="mb-1 ml-7 h-0.5 rounded-full bg-muted">
                <div
                  className={cn('h-0.5 rounded-full', meta.dot)}
                  style={{
                    width: `${Math.max(2, Math.round(((count ?? 0) / maxCount) * 100))}%`,
                  }}
                />
              </div>
              {status === V1TaskStatus.RUNNING &&
                runningSubs.map((sub) => {
                  const subId = `runs-sidebar-running-${sub.value}`;
                  return (
                    <div
                      key={sub.value}
                      className="flex h-6 items-center gap-2 rounded py-0 pl-7 pr-1 hover:bg-muted/50"
                    >
                      <Checkbox
                        id={subId}
                        checked={isRunningSubChecked(sub.value)}
                        onCheckedChange={() =>
                          toggleRunningSub(sub.value, sub.other)
                        }
                      />
                      <Label
                        htmlFor={subId}
                        className="flex-1 cursor-pointer text-xs font-normal text-muted-foreground"
                      >
                        {RUNNING_SUB_LABELS[sub.value]}
                      </Label>
                      <span className="font-mono text-[11px] text-muted-foreground">
                        {sub.count?.toLocaleString('en-US') ?? '–'}
                      </span>
                    </div>
                  );
                })}
            </div>
          );
        })}
      </div>
    </FilterBlock>
  );
}

function FlattenDAGsBlock() {
  const { filters } = useRunsContext();
  const { setFilter } = useRunFilterActions();
  const id = 'runs-sidebar-flatten-dags';

  return (
    <section className="border-b px-3.5 py-3">
      <div className="flex h-7 items-center gap-2 rounded px-1 hover:bg-muted/50">
        <Checkbox
          id={id}
          checked={!!filters.apiFilters.flattenDAGs}
          onCheckedChange={(checked) =>
            setFilter(flattenDAGsKey, checked === true ? true : undefined)
          }
        />
        <Label
          htmlFor={id}
          className="flex-1 cursor-pointer text-[13px] font-normal"
        >
          {TaskRunColumn.flattenDAGs}
        </Label>
      </div>
    </section>
  );
}

function WorkflowBlock() {
  const { filters, toolbarFilters } = useRunsContext();
  const { setFilter } = useRunFilterActions();
  const [search, setSearch] = useState('');
  const config = toolbarFilters.find((f) => f.columnId === workflowKey);
  const selected = filters.apiFilters.workflowIds ?? [];
  const options = config?.options ?? [];

  return (
    <FilterBlock
      title="Workflow"
      aside={
        selected.length > 0 ? (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={() => setFilter(workflowKey, undefined)}
          >
            Clear
          </Button>
        ) : undefined
      }
    >
      <div className="relative mb-1.5">
        <Search
          className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          aria-label="Search workflows"
          placeholder="Search workflows"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            startTransition(() =>
              config?.searchConfig?.onSearch(e.target.value),
            );
          }}
          className="h-7 pl-7 text-xs"
        />
      </div>
      <div className="max-h-56 space-y-0.5 overflow-y-auto">
        {options.length === 0 ? (
          <p className="py-2 text-center text-xs text-muted-foreground">
            No workflows found
          </p>
        ) : (
          options.map((option) => {
            const id = `runs-sidebar-workflow-${option.value}`;
            const checked = selected.includes(option.value);
            return (
              <div
                key={option.value}
                className="flex h-7 items-center gap-2 rounded px-1 hover:bg-muted/50"
              >
                <Checkbox
                  id={id}
                  checked={checked}
                  onCheckedChange={() =>
                    setFilter(
                      workflowKey,
                      checked
                        ? selected.filter((v) => v !== option.value)
                        : [...selected, option.value],
                    )
                  }
                />
                <Label
                  htmlFor={id}
                  className="flex-1 cursor-pointer truncate text-[13px] font-normal"
                >
                  {option.label}
                </Label>
              </div>
            );
          })
        )}
      </div>
    </FilterBlock>
  );
}

function MetadataBlock() {
  const { filters } = useRunsContext();
  const { setFilter } = useRunFilterActions();
  const [adding, setAdding] = useState(false);
  const [key, setKey] = useState('');
  const [value, setValue] = useState('');
  const pairs = filters.apiFilters.additionalMetadata ?? [];
  const operator = filters.apiFilters.additionalMetadataOperator;

  const add = () => {
    if (!key.trim() || !value.trim()) {
      return;
    }
    setFilter(additionalMetadataKey, [
      ...pairs,
      `${key.trim()}:${value.trim()}`,
    ]);
    setKey('');
    setValue('');
    setAdding(false);
  };

  return (
    <FilterBlock
      title="Metadata"
      aside={
        <div className="w-24">
          <Segmented
            label="Metadata match"
            value={operator}
            options={[
              { value: V1AdditionalMetadataOperator.AND, label: 'AND' },
              { value: V1AdditionalMetadataOperator.OR, label: 'OR' },
            ]}
            onChange={filters.setAdditionalMetadataOperator}
          />
        </div>
      }
    >
      <div className="space-y-1">
        {pairs.map((pair, index) => {
          const [k, ...rest] = pair.split(':');
          return (
            <RemovableRow
              key={pair}
              removeLabel={`Remove ${pair}`}
              onRemove={() =>
                setFilter(
                  additionalMetadataKey,
                  pairs.filter((_, i) => i !== index),
                )
              }
            >
              <span className="font-medium">{k}</span>
              <span className="text-muted-foreground"> = </span>
              <span>{rest.join(':')}</span>
            </RemovableRow>
          );
        })}

        {adding ? (
          <div className="space-y-1.5 pt-1">
            <div className="grid grid-cols-2 gap-1.5">
              <Input
                aria-label="Metadata key"
                placeholder="key"
                value={key}
                autoFocus
                onChange={(e) => setKey(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && add()}
                className="h-7 text-xs"
              />
              <Input
                aria-label="Metadata value"
                placeholder="value"
                value={value}
                onChange={(e) => setValue(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && add()}
                className="h-7 text-xs"
              />
            </div>
            <div className="flex gap-1.5">
              <Button
                size="sm"
                variant="outline"
                className="h-6 flex-1 px-2 text-xs"
                disabled={!key.trim() || !value.trim()}
                onClick={add}
              >
                Add
              </Button>
              <Button
                size="sm"
                className="h-6 px-2 text-xs"
                variant="ghost"
                onClick={() => {
                  setAdding(false);
                  setKey('');
                  setValue('');
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <Button
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs text-muted-foreground"
            onClick={() => setAdding(true)}
          >
            + Add condition
          </Button>
        )}
      </div>
    </FilterBlock>
  );
}

function IdempotencyKeyBlock() {
  const { filters } = useRunsContext();
  const { setFilter } = useRunFilterActions();
  const [draft, setDraft] = useState('');
  const keys = filters.apiFilters.idempotencyKeys ?? [];

  const add = () => {
    if (!draft.trim() || keys.includes(draft.trim())) {
      return;
    }
    setFilter(idempotencyKeyKey, [...keys, draft.trim()]);
    setDraft('');
  };

  return (
    <FilterBlock title="Idempotency key">
      <div className="space-y-1">
        {keys.map((k) => (
          <RemovableRow
            key={k}
            removeLabel={`Remove ${k}`}
            onRemove={() =>
              setFilter(
                idempotencyKeyKey,
                keys.filter((x) => x !== k),
              )
            }
          >
            {k}
          </RemovableRow>
        ))}
        <Input
          aria-label="Add idempotency key"
          placeholder="Add a key and press Enter"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && add()}
          className="h-7 text-xs"
        />
      </div>
    </FilterBlock>
  );
}

export function RunsFilterSidebar() {
  const {
    filters,
    toolbarFilters,
    display: { hiddenFilters = [], hideDateFilter },
  } = useRunsContext();

  const visible = useMemo(() => {
    const ids = new Set(
      toolbarFilters
        .map((f) => f.columnId)
        .filter((id) => !hiddenFilters.includes(id as never)),
    );
    return (id: string) => ids.has(id);
  }, [toolbarFilters, hiddenFilters]);

  return (
    <aside aria-label="Filters" className="runs-filter-sidebar bg-background">
      <div className="flex items-center justify-between border-b px-3.5 py-2.5">
        <h2 className="text-[13px] font-semibold">Filters</h2>
        <Button
          variant="ghost"
          size="sm"
          className="h-6 px-2 text-xs"
          onClick={filters.resetFilters}
        >
          Reset
        </Button>
      </div>
      {visible(createdAtKey) && !hideDateFilter && <TimeRangeBlock />}
      {visible(statusKey) && <StatusBlock />}
      {visible(workflowKey) && <WorkflowBlock />}
      {visible(additionalMetadataKey) && <MetadataBlock />}
      {visible(idempotencyKeyKey) && <IdempotencyKeyBlock />}
      {visible(flattenDAGsKey) && <FlattenDAGsBlock />}
    </aside>
  );
}
