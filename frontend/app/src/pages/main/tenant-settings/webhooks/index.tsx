import { Separator } from '@/components/ui/separator';
import { TenantContextType } from '@/lib/outlet';
import { useOutletContext } from 'react-router-dom';
import { useMutation, useQuery } from '@tanstack/react-query';
import api, { queries, WebhookWorkerCreateRequest } from '@/lib/api';
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card.tsx';
import CopyToClipboard from '@/components/ui/copy-to-clipboard';
import { Button } from '@/components/ui/button.tsx';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks.ts';
import { Dialog } from '@/components/ui/dialog.tsx';
import { CreateWebhookWorkerDialog } from '@/pages/main/tenant-settings/webhooks/components/create-webhook-worker-dialog.tsx';

export default function Webhooks() {
  const { tenant } = useOutletContext<TenantContextType>();
  const [showTokenDialog, setShowTokenDialog] = useState(false);

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

          <Button
            key="create-api-token"
            onClick={() => setShowTokenDialog(true)}
          >
            Create Webhook Endpoint
          </Button>
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
                <Card>
                  <CardHeader>
                    <CardTitle>{worker.metadata.id}</CardTitle>
                    <CardDescription>
                      <div className="text-sm mt-2">{worker.url}</div>

                      <div className="flex items-center gap-2 mt-2">
                        <pre className="text-xs">
                          {worker.secret.slice(0, 4)}****
                        </pre>
                        <CopyToClipboard text={worker.secret} />
                      </div>
                    </CardDescription>
                  </CardHeader>
                </Card>
              </div>
            </div>
          ))}

          {showTokenDialog && (
            <CreateWebhookWorker
              tenant={tenant.metadata.id}
              showTokenDialog={showTokenDialog}
              setShowTokenDialog={setShowTokenDialog}
              onSuccess={() => {
                listWebhookWorkersQuery.refetch();
              }}
            />
          )}
        </div>
      </div>
    </div>
  );
}

function CreateWebhookWorker({
  tenant,
  showTokenDialog,
  setShowTokenDialog,
  onSuccess,
}: {
  tenant: string;
  onSuccess: () => void;
  showTokenDialog: boolean;
  setShowTokenDialog: (show: boolean) => void;
}) {
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const listWorkflowQuery = useQuery({
    ...queries.workflows.list(tenant),
    refetchInterval: 5000,
  });

  const createWebhookWorkerMutation = useMutation({
    mutationKey: ['webhook-worker:create', tenant],
    mutationFn: async (data: WebhookWorkerCreateRequest) => {
      const res = await api.webhookCreate(tenant, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedToken(data.secret);
      onSuccess();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={showTokenDialog} onOpenChange={setShowTokenDialog}>
      <CreateWebhookWorkerDialog
        workflows={listWorkflowQuery.data?.rows || []}
        isLoading={createWebhookWorkerMutation.isPending}
        onSubmit={createWebhookWorkerMutation.mutate}
        token={generatedToken}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}
