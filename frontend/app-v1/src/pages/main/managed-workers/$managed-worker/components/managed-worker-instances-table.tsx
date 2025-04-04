import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import invariant from 'tiny-invariant';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './managed-worker-instances-columns';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/v1/ui/card';
import { capitalize } from '@/lib/utils';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { VisibilityState } from '@tanstack/react-table';
import { BiCard, BiTable } from 'react-icons/bi';
import { Instance } from '@/lib/api/generated/cloud/data-contracts';
import { Badge } from '@/components/v1/ui/badge';

export function ManagedWorkerInstancesTable({
  managedWorkerId,
}: {
  managedWorkerId: string;
}) {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rotate, setRotate] = useState(false);

  const [cardToggle, setCardToggle] = useState(true);

  const listManagedWorkerInstancesQuery = useQuery({
    ...queries.cloud.listManagedWorkerInstances(managedWorkerId),
    refetchInterval: 5000,
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
          <p className="text-gray-700 dark:text-gray-300 mb-4">
            There are no instances currently active for this managed worker
            pool.
          </p>
        </CardDescription>
      </CardHeader>
      <CardFooter></CardFooter>
    </Card>
  );

  const card: React.FC<{ data: Instance }> = ({ data }) => (
    <div
      key={data.instanceId}
      className="border overflow-hidden shadow rounded-lg"
    >
      <div className="p-4 sm:p-6">
        <div className="flex items-center justify-between">
          <h3 className="text-lg leading-6 font-medium text-foreground">
            {data.name}
          </h3>
          <StateBadge state={data.state} />
        </div>
        <div className="mt-2 flex items-center text-sm text-background-secondary">
          CPUs: {data.cpus} {data.cpuKind}
        </div>
        <div className="mt-2 flex items-center text-sm text-background-secondary">
          Memory: {data.memoryMb} MB
        </div>
      </div>
      <div className="px-4 py-4 sm:px-6"></div>
    </div>
  );

  const actions = [
    <Button
      key="card-toggle"
      className="h-8 px-2 lg:px-3"
      size="sm"
      onClick={() => {
        setCardToggle((t) => !t);
      }}
      variant={'outline'}
      aria-label="Toggle card/table view"
    >
      {!cardToggle ? (
        <BiCard className={`h-4 w-4 `} />
      ) : (
        <BiTable className={`h-4 w-4 `} />
      )}
    </Button>,
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
        className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
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
      filters={[]}
      emptyState={emptyState}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      manualSorting={false}
      actions={actions}
      manualFiltering={false}
      card={
        cardToggle
          ? {
              component: card,
            }
          : undefined
      }
    />
  );
}

const INSTANCE_STATUSES: Record<
  string,
  {
    text: string;
    variant: 'successful' | 'failed' | 'outline';
  }
> = {
  started: {
    text: 'Running',
    variant: 'successful',
  },
  suspended: {
    text: 'Suspended',
    variant: 'failed',
  },
  destroyed: {
    text: 'Destroyed',
    variant: 'failed',
  },
  stopped: {
    text: 'Stopped',
    variant: 'outline',
  },
};

const StateBadge = ({ state }: { state: string }) => {
  let instanceStatus = INSTANCE_STATUSES[state];

  if (!instanceStatus) {
    instanceStatus = {
      text: capitalize(state),
      variant: 'outline',
    };
  }

  return (
    <Badge variant={instanceStatus.variant}>
      {capitalize(instanceStatus.text)}
    </Badge>
  );
};
