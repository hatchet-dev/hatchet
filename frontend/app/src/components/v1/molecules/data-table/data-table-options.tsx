import * as React from 'react';
import { Cross2Icon, MixerHorizontalIcon } from '@radix-ui/react-icons';
import { Table } from '@tanstack/react-table';
import { Button } from '@/components/v1/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import { Badge } from '@/components/v1/ui/badge';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { flattenDAGsKey } from '@/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns';
import { ToolbarFilters } from './data-table-toolbar';
import {
  ToolbarType,
  FilterOption,
  TimeRangeConfig,
} from './data-table-toolbar';
import { Input } from '@/components/v1/ui/input';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { Label } from '@/components/v1/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { Column } from '@tanstack/react-table';

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
              <div className="flex justify-between items-center">
                <span className="text-xs font-medium text-muted-foreground">
                  Custom Range
                </span>
                <Button
                  onClick={config.onClearTimeRange}
                  variant="ghost"
                  size="sm"
                  className="h-6 px-2 text-xs"
                >
                  <XCircleIcon className="h-3 w-3 mr-1" />
                  Clear
                </Button>
              </div>
              <div className="space-y-2">
                <div className="space-y-1 w-full">
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
                <div className="space-y-1 w-full">
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
                <SelectContent className="z-[80]">
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
        <div className="flex items-center justify-between hover:bg-muted/50 rounded-md px-3 py-2 bg-muted/10 border">
          <Label
            htmlFor={`filter-${filter.columnId}`}
            className="text-sm font-medium cursor-pointer flex-1"
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
                    className="flex items-center justify-between bg-muted/50 rounded-md px-2 py-1 text-xs"
                  >
                    <div className="flex items-center gap-1 font-mono">
                      <span className="text-blue-600 font-medium">{key}</span>
                      <span className="text-muted-foreground">{separator}</span>
                      <span className="text-green-600">{value}</span>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        const newValues = currentKVPairs.filter(
                          (_, i) => i !== index,
                        );
                        column?.setFilterValue(
                          newValues.length > 0 ? newValues : undefined,
                        );
                      }}
                      className="h-5 w-5 p-0 hover:bg-destructive/10 hover:text-destructive"
                    >
                      <Cross2Icon className="h-3 w-3" />
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
              className="w-full h-8 text-xs"
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
                    className="flex items-center justify-between bg-muted/50 rounded-md px-2 py-1 text-xs"
                  >
                    <div className="flex items-center gap-1 font-mono">
                      <span className="text-muted-foreground">{val}</span>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        const newValues = currentArrayValues.filter(
                          (_, i) => i !== index,
                        );
                        column?.setFilterValue(
                          newValues.length > 0 ? newValues : undefined,
                        );
                      }}
                      className="h-5 w-5 p-0 hover:bg-destructive/10 hover:text-destructive"
                    >
                      <Cross2Icon className="h-3 w-3" />
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
              className="h-8 text-xs placeholder:text-muted-foreground/50 w-full"
            />
            <Button
              variant="outline"
              size="sm"
              disabled={!newArrayValue.trim()}
              onClick={addArrayValue}
              className="w-full h-8 text-xs"
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
          <div className="max-h-56 overflow-y-auto space-y-1 border rounded-md p-2 bg-muted/10">
            {filteredOptions.length > 0 ? (
              filteredOptions.map((option) => (
                <div
                  key={option.value}
                  className="flex items-center space-x-2 hover:bg-muted/50 rounded-md px-2 py-1.5"
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
                    className="text-sm cursor-pointer flex-1 truncate"
                  >
                    {option.label}
                  </Label>
                </div>
              ))
            ) : (
              <div className="text-xs text-muted-foreground text-center py-3">
                No options found
              </div>
            )}
          </div>
        </div>
      );
    default:
      const exhaustiveCheck: never = filter.type;
      return exhaustiveCheck;
  }
}

interface DataTableOptionsProps<TData> {
  table: Table<TData>;
  filters: ToolbarFilters;
  hideFlatten?: boolean;
  columnKeyToName?: Record<string, string>;
}

export function DataTableOptions<TData>({
  table,
  filters,
  hideFlatten,
  columnKeyToName,
}: DataTableOptionsProps<TData>) {
  const activeFiltersCount = table.getState().columnFilters?.length || 0;

  const visibleFilters = filters.filter((filter) => {
    if (hideFlatten && filter.columnId === flattenDAGsKey) {
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
        <Button variant="outline" size="sm" className="h-8 flex-shrink-0">
          <MixerHorizontalIcon className="h-4 w-4" />
          <span className="cq-xl:inline hidden ml-2">Options</span>
          {activeFiltersCount > 0 && (
            <Badge variant="secondary" className="ml-2 px-1 py-0 text-xs">
              {activeFiltersCount}
            </Badge>
          )}
          <ChevronDownIcon className="h-4 w-4 ml-2 hidden cq-xl:inline" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="w-96 max-h-[32rem] overflow-y-auto z-[70] shadow-lg"
      >
        {hasFilters && (
          <>
            <div className="px-3 py-2 bg-muted/30">
              <div className="flex items-center gap-2">
                <div className="w-1 h-4 bg-primary rounded-full"></div>
                <span className="text-sm font-semibold text-foreground">
                  Filters
                </span>
              </div>
            </div>
            <div className="p-3 space-y-4">
              {visibleFilters.map((filter, index) => (
                <div key={filter.columnId} className="space-y-2">
                  <label className="text-sm font-medium text-foreground">
                    {filter.title}
                  </label>
                  <FilterControl
                    column={table.getColumn(filter.columnId)}
                    filter={filter}
                  />
                  {index < visibleFilters.length - 1 && (
                    <div className="border-t border-border/30 my-3" />
                  )}
                </div>
              ))}
            </div>

            {hasVisibleColumns && <div className="border-t border-border/50" />}
          </>
        )}

        {hasVisibleColumns && (
          <>
            <div className="px-3 py-2 bg-muted/30">
              <div className="flex items-center gap-2">
                <div className="w-1 h-4 bg-secondary rounded-full"></div>
                <span className="text-sm font-semibold text-foreground">
                  Column Visibility
                </span>
              </div>
            </div>
            <div className="p-3 space-y-1">
              {table
                .getAllColumns()
                .filter(
                  (column) =>
                    typeof column.accessorFn !== 'undefined' &&
                    column.getCanHide(),
                )
                .map((column) => {
                  const columnName =
                    (columnKeyToName ?? {})[
                      column.id as keyof typeof columnKeyToName
                    ] || column.id;

                  return (
                    <div
                      key={column.id}
                      className="flex items-center space-x-2 hover:bg-muted/50 rounded-md px-2 py-1.5"
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
                        className="text-sm cursor-pointer flex-1 truncate"
                      >
                        {columnName}
                      </Label>
                    </div>
                  );
                })}
            </div>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
