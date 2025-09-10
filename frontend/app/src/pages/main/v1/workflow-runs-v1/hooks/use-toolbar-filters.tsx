import { V1TaskStatus } from '@/lib/api';
import { useMemo } from 'react';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
  TimeRangeConfig,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import {
  additionalMetadataKey,
  flattenDAGsKey,
  statusKey,
  workflowKey,
} from '../components/v1/task-runs-columns';
import { useWorkflows } from './use-workflows';
import { RunsTableState } from './use-runs-table-state';
import { FilterActions } from './use-runs-table-filters';

const workflowRunStatusFilters = [
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
  state,
  filterActions,
}: {
  filterVisibility: { [key: string]: boolean };
  state: RunsTableState;
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
          start: state.createdAfter || new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
          end: state.finishedBefore || new Date().toISOString(),
        });
      }
    },
    onCreatedAfterChange: (date?: string) => {
      if (state.isCustomTimeRange && state.finishedBefore) {
        filterActions.setCustomTimeRange({
          start: date || new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
          end: state.finishedBefore,
        });
      }
    },
    onFinishedBeforeChange: (date?: string) => {
      if (state.isCustomTimeRange && state.createdAfter) {
        filterActions.setCustomTimeRange({
          start: state.createdAfter,
          end: date || new Date().toISOString(),
        });
      }
    },
    onClearTimeRange: () => filterActions.setCustomTimeRange(null),
    currentTimeWindow: state.timeWindow,
    isCustomTimeRange: state.isCustomTimeRange,
    createdAfter: state.createdAfter,
    finishedBefore: state.finishedBefore,
  };

  return [
    {
      columnId: 'time-range',
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
