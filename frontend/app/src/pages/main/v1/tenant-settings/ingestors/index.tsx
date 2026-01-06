import { CreateSNSDialog } from './components/create-sns-dialog';
import { DeleteSNSForm } from './components/delete-sns-form';
import {
  CopyIngestURL,
  SNSActions,
} from './components/sns-integrations-columns';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import api, {
  CreateSNSIntegrationRequest,
  SNSIntegration,
  queries,
} from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { Dialog } from '@radix-ui/react-dialog';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState, useMemo } from 'react';

export default function Ingestors() {
  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-semibold leading-tight text-foreground">
          Ingestors
        </h2>
        <p className="my-4 text-gray-700 dark:text-gray-300">
          Ingestors are integrations that allow you to send events to Hatchet.
        </p>
        <Separator className="my-4" />
        <SNSIntegrationsList />
      </div>
    </div>
  );
}

function SNSIntegrationsList() {
  const { tenantId } = useCurrentTenantId();
  const [showSNSDialog, setShowSNSDialog] = useState(false);
  const [deleteSNS, setDeleteSNS] = useState<SNSIntegration | null>(null);

  const listIntegrationsQuery = useQuery({
    ...queries.snsIntegrations.list(tenantId),
  });

  const snsColumns = useMemo(
    () => [
      {
        columnLabel: 'Topic ARN',
        cellRenderer: (integration: SNSIntegration) => (
          <div>{integration.topicArn}</div>
        ),
      },
      {
        columnLabel: 'Ingest URL',
        cellRenderer: (integration: SNSIntegration) => (
          <CopyIngestURL ingestUrl={integration.ingestUrl || ''} />
        ),
      },
      {
        columnLabel: 'Created',
        cellRenderer: (integration: SNSIntegration) => (
          <RelativeDate date={integration.metadata.createdAt} />
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (integration: SNSIntegration) => (
          <SNSActions
            integration={integration}
            onDeleteClick={(integration) => {
              setDeleteSNS(integration);
            }}
          />
        ),
      },
    ],
    [],
  );

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          SNS Integrations
        </h3>
        <Button key="create-api-token" onClick={() => setShowSNSDialog(true)}>
          Create SNS Endpoint
        </Button>
      </div>
      <Separator className="my-4" />
      {(listIntegrationsQuery.data?.rows || []).length > 0 ? (
        <SimpleTable
          columns={snsColumns}
          data={listIntegrationsQuery.data?.rows || []}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No SNS integrations found. Create an endpoint to receive events from
          AWS SNS.
        </div>
      )}
      {showSNSDialog && (
        <CreateSNSIntegration
          showSNSDialog={showSNSDialog}
          setShowSNSDialog={setShowSNSDialog}
          onSuccess={() => {
            listIntegrationsQuery.refetch();
          }}
        />
      )}
      {deleteSNS && (
        <DeleteSNSIntegration
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
  showSNSDialog,
  setShowSNSDialog,
  onSuccess,
}: {
  onSuccess: () => void;
  showSNSDialog: boolean;
  setShowSNSDialog: (show: boolean) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const [generatedIngestUrl, setGeneratedIngestUrl] = useState<
    string | undefined
  >();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createSNSIntegrationMutation = useMutation({
    mutationKey: ['sns:create', tenantId],
    mutationFn: async (data: CreateSNSIntegrationRequest) => {
      const res = await api.snsCreate(tenantId, data);
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
  snsIntegration,
  setShowSNSDelete,
  onSuccess,
}: {
  snsIntegration: SNSIntegration;
  setShowSNSDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['sns:delete', tenantId, snsIntegration],
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
