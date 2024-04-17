import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Worker, queries } from '@/lib/api';
import invariant from 'tiny-invariant';
import { TenantContextType } from '@/lib/outlet';
import { Link, useOutletContext } from 'react-router-dom';
import { DataTable } from '@/components/molecules/data-table/data-table.tsx';
import { columns } from './worker-columns';
import { Loading } from '@/components/ui/loading.tsx';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/ui/card';
import { cn, relativeDate } from '@/lib/utils';
import {
  ArrowPathIcon,
  QuestionMarkCircleIcon,
} from '@heroicons/react/24/outline';
import { BiCard, BiTable } from 'react-icons/bi';

export function WorkersTable() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const [rotate, setRotate] = useState(false);
  const [cardToggle, setCardToggle] = useState(true);

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenant.metadata.id),
    refetchInterval: 5000,
  });

  const data = useMemo(() => {
    return listWorkersQuery.data?.rows || [];
  }, [listWorkersQuery.data?.rows]);

  if (listWorkersQuery.isLoading) {
    return <Loading />;
  }

  const emptyState = (
    <Card className="w-full text-justify">
      <CardHeader>
        <CardTitle>No Active Workers</CardTitle>
        <CardDescription>
          <p className="text-gray-300 mb-4">
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
        <h3 className="text-lg leading-6 font-medium text-foreground">
          <Link to={`/workers/${data.metadata?.id}`}>{data.name}</Link>
        </h3>
        <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
          Started {relativeDate(data.metadata?.createdAt)}
          <br />
          Last seen {relativeDate(data?.lastHeartbeatAt)}
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
      filters={[]}
      emptyState={emptyState}
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
