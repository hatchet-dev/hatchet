import { V1TaskStatus } from '@/lib/api';
import { useMemo } from 'react';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { TaskRunColumn } from '../components/v1/task-runs-columns';
import { useWorkflows } from './use-workflows';

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
  const workflows = useWorkflows();

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflows.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflows]);

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
