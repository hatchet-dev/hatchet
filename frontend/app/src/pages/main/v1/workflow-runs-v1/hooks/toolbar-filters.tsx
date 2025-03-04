import { V1TaskStatus } from '@/lib/api';
import { useWorkflow } from './workflow';
import { useMemo } from 'react';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { TaskRunColumn } from '../components/v1/task-runs-columns';

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
}: {
  filterVisibility: { [key: string]: boolean };
}) => {
  const { workflowKeys } = useWorkflow();

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const filters: ToolbarFilters = [
    {
      columnId: TaskRunColumn.workflow,
      title: 'Workflow',
      options: workflowKeyFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: TaskRunColumn.status,
      title: 'Status',
      options: workflowRunStatusFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: TaskRunColumn.additionalMetadata,
      title: 'Metadata',
      type: ToolbarType.KeyValue,
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);

  return filters;
};
