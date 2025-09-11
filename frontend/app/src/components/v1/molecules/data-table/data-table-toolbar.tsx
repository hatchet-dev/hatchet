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
  metrics?: JSX.Element;
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
  metrics,
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
      <div className="flex flex-1 items-center space-x-2 overflow-x-auto pr-4 min-w-0">
        {setSearch && (
          <Input
            placeholder="Search..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="h-8 w-[150px] lg:w-[200px] flex-shrink-0"
          />
        )}
        {metrics && <div className="flex-shrink-0">{metrics}</div>}
      </div>

      <div className="flex flex-row gap-2 items-center flex-shrink-0">
        {isLoading && <Spinner />}
        {actions && actions.length > 0 && actions}
        {(hasFilters || showColumnToggle) && (
          <DataTableOptions
            table={table}
            filters={visibleFilters}
            onReset={onReset}
            hideFlatten={hideFlatten}
            columnKeyToName={columnKeyToName}
          />
        )}{' '}
      </div>
    </div>
  );
}
