import GithubButton from '../$managed-worker/components/github-button';
import RelativeDate from '@/components/v1/molecules/relative-date';
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
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { ManagedWorker } from '@/lib/api/generated/cloud/data-contracts';
import { appRoutes } from '@/router';
import { ArrowPathIcon, CpuChipIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { useMemo, useState } from 'react';

const ManagedWorkerCard: React.FC<{ data: ManagedWorker }> = ({ data }) => {
  const { tenantId } = useCurrentTenantId();

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
      className="overflow-hidden rounded-lg border shadow"
    >
      <div className="flex flex-col gap-1 px-4 py-5 sm:p-6">
        <div className="flex flex-row items-center gap-2">
          <CpuChipIcon className="mt-1 h-6 w-6 text-foreground" />
          <h3 className="text-lg font-semibold leading-tight text-foreground">
            {data.name}
          </h3>
        </div>
        <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
          Created <RelativeDate date={data.metadata?.createdAt} />
        </p>
        {data.buildConfig && (
          <GithubButton buildConfig={data.buildConfig} prefix="Deploys from" />
        )}
        <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
          {totReplicas} {totReplicas == 1 ? 'instance' : 'instances'} with{' '}
          {totCpus} CPUs and {totMemory} MB memory
        </p>
      </div>
      <div className="px-4 py-4 sm:px-6">
        <div className="text-background-secondary text-sm">
          <Link
            to={appRoutes.tenantManagedWorkerRoute.to}
            params={{ tenant: tenantId, managedWorker: data.metadata?.id }}
          >
            <Button>View Compute Instance</Button>
          </Link>
        </div>
      </div>
    </div>
  );
};

export function ManagedWorkersTable() {
  const { tenantId } = useCurrentTenantId();
  const { refetchInterval } = useRefetchInterval();

  const [rotate, setRotate] = useState(false);

  const listManagedWorkersQuery = useQuery({
    ...queries.cloud.listManagedWorkers(tenantId),
    refetchInterval,
  });

  const data = useMemo(() => {
    const data = listManagedWorkersQuery.data?.rows || [];

    return data.sort((a, b) =>
      a.metadata?.created_at < b.metadata?.created_at ? 1 : -1,
    );
  }, [listManagedWorkersQuery.data?.rows]);

  if (listManagedWorkersQuery.isLoading) {
    return <Loading />;
  }

  return (
    <div className="flex flex-col">
      <div className="flex flex-row justify-end mb-4">
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
            className={`size-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
          />
        </Button>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {data.map((w) => (
          <ManagedWorkerCard key={w.metadata.id} data={w} />
        ))}
        {data.length === 0 && (
          <Card className="w-full text-justify">
            <CardHeader>
              <CardTitle>No Managed Services</CardTitle>
              <CardDescription>
                <p className="mb-4 text-gray-700 dark:text-gray-300">
                  There are no managed services created in this tenant.
                </p>
              </CardDescription>
            </CardHeader>
            <CardFooter></CardFooter>
          </Card>
        )}
      </div>
    </div>
  );
}
