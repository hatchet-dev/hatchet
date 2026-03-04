import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../ui/tabs';
import { ToolbarFilters } from './data-table-toolbar';
import {
  ToolbarType,
  FilterOption,
  TimeRangeConfig,
} from './data-table-toolbar';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import { Checkbox } from '@/components/v1/ui/checkbox';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Input } from '@/components/v1/ui/input';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { V1TaskStatus } from '@/lib/api';
import {
  flattenDAGsKey,
  createdAfterKey,
  finishedBeforeKey,
  statusKey,
  isCustomTimeRangeKey,
  timeWindowKey,
} from '@/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { Cross2Icon, MixerHorizontalIcon } from '@radix-ui/react-icons';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { ColumnFiltersState, Table } from '@tanstack/react-table';
import { Column } from '@tanstack/react-table';
import * as React from 'react';

interface FilterControlProps<TData> {
  column?: Column<TData, any>;
  filter: {
    columnId: string;
    title: string;
    type?: ToolbarType;
    options?: FilterOption[];
    timeRangeConfig?: TimeRangeConfig;
  };
}

function FilterControl<TData>({ column, filter }: FilterControlProps<TData>) {
  const value = column?.getFilterValue();
  const [searchTerm, setSearchTerm] = React.useState('');
  const keyInputRef = React.useRef<HTMLInputElement>(null);
  const valueInputRef = React.useRef<HTMLInputElement>(null);
  const arrayValueInputRef = React.useRef<HTMLInputElement>(null);
  const [newKey, setNewKey] = React.useState('');
  const [newValue, setNewValue] = React.useState('');
  const [newArrayValue, setNewArrayValue] = React.useState('');

  if (!filter.type) {
    return null;
  }

  switch (filter.type) {
    case ToolbarType.TimeRange:
      const config = filter.timeRangeConfig;
      if (!config) {
        return null;
      }

      return (
        <div className="space-y-3">
          {config.isCustomTimeRange && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-muted-foreground">
                  Custom Range
                </span>
                <Button
                  onClick={config.onClearTimeRange}
                  variant="ghost"
                  size="sm"
                  leftIcon={<XCircleIcon className="size-4" />}
                >
                  Clear
                </Button>
              </div>
              <div className="space-y-2">
                <div className="w-full space-y-1">
                  <DateTimePicker
                    label="After"
                    date={
                      config.createdAfter
                        ? new Date(config.createdAfter)
                        : undefined
                    }
                    setDate={(date) =>
                      config.onCreatedAfterChange?.(date?.toISOString())
                    }
                    triggerClassName="w-full"
                  />
                </div>
                <div className="w-full space-y-1">
                  <DateTimePicker
                    label="Before"
                    date={
                      config.finishedBefore
                        ? new Date(config.finishedBefore)
                        : undefined
                    }
                    setDate={(date) =>
                      config.onFinishedBeforeChange?.(date?.toISOString())
                    }
                    triggerClassName="w-full"
                  />
                </div>
              </div>
            </div>
          )}

          {!config.isCustomTimeRange && (
            <div className="space-y-1">
              <Select
                value={
                  config.isCustomTimeRange ? 'custom' : config.currentTimeWindow
                }
                onValueChange={(value) => config.onTimeWindowChange?.(value)}
              >
                <SelectTrigger className="h-8 text-xs">
                  <SelectValue placeholder="Choose time range" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1h">1 hour</SelectItem>
                  <SelectItem value="6h">6 hours</SelectItem>
                  <SelectItem value="1d">1 day</SelectItem>
                  <SelectItem value="7d">7 days</SelectItem>
                  <SelectItem value="custom">Custom</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}
        </div>
      );
    case ToolbarType.Switch:
      return (
        <div className="flex items-center justify-between rounded-md border bg-muted/10 px-3 py-2 hover:bg-muted/50">
          <Label
            htmlFor={`filter-${filter.columnId}`}
            className="flex-1 cursor-pointer text-sm font-medium"
          >
            {filter.title}
          </Label>
          <Checkbox
            id={`filter-${filter.columnId}`}
            checked={!!value}
            onCheckedChange={(checked) =>
              column?.setFilterValue(checked === true ? true : undefined)
            }
          />
        </div>
      );
    case ToolbarType.KeyValue:
      const currentKVPairs = Array.isArray(value)
        ? value
        : value
          ? [value]
          : [];

      const addKeyValue = () => {
        if (newKey.trim() && newValue.trim()) {
          const keyValuePair = `${newKey.trim()}:${newValue.trim()}`;
          column?.setFilterValue([...currentKVPairs, keyValuePair]);
          setNewKey('');
          setNewValue('');
        }
      };

      return (
        <div className="space-y-3">
          {currentKVPairs.length > 0 && (
            <div className="space-y-2">
              {currentKVPairs.map((val: string, index: number) => {
                const separator = val.includes(':') ? ':' : '=';
                const [key, value] = val.split(separator);
                return (
                  <div
                    key={index}
                    className="flex items-center justify-between rounded-md bg-muted/50 px-2 py-1 text-xs"
                  >
                    <div className="flex items-center gap-1 font-mono">
                      <span className="font-medium text-blue-600">{key}</span>
                      <span className="text-muted-foreground">{separator}</span>
                      <span className="text-green-600">{value}</span>
                    </div>
                    <Button
                      variant="icon"
                      size="xs"
                      onClick={() => {
                        const newValues = currentKVPairs.filter(
                          (_, i) => i !== index,
                        );
                        column?.setFilterValue(
                          newValues.length > 0 ? newValues : undefined,
                        );
                      }}
                    >
                      <Cross2Icon className="size-3" />
                    </Button>
                  </div>
                );
              })}
            </div>
          )}

          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <div className="space-y-1">
                <label className="text-xs text-muted-foreground">Key</label>
                <Input
                  ref={keyInputRef}
                  placeholder="ENV"
                  value={newKey}
                  onChange={(e) => setNewKey(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      addKeyValue();
                    } else if (e.key === 'Tab' && !e.shiftKey) {
                      e.preventDefault();
                      valueInputRef.current?.focus();
                    }
                  }}
                  className="h-8 text-xs placeholder:text-muted-foreground/50"
                />
              </div>
              <div className="space-y-1">
                <label className="text-xs text-muted-foreground">Value</label>
                <Input
                  ref={valueInputRef}
                  placeholder="PRODUCTION"
                  value={newValue}
                  onChange={(e) => setNewValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      addKeyValue();
                    } else if (e.key === 'Tab' && e.shiftKey) {
                      e.preventDefault();
                      keyInputRef.current?.focus();
                    }
                  }}
                  className="h-8 text-xs placeholder:text-muted-foreground/50"
                />
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              disabled={!newKey.trim() || !newValue.trim()}
              onClick={addKeyValue}
              className="w-full"
            >
              Add Filter
            </Button>
          </div>
        </div>
      );
    case ToolbarType.Array:
      const currentArrayValues = Array.isArray(value)
        ? value
        : value
          ? [value]
          : [];

      const addArrayValue = () => {
        if (newArrayValue.trim()) {
          column?.setFilterValue([...currentArrayValues, newArrayValue]);
          setNewArrayValue('');
        }
      };

      return (
        <div className="space-y-3">
          {currentArrayValues.length > 0 && (
            <div className="space-y-2">
              {currentArrayValues.map((val: string, index: number) => {
                return (
                  <div
                    key={index}
                    className="flex items-center justify-between rounded-md bg-muted/50 px-2 py-1 text-xs"
                  >
                    <div className="flex items-center gap-1 font-mono">
                      <span className="text-muted-foreground">{val}</span>
                    </div>
                    <Button
                      variant="icon"
                      size="xs"
                      onClick={() => {
                        const newValues = currentArrayValues.filter(
                          (_, i) => i !== index,
                        );
                        column?.setFilterValue(
                          newValues.length > 0 ? newValues : undefined,
                        );
                      }}
                    >
                      <Cross2Icon className="size-3" />
                    </Button>
                  </div>
                );
              })}
            </div>
          )}

          <div className="space-y-3">
            <Input
              ref={arrayValueInputRef}
              placeholder="foobar"
              value={newArrayValue}
              onChange={(e) => setNewArrayValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  addArrayValue();
                }
              }}
              className="h-8 w-full text-xs placeholder:text-muted-foreground/50"
            />
            <Button
              variant="outline"
              size="sm"
              disabled={!newArrayValue.trim()}
              onClick={addArrayValue}
              className="w-full"
            >
              Add Filter
            </Button>
          </div>
        </div>
      );
    case ToolbarType.Checkbox:
    case ToolbarType.Radio:
      if (!filter.options) {
        return null;
      }

      const selectedValues = Array.isArray(value)
        ? value
        : value
          ? [value]
          : [];
      const filteredOptions = filter.options.filter((option) =>
        option.label.toLowerCase().includes(searchTerm.toLowerCase()),
      );

      return (
        <div className="space-y-2">
          {filter.options.length > 5 && (
            <Input
              placeholder={`Search ${filter.title.toLowerCase()}...`}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="h-8 text-xs"
            />
          )}
          <div className="max-h-56 space-y-1 overflow-y-auto rounded-md border bg-muted/10 p-2">
            {filteredOptions.length > 0 ? (
              filteredOptions.map((option) => (
                <div
                  key={option.value}
                  className="flex items-center space-x-2 rounded-md px-2 py-1.5 hover:bg-muted/50"
                >
                  <Checkbox
                    id={`${filter.columnId}-${option.value}`}
                    checked={selectedValues.includes(option.value)}
                    onCheckedChange={(checked) => {
                      let newValue;
                      if (checked) {
                        newValue = [...selectedValues, option.value];
                      } else {
                        newValue = selectedValues.filter(
                          (v) => v !== option.value,
                        );
                      }
                      column?.setFilterValue(
                        newValue.length > 0 ? newValue : undefined,
                      );
                    }}
                  />
                  <Label
                    htmlFor={`${filter.columnId}-${option.value}`}
                    className="flex-1 cursor-pointer truncate text-sm"
                  >
                    {option.label}
                  </Label>
                </div>
              ))
            ) : (
              <div className="py-3 text-center text-xs text-muted-foreground">
                No options found
              </div>
            )}
          </div>
        </div>
      );
    case ToolbarType.Search:
      const currentSearchTerm = value ? String(value) || '' : '';

      return (
        <Input
          ref={keyInputRef}
          placeholder={`Search ${filter.title.toLowerCase()}...`}
          value={currentSearchTerm}
          onChange={(e) => column?.setFilterValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              column?.setFilterValue(undefined);
            }
          }}
          className="h-8 w-full text-xs placeholder:text-muted-foreground/50"
        />
      );

    default:
      const exhaustiveCheck: never = filter.type;
      return exhaustiveCheck;
  }
}

interface DataTableOptionsProps<TData> {
  table: Table<TData>;
  filters: ToolbarFilters;
  hiddenFilters: string[];
  columnKeyToName?: Record<string, string>;
  onResetFilters?: () => void;
}

function arraysEqual<T>(a: T[], b: T[]) {
  return (
    Array.isArray(a) &&
    Array.isArray(b) &&
    a.length === b.length &&
    a.every((val, index) => val === b[index])
  );
}

export function DataTableOptions<TData>({
  table,
  filters,
  hiddenFilters,
  columnKeyToName,
  onResetFilters,
}: DataTableOptionsProps<TData>) {
  const cf: ColumnFiltersState | undefined = table.getState().columnFilters;
  const activeFiltersCount = React.useMemo(
    () =>
      cf?.filter((f) => {
        if (
          f.id === statusKey &&
          arraysEqual(f.value as V1TaskStatus[], Object.values(V1TaskStatus))
        ) {
          return false;
        }

        if (
          f.id === createdAfterKey ||
          f.id === finishedBeforeKey ||
          (f.id === isCustomTimeRangeKey && f.value !== true) ||
          (f.id === timeWindowKey && f.value === '1d')
        ) {
          return false;
        }

        if (f.id === flattenDAGsKey && !f.value) {
          return false;
        }

        if (hiddenFilters.includes(f.id)) {
          return false;
        }

        if (f.value === undefined || f.value === null || f.value === '') {
          return false;
        }

        if (Array.isArray(f.value) && f.value.length === 0) {
          return false;
        }

        return true;
      })?.length || 0,
    [hiddenFilters, cf],
  );

  const visibleFilters = filters.filter((filter) => {
    if (hiddenFilters.includes(filter.columnId)) {
      return false;
    }
    return true;
  });

  const hasFilters = visibleFilters.length > 0;
  const hasVisibleColumns =
    table
      .getAllColumns()
      .filter(
        (column) =>
          typeof column.accessorFn !== 'undefined' && column.getCanHide(),
      ).length > 0;

  if (!hasFilters && !hasVisibleColumns) {
    return null;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className="flex-shrink-0">
          <MixerHorizontalIcon className="size-4" />
          <span className="cq-xl:inline ml-2 hidden text-sm">Filters</span>
          {activeFiltersCount > 0 && (
            <Badge variant="secondary" className="ml-2 px-1 py-0 text-xs">
              {activeFiltersCount}
            </Badge>
          )}
          <ChevronDownIcon className="cq-xl:inline ml-2 hidden size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="max-h-[32rem] w-96 overflow-y-auto p-0 shadow-lg"
      >
        <DataTableOptionsContent
          table={table}
          filters={filters}
          hiddenFilters={hiddenFilters}
          columnKeyToName={columnKeyToName}
          showColumnVisibility
          onResetFilters={onResetFilters}
          activeFiltersCount={activeFiltersCount}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export interface DataTableOptionsContentProps<TData> {
  table: Table<TData>;
  filters: ToolbarFilters;
  columnKeyToName?: Record<string, string>;
  hiddenFilters: string[];
  showColumnVisibility: boolean;
  onResetFilters?: () => void;
  activeFiltersCount?: number;
}

interface FiltersContentProps<TData> {
  table: Table<TData>;
  filters: ToolbarFilters;
  hiddenFilters: string[];
  onResetFilters?: () => void;
  activeFiltersCount?: number;
}

function FiltersContent<TData>({
  table,
  filters,
  hiddenFilters,
  onResetFilters,
  activeFiltersCount = 0,
}: FiltersContentProps<TData>) {
  const visibleFilters = filters.filter((filter) => {
    if (hiddenFilters.includes(filter.columnId)) {
      return false;
    }
    return true;
  });

  return (
    <div className="space-y-0">
      {onResetFilters && activeFiltersCount > 0 && (
        <div className="border-b bg-muted/10 p-3">
          <Button
            variant="outline"
            size="sm"
            onClick={onResetFilters}
            className="w-full"
          >
            <Cross2Icon className="mr-2 size-3" />
            Clear All Filters
          </Button>
        </div>
      )}
      <div className="max-h-96 overflow-y-auto">
        <div className="space-y-4 p-3">
          {visibleFilters.map((filter, index) => (
            <div key={filter.columnId} className="space-y-2">
              <div className="flex items-center justify-between">
                <label className="text-sm font-medium text-foreground">
                  {filter.title}
                </label>
                {table.getColumn(filter.columnId)?.getFilterValue() !==
                  undefined &&
                  [
                    ToolbarType.Array,
                    ToolbarType.KeyValue,
                    ToolbarType.Checkbox,
                    ToolbarType.TimeRange,
                    ToolbarType.Radio,
                  ].includes(filter.type) && (
                    <Button
                      variant="icon"
                      size="xs"
                      onClick={() =>
                        table
                          .getColumn(filter.columnId)
                          ?.setFilterValue(undefined)
                      }
                    >
                      <Cross2Icon className="size-3" />
                    </Button>
                  )}
              </div>
              <FilterControl
                column={table.getColumn(filter.columnId)}
                filter={filter}
              />
              {index < visibleFilters.length - 1 && (
                <div className="border-t border-border/20 pt-3" />
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

interface ColumnsContentProps<TData> {
  table: Table<TData>;
  columnKeyToName?: Record<string, string>;
}

function ColumnsContent<TData>({
  table,
  columnKeyToName,
}: ColumnsContentProps<TData>) {
  return (
    <div className="space-y-0">
      <div className="max-h-80 overflow-y-auto">
        <div className="space-y-1 p-3">
          {table
            .getAllColumns()
            .filter(
              (column) =>
                typeof column.accessorFn !== 'undefined' && column.getCanHide(),
            )
            .map((column) => {
              const columnName =
                (columnKeyToName ?? {})[
                  column.id as keyof typeof columnKeyToName
                ] || column.id;

              return (
                <div
                  key={column.id}
                  className="flex items-center space-x-3 rounded-md px-2 py-2 transition-colors hover:bg-muted/50"
                >
                  <Checkbox
                    id={`column-${column.id}`}
                    checked={column.getIsVisible()}
                    onCheckedChange={(value) =>
                      column.toggleVisibility(!!value)
                    }
                  />
                  <Label
                    htmlFor={`column-${column.id}`}
                    className="flex-1 cursor-pointer truncate text-sm font-medium"
                  >
                    {columnName}
                  </Label>
                </div>
              );
            })}
        </div>
      </div>
      <div className="border-t p-3">
        <div className="flex w-full gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => table.toggleAllColumnsVisible(false)}
            className="flex-1"
          >
            Hide All
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => table.toggleAllColumnsVisible(true)}
            className="flex-1"
          >
            Show All
          </Button>
        </div>
      </div>
    </div>
  );
}

export function DataTableOptionsContent<TData>({
  table,
  filters,
  columnKeyToName,
  hiddenFilters,
  showColumnVisibility,
  onResetFilters,
  activeFiltersCount = 0,
}: DataTableOptionsContentProps<TData>) {
  const [selectedTab, setSelectedTab] = React.useState<'filters' | 'columns'>(
    'filters',
  );

  const visibleFilters = filters.filter((filter) => {
    if (hiddenFilters.includes(filter.columnId)) {
      return false;
    }
    return true;
  });

  const hasFilters = visibleFilters.length > 0;
  const hasVisibleColumns =
    table
      .getAllColumns()
      .filter(
        (column) =>
          typeof column.accessorFn !== 'undefined' && column.getCanHide(),
      ).length > 0;

  const showBothSections =
    hasFilters && hasVisibleColumns && showColumnVisibility;

  if (!showBothSections) {
    return (
      <div className="w-full">
        {hasFilters ? (
          <FiltersContent
            table={table}
            filters={filters}
            hiddenFilters={hiddenFilters}
            onResetFilters={onResetFilters}
            activeFiltersCount={activeFiltersCount}
          />
        ) : hasVisibleColumns && showColumnVisibility ? (
          <ColumnsContent table={table} columnKeyToName={columnKeyToName} />
        ) : (
          <div className="p-6 text-center text-sm text-muted-foreground">
            No options available
          </div>
        )}
      </div>
    );
  }

  return (
    <Tabs
      value={selectedTab}
      onValueChange={(value) => setSelectedTab(value as 'filters' | 'columns')}
      className="w-full rounded-none p-0"
    >
      <TabsList className="grid w-full grid-cols-2 rounded-none bg-muted/30 px-2">
        <TabsTrigger
          value="filters"
          className="text-xs font-medium data-[state=active]:bg-background data-[state=active]:text-foreground"
        >
          <div className="flex items-center gap-2">
            Filters
            {activeFiltersCount > 0 && (
              <Badge
                variant="secondary"
                className="h-4 px-1.5 text-[10px] leading-none"
              >
                {activeFiltersCount}
              </Badge>
            )}
          </div>
        </TabsTrigger>
        <TabsTrigger
          value="columns"
          className="text-xs font-medium data-[state=active]:bg-background data-[state=active]:text-foreground"
        >
          Column Visibility
        </TabsTrigger>
      </TabsList>

      <TabsContent value="filters" className="mt-0 space-y-0">
        <FiltersContent
          table={table}
          filters={filters}
          hiddenFilters={hiddenFilters}
          onResetFilters={onResetFilters}
          activeFiltersCount={activeFiltersCount}
        />
      </TabsContent>

      <TabsContent value="columns" className="mt-0">
        <ColumnsContent table={table} columnKeyToName={columnKeyToName} />
      </TabsContent>
    </Tabs>
  );
}
