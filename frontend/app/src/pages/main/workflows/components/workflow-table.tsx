import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Workflow, queries } from '@/lib/api';
import invariant from 'tiny-invariant';
import { TenantContextType } from '@/lib/outlet';
import { Link, useOutletContext } from 'react-router-dom';
import { DataTable } from '@/components/molecules/data-table/data-table.tsx';
import { columns } from './workflow-columns';
import { Loading } from '@/components/ui/loading.tsx';
import { Button } from '@/components/ui/button';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/ui/card';
import { cn } from '@/lib/utils';
import {
  ArrowPathIcon,
  QuestionMarkCircleIcon,
} from '@heroicons/react/24/outline';
import { SortingState, VisibilityState } from '@tanstack/react-table';
import { BiCard, BiTable } from 'react-icons/bi';
import RelativeDate from '@/components/molecules/relative-date';

export function WorkflowTable() {
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

  const listWorkflowQuery = useQuery({
    ...queries.workflows.list(tenant.metadata.id),
    refetchInterval: 5000,
  });

  const data = useMemo(() => {
    const data = listWorkflowQuery.data?.rows || [];

    return data;
  }, [listWorkflowQuery.data?.rows]);

  if (listWorkflowQuery.isLoading) {
    return <Loading />;
  }

  const emptyState = (
    <Card className="w-full text-justify">
      <CardHeader>
        <CardTitle>No Registered Workflows</CardTitle>
        <CardDescription>
          <p className="text-gray-700 dark:text-gray-300 mb-4">
            There are no workflows registered in this tenant. To enable workflow
            execution, please register a workflow with a worker or{' '}
            <a href="support@hatchet.run">contact support</a>.
          </p>
        </CardDescription>
      </CardHeader>
      <CardFooter>
        <a
          href="https://docs.hatchet.run/home/basics/workflows"
          className="flex flex-row item-center"
        >
          <Button onClick={() => {}} variant="link" className="p-0 w-fit">
            <QuestionMarkCircleIcon className={cn('h-4 w-4 mr-2')} />
            Docs: Understanding Workflows in Hatchet
          </Button>
        </a>
      </CardFooter>
    </Card>
  );

  const card: React.FC<{ data: Workflow }> = ({ data }) => (
    <div
      key={data.metadata?.id}
      className="border overflow-hidden shadow rounded-lg"
    >
      <div className="px-4 py-5 sm:p-6">
        <h3 className="text-lg leading-6 font-medium text-foreground">
          <Link to={`/workflows/${data.metadata?.id}`}>{data.name}</Link>
        </h3>
        <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
          Last run{' '}
          {data.lastRun?.metadata?.createdAt ? (
            <RelativeDate date={data.lastRun?.metadata?.createdAt} />
          ) : (
            'never'
          )}
          <br />
          Created at <RelativeDate date={data.metadata?.createdAt} />
        </p>
      </div>
      <div className="px-4 py-4 sm:px-6">
        <div className="text-sm text-background-secondary">
          <Link to={`/workflows/${data.metadata?.id}`}>
            <Button>View Workflow</Button>
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
        listWorkflowQuery.refetch();
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
