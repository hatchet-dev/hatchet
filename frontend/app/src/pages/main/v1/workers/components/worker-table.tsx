import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Worker, queries } from '@/lib/api';
import invariant from 'tiny-invariant';
import { TenantContextType } from '@/lib/outlet';
import { Link, useOutletContext } from 'react-router-dom';
import { DataTable } from '@/components/v1/molecules/data-table/data-table.tsx';
import { columns } from './worker-columns';
import { Loading } from '@/components/v1/ui/loading.tsx';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/v1/ui/card';
import { cn } from '@/lib/utils';
import {
  ArrowPathIcon,
  QuestionMarkCircleIcon,
} from '@heroicons/react/24/outline';
import { BiCard, BiTable } from 'react-icons/bi';
import { WorkerStatus, isHealthy } from '../$worker';
import { ColumnFiltersState } from '@tanstack/react-table';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { SdkInfo } from './sdk-info';

export function WorkersTable() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [rotate, setRotate] = useState(false);
  const [cardToggle, setCardToggle] = useState(true);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([
    {
      id: 'status',
      value: ['ACTIVE', 'PAUSED'],
    },
  ]);

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenant.metadata.id),
    refetchInterval: 3000,
  });

  const data = useMemo(() => {
    let rows = listWorkersQuery.data?.rows || [];

    columnFilters.map((filter) => {
      if (filter.id === 'status') {
        rows = rows.filter((row) =>
          (filter.value as any[]).includes(row.status),
        );
      }
    });

    return rows.sort(
      (a, b) =>
        new Date(b.metadata?.createdAt).getTime() -
        new Date(a.metadata?.createdAt).getTime(),
    );
  }, [listWorkersQuery.data?.rows, columnFilters]);

  if (listWorkersQuery.isLoading) {
    return <Loading />;
  }

  const emptyState = (
    <Card className="w-full text-justify">
      <CardHeader>
        <CardTitle>No Active Workers</CardTitle>
        <CardDescription>
          <p className="text-gray-700 dark:text-gray-300 mb-4">
            There are no worker processes currently running and connected to the
            Hatchet engine for this tenant. To enable workflow execution, please
            attempt to start a worker process or{' '}
            <a href="support@hatchet.run">contact support</a>.
          </p>
        </CardDescription>
      </CardHeader>
      <CardFooter>
        <a
          href="https://docs.hatchet.run/home/basics/workers"
          className="flex flex-row item-center"
        >
          <Button onClick={() => {}} variant="link" className="p-0 w-fit">
            <QuestionMarkCircleIcon className={cn('h-4 w-4 mr-2')} />
            Docs: Understanding Workers in Hatchet
          </Button>
        </a>
      </CardFooter>
    </Card>
  );

  const card: React.FC<{ data: Worker }> = ({ data }) => (
    <div
      key={data.metadata?.id}
      className="border overflow-hidden shadow rounded-lg"
    >
      <div className="px-4 py-5 sm:p-6">
        <div className="flex flex-row items-center justify-between mb-2">
          <Badge variant="secondary">{data.type}</Badge>{' '}
          <WorkerStatus status={data.status} health={isHealthy(data)} />
        </div>
        <h3 className="text-lg leading-6 font-medium text-foreground">
          <Link
            to={`/workers/${data.metadata?.id}`}
            className="flex flex-row gap-2 hover:underline"
          >
            <SdkInfo runtimeInfo={data?.runtimeInfo} iconOnly={true} />
            {data.webhookUrl || data.name}
          </Link>
        </h3>
        <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
          Started <RelativeDate date={data.metadata?.createdAt} />
          <br />
          Last seen{' '}
          {data?.lastHeartbeatAt ? (
            <RelativeDate date={data?.lastHeartbeatAt} />
          ) : (
            'N/A'
          )}
          <br />
          {(data.maxRuns ?? 0) > 0
            ? `${data.availableRuns} / ${data.maxRuns ?? 0}`
            : '100'}{' '}
          available run slots
        </p>
      </div>
      <div className="px-4 py-4 sm:px-6">
        <div className="text-sm text-background-secondary">
          <Link to={`/workers/${data.metadata?.id}`}>
            <Button>View worker</Button>
          </Link>
        </div>
      </div>
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
        listWorkersQuery.refetch();
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
      filters={[
        {
          columnId: 'status',
          title: 'Status',
          options: [
            { value: 'ACTIVE', label: 'Active' },
            { value: 'PAUSED', label: 'Paused' },
            { value: 'INACTIVE', label: 'Inactive' },
          ],
        },
      ]}
      emptyState={emptyState}
      columnFilters={columnFilters}
      setColumnFilters={setColumnFilters}
      card={
        cardToggle
          ? {
              component: card,
            }
          : undefined
      }
      actions={actions}
    />
  );
}
