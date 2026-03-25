import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries } from '@/lib/api';
import { Instance } from '@/lib/api/generated/cloud/data-contracts';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';

type InstanceWithMetadata = Instance & {
  metadata: {
    id: string;
  };
};

export function ManagedWorkerInstancesTable({
  managedWorkerId,
}: {
  managedWorkerId: string;
}) {
  const { refetchInterval } = useRefetchInterval();
  const [rotate, setRotate] = useState(false);

  const listManagedWorkerInstancesQuery = useQuery({
    ...queries.cloud.listManagedWorkerInstances(managedWorkerId),
    refetchInterval,
  });

  const data = useMemo(() => {
    const data = listManagedWorkerInstancesQuery.data?.rows || [];

    return data;
  }, [listManagedWorkerInstancesQuery.data?.rows]);

  const dataWithMetadata: InstanceWithMetadata[] = data.map((d) => ({
    ...d,
    metadata: {
      id: d.instanceId,
    },
  }));

  const instanceColumns = useMemo(
    () => [
      {
        columnLabel: 'Name',
        cellRenderer: (instance: InstanceWithMetadata) => (
          <div className="text-md min-w-fit whitespace-nowrap p-2">
            {instance.name}
          </div>
        ),
      },
      {
        columnLabel: 'State',
        cellRenderer: (instance: InstanceWithMetadata) => (
          <div className="whitespace-nowrap">{instance.state}</div>
        ),
      },
      {
        columnLabel: 'Commit',
        cellRenderer: (instance: InstanceWithMetadata) => (
          <div className="whitespace-nowrap">
            {instance.commitSha.substring(0, 7)}
          </div>
        ),
      },
    ],
    [],
  );

  if (listManagedWorkerInstancesQuery.isLoading) {
    return <Loading />;
  }

  return (
    <div>
      <div className="mb-4 flex justify-end">
        <Button
          className="h-8 px-2 lg:px-3"
          size="sm"
          onClick={() => {
            listManagedWorkerInstancesQuery.refetch();
            setRotate(!rotate);
          }}
          variant={'outline'}
          aria-label="Refresh instances list"
        >
          <ArrowPathIcon
            className={`size-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
          />
        </Button>
      </div>
      {dataWithMetadata.length > 0 ? (
        <SimpleTable columns={instanceColumns} data={dataWithMetadata} />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          There are no instances currently active for this managed worker pool.
        </div>
      )}
    </div>
  );
}
