import { Separator } from '@/components/v1/ui/separator';
import { useState } from 'react';
import { SNSIntegration } from '@/lib/api';
import { DataTable } from '@/components/v1/molecules/data-table/data-table';
import { Button } from '@/components/v1/ui/button';
import { Dialog } from '@radix-ui/react-dialog';
import { CreateSNSDialog } from './components/create-sns-dialog';
import { columns as snsIntegrationsColumns } from './components/sns-integrations-columns';
import { PageTitle } from '@/next/components/ui/page-header';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Headline } from '@/next/components/ui/page-header';
import { useIngestors } from '@/next/hooks/use-ingestors';
import { DestructiveDialog } from '@/next/components/ui/dialog/index';
import { IngestorsProvider } from '@/next/hooks/use-ingestors';
export default function Ingestors() {
  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Ingestors are integrations that allow you to send events to Hatchet.">
          Ingestors
        </PageTitle>
      </Headline>
      <Separator className="my-4" />
      <IngestorsProvider>
        <SNSIntegrationsList />
      </IngestorsProvider>
    </BasicLayout>
  );
}

function SNSIntegrationsList() {
  const {
    sns: { list, create, remove },
  } = useIngestors();

  const [showSNSDialog, setShowSNSDialog] = useState(false);
  const [deleteSNS, setDeleteSNS] = useState<SNSIntegration | null>(null);

  const cols = snsIntegrationsColumns({
    onDeleteClick: (row) => {
      setDeleteSNS(row);
    },
  });

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground flex items-center gap-2">
          AWS SNS Integrations
        </h3>
        <Button key="create-api-token" onClick={() => setShowSNSDialog(true)}>
          Create SNS Endpoint
        </Button>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={list.isLoading}
        columns={cols}
        data={list.data?.rows || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      {showSNSDialog && (
        <Dialog open={showSNSDialog} onOpenChange={setShowSNSDialog}>
          <CreateSNSDialog
            isLoading={create.isPending}
            closeDialog={() => setShowSNSDialog(false)}
          />
        </Dialog>
      )}
      {deleteSNS && (
        <DestructiveDialog
          open={true}
          onOpenChange={() => {}}
          title="Delete Slack Webhook"
          description="Are you sure you want to delete this Slack Webhook?"
          confirmationText={deleteSNS.topicArn}
          confirmButtonText="Delete Webhook"
          onConfirm={async () => {
            await remove.mutate(deleteSNS);
            setDeleteSNS(null);
          }}
        />
      )}
    </div>
  );
}
