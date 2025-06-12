import { useState, useMemo, useCallback } from 'react';
import { ColumnFiltersState } from '@tanstack/react-table';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { WorkerType } from '@/lib/api';
import { useWorkers } from '@/next/hooks/use-workers';
import { useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';
import { createWorkerColumns, WorkerWithPoolInfo } from './worker-columns';

interface WorkerTableProps {
  poolName: string;
}

export function WorkerTable({ poolName }: WorkerTableProps) {
  const navigate = useNavigate();
  const { tenantId } = useCurrentTenantId();
  const { pools, isLoading, update } = useWorkers();

  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([
    {
      id: 'status',
      value: ['ACTIVE'],
    },
  ]);

  const pool = pools.find((worker) => worker.name === poolName);

  // Transform pool workers to include pool info
  const data: WorkerWithPoolInfo[] = useMemo(() => {
    if (!pool) {
      return [];
    }

    return pool.workers.map((worker) => ({
      ...worker,
      poolName: pool.name,
    }));
  }, [pool]);

  // Use original data, let DataTable handle filtering
  const tableData = useMemo(() => data, [data]);

  const handleWorkerAction = useCallback(
    async (workerId: string, action: 'pause' | 'resume') => {
      try {
        await update.mutateAsync({
          workerId,
          data: { isPaused: action !== 'resume' },
        });
      } catch (error) {
        // Error handling is done in the mutation
      }
    },
    [update],
  );

  const handleWorkerClick = useCallback(
    (workerId: string) => {
      navigate(
        ROUTES.workers.workerDetail(
          tenantId,
          encodeURIComponent(poolName),
          workerId,
          pool?.type || WorkerType.SELFHOSTED,
        ),
      );
    },
    [navigate, tenantId, poolName, pool?.type],
  );

  const columns = useMemo(
    () => createWorkerColumns(handleWorkerClick, handleWorkerAction),
    [handleWorkerClick, handleWorkerAction],
  );

  const filters = useMemo(
    () => [
      {
        columnId: 'status',
        title: 'Status',
        options: [
          { value: 'ACTIVE', label: `Active (${pool?.activeCount || 0})` },
          { value: 'PAUSED', label: `Paused (${pool?.pausedCount || 0})` },
          {
            value: 'INACTIVE',
            label: `Inactive (${pool?.inactiveCount || 0})`,
          },
        ],
      },
    ],
    [pool?.activeCount, pool?.pausedCount, pool?.inactiveCount],
  );

  const emptyState = useMemo(
    () =>
      pool ? (
        <div className="flex flex-col items-center justify-center gap-4 py-8">
          <p className="text-md">
            {columnFilters.some(
              (f) =>
                f.id === 'status' &&
                Array.isArray(f.value) &&
                f.value.length > 0,
            )
              ? `No ${(columnFilters.find((f) => f.id === 'status')?.value as string[])?.join(', ').toLowerCase()} workers in this pool.`
              : 'No workers in this pool.'}
          </p>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center gap-4 py-8">
          <p className="text-md">No worker pool found.</p>
        </div>
      ),
    [pool, columnFilters],
  );

  return (
    <DataTable
      columns={columns}
      data={tableData}
      pageCount={1}
      filters={filters}
      emptyState={emptyState}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      isLoading={isLoading}
      manualFiltering={false}
    />
  );
}
