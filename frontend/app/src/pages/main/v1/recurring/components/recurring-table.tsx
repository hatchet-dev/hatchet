import { useState } from 'react';
import { VisibilityState } from '@tanstack/react-table';
import { CronWorkflows } from '@/lib/api';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { columns } from './recurring-columns';
import { Button } from '@/components/v1/ui/button';
import { DeleteCron } from './delete-cron';
import {
  ToolbarFilters,
  ToolbarType,
} from '@/components/v1/molecules/data-table/data-table-toolbar';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { docsPages } from '@/lib/generated/docs';
import { useCrons } from '../hooks/use-crons';
import { CronColumn, workflowKey, metadataKey } from './recurring-columns';

export function CronsTable() {
  const { tenantId } = useCurrentTenantId();
  const [triggerWorkflow, setTriggerWorkflow] = useState(false);
  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  const {
    crons,
    numPages,
    isLoading,
    refetch,
    error,
    pagination,
    setPagination,
    setPageSize,
    columnFilters,
    setColumnFilters,
    workflowKeyFilters,
    isRefetching,
    resetFilters,
  } = useCrons({
    key: 'table',
  });

  const [showDeleteCron, setShowDeleteCron] = useState<
    CronWorkflows | undefined
  >();

  const handleDeleteClick = (cron: CronWorkflows) => {
    setShowDeleteCron(cron);
  };

  const handleConfirmDelete = () => {
    if (showDeleteCron) {
      setShowDeleteCron(undefined);
      refetch();
    }
  };

  const filters: ToolbarFilters = [
    {
      columnId: workflowKey,
      title: CronColumn.workflow,
      options: workflowKeyFilters,
      type: ToolbarType.Radio,
    },
    {
      columnId: metadataKey,
      title: CronColumn.metadata,
      type: ToolbarType.KeyValue,
    },
  ];

  const actions = [
    <Button
      key="create-cron"
      onClick={() => setTriggerWorkflow(true)}
      className="h-8 border px-3"
    >
      Create Cron Job
    </Button>,
  ];

  return (
    <>
      {showDeleteCron && (
        <DeleteCron
          cron={showDeleteCron}
          setShowCronRevoke={setShowDeleteCron}
          onSuccess={handleConfirmDelete}
        />
      )}
      <TriggerWorkflowForm
        defaultTimingOption="cron"
        defaultWorkflow={undefined}
        show={triggerWorkflow}
        onClose={() => setTriggerWorkflow(false)}
      />

      <DataTable
        error={error}
        isLoading={isLoading}
        columns={columns({
          tenantId,
          onDeleteClick: handleDeleteClick,
          selectedJobId,
          setSelectedJobId,
        })}
        data={crons}
        filters={filters}
        showColumnToggle={true}
        columnVisibility={columnVisibility}
        setColumnVisibility={setColumnVisibility}
        columnFilters={columnFilters}
        setColumnFilters={setColumnFilters}
        pagination={pagination}
        setPagination={setPagination}
        onSetPageSize={setPageSize}
        pageCount={numPages}
        rightActions={actions}
        getRowId={(row) => row.metadata.id}
        columnKeyToName={CronColumn}
        refetchProps={{
          isRefetching,
          onRefetch: refetch,
        }}
        onResetFilters={resetFilters}
        showSelectedRows={false}
        emptyState={
          <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8 justify-center items-center">
            <p className="text-lg font-semibold">No crons found</p>
            <div className="w-fit">
              <DocsButton
                doc={docsPages.home['cron-runs']}
                size="full"
                variant="outline"
                label="Learn about cron jobs in Hatchet"
              />
            </div>
          </div>
        }
      />
    </>
  );
}
