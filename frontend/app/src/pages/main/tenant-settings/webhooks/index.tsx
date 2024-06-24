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
import { Button } from '@/components/ui/button.tsx';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks.ts';
import { Dialog } from '@/components/ui/dialog.tsx';
import { CreateWebhookWorkerDialog } from '@/pages/main/tenant-settings/webhooks/components/create-webhook-worker-dialog.tsx';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu.tsx';
import { BiDotsVertical } from 'react-icons/bi';
import { DeleteWebhookWorkerDialog } from '@/pages/main/tenant-settings/webhooks/components/delete-webhook-worker-dialog.tsx';

export default function Webhooks() {
  const { tenant } = useOutletContext<TenantContextType>();
  const [showCreateTokenDialog, setShowCreateTokenDialog] = useState(false);
  const [showDeleteTokenDialog, setShowDeleteTokenDialog] = useState('');

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
            key="create-webhook-worker"
            onClick={() => setShowCreateTokenDialog(true)}
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
                    <CardTitle className="flex items-center justify-between">
                      {worker.name}
                      <DropdownMenu>
                        <DropdownMenuTrigger>
                          <Button
                            aria-label="Workflow Actions"
                            size="icon"
                            variant="ghost"
                          >
                            <BiDotsVertical />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent>
                          <DropdownMenuItem
                            onClick={() => {
                              setShowDeleteTokenDialog(worker.metadata.id);
                            }}
                          >
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </CardTitle>
                    <CardDescription>
                      <div className="text-sm mt-2 font-mono">
                        {worker.metadata.id}
                      </div>
                      <div className="text-sm mt-2">{worker.url}</div>
                    </CardDescription>
                  </CardHeader>
                </Card>
              </div>
            </div>
          ))}

          <CreateWebhookWorker
            tenant={tenant.metadata.id}
            showDialog={showCreateTokenDialog}
            setShowDialog={setShowCreateTokenDialog}
            onSuccess={() => {
              void listWebhookWorkersQuery.refetch();
            }}
          />

          <DeleteWebhookWorker
            showDialog={showDeleteTokenDialog}
            setShowDialog={setShowDeleteTokenDialog}
            onSuccess={() => {
              void listWebhookWorkersQuery.refetch();
            }}
          />
        </div>
      </div>
    </div>
  );
}

function DeleteWebhookWorker({
  showDialog,
  setShowDialog,
  onSuccess,
}: {
  onSuccess: () => void;
  showDialog: string;
  setShowDialog: (show: string) => void;
}) {
  const { handleApiError } = useApiError({});

  const deleteWebhookWorkerMutation = useMutation({
    mutationKey: ['webhook-worker:delete', showDialog],
    mutationFn: async (id: string) => {
      const res = await api.webhookDelete(id);
      return res.data;
    },
    onSuccess: () => {
      onSuccess();
    },
    onError: handleApiError,
  });

  return (
    <Dialog
      open={!!showDialog}
      onOpenChange={(open) => {
        if (!open) {
          setShowDialog('');
        }
      }}
    >
      <DeleteWebhookWorkerDialog
        isLoading={deleteWebhookWorkerMutation.isPending}
        onSubmit={() => {
          deleteWebhookWorkerMutation.mutate(showDialog);
          setShowDialog('');
        }}
      />
    </Dialog>
  );
}

function CreateWebhookWorker({
  tenant,
  showDialog,
  setShowDialog,
  onSuccess,
}: {
  tenant: string;
  onSuccess: () => void;
  showDialog: boolean;
  setShowDialog: (show: boolean) => void;
}) {
  const [generatedToken, setGeneratedToken] = useState<string | undefined>();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
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
    <Dialog open={showDialog} onOpenChange={setShowDialog}>
      <CreateWebhookWorkerDialog
        isLoading={createWebhookWorkerMutation.isPending}
        onSubmit={createWebhookWorkerMutation.mutate}
        secret={generatedToken}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}
