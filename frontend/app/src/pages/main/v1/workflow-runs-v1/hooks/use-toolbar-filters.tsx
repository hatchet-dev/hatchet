import { V1TaskStatus } from '@/lib/api';
import { useMemo } from 'react';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import {
  additionalMetadataKey,
  flattenDAGsKey,
  statusKey,
  workflowKey,
} from '../components/v1/task-runs-columns';
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
