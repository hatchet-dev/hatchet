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

  return [
    {
      columnId: TaskRunColumn.workflow,
      title: 'Workflow',
      options: workflowKeyFilters,
      type: ToolbarType.Checkbox,
    },
    {
      columnId: TaskRunColumn.status,
      title: 'Status',
      options: workflowRunStatusFilters,
      type: ToolbarType.Checkbox,
    },
    {
      columnId: TaskRunColumn.additionalMetadata,
      title: 'Metadata',
      type: ToolbarType.KeyValue,
    },
    {
      columnId: TaskRunColumn.flattenDAGs,
      title: 'Flatten DAGs',
      type: ToolbarType.Switch,
      options: [
        { value: 'true', label: 'Flatten' },
        { value: 'false', label: 'All' },
      ],
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);
};
