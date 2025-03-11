import { Separator } from '@/components/v1/ui/separator';
import { useState } from 'react';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import api, {
  CreateSNSIntegrationRequest,
  SNSIntegration,
  queries,
} from '@/lib/api';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { Button } from '@/components/v1/ui/button';
import { TenantContextType } from '@/lib/outlet';
import { Dialog } from '@radix-ui/react-dialog';
import { useOutletContext } from 'react-router-dom';
import { CreateSNSDialog } from './components/create-sns-dialog';
import { DeleteSNSForm } from './components/delete-sns-form';
import { columns as snsIntegrationsColumns } from './components/sns-integrations-columns';

export default function Ingestors() {
  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-semibold leading-tight text-foreground">
          Ingestors
        </h2>
        <p className="text-gray-700 dark:text-gray-300 my-4">
          Ingestors are integrations that allow you to send events to Hatchet.
        </p>
        <Separator className="my-4" />
        <SNSIntegrationsList />
      </div>
    </div>
  );
}

function SNSIntegrationsList() {
  const { tenant } = useOutletContext<TenantContextType>();
  const [showSNSDialog, setShowSNSDialog] = useState(false);
  const [deleteSNS, setDeleteSNS] = useState<SNSIntegration | null>(null);

  const listIntegrationsQuery = useQuery({
    ...queries.snsIntegrations.list(tenant.metadata.id),
  });

  const cols = snsIntegrationsColumns({
    onDeleteClick: (row) => {
      setDeleteSNS(row);
    },
  });

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          SNS Integrations
        </h3>
        <Button key="create-api-token" onClick={() => setShowSNSDialog(true)}>
          Create SNS Endpoint
        </Button>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={listIntegrationsQuery.isLoading}
        columns={cols}
        data={listIntegrationsQuery.data?.rows || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      {showSNSDialog && (
        <CreateSNSIntegration
          tenant={tenant.metadata.id}
          showSNSDialog={showSNSDialog}
          setShowSNSDialog={setShowSNSDialog}
          onSuccess={() => {
            listIntegrationsQuery.refetch();
          }}
        />
      )}
      {deleteSNS && (
        <DeleteSNSIntegration
          tenant={tenant.metadata.id}
          snsIntegration={deleteSNS}
          setShowSNSDelete={() => setDeleteSNS(null)}
          onSuccess={() => {
            setDeleteSNS(null);
            listIntegrationsQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function CreateSNSIntegration({
  tenant,
  showSNSDialog,
  setShowSNSDialog,
  onSuccess,
}: {
  tenant: string;
  onSuccess: () => void;
  showSNSDialog: boolean;
  setShowSNSDialog: (show: boolean) => void;
}) {
  const [generatedIngestUrl, setGeneratedIngestUrl] = useState<
    string | undefined
  >();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createSNSIntegrationMutation = useMutation({
    mutationKey: ['sns:create', tenant],
    mutationFn: async (data: CreateSNSIntegrationRequest) => {
      const res = await api.snsCreate(tenant, data);
      return res.data;
    },
    onSuccess: (data) => {
      setGeneratedIngestUrl(data.ingestUrl);
      onSuccess();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={showSNSDialog} onOpenChange={setShowSNSDialog}>
      <CreateSNSDialog
        isLoading={createSNSIntegrationMutation.isPending}
        onSubmit={createSNSIntegrationMutation.mutate}
        snsIngestionUrl={generatedIngestUrl}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function DeleteSNSIntegration({
  tenant,
  snsIntegration,
  setShowSNSDelete,
  onSuccess,
}: {
  tenant: string;
  snsIntegration: SNSIntegration;
  setShowSNSDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['sns:delete', tenant, snsIntegration],
    mutationFn: async () => {
      await api.snsDelete(snsIntegration.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!snsIntegration} onOpenChange={setShowSNSDelete}>
      <DeleteSNSForm
        snsIntegration={snsIntegration}
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowSNSDelete(false)}
      />
    </Dialog>
  );
}
