import * as React from 'react';
import { Table } from '@tanstack/react-table';
import { DataTableOptions } from './data-table-options';
import { Input } from '@/components/v1/ui/input.tsx';
import { Spinner } from '@/components/v1/ui/loading';
import { flattenDAGsKey } from '@/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns';

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
  type: ToolbarType;
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
  hideFlatten?: boolean;
  columnKeyToName?: Record<string, string>;
}

export function DataTableToolbar<TData>({
  table,
  filters,
  actions,
  setSearch,
  search,
  showColumnToggle,
  isLoading = false,
  hideFlatten,
  columnKeyToName,
}: DataTableToolbarProps<TData>) {
  const visibleFilters = filters.filter((filter) => {
    if (hideFlatten && filter.columnId === flattenDAGsKey) {
      return false;
    }
    return true;
  });

  const hasFilters = visibleFilters.length > 0;

  return (
    <div className="flex items-center justify-between">
      {setSearch && (
        <div className="flex flex-1 items-center space-x-2 overflow-x-auto pr-4 min-w-0">
          <Input
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-8 w-[150px] lg:w-[200px] flex-shrink-0"
          />
        </div>
      )}

      <div className="flex flex-row items-center flex-shrink-0 w-full justify-between overflow-x-auto">
        <div className="flex items-center min-w-0 flex-shrink-0">
          {isLoading && <Spinner />}
          {actions && actions.length > 0 && actions[0]}
        </div>
        <div className="flex flex-row gap-2 items-center flex-shrink-0">
          {actions && actions.length > 0 && actions.slice(1)}
          {(hasFilters || showColumnToggle) && (
            <DataTableOptions
              table={table}
              filters={visibleFilters}
              hideFlatten={hideFlatten}
              columnKeyToName={columnKeyToName}
            />
          )}
        </div>
      </div>
    </div>
  );
}
