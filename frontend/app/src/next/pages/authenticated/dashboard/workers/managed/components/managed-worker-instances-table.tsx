import { useManagedComputeDetail } from '@/next/hooks/use-managed-compute-detail';
import { DataTable } from '@/next/components/ui/data-table';
import { Spinner } from '@/next/components/ui/spinner';
import { Button } from '@/next/components/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/next/components/ui/card';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/next/components/ui/badge';
import { Instance } from '@/lib/api/generated/cloud/data-contracts';
import { useState } from 'react';
import { capitalize } from '@/next/lib/utils';

const columns: ColumnDef<Instance>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
  },
  {
    accessorKey: 'state',
    header: 'State',
    cell: ({ row }) => <StateBadge state={row.getValue('state')} />,
  },
  {
    accessorKey: 'cpus',
    header: 'CPUs',
    cell: ({ row }) => `${row.getValue('cpus')} ${row.original.cpuKind}`,
  },
  {
    accessorKey: 'memoryMb',
    header: 'Memory',
    cell: ({ row }) => `${row.getValue('memoryMb')} MB`,
  },
];

export function ManagedWorkerInstancesTable() {
  const { instances } = useManagedComputeDetail();
  const [rotate, setRotate] = useState(false);

  if (instances?.isLoading) {
    return <Spinner />;
  }

  const data = instances?.data?.rows || [];

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

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button
          className="h-8 px-2 lg:px-3"
          size="sm"
          onClick={async () => {
            await instances?.refetch();
            setRotate(!rotate);
          }}
          variant="outline"
          aria-label="Refresh instances list"
        >
          <ArrowPathIcon
            className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
          />
        </Button>
      </div>
      <DataTable columns={columns} data={data} emptyState={emptyState} />
    </div>
  );
}

type InstanceStatus = {
  text: string;
  variant: 'successful' | 'secondary' | 'destructive' | 'outline' | 'default';
};

type InstanceStatusMap = {
  started: InstanceStatus;
  suspended: InstanceStatus;
  destroyed: InstanceStatus;
  stopped: InstanceStatus;
};

const INSTANCE_STATUSES: InstanceStatusMap = {
  started: {
    text: 'Running',
    variant: 'successful',
  },
  suspended: {
    text: 'Suspended',
    variant: 'destructive',
  },
  destroyed: {
    text: 'Destroyed',
    variant: 'destructive',
  },
  stopped: {
    text: 'Stopped',
    variant: 'outline',
  },
};

const StateBadge = ({ state }: { state: string }) => {
  let instanceStatus = INSTANCE_STATUSES[state as keyof InstanceStatusMap];

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
