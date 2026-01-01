import { columns } from './managed-worker-instances-columns';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/v1/ui/card';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries } from '@/lib/api';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { VisibilityState } from '@tanstack/react-table';
import { useMemo, useState } from 'react';

export function ManagedWorkerInstancesTable({
  managedWorkerId,
}: {
  managedWorkerId: string;
}) {
  const { refetchInterval } = useRefetchInterval();
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rotate, setRotate] = useState(false);

  const listManagedWorkerInstancesQuery = useQuery({
    ...queries.cloud.listManagedWorkerInstances(managedWorkerId),
    refetchInterval,
  });

  const data = useMemo(() => {
    const data = listManagedWorkerInstancesQuery.data?.rows || [];

    return data;
  }, [listManagedWorkerInstancesQuery.data?.rows]);

  if (listManagedWorkerInstancesQuery.isLoading) {
    return <Loading />;
  }

  const emptyState = (
    <Card className="w-full text-justify">
      <CardHeader>
        <CardTitle>No Instances</CardTitle>
        <CardDescription>
          <p className="mb-4 text-gray-700 dark:text-gray-300">
            There are no instances currently active for this managed worker
            pool.
          </p>
        </CardDescription>
      </CardHeader>
      <CardFooter></CardFooter>
    </Card>
  );

  const actions = [
    <Button
      key="refresh"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        listManagedWorkerInstancesQuery.refetch();
        setRotate(!rotate);
      }}
      variant={'outline'}
      aria-label="Refresh events list"
    >
      <ArrowPathIcon
        className={`size-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
      />
    </Button>,
  ];

  const dataWithMetadata = data.map((d) => ({
    ...d,
    metadata: {
      id: d.instanceId,
    },
  }));

  return (
    <DataTable
      columns={columns}
      data={dataWithMetadata}
      pageCount={1}
      emptyState={emptyState}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      manualSorting={false}
      rightActions={actions}
      manualFiltering={false}
    />
  );
}
