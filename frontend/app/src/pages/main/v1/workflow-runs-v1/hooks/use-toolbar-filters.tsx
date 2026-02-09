import {
  additionalMetadataKey,
  createdAtKey,
  flattenDAGsKey,
  statusKey,
  workflowKey,
} from '../components/v1/task-runs-columns';
import { FilterActions } from './use-runs-table-filters';
import { useWorkflows } from './use-workflows';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
  TimeRangeConfig,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { V1TaskStatus } from '@/lib/api';
import { useMemo } from 'react';

export const workflowRunStatusFilters = [
  {
    value: V1TaskStatus.COMPLETED,
    label: 'Succeeded',
  },
  {
    value: V1TaskStatus.FAILED,
    label: 'Failed',
  },
  {
    value: V1TaskStatus.RUNNING,
    label: 'Running',
  },
  {
    value: V1TaskStatus.QUEUED,
    label: 'Queued',
  },
  {
    value: V1TaskStatus.CANCELLED,
    label: 'Cancelled',
  },
];

export const useToolbarFilters = ({
  filterVisibility,
  filterActions,
}: {
  filterVisibility: { [key: string]: boolean };
  filterActions: FilterActions;
}): ToolbarFilters => {
  const workflows = useWorkflows();

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflows.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflows]);

  const timeRangeConfig: TimeRangeConfig = {
    onTimeWindowChange: (value: string) => {
      if (value !== 'custom') {
        filterActions.setTimeWindow(value as any);
      } else {
        filterActions.setCustomTimeRange({
          start:
            filterActions.apiFilters.since ||
            new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
          end: filterActions.apiFilters.until || new Date().toISOString(),
        });
      }
    },
    onCreatedAfterChange: (date?: string) => {
      if (filterActions.isCustomTimeRange && filterActions.apiFilters.until) {
        filterActions.setCustomTimeRange({
          start:
            date || new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
          end: filterActions.apiFilters.until,
        });
      }
    },
    onFinishedBeforeChange: (date?: string) => {
      if (filterActions.isCustomTimeRange && filterActions.apiFilters.since) {
        filterActions.setCustomTimeRange({
          start: filterActions.apiFilters.since,
          end: date || new Date().toISOString(),
        });
      }
    },
    onClearTimeRange: () => filterActions.setCustomTimeRange(null),
    currentTimeWindow: filterActions.timeWindow,
    isCustomTimeRange: filterActions.isCustomTimeRange,
    createdAfter: filterActions.apiFilters.since,
    finishedBefore: filterActions.apiFilters.until,
  };

  return [
    {
      columnId: createdAtKey,
      title: 'Time Range',
      type: ToolbarType.TimeRange,
      timeRangeConfig,
    },
    {
      columnId: workflowKey,
      title: 'Workflow',
      options: workflowKeyFilters,
      type: ToolbarType.Checkbox,
    },
    {
      columnId: statusKey,
      title: 'Status',
      options: workflowRunStatusFilters,
      type: ToolbarType.Checkbox,
    },
    {
      columnId: additionalMetadataKey,
      title: 'Metadata',
      type: ToolbarType.KeyValue,
    },
    {
      columnId: flattenDAGsKey,
      title: 'Flatten DAGs',
      type: ToolbarType.Switch,
      options: [
        { value: 'true', label: 'Flatten' },
        { value: 'false', label: 'All' },
      ],
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);
};
