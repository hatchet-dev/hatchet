import { Separator } from '@/components/ui/separator';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';
import invariant from 'tiny-invariant';
import { cn, relativeDate } from '@/lib/utils';
import { Link, useOutletContext } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import { Loading } from '@/components/ui/loading.tsx';
import { TenantContextType } from '@/lib/outlet';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardFooter,
} from '@/components/ui/card';
import { QuestionMarkCircleIcon } from '@heroicons/react/24/outline';

export default function Workers() {
  const { tenant } = useOutletContext<TenantContextType>();
  invariant(tenant);

  const listWorkersQuery = useQuery({
    ...queries.workers.list(tenant.metadata.id),
    refetchInterval: 5000,
  });

  if (listWorkersQuery.isLoading || !listWorkersQuery.data?.rows) {
    return <Loading />;
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-bold leading-tight text-foreground">
          Workers
        </h2>
        <Separator className="my-4" />
        {/* Grid of workers */}
        {listWorkersQuery.data?.rows.length === 0 && (
          <Card className="w-full">
            <CardHeader>
              <CardTitle>No Active Workers</CardTitle>
              <CardDescription>
                <p className="text-gray-300 mb-4">
                  There are no worker processes currently running and connected
                  to the Hatchet engine for this tenant. To enable workflow
                  execution, please attempt to start a worker process or{' '}
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
        )}

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {listWorkersQuery.data?.rows.map((worker) => (
            <div
              key={worker.metadata.id}
              className="border overflow-hidden shadow rounded-lg"
            >
              <div className="px-4 py-5 sm:p-6">
                <h3 className="text-lg leading-6 font-medium text-foreground">
                  {worker.name}
                </h3>
                <p className="mt-1 max-w-2xl text-sm text-gray-700 dark:text-gray-300">
                  Last seen {relativeDate(worker.lastHeartbeatAt)}
                </p>
              </div>
              <div className="px-4 py-4 sm:px-6">
                <div className="text-sm text-background-secondary">
                  <Link to={`/workers/${worker.metadata.id}`}>
                    <Button>View worker</Button>
                  </Link>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
