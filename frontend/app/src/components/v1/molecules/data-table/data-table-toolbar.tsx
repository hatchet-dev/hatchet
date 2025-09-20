import * as React from 'react';
import { Table } from '@tanstack/react-table';
import { DataTableOptions } from './data-table-options';
import { Spinner } from '@/components/v1/ui/loading';
import { flattenDAGsKey } from '@/pages/main/v1/workflow-runs-v1/components/v1/task-runs-columns';
import { RefetchIntervalDropdown } from '@/components/refetch-interval-dropdown';

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
  Search = 'search',
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
  leftActions?: JSX.Element[];
  rightActions?: JSX.Element[];
  showColumnToggle?: boolean;
  isLoading?: boolean;
  hideFlatten?: boolean;
  columnKeyToName?: Record<string, string>;
  isRefetching: boolean;
  onRefetch: () => void;
}

export function DataTableToolbar<TData>({
  table,
  filters,
  leftActions,
  rightActions,
  showColumnToggle,
  isLoading = false,
  hideFlatten,
  columnKeyToName,
  isRefetching,
  onRefetch,
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
      <div className="flex flex-row items-center flex-shrink-0 w-full justify-between overflow-x-auto">
        <div className="flex items-center gap-2 min-w-0 flex-shrink-0">
          {isLoading && <Spinner />}
          {leftActions}
        </div>
        <div className="flex flex-row gap-2 items-center flex-shrink-0">
          {rightActions}
          <RefetchIntervalDropdown
            isRefetching={isRefetching}
            onRefetch={onRefetch}
          />
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
