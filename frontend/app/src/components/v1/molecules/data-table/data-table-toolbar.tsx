import * as React from 'react';
import { Cross2Icon, MixerHorizontalIcon } from '@radix-ui/react-icons';
import { XCircleIcon } from '@heroicons/react/24/outline';
import { Table } from '@tanstack/react-table';
import { Button } from '@/components/v1/ui/button';
import { DataTableViewOptions } from './data-table-view-options';
import { Input } from '@/components/v1/ui/input.tsx';
import { Spinner } from '@/components/v1/ui/loading';
import { flattenDAGsKey } from '@/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns';
import { Badge } from '@/components/v1/ui/badge';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/v1/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { Checkbox } from '@/components/v1/ui/checkbox';
import { Label } from '@/components/v1/ui/label';
import { Column } from '@tanstack/react-table';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';

export interface FilterOption {
  label: string;
  value: string;
  icon?: React.ComponentType<{ className?: string }>;
}

export enum ToolbarType {
  Checkbox = 'checkbox',
  Radio = 'radio',
  KeyValue = 'key-value',
  Array = 'array',
  Switch = 'switch',
  TimeRange = 'time-range',
}

export interface TimeRangeConfig {
  onTimeWindowChange?: (value: string) => void;
  onCreatedAfterChange?: (date?: string) => void;
  onFinishedBeforeChange?: (date?: string) => void;
  onClearTimeRange?: () => void;
  currentTimeWindow?: string;
  isCustomTimeRange?: boolean;
  createdAfter?: string;
  finishedBefore?: string;
}

export type ToolbarFilters = {
  columnId: string;
  title: string;
  type?: ToolbarType;
  options?: FilterOption[];
  timeRangeConfig?: TimeRangeConfig;
}[];

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
  filters: ToolbarFilters;
  actions: JSX.Element[];
  setSearch?: (search: string) => void;
  search?: string;
  showColumnToggle?: boolean;
  isLoading?: boolean;
  onReset?: () => void;
  hideFlatten?: boolean;
  columnKeyToName?: Record<string, string>;
}

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
        <div className="flex items-center justify-between p-2 bg-muted/30 rounded-md">
          <Label
            htmlFor={`filter-${filter.columnId}`}
            className="text-sm font-medium"
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
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={!newKey.trim() || !newValue.trim()}
                onClick={addKeyValue}
                className="flex-1 h-8 text-xs"
              >
                Add Filter
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={!newKey.trim() && !newValue.trim()}
                onClick={() => {
                  setNewKey('');
                  setNewValue('');
                  keyInputRef.current?.focus();
                }}
                className="h-8 px-3 text-xs"
              >
                Reset
              </Button>
            </div>
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
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={!newArrayValue.trim() || !newArrayValue.trim()}
                onClick={addArrayValue}
                className="flex-1 h-8 text-xs"
              >
                Add Filter
              </Button>
              <Button
                variant="ghost"
                size="sm"
                disabled={!newArrayValue.trim()}
                onClick={() => {
                  setNewArrayValue('');
                  arrayValueInputRef.current?.focus();
                }}
                className="h-8 px-3 text-xs"
              >
                Reset
              </Button>
            </div>
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
        <div className="space-y-3">
          <div className="space-y-2">
            {filter.options.length > 5 && (
              <Input
                placeholder={`Search ${filter.title.toLowerCase()}...`}
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="h-7 text-xs"
              />
            )}
            <div className="max-h-48 overflow-y-auto space-y-1 border rounded-md p-2 bg-muted/20">
              {filteredOptions.length > 0 ? (
                filteredOptions.map((option) => (
                  <div
                    key={option.value}
                    className="flex items-center space-x-2 hover:bg-muted/50 rounded px-1 py-0.5"
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
                <div className="text-xs text-muted-foreground text-center py-2">
                  No workflows found
                </div>
              )}
            </div>
          </div>
        </div>
      );
    default:
      const exhaustiveCheck: never = filter.type;
      return exhaustiveCheck;
  }
}

export function DataTableToolbar<TData>({
  table,
  filters,
  actions,
  setSearch,
  search,
  showColumnToggle,
  isLoading = false,
  onReset,
  hideFlatten,
  columnKeyToName,
}: DataTableToolbarProps<TData>) {
  const isFiltered = table.getState().columnFilters?.length > 0;
  const activeFiltersCount = table.getState().columnFilters?.length || 0;

  const visibleFilters = filters.filter((filter) => {
    if (hideFlatten && filter.columnId === flattenDAGsKey) {
      return false;
    }
    return true;
  });

  const hasFilters = visibleFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-1 items-center space-x-2 overflow-x-auto pr-4 min-w-0">
        {setSearch && (
          <Input
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-8 w-[150px] lg:w-[200px] flex-shrink-0"
          />
        )}

        {hasFilters && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="h-8 flex-shrink-0">
                <MixerHorizontalIcon className="mr-2 h-4 w-4" />
                Filters
                {activeFiltersCount > 0 && (
                  <Badge variant="secondary" className="ml-2 px-1 py-0 text-xs">
                    {activeFiltersCount}
                  </Badge>
                )}
                <ChevronDownIcon className="ml-2 h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="start"
              className="w-96 max-h-96 overflow-y-auto"
            >
              <div className="p-4 space-y-6">
                {visibleFilters.map((filter, index) => (
                  <div key={filter.columnId} className="space-y-3">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full bg-primary/60"></div>
                      <label className="text-sm font-semibold text-foreground">
                        {filter.title}
                      </label>
                    </div>
                    <FilterControl
                      column={table.getColumn(filter.columnId)}
                      filter={filter}
                    />
                    {index < visibleFilters.length - 1 && (
                      <div className="border-b border-border/50" />
                    )}
                  </div>
                ))}
              </div>

              {isFiltered && (
                <>
                  <DropdownMenuSeparator />
                  <div className="p-4">
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => {
                        if (onReset) {
                          onReset();
                        } else {
                          table.resetColumnFilters();
                        }
                      }}
                      className="w-full justify-center font-medium"
                    >
                      <Cross2Icon className="mr-2 h-4 w-4" />
                      Clear All Filters
                    </Button>
                  </div>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <div className="flex flex-row gap-2 items-center flex-shrink-0">
        {isLoading && <Spinner />}
        {actions && actions.length > 0 && actions}
        {showColumnToggle && (
          <DataTableViewOptions
            table={table}
            columnKeyToName={columnKeyToName}
          />
        )}
      </div>
    </div>
  );
}
