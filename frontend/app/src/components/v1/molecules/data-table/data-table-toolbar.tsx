import { DataTableOptions } from './data-table-options';
import { RefetchIntervalDropdown } from '@/components/refetch-interval-dropdown';
import { Spinner } from '@/components/v1/ui/loading';
import {
  ActionType,
  BaseTaskRunActionParams,
} from '@/pages/main/v1/task-runs-v1/actions';
import { TableActions } from '@/pages/main/v1/workflow-runs-v1/components/task-runs-table/table-actions';
import { Table } from '@tanstack/react-table';
import * as React from 'react';

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

type RefetchProps = {
  isRefetching: boolean;
  onRefetch: () => void;
};

export type ShowTableActionsProps = {
  showTableActions: true;
  onTriggerWorkflow: () => void;
  selectedActionType: ActionType | null;
  actionModalParams: BaseTaskRunActionParams;
};

interface DataTableToolbarProps<TData> {
  table: Table<TData>;
  filters: ToolbarFilters;
  leftActions?: JSX.Element[];
  rightActions?: JSX.Element[];
  showColumnToggle?: boolean;
  isLoading?: boolean;
  hiddenFilters: string[];
  columnKeyToName?: Record<string, string>;
  refetchProps?: RefetchProps;
  tableActions?: ShowTableActionsProps;
  onResetFilters?: () => void;
}

export function DataTableToolbar<TData>({
  table,
  filters,
  leftActions,
  rightActions = [],
  showColumnToggle,
  isLoading = false,
  hiddenFilters,
  columnKeyToName,
  refetchProps,
  tableActions,
  onResetFilters,
}: DataTableToolbarProps<TData>) {
  const visibleFilters = filters.filter((filter) => {
    if (hiddenFilters.includes(filter.columnId)) {
      return false;
    }
    return true;
  });

  const hasFilters = visibleFilters.length > 0;
  if (tableActions) {
    rightActions.push(
      <TableActions
        key="table-actions"
        onTriggerWorkflow={tableActions.onTriggerWorkflow}
      />,
    );
  }

  return (
    <div className="flex items-center justify-between">
      <div className="flex w-full flex-shrink-0 flex-row items-center justify-between overflow-x-auto">
        <div className="flex min-w-0 flex-shrink-0 items-center gap-2">
          {isLoading && <Spinner />}
          {leftActions}
        </div>
        <div className="flex flex-shrink-0 flex-row items-center gap-2">
          {rightActions}
          {refetchProps && (
            <RefetchIntervalDropdown
              isRefetching={refetchProps.isRefetching}
              onRefetch={refetchProps.onRefetch}
            />
          )}
          {(hasFilters || showColumnToggle) && (
            <DataTableOptions
              table={table}
              filters={visibleFilters}
              hiddenFilters={hiddenFilters}
              columnKeyToName={columnKeyToName}
              onResetFilters={onResetFilters}
            />
          )}
        </div>
      </div>
    </div>
  );
}
