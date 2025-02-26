import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import invariant from 'tiny-invariant';
import { TenantContextType } from '@/lib/outlet';
import { Link, useOutletContext } from 'react-router-dom';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './managed-worker-columns';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/v1/ui/card';
import { ArrowPathIcon, CpuChipIcon } from '@heroicons/react/24/outline';
import { SortingState, VisibilityState } from '@tanstack/react-table';
import { BiCard, BiTable } from 'react-icons/bi';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { ManagedWorker } from '@/lib/api/generated/cloud/data-contracts';
import GithubButton from '../$managed-worker/components/github-button';

export function ManagedWorkersTable() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [sorting, setSorting] = useState<SortingState>([
    {
      id: 'lastRun',
      desc: true,
    },
  ]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const [rotate, setRotate] = useState(false);

  const [cardToggle, setCardToggle] = useState(true);

  const listManagedWorkersQuery = useQuery({
    ...queries.cloud.listManagedWorkers(tenant.metadata.id),
    refetchInterval: 5000,
  });

  const data = useMemo(() => {
    const data = listManagedWorkersQuery.data?.rows || [];

    return data;
  }, [listManagedWorkersQuery.data?.rows]);

  if (listManagedWorkersQuery.isLoading) {
    return <Loading />;
  }

  const emptyState = (
    <Card className="w-full text-justify">
      <CardHeader>
        <CardTitle>No Managed Workers</CardTitle>
        <CardDescription>
          <p className="text-gray-700 dark:text-gray-300 mb-4">
            There are no managed workers created in this tenant.
          </p>
        </CardDescription>
      </CardHeader>
      <CardFooter></CardFooter>
    </Card>
  );

  const card: React.FC<{ data: ManagedWorker }> = ({ data }) => {
    const totReplicas = data.runtimeConfigs?.reduce(
      (acc, curr) => acc + curr.numReplicas,
      0,
    );
    const totCpus = data.runtimeConfigs?.reduce(
      (acc, curr) => acc + curr.cpus,
      0,
    );
    const totMemory = data.runtimeConfigs?.reduce(
      (acc, curr) => acc + curr.memoryMb,
      0,
    );

    return (
      <div
        key={data.metadata?.id}
        className="border overflow-hidden shadow rounded-lg"
      >
        <div className="px-4 py-5 sm:p-6 gap-1 flex flex-col">
          <div className="flex flex-row gap-2 items-center">
            <CpuChipIcon className="h-6 w-6 text-foreground mt-1" />
            <h3 className="text-lg font-semibold leading-tight text-foreground">
              {data.name}
            </h3>
          </div>
          <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
            Created <RelativeDate date={data.metadata?.createdAt} />
          </p>
          <GithubButton buildConfig={data.buildConfig} prefix="Deploys from" />
          <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
            {totReplicas} {totReplicas == 1 ? 'instance' : 'instances'} with{' '}
            {totCpus} CPUs and {totMemory} MB memory
          </p>
        </div>
        <div className="px-4 py-4 sm:px-6">
          <div className="text-sm text-background-secondary">
            <Link to={`/managed-workers/${data.metadata?.id}`}>
              <Button>View Compute Instance</Button>
            </Link>
          </div>
        </div>
      </div>
    );
  };

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
        listManagedWorkersQuery.refetch();
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

  return (
    <DataTable
      columns={columns}
      data={data}
      pageCount={1}
      filters={[]}
      emptyState={emptyState}
      columnVisibility={columnVisibility}
      setColumnVisibility={setColumnVisibility}
      sorting={sorting}
      setSorting={setSorting}
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
