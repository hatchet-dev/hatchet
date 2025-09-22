import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { useEffect, useMemo, useState } from 'react';
import {
  ColumnFiltersState,
  PaginationState,
  SortingState,
  VisibilityState,
} from '@tanstack/react-table';
import { useQuery } from '@tanstack/react-query';
import {
  ScheduledRunStatus,
  ScheduledWorkflows,
  ScheduledWorkflowsOrderByField,
  WorkflowRunOrderByDirection,
  queries,
} from '@/lib/api';
import { useSearchParams } from 'react-router-dom';
import {
  FilterOption,
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { Button } from '@/components/v1/ui/button';
import { columns } from './scheduled-runs-columns';
import { DeleteScheduledRun } from './delete-scheduled-runs';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';

export interface ScheduledWorkflowRunsTableProps {
  createdAfter?: string;
  createdBefore?: string;
  workflowId?: string;
  parentWorkflowRunId?: string;
  parentStepRunId?: string;
  initColumnVisibility?: VisibilityState;
  filterVisibility?: { [key: string]: boolean };
  showMetrics?: boolean;
}

export function ScheduledRunsTable({
  workflowId,
  initColumnVisibility = {
    createdAt: false,
  },
  filterVisibility = {},
  parentWorkflowRunId,
  parentStepRunId,
}: ScheduledWorkflowRunsTableProps) {
  const { tenantId } = useCurrentTenantId();
  const [searchParams, setSearchParams] = useSearchParams();
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [selectedAdditionalMetaJobId, setSelectedAdditionalMetaJobId] =
    useState<string | null>(null);
  const { refetchInterval } = useRefetchInterval();

  const [sorting, setSorting] = useState<SortingState>(() => {
    const sortParam = searchParams.get('sort');
    if (sortParam) {
      return sortParam.split(',').map((param) => {
        const [id, desc] = param.split(':');
        return { id, desc: desc === 'desc' };
      });
    }
    return [];
  });

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(() => {
    const filtersParam = searchParams.get('filters');
    if (filtersParam) {
      return JSON.parse(filtersParam);
    }
    return [];
  });

  const [columnVisibility, setColumnVisibility] =
    useState<VisibilityState>(initColumnVisibility);

  const [pagination, setPagination] = useState<PaginationState>(() => {
    const pageIndex = Number(searchParams.get('pageIndex')) || 0;
    const pageSize = Number(searchParams.get('pageSize')) || 50;
    return { pageIndex, pageSize };
  });

  useEffect(() => {
    const newSearchParams = new URLSearchParams(searchParams);
    if (sorting.length) {
      newSearchParams.set(
        'orderDirection',
        sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(','),
      );
    } else {
      newSearchParams.delete('orderDirection');
    }
    if (columnFilters.length) {
      newSearchParams.set('filters', JSON.stringify(columnFilters));
    } else {
      newSearchParams.delete('filters');
    }
    newSearchParams.set('pageIndex', pagination.pageIndex.toString());
    newSearchParams.set('pageSize', pagination.pageSize.toString());

    if (newSearchParams.toString() !== searchParams.toString()) {
      setSearchParams(newSearchParams, { replace: true });
    }
  }, [sorting, columnFilters, pagination, setSearchParams, searchParams]);

  const [pageSize, setPageSize] = useState<number>(50);

  const offset = useMemo(() => {
    if (!pagination) {
      return;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  const workflow = useMemo<string | undefined>(() => {
    if (workflowId) {
      return workflowId;
    }

    const filter = columnFilters.find((filter) => filter.id === 'Workflow');

    if (!filter) {
      return;
    }

    const vals = filter?.value as Array<string>;
    return vals[0];
  }, [columnFilters, workflowId]);

  const statuses = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'status');

    if (!filter) {
      return;
    }

    return filter?.value as Array<ScheduledRunStatus>;
  }, [columnFilters]);

  const AdditionalMetadataFilter = useMemo(() => {
    const filter = columnFilters.find((filter) => filter.id === 'Metadata');

    if (!filter) {
      return;
    }

    return filter?.value as Array<string>;
  }, [columnFilters]);

  const orderByDirection = useMemo(():
    | WorkflowRunOrderByDirection
    | undefined => {
    if (!sorting.length) {
      return;
    }

    return sorting[0]?.desc
      ? WorkflowRunOrderByDirection.DESC
      : WorkflowRunOrderByDirection.ASC;
  }, [sorting]);

  const orderByField = useMemo(():
    | ScheduledWorkflowsOrderByField
    | undefined => {
    if (!sorting.length) {
      return;
    }

    switch (sorting[0]?.id) {
      case 'createdAt':
        return ScheduledWorkflowsOrderByField.CreatedAt;
      case 'triggerAt':
      default:
        return ScheduledWorkflowsOrderByField.TriggerAt;
    }
  }, [sorting]);

  const listWorkflowRunsQuery = useQuery({
    ...queries.scheduledRuns.list(tenantId, {
      offset,
      limit: pageSize,
      statuses,
      workflowId: workflow,
      parentWorkflowRunId,
      parentStepRunId,
      orderByDirection,
      orderByField,
      additionalMetadata: AdditionalMetadataFilter,
    }),
    placeholderData: (prev) => prev,
    refetchInterval,
  });

  const {
    data: workflowKeys,
    isLoading: workflowKeysIsLoading,
    error: workflowKeysError,
  } = useQuery({
    ...queries.workflows.list(tenantId, { limit: 200 }),
    refetchInterval,
  });

  const workflowKeyFilters = useMemo((): FilterOption[] => {
    return (
      workflowKeys?.rows?.map((key) => ({
        value: key.metadata.id,
        label: key.name,
      })) || []
    );
  }, [workflowKeys]);

  const workflowRunStatusFilters = useMemo((): FilterOption[] => {
    return [
      {
        value: ScheduledRunStatus.SCHEDULED,
        label: 'Scheduled',
      },
      {
        value: ScheduledRunStatus.SUCCEEDED,
        label: 'Succeeded',
      },
      {
        value: ScheduledRunStatus.FAILED,
        label: 'Failed',
      },
      {
        value: ScheduledRunStatus.RUNNING,
        label: 'Running',
      },
      {
        value: ScheduledRunStatus.QUEUED,
        label: 'Queued',
      },
      {
        value: ScheduledRunStatus.PENDING,
        label: 'Pending',
      },
    ];
  }, []);

  const filters: ToolbarFilters = [
    {
      columnId: 'Workflow',
      title: 'Workflow',
      options: workflowKeyFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: 'status',
      title: 'Status',
      options: workflowRunStatusFilters,
      type: ToolbarType.Checkbox,
    },
    {
      columnId: 'Metadata',
      title: 'Metadata',
      type: ToolbarType.KeyValue,
    },
  ].filter((filter) => filterVisibility[filter.columnId] != false);

  const actions = [
    <Button
      key="schedule-run"
      onClick={() => setTriggerWorkflow(true)}
      className="h-8 border px-3"
    >
      Schedule Run
    </Button>,
  ];

  const [showScheduledRunRevoke, setShowScheduledRunRevoke] = useState<
    ScheduledWorkflows | undefined
  >(undefined);

  const isLoading = listWorkflowRunsQuery.isFetching || workflowKeysIsLoading;

  return (
    <>
      <DeleteScheduledRun
        scheduledRun={showScheduledRunRevoke}
        setShowScheduledRunRevoke={setShowScheduledRunRevoke}
        onSuccess={() => {
          listWorkflowRunsQuery.refetch();
          setShowScheduledRunRevoke(undefined);
        }}
      />
      <TriggerWorkflowForm
        defaultTimingOption="schedule"
        defaultWorkflow={undefined}
        show={triggerWorkflow}
        onClose={() => setTriggerWorkflow(false)}
      />

      <DataTable
        emptyState={
          <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
            <p className="text-lg font-semibold">No runs found</p>
            <div className="w-fit">
              <DocsButton
                doc={docsPages.home['scheduled-runs']}
                size="full"
                variant="outline"
                label="Learn about scheduled runs"
              />
            </div>
          </div>
        }
        error={workflowKeysError}
        isLoading={isLoading}
        columns={columns({
          tenantId,
          onDeleteClick: (row) => {
            setShowScheduledRunRevoke(row);
          },
          selectedAdditionalMetaJobId,
          handleSetSelectedAdditionalMetaJobId: setSelectedAdditionalMetaJobId,
        })}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        data={listWorkflowRunsQuery.data?.rows || []}
        filters={filters}
        rightActions={actions}
        sorting={sorting}
        setSorting={setSorting}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={listWorkflowRunsQuery.data?.pagination?.num_pages || 0}
        showColumnToggle={true}
        refetchProps={{
          isRefetching: listWorkflowRunsQuery.isRefetching,
          onRefetch: listWorkflowRunsQuery.refetch,
        }}
        showSelectedRows={false}
      />
    </>
  );
}
