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
import { cn, relativeDate } from '@/lib/utils';
import { QuestionMarkCircleIcon } from '@heroicons/react/24/outline';
import { SortingState, VisibilityState } from '@tanstack/react-table';

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

  const listWorkflowQuery = useQuery({
    ...queries.workflows.list(tenant.metadata.id),
    refetchInterval: 5000,
  });

  const data = useMemo(() => {
    console.log(sorting);

    const data = listWorkflowQuery.data?.rows || [];

    return data;
  }, [listWorkflowQuery.data?.rows, sorting]);

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
          {data.name}
        </h3>
        <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
          Last run {relativeDate(data.lastRun?.metadata?.createdAt)} <br />
          Created at {relativeDate(data.metadata?.createdAt)}
          <br />
          Updated at {relativeDate(data.metadata?.updatedAt)}
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
      // card={{
      //   component: card,
      // }}
    />
  );
}
