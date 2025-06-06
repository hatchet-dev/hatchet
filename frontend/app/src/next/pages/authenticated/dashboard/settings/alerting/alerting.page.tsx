import { Separator } from '@/next/components/ui/separator';
import useApiMeta from '@/next/hooks/use-api-meta';
import {
  TenantAlertsProvider,
  useTenantDetailsAlerts,
} from '@/next/hooks/use-tenant-alerts';
import { UpdateTenantAlertingSettings } from './components/update-tenant-alerting-settings-form';
import { SlackWebhook, TenantAlertEmailGroup } from '@/lib/api';
import { useMemo, useState } from 'react';
import { Button } from '@/next/components/ui/button';
import { DataTable } from '@/next/components/ui/data-table';
import { emailGroupColumns } from './components/email-groups-columns';
import { CreateEmailGroupDialog } from './components/create-email-group-dialog';
import { Dialog } from '@/next/components/ui/dialog';
import { DestructiveDialog } from '@/next/components/ui/dialog/index';
import { slackWebhookColumns } from './components/slack-webhooks-columns';
import BasicLayout from '@/next/components/layouts/basic.layout';
import { Headline, PageTitle } from '@/next/components/ui/page-header';

export default function Alerting() {
  const { integrations } = useApiMeta();

  const hasEmailIntegration = integrations?.find((i) => i.name === 'email');
  const hasSlackIntegration = integrations?.find((i) => i.name === 'slack');

  return (
    <BasicLayout>
      <Headline>
        <PageTitle description="Manage your tenant alerting settings">
          Alerting
        </PageTitle>
      </Headline>
      <Separator className="my-4" />
      <TenantAlertsProvider>
        <AlertingSettings />
        {hasEmailIntegration ? (
          <>
            <Separator className="my-4" />
            <EmailGroupsList />
          </>
        ) : null}
        {hasSlackIntegration ? (
          <>
            <Separator className="my-4" />
            <SlackWebhooksList />
          </>
        ) : null}
      </TenantAlertsProvider>
    </BasicLayout>
  );
}

const AlertingSettings: React.FC = () => {
  const {
    data: alertingSettings,
    isLoading,
    update,
  } = useTenantDetailsAlerts();

  return (
    <div>
      <div className="flex items-center space-x-2">
        {alertingSettings ? (
          <UpdateTenantAlertingSettings
            alertingSettings={alertingSettings}
            isLoading={isLoading}
            onSubmit={async (opts) => {
              await update.mutateAsync(opts);
            }}
            fieldErrors={{}}
          />
        ) : null}
      </div>
    </div>
  );
};

function EmailGroupsList() {
  const {
    emailGroups: { list, create, remove },
    data,
    update,
  } = useTenantDetailsAlerts();

  const [showGroupsDialog, setShowGroupsDialog] = useState(false);
  const [deleteEmailGroup, setDeleteEmailGroup] =
    useState<TenantAlertEmailGroup | null>(null);

  const [isAlertMemberEmails, setIsAlertMemberEmails] = useState(
    data?.alertMemberEmails || false,
  );

  const cols = emailGroupColumns({
    onDeleteClick: (row) => {
      setDeleteEmailGroup(row);
    },
    alertTenantEmailsSet: isAlertMemberEmails,
    onToggleMembersClick: async (value) => {
      setIsAlertMemberEmails(value);
      await update.mutateAsync({
        alertMemberEmails: value,
      });
    },
  });

  const groups: TenantAlertEmailGroup[] = useMemo(() => {
    const customGroups = list?.data?.rows || [];

    return [
      {
        // Special group for all tenant members
        emails: [],
        metadata: {
          id: 'default',
          createdAt: 'default',
          updatedAt: 'default',
        },
      },
      ...customGroups,
    ];
  }, [list?.data?.rows]);

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Email Groups
        </h3>
        <Button
          key="create-slack-webhook"
          onClick={() => {
            setShowGroupsDialog(true);
          }}
        >
          Create new group
        </Button>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={list.isLoading}
        columns={cols}
        data={groups}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      {showGroupsDialog ? (
        <Dialog open={showGroupsDialog} onOpenChange={setShowGroupsDialog}>
          <CreateEmailGroupDialog
            isLoading={create.isPending}
            onSubmit={async (data) => {
              await create.mutate(data);
              setShowGroupsDialog(false);
            }}
            // fieldErrors={create}
          />
        </Dialog>
      ) : null}
      {deleteEmailGroup ? (
        <DestructiveDialog
          open={true}
          onOpenChange={() => {}}
          title="Delete email group"
          description="Are you sure you want to delete this email group?"
          confirmationText="delete"
          confirmButtonText="Delete group"
          onConfirm={async () => {
            await remove.mutate(deleteEmailGroup.metadata.id);
            setDeleteEmailGroup(null);
          }}
        />
      ) : null}
    </div>
  );
}

function SlackWebhooksList() {
  const {
    slackWebhooks: { list, startUrl, remove },
  } = useTenantDetailsAlerts();

  const [deleteSlack, setDeleteSlack] = useState<SlackWebhook | null>(null);

  const cols = slackWebhookColumns({
    onDeleteClick: (row) => {
      setDeleteSlack(row);
    },
  });

  return (
    <div>
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Slack Webhooks
        </h3>
        <a href={startUrl}>
          <Button key="create-slack-webhook">Add Slack Webhook</Button>
        </a>
      </div>
      <Separator className="my-4" />
      <DataTable
        isLoading={list.isLoading}
        columns={cols}
        data={list.data?.rows || []}
        filters={[]}
        getRowId={(row) => row.metadata.id}
      />
      {deleteSlack ? (
        <DestructiveDialog
          open={true}
          onOpenChange={() => {}}
          title="Delete Slack Webhook"
          description="Are you sure you want to delete this Slack Webhook?"
          confirmationText="delete"
          confirmButtonText="Delete Webhook"
          onConfirm={async () => {
            await remove.mutate(deleteSlack.metadata.id);
            setDeleteSlack(null);
          }}
        />
      ) : null}
    </div>
  );
}
