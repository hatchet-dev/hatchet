import { Separator } from '@/components/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { queries } from '@/lib/api';

export default function Webhooks() {
  const { tenant } = useOutletContext<TenantContextType>();

  const listWebhookWorkersQuery = useQuery({
    ...queries.webhookWorkers.list(tenant.metadata.id),
  });

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-semibold leading-tight text-foreground">
            Webhooks
          </h2>
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Assign webhook workers to workflows.
        </p>
        <Separator className="my-4" />

        <div className="grid gap-2 grid-cols-1 sm:grid-cols-2">
          <div className="">
            {listWebhookWorkersQuery.isLoading && 'Loading...'}
          </div>
          <div className="">{listWebhookWorkersQuery.isError && 'Error'}</div>

          {listWebhookWorkersQuery.data?.rows?.map((worker) => (
            <div key={worker.metadata!.id}>
              <div className="flex flex-row justify-between items-center">
                <div className="text-sm">{worker.url}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
