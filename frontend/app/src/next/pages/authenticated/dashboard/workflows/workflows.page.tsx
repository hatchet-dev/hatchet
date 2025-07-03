import RelativeDate from '@/components/v1/molecules/relative-date';
import { Workflow } from '@/lib/api';
import { Badge } from '@/next/components/ui/badge';
import { Button } from '@/next/components/ui/button';
import { Skeleton } from '@/next/components/ui/skeleton';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';
import { WorkflowsProvider, useWorkflows } from '@/next/hooks/use-workflows';
import { ROUTES } from '@/next/lib/routes';
import { ArrowPathIcon } from '@heroicons/react/24/outline';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/next/components/ui/card';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/lib/docs';
import {
  HeadlineActionItem,
  HeadlineActions,
  PageTitle,
  Headline,
} from '@/next/components/ui/page-header';
import BasicLayout from '@/next/components/layouts/basic.layout';
import {
  PageSelector,
  PageSizeSelector,
  Pagination,
} from '@/next/components/ui/pagination';

const WorkflowCard: React.FC<{ data: Workflow }> = ({ data }) => {
  const { tenantId } = useCurrentTenantId();

  return (
    <div
      key={data.metadata?.id}
      className="border overflow-hidden shadow rounded-lg"
    >
      <div className="px-4 py-5 sm:p-6">
        <div className="flex flex-row justify-between items-center">
          <h3 className="text-lg leading-6 font-medium text-foreground">
            <Link to={ROUTES.workflows.detail(tenantId, data.metadata.id)}>
              {data.name}
            </Link>
          </h3>
          {data.isPaused ? (
            <Badge variant="default">Paused</Badge> // TODO: This should be `inProgress`
          ) : (
            <Badge variant="outline">Active</Badge> // TODO: This should be `successful`
          )}
        </div>
        <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
          Created at <RelativeDate date={data.metadata?.createdAt} />
        </p>
      </div>
      <div className="px-4 py-4 sm:px-6">
        <div className="text-sm text-background-secondary">
          <Link to={ROUTES.workflows.detail(tenantId, data.metadata.id)}>
            <Button>View Workflow</Button>
          </Link>
        </div>
      </div>
    </div>
  );
};

function WorkflowsContent() {
  const { data, isLoading, invalidate } = useWorkflows();
  const [rotate, setRotate] = useState(false);

  if (isLoading) {
    return (
      <div className="flex flex-1 flex-col gap-4 p-4 pt-16">
        <div className="grid auto-rows-min gap-4 md:grid-cols-3">
          {Array.from({ length: 9 }).map((_, key) => (
            <Skeleton key={key} className="h-40 rounded-md" />
          ))}
        </div>
      </div>
    );
  }

  const emptyState = (
    <Card className="w-full text-justify">
      <CardHeader>
        <CardTitle>No Tasks or Workflows Found</CardTitle>
        <CardDescription>
          <p className="text-gray-700 dark:text-gray-300 mb-4">
            There are no tasks or workflows registered in this tenant, please
            register a task or workflow with a worker.
          </p>
        </CardDescription>
      </CardHeader>
      <CardFooter className="flex flex-col gap-2">
        <DocsButton
          variant="default"
          doc={docs.home.your_first_task}
          titleOverride="declaring tasks"
        />
        <DocsButton doc={docs.home.workers} titleOverride="registering tasks" />
      </CardFooter>
    </Card>
  );

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="View and manage workload that is registered on this tenant">
          Tasks and Workflows
        </PageTitle>
        <HeadlineActions>
          <HeadlineActionItem>
            <DocsButton doc={docs.home.your_first_task} size="icon" />
          </HeadlineActionItem>
          <HeadlineActionItem>
            <Button
              key="refresh"
              className="h-8 px-2 lg:px-3"
              size="sm"
              onClick={async () => {
                await invalidate();
                setRotate(!rotate);
              }}
              variant={'outline'}
              aria-label="Refresh workflows list"
            >
              <ArrowPathIcon
                className={`h-4 w-4 transition-transform ${rotate ? 'rotate-180' : ''}`}
              />
            </Button>
          </HeadlineActionItem>
        </HeadlineActions>
      </Headline>

      {!data || data.length === 0 ? (
        emptyState
      ) : (
        <div className="flex flex-col gap-4">
          <div className="grid md:grid-cols-2 xl:grid-cols-3 gap-2">
            {data.map((workflow) => (
              <WorkflowCard key={workflow.metadata?.id} data={workflow} />
            ))}
          </div>
          <Pagination className="mt-4 justify-between flex flex-row">
            <PageSizeSelector />
            <PageSelector variant="dropdown" />
          </Pagination>
        </div>
      )}
    </BasicLayout>
  );
}

export default function WorkflowsPage() {
  return (
    <WorkflowsProvider>
      <WorkflowsContent />
    </WorkflowsProvider>
  );
}
