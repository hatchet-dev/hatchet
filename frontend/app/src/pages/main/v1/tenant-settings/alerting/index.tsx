import { CreateEmailGroupDialog } from './components/create-email-group-dialog';
import { DeleteEmailGroupForm } from './components/delete-email-group-form';
import { DeleteSlackForm } from './components/delete-slack-form';
import {
  EmailGroupCell,
  EmailGroupStatusCell,
  EmailGroupActions,
} from './components/email-groups-columns';
import { SlackActions } from './components/slack-webhooks-columns';
import { UpdateTenantAlertingSettings } from './components/update-tenant-alerting-settings-form';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import api, {
  CreateTenantAlertEmailGroupRequest,
  SlackWebhook,
  TenantAlertEmailGroup,
  UpdateTenantRequest,
  queries,
} from '@/lib/api';
import { useApiError, useApiMetaIntegrations } from '@/lib/hooks';
import { Dialog } from '@radix-ui/react-dialog';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';

export default function Alerting() {
  const integrations = useApiMetaIntegrations();

  const hasEmailIntegration = integrations?.find((i) => i.name === 'email');
  const hasSlackIntegration = integrations?.find((i) => i.name === 'slack');

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <h2 className="text-2xl font-semibold leading-tight text-foreground">
          Alerting
        </h2>
        <p className="my-4 text-gray-700 dark:text-gray-300">
          Manage alerts to get notified on task failure.
        </p>
        <Separator className="my-4" />
        <AlertingSettings />
        {hasEmailIntegration && <Separator className="my-4" />}
        {hasEmailIntegration && <EmailGroupsList />}
        {hasSlackIntegration && <Separator className="my-4" />}
        {hasSlackIntegration && <SlackWebhooksList />}
      </div>
    </div>
  );
}

const AlertingSettings: React.FC = () => {
  const { tenantId } = useCurrentTenantId();
  const alertingSettings = useQuery({
    ...queries.alertingSettings.get(tenantId),
  });

  const [isLoading, setIsLoading] = useState(false);

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onMutate: () => {
      setIsLoading(true);
    },
    onSuccess: () => {
      setIsLoading(false);
      alertingSettings.refetch();
    },
    onError: handleApiError,
  });

  if (!alertingSettings.data) {
    return <Spinner />;
  }

  return (
    <div>
      <h3 className="text-xl font-semibold leading-tight text-foreground">
        Settings
      </h3>
      <Separator className="my-4" />
      <div className="flex items-center space-x-2">
        <UpdateTenantAlertingSettings
          alertingSettings={alertingSettings.data}
          isLoading={isLoading}
          onSubmit={(opts) => {
            updateMutation.mutate(opts);
          }}
          fieldErrors={{}}
        />
      </div>
    </div>
  );
};

function EmailGroupsList() {
  const { tenant } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();
  const [showGroupsDialog, setShowGroupsDialog] = useState(false);
  const [deleteEmailGroup, setDeleteEmailGroup] =
    useState<TenantAlertEmailGroup | null>(null);

  const [isAlertMemberEmails, setIsAlertMemberEmails] = useState(
    tenant?.alertMemberEmails || false,
  );

  const { handleApiError } = useApiError({});

  const updateMutation = useMutation({
    mutationKey: ['tenant:update'],
    mutationFn: async (data: UpdateTenantRequest) => {
      await api.tenantUpdate(tenantId, data);
    },
    onError: handleApiError,
  });

  const listEmailGroupQuery = useQuery({
    ...queries.emailGroups.list(tenantId),
  });

  const groups: TenantAlertEmailGroup[] = useMemo(() => {
    const customGroups = listEmailGroupQuery.data?.rows || [];

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
  }, [listEmailGroupQuery.data]);

  const emailGroupColumns = useMemo(
    () => [
      {
        columnLabel: 'Emails',
        cellRenderer: (group: TenantAlertEmailGroup) => (
          <EmailGroupCell group={group} />
        ),
      },
      {
        columnLabel: 'Created',
        cellRenderer: (group: TenantAlertEmailGroup) =>
          group.metadata.id != 'default' ? (
            <RelativeDate date={group.metadata.createdAt} />
          ) : (
            <div />
          ),
      },
      {
        columnLabel: '',
        cellRenderer: (group: TenantAlertEmailGroup) => (
          <EmailGroupStatusCell
            group={group}
            alertTenantEmailsSet={isAlertMemberEmails}
          />
        ),
      },
      {
        columnLabel: '',
        cellRenderer: (group: TenantAlertEmailGroup) => (
          <EmailGroupActions
            group={group}
            alertTenantEmailsSet={isAlertMemberEmails}
            onDeleteClick={(group) => {
              setDeleteEmailGroup(group);
            }}
            onToggleMembersClick={(value) => {
              setIsAlertMemberEmails(value);
              updateMutation.mutate({
                alertMemberEmails: value,
              });
            }}
          />
        ),
      },
    ],
    [isAlertMemberEmails, updateMutation],
  );

  return (
    <div>
      <div className="flex flex-row items-center justify-between">
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
      {groups.length > 0 ? (
        <SimpleTable columns={emailGroupColumns} data={groups} />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No email groups found. Create a group to receive alerts via email.
        </div>
      )}
      {showGroupsDialog && (
        <CreateEmailGroup
          onSuccess={() => {
            setShowGroupsDialog(false);
            listEmailGroupQuery.refetch();
          }}
          showGroupDialog={showGroupsDialog}
          setShowGroupDialog={setShowGroupsDialog}
        />
      )}
      {deleteEmailGroup && (
        <DeleteEmailGroup
          emailGroup={deleteEmailGroup}
          setShowEmailGroupDelete={() => setDeleteEmailGroup(null)}
          onSuccess={() => {
            setDeleteEmailGroup(null);
            listEmailGroupQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function CreateEmailGroup({
  showGroupDialog,
  setShowGroupDialog,
  onSuccess,
}: {
  onSuccess: () => void;
  showGroupDialog: boolean;
  setShowGroupDialog: (show: boolean) => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const { handleApiError } = useApiError({
    setFieldErrors: setFieldErrors,
  });

  const createTokenMutation = useMutation({
    mutationKey: ['api-token:create', tenantId],
    mutationFn: async (data: CreateTenantAlertEmailGroupRequest) => {
      const res = await api.alertEmailGroupCreate(tenantId, data);
      return res.data;
    },
    onSuccess: () => {
      onSuccess();
    },
    onError: handleApiError,
  });

  return (
    <Dialog open={showGroupDialog} onOpenChange={setShowGroupDialog}>
      <CreateEmailGroupDialog
        isLoading={createTokenMutation.isPending}
        onSubmit={createTokenMutation.mutate}
        fieldErrors={fieldErrors}
      />
    </Dialog>
  );
}

function DeleteEmailGroup({
  emailGroup,
  setShowEmailGroupDelete,
  onSuccess,
}: {
  emailGroup: TenantAlertEmailGroup;
  setShowEmailGroupDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['alert-email-group:delete', tenantId, emailGroup],
    mutationFn: async () => {
      await api.alertEmailGroupDelete(emailGroup.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!emailGroup} onOpenChange={setShowEmailGroupDelete}>
      <DeleteEmailGroupForm
        emailGroup={emailGroup}
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowEmailGroupDelete(false)}
      />
    </Dialog>
  );
}

function SlackWebhooksList() {
  const { tenantId } = useCurrentTenantId();
  const [deleteSlack, setDeleteSlack] = useState<SlackWebhook | null>(null);

  const listWebhooksQuery = useQuery({
    ...queries.slackWebhooks.list(tenantId),
  });

  const slackColumns = useMemo(
    () => [
      {
        columnLabel: 'Team',
        cellRenderer: (webhook: SlackWebhook) => <div>{webhook.teamName}</div>,
      },
      {
        columnLabel: 'Channel',
        cellRenderer: (webhook: SlackWebhook) => (
          <div>{webhook.channelName}</div>
        ),
      },
      {
        columnLabel: 'Created',
        cellRenderer: (webhook: SlackWebhook) => (
          <RelativeDate date={webhook.metadata.createdAt} />
        ),
      },
      {
        columnLabel: 'Actions',
        cellRenderer: (webhook: SlackWebhook) => (
          <SlackActions
            webhook={webhook}
            onDeleteClick={(webhook) => {
              setDeleteSlack(webhook);
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
          Slack Webhooks
        </h3>
        <a href={'/api/v1/tenants/' + tenantId + '/slack/start'}>
          <Button key="create-slack-webhook">Add Slack Webhook</Button>
        </a>
      </div>
      <Separator className="my-4" />
      {(listWebhooksQuery.data?.rows || []).length > 0 ? (
        <SimpleTable
          columns={slackColumns}
          data={listWebhooksQuery.data?.rows || []}
        />
      ) : (
        <div className="py-8 text-center text-sm text-muted-foreground">
          No Slack webhooks found. Add a webhook to receive alerts in Slack.
        </div>
      )}
      {deleteSlack && (
        <DeleteSlackWebhook
          slackWebhook={deleteSlack}
          setShowSlackWebhookDelete={() => setDeleteSlack(null)}
          onSuccess={() => {
            setDeleteSlack(null);
            listWebhooksQuery.refetch();
          }}
        />
      )}
    </div>
  );
}

function DeleteSlackWebhook({
  slackWebhook,
  setShowSlackWebhookDelete,
  onSuccess,
}: {
  slackWebhook: SlackWebhook;
  setShowSlackWebhookDelete: (show: boolean) => void;
  onSuccess: () => void;
}) {
  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});

  const deleteMutation = useMutation({
    mutationKey: ['slack-webhook:delete', tenantId, slackWebhook],
    mutationFn: async () => {
      await api.slackWebhookDelete(slackWebhook.metadata.id);
    },
    onSuccess: onSuccess,
    onError: handleApiError,
  });

  return (
    <Dialog open={!!slackWebhook} onOpenChange={setShowSlackWebhookDelete}>
      <DeleteSlackForm
        slackWebhook={slackWebhook}
        isLoading={deleteMutation.isPending}
        onSubmit={() => deleteMutation.mutate()}
        onCancel={() => setShowSlackWebhookDelete(false)}
      />
    </Dialog>
  );
}
