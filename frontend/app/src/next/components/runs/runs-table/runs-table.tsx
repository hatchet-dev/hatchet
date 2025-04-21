import { useRuns, RunsFilters } from '@/next/hooks/use-runs';
import { useMemo } from 'react';
import { DataTable } from './data-table';
import { columns } from './columns';
import {
  Pagination,
  PageSizeSelector,
  PageSelector,
  usePagination,
} from '@/next/components/ui/pagination';
import { useFilters } from '@/next/hooks/use-filters';
import {
  FilterGroup,
  FilterSelect,
  FilterTaskSelect,
  FilterKeyValue,
} from '@/next/components/ui/filters/filters';
import { V1TaskStatus, V1TaskSummary } from '@/lib/api';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/docs-meta-data';
import { RunsMetricsView } from '../runs-metrics/runs-metrics';

interface RunsTableProps {
  rowClicked: (row: V1TaskSummary) => void;
  selectedTaskId?: string;
}

export function RunsTable({ rowClicked, selectedTaskId }: RunsTableProps) {
  const { filters } = useFilters<RunsFilters>();
  const pagination = usePagination();

  const {
    data: runs,
    metrics,
    isLoading,
  } = useRuns({
    pagination,
    filters,
    refetchInterval: 3000,
  });

  const tableColumns = useMemo(() => columns(rowClicked), [rowClicked]);

  const additionalMetaOpts = useMemo(() => {
    if (!runs || runs.length === 0) {
      return [];
    }

    const allKeys = new Set<string>();
    runs.forEach((run) => {
      if (run.additionalMetadata) {
        Object.keys(run.additionalMetadata).forEach((key) => allKeys.add(key));
      }
    });

    return Array.from(allKeys).map((key) => ({
      label: key,
      value: key,
    }));
  }, [runs]);

  const emptyState = useMemo(
    () => (
      <div className="flex flex-col items-center justify-center gap-4 py-8">
        <p className="text-md">No runs found.</p>
        <p className="text-sm text-muted-foreground">
          Trigger a new run to get started.
        </p>
        <DocsButton
          doc={docs.home['running-tasks']}
          titleOverride="Running Tasks"
        />
      </div>
    ),
    [],
  );

  return (
    <div className="flex flex-col gap-4 mt-4">
      <RunsMetricsView metrics={metrics} />
      <FilterGroup>
        <FilterSelect<RunsFilters, V1TaskStatus[]>
          name="statuses"
          value={filters.statuses}
          placeholder="Status"
          multi
          options={[
            { label: 'Running', value: V1TaskStatus.RUNNING },
            { label: 'Completed', value: V1TaskStatus.COMPLETED },
            { label: 'Failed', value: V1TaskStatus.FAILED },
            { label: 'Cancelled', value: V1TaskStatus.CANCELLED },
            { label: 'Queued', value: V1TaskStatus.QUEUED },
          ]}
        />
        <FilterTaskSelect<RunsFilters>
          name="workflow_ids"
          placeholder="Name"
          multi
        />
        <FilterSelect<RunsFilters, boolean>
          name="is_root_task"
          value={filters.is_root_task}
          placeholder="Only Root Tasks"
          options={[
            { label: 'Yes', value: true },
            { label: 'No', value: false },
          ]}
        />
        <FilterTaskSelect<RunsFilters>
          name="workflow_ids"
          placeholder="Task Name"
          multi
        />
        <FilterKeyValue<RunsFilters>
          name="additional_metadata"
          placeholder="Metadata"
          options={additionalMetaOpts}
        />
      </FilterGroup>
      <DataTable
        columns={tableColumns}
        data={runs || []}
        emptyState={emptyState}
        isLoading={isLoading}
        selectedTaskId={selectedTaskId}
      />
      <Pagination className="p-2 justify-between flex flex-row">
        <PageSizeSelector />
        <PageSelector variant="dropdown" />
      </Pagination>
    </div>
  );
}
